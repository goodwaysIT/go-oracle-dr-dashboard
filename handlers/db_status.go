package handlers

import (
	"github.com/goodwaysIT/go-oracle-dr-dashboard/models"
	"github.com/goodwaysIT/go-oracle-dr-dashboard/util"
	"log"
	"sync"
	"time"
)

// GetAllDatabaseStatus retrieves the status of all configured databases.
func GetAllDatabaseStatus() []models.DatabaseStatus {
	currentConfig := models.GetConfig()
	statusList := make([]models.DatabaseStatus, len(currentConfig.DBs))
	var wg sync.WaitGroup

	for i, db := range currentConfig.DBs {
		wg.Add(1)
		go func(idx int, dbConfig models.DatabaseConfig) {
			defer wg.Done()
			statusList[idx] = checkDatabaseSystem(dbConfig)
		}(i, db)
	}
	wg.Wait()
	return statusList
}

// checkOracleInstanceDetailed performs a comprehensive check of a single Oracle instance.
// It checks ping, port, DB connection, and gathers DB-specific info.
// instanceType is "Production" or "Disaster Recovery" for logging.
func checkOracleInstanceDetailed(instanceIP string, dbConfig models.DatabaseConfig, instanceType string) models.OracleInstanceStatus {
	res := models.OracleInstanceStatus{
		Role:          "UNKNOWN",
		CurrentStatus: "CHECKING",

		DgDelay:     -1,
		Connections: -1,
	}

	var pingErr, portErr error
	res.IsAlive, pingErr = util.PingHost(instanceIP, 3*time.Second)
	if pingErr != nil {
		log.Printf("Error pinging %s %s (%s): %v", instanceType, dbConfig.Name, instanceIP, pingErr)
	}

	if !res.IsAlive {
		res.CurrentStatus = "OFFLINE"
		return res
	}

	res.PortOpen, portErr = util.CheckTCPPort(instanceIP, dbConfig.Port, 3*time.Second)
	if portErr != nil {
		log.Printf("Error checking port for %s %s (%s:%d): %v", instanceType, dbConfig.Name, instanceIP, dbConfig.Port, portErr)
	}

	if !res.PortOpen {
		res.CurrentStatus = "PORT_ERROR"
		return res
	}

	oraCfg := util.CreateOraUtilConfig(instanceIP, dbConfig)
	oraDB, err := util.NewOracleDB(oraCfg)
	if err != nil {
		log.Printf("Warning: Could not connect to %s database %s (%s:%d): %v", instanceType, dbConfig.Name, instanceIP, dbConfig.Port, err)
		res.CurrentStatus = "DB_CONNECTION_ERROR"
		return res
	}
	res.DbConnected = true
	defer oraDB.Close()

	dbInfo, infoErr := oraDB.GetDatabaseInfo()
	if infoErr != nil {
		log.Printf("Warning: Failed to get %s database info for %s (%s:%d): %v", instanceType, dbConfig.Name, instanceIP, dbConfig.Port, infoErr)
		res.CurrentStatus = "INFO_FETCH_FAILED"
		return res
	}

	role, _ := dbInfo["DATABASE_ROLE"].(string)
	openMode, _ := dbInfo["OPEN_MODE"].(string)

	if role != "" {
		res.Role = role // Return the raw role
	}
	res.CurrentStatus = openMode // Return the raw open_mode

	// Get Lag or Connections based on Open Mode
	if openMode != "READ WRITE" && openMode != "" { // Typically STANDBY or READ ONLY
		delay, lagErr := oraDB.GetADGLag()
		if lagErr != nil {
			log.Printf("Warning: Failed to get ADG lag for %s %s (%s:%d): %v", instanceType, dbConfig.Name, instanceIP, dbConfig.Port, lagErr)
			// DgDelay remains -1
		} else {
			res.DgDelay = delay
		}
	} else if openMode == "READ WRITE" { // Typically PRIMARY
		// Only fetch connections if it's the "Production" instance type,
		// as "Connections" field in DatabaseStatus is for the primary.
		if instanceType == "Production" {
			conns, connErr := oraDB.GetBusinessConnectionCount()
			if connErr != nil {
				log.Printf("Warning: Failed to get business connection count for %s %s (%s:%d): %v", instanceType, dbConfig.Name, instanceIP, dbConfig.Port, connErr)
				// Connections remains -1
			} else {
				res.Connections = conns
			}
		}
	}
	return res
}

