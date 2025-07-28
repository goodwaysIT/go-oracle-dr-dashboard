package util

import (
	"database/sql"
	"fmt"
	"github.com/goodwaysIT/go-oracle-dr-dashboard/models"
	"log"
	"strconv"
	"strings"

	go_ora "github.com/sijms/go-ora/v2"
)

// OracleConfig holds Oracle connection parameters.
type OracleConfig struct {
	Host        string
	Port        int
	ServiceName string
	Username    string
	Password    string
	ConnTimeout int               // Connection timeout in seconds
	ConnectType string            // "service_name" or "sid"
	URLOptions  map[string]string // For additional URL parameters
}

// OracleDB wraps a *sql.DB connection and provides Oracle-specific methods.
type OracleDB struct {
	db  *sql.DB
	cfg *OracleConfig
}

// NewOracleDB creates a new OracleDB instance and establishes a connection.
func NewOracleDB(cfg *OracleConfig) (*OracleDB, error) {
	if cfg == nil {
		return nil, fmt.Errorf("database configuration cannot be nil")
	}

	dsn := fmt.Sprintf("oracle://%s:%s@%s:%d/%s",
		cfg.Username,
		cfg.Password,
		cfg.Host,
		cfg.Port,
		cfg.ServiceName,
	)

	if len(cfg.URLOptions) > 0 {
		var opts []string
		for k, v := range cfg.URLOptions {
			opts = append(opts, fmt.Sprintf("%s=%s", k, v))
		}
		dsn += "?" + strings.Join(opts, "&")
	}

	if cfg.ConnTimeout > 0 {
		if strings.Contains(dsn, "?") {
			dsn += "&"
		} else {
			dsn += "?"
		}
		dsn += fmt.Sprintf("connection timeout=%d", cfg.ConnTimeout)
	}

	db, err := sql.Open("oracle", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}

	if err = db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w (DSN: %s)", err, dsn)
	}

	return &OracleDB{db: db, cfg: cfg}, nil
}

// Close closes the database connection.
func (o *OracleDB) Close() error {
	if o.db != nil {
		return o.db.Close()
	}
	return nil
}

// TestConnection provides a simple way to check if a connection can be established.
func TestConnection(host string, port int, user, password, serviceName string) bool {
	connStr := go_ora.BuildUrl(host, port, serviceName, user, password, nil)
	db, err := sql.Open("oracle", connStr)
	if err != nil {
		return false
	}
	defer db.Close()
	err = db.Ping()
	return err == nil
}

// GetDatabaseInfo retrieves basic database information like role and open mode.
func (o *OracleDB) GetDatabaseInfo() (map[string]interface{}, error) {
	query := "SELECT DATABASE_ROLE, OPEN_MODE FROM V$DATABASE"
	row := o.db.QueryRow(query)

	var databaseRole, openMode string
	err := row.Scan(&databaseRole, &openMode)
	if err != nil {
		return nil, fmt.Errorf("failed to query V$DATABASE: %w", err)
	}

	return map[string]interface{}{
		"DATABASE_ROLE": databaseRole,
		"OPEN_MODE":     openMode,
	}, nil
}

