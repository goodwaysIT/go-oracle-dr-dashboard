package models

// DatabaseStatus represents the status of a single database system.
type DatabaseStatus struct {
	Name                  string `json:"name"`
	LoadBalancerIP        string `json:"load_balancer_ip"`
	LoadBalancerAlive     bool   `json:"load_balancer_alive"`
	LoadBalancerPort1521  bool   `json:"load_balancer_port_1521"`
	LoadBalancerDbConnect bool   `json:"load_balancer_db_connect"`
	Connections           int    `json:"connections"` // Typically for Primary DB
	ProductionIP          string `json:"production_ip"`
	ProductionAlive       bool   `json:"production_alive"`
	ProductionPort1521    bool   `json:"production_port_1521"`
	ProductionDbConnect   bool   `json:"production_db_connect"`
	ProductionStatus      string `json:"production_status"`
	ProductionRole        string `json:"production_role"`
	ProductionDgDelay     int    `json:"production_dgdelay"` // DG Lag in seconds
	DisasterIP            string `json:"disaster_ip"`
	DisasterAlive         bool   `json:"disaster_alive"`
	DisasterPort1521      bool   `json:"disaster_port_1521"`
	DisasterDbConnect     bool   `json:"disaster_db_connect"`
	DisasterStatus        string `json:"disaster_status"`
	DisasterRole          string `json:"disaster_role"`
	DisasterDgDelay       int    `json:"disaster_dgdelay"` // DG Lag in seconds
}

// OracleInstanceStatus holds the detailed status of a single Oracle instance.
// This struct is used internally by checkOracleInstanceDetailed.
type OracleInstanceStatus struct {
	IsAlive       bool
	PortOpen      bool
	DbConnected   bool
	CurrentStatus string
	Role          string
	DgDelay       int
	Connections   int // Only relevant for Primary
} 