// checkDatabaseSystem gets the status of a single database system.
func checkDatabaseSystem(db models.DatabaseConfig) models.DatabaseStatus {
	status := models.DatabaseStatus{
		Name:              db.Name,
		LoadBalancerIP:    db.LBIP,
		ProductionIP:      db.ProdIP,
		DisasterIP:        db.DRIP,
		ProductionDgDelay: -1, // Initialize defaults
		DisasterDgDelay:   -1,
		Connections:       -1,
		ProductionRole:    "UNKNOWN",
		DisasterRole:      "UNKNOWN",
		ProductionStatus:  "CHECKING",
		DisasterStatus:    "CHECKING",
	}

	var wg sync.WaitGroup
	wg.Add(3) // One goroutine for LB, one for Production, one for DR

	// --- Load Balancer Checks ---
	go func() {
		defer wg.Done()
		var pingErr, portErr error
		status.LoadBalancerAlive, pingErr = util.PingHost(db.LBIP, 3*time.Second)
		if pingErr != nil {
			log.Printf("Error pinging Load Balancer %s (%s): %v", db.Name, db.LBIP, pingErr)
		}
		if status.LoadBalancerAlive {
			status.LoadBalancerPort1521, portErr = util.CheckTCPPort(db.LBIP, db.Port, 3*time.Second)
			if portErr != nil {
				log.Printf("Error checking port for Load Balancer %s (%s:%d): %v", db.Name, db.LBIP, db.Port, portErr)
			}
			if status.LoadBalancerPort1521 {
				status.LoadBalancerDbConnect = util.TestConnection(db.LBIP, db.Port, db.Username, db.Password, db.ServiceName)
			}
		}
	}()

	// --- Production Checks ---
	go func() {
		defer wg.Done()
		prodStatus := checkOracleInstanceDetailed(db.ProdIP, db, "Production")
		status.ProductionAlive = prodStatus.IsAlive
		status.ProductionPort1521 = prodStatus.PortOpen
		status.ProductionDbConnect = prodStatus.DbConnected
		status.ProductionStatus = prodStatus.CurrentStatus
		status.ProductionRole = prodStatus.Role
		status.ProductionDgDelay = prodStatus.DgDelay
		if prodStatus.Connections != -1 { // Only update if valid connections count was fetched
			status.Connections = prodStatus.Connections
		}
	}()

	// --- Disaster Recovery Checks ---
	go func() {
		defer wg.Done()
		drStatus := checkOracleInstanceDetailed(db.DRIP, db, "Disaster Recovery")
		status.DisasterAlive = drStatus.IsAlive
		status.DisasterPort1521 = drStatus.PortOpen
		status.DisasterDbConnect = drStatus.DbConnected
		status.DisasterStatus = drStatus.CurrentStatus
		status.DisasterRole = drStatus.Role
		status.DisasterDgDelay = drStatus.DgDelay
		// Connections field is typically not set for DR unless it becomes primary.
	}()

	wg.Wait()
	// Final status refinement is mostly handled within checkOracleInstanceDetailed now.
	// This block can be simplified or removed if statuses are definitive from checkers.
	// For instance, if prodStatus.CurrentStatus is "Offline", status.ProductionStatus will be "Offline".
	return status
}

// createOraUtilConfig is a helper function to create OracleConfig.
func createOraUtilConfig(ip string, dbCfg models.DatabaseConfig) *util.OracleConfig {
	return &util.OracleConfig{
		Host:        ip,
		Port:        dbCfg.Port,
		ServiceName: dbCfg.ServiceName,
		Username:    dbCfg.Username,
		Password:    dbCfg.Password,
		ConnTimeout: 5, // Short timeout for status check connection attempt
		ConnectType: "service_name",
		URLOptions:  make(map[string]string),
	}
} 