// GetADGLag retrieves ADG (Active Data Guard) transport and apply lag, then sums them in seconds.
func (o *OracleDB) GetADGLag() (int, error) {
	query := `
        SELECT name, value
		FROM V$DATAGUARD_STATS
		WHERE name IN ('apply lag', 'transport lag')`

	results, err := o.db.Query(query)
	if err != nil {
		return 0, fmt.Errorf("failed to query v$dataguard_stats: %w", err)
	}
	defer results.Close()

	var transportLagStr, applyLagStr string
	foundTransportLag, foundApplyLag := false, false

	cols, err := results.Columns()
	if err != nil {
		return 0, fmt.Errorf("failed to get columns from v$dataguard_stats: %w", err)
	}

	rowCount := 0
	for results.Next() {
		rowCount++
		values := make([]interface{}, len(cols))
		valuePtrs := make([]interface{}, len(cols))
		for i := range cols {
			valuePtrs[i] = &values[i]
		}

		if err := results.Scan(valuePtrs...); err != nil {
			return 0, fmt.Errorf("failed to scan v$dataguard_stats row: %w", err)
		}

		var name, valueString string
		if len(values) < 2 {
			return 0, fmt.Errorf("unexpected number of columns in v$dataguard_stats result, expected at least 2, got %d", len(values))
		}

		switch v := values[0].(type) {
		case []byte:
			name = string(v)
		case string:
			name = v
		case nil:
			return 0, fmt.Errorf("NAME column is NULL, which is unexpected for V$DATAGUARD_STATS")
		default:
			return 0, fmt.Errorf("NAME column has unexpected type: %T", values[0])
		}

		if values[1] == nil {
			log.Printf("Trace: VALUE column is NULL for NAME='%s'. Treating as zero lag.", name)
			valueString = ""
		} else {
			switch v := values[1].(type) {
			case []byte:
				valueString = string(v)
			case string:
				valueString = v
			default:
				return 0, fmt.Errorf("VALUE column for NAME='%s' has unexpected type: %T", name, values[1])
			}
		}

		switch name {
		case "transport lag":
			transportLagStr = valueString
			foundTransportLag = true
		case "apply lag":
			applyLagStr = valueString
			foundApplyLag = true
		}
	}
	if err = results.Err(); err != nil {
		return 0, fmt.Errorf("error iterating v$dataguard_stats results: %w", err)
	}

	if rowCount == 0 {
		return -1, nil
	}

	if !foundTransportLag && !foundApplyLag {
		log.Printf("Warning: Neither transport lag nor apply lag found in V$DATAGUARD_STATS, though rows were present. This is unexpected.")
		return -1, nil
	}

	transportLagSeconds, err := parseLag(transportLagStr)
	if err != nil {
		return 0, fmt.Errorf("failed to parse transport lag ('%s'): %w", transportLagStr, err)
	}

	applyLagSeconds, err := parseLag(applyLagStr)
	if err != nil {
		return 0, fmt.Errorf("failed to parse apply lag ('%s'): %w", applyLagStr, err)
	}

	totalLagSeconds := transportLagSeconds + applyLagSeconds

	return totalLagSeconds, nil
}

func parseLag(lag string) (int, error) {
	lag = strings.TrimSpace(lag)

	if lag == "" || lag == "+00 00:00:00" {
		return 0, nil
	}

	if !strings.HasPrefix(lag, "+") {
		return 0, fmt.Errorf("invalid lag format: expected '+DD HH:MI:SS', got '%s'", lag)
	}

	parts := strings.Split(lag[1:], " ")
	if len(parts) != 2 {
		return 0, fmt.Errorf("invalid lag format structure (days part): '%s'", lag)
	}

	days, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, fmt.Errorf("invalid days value in lag '%s': %w", parts[0], err)
	}

	timeParts := strings.Split(parts[1], ":")
	if len(timeParts) != 3 {
		return 0, fmt.Errorf("invalid time format structure in lag '%s': '%s'", lag, parts[1])
	}

	hours, err := strconv.Atoi(timeParts[0])
	if err != nil {
		return 0, fmt.Errorf("invalid hours value in lag '%s': %w", timeParts[0], err)
	}

	minutes, err := strconv.Atoi(timeParts[1])
	if err != nil {
		return 0, fmt.Errorf("invalid minutes value in lag '%s': %w", timeParts[1], err)
	}

	seconds, err := strconv.Atoi(timeParts[2])
	if err != nil {
		return 0, fmt.Errorf("invalid seconds value in lag '%s': %w", timeParts[2], err)
	}

	totalSeconds := (days * 24 * 3600) + (hours * 3600) + (minutes * 60) + seconds
	return totalSeconds, nil
}

// GetBusinessConnectionCount retrieves the count of non-background connections.
func (o *OracleDB) GetBusinessConnectionCount() (int, error) {
	query := "SELECT COUNT(*) FROM V$SESSION WHERE TYPE != 'BACKGROUND' AND STATUS = 'ACTIVE'"
	var count int
	err := o.db.QueryRow(query).Scan(&count)
	if err != nil {
		return -1, fmt.Errorf("failed to query business connection count: %w", err)
	}
	return count, nil
}

func CreateOraUtilConfig(ip string, dbCfg models.DatabaseConfig) *OracleConfig {
	return &OracleConfig{
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