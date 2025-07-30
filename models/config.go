package models

import (
	"fmt"
	"io/ioutil"
	"sync"

	"gopkg.in/yaml.v3"
)

var (
	appConfig Config
	configLock = &sync.RWMutex{}
)

// GetConfig returns a thread-safe copy of the current application configuration.
func GetConfig() Config {
	configLock.RLock()
	defer configLock.RUnlock()
	return appConfig
}

// LoadConfig reads the configuration from the specified file and updates the global config.
func LoadConfig(configFile string) error {
	data, err := ioutil.ReadFile(configFile)
	if err != nil {
		return fmt.Errorf("failed to read config file: %v", err)
	}

	var newConfig Config
	err = yaml.Unmarshal(data, &newConfig)
	if err != nil {
		return fmt.Errorf("failed to parse config file: %v", err)
	}

	// Set default values
	if newConfig.Server.RefreshInterval <= 0 {
		newConfig.Server.RefreshInterval = 30 // Default to 30 seconds refresh
	}
	if newConfig.Frontend.DefaultIntervalMs <= 0 {
		newConfig.Frontend.DefaultIntervalMs = 600000 // Default to 10 minutes
	}

	configLock.Lock()
	appConfig = newConfig
	configLock.Unlock()

	return nil
}


// Config defines the overall application configuration structure.
type Config struct {
	Server   ServerConfig     `yaml:"server"`
	Logging  LoggingConfig    `yaml:"logging"`
	DBs      []DatabaseConfig `yaml:"databases"`
	Titles   TitlesConfig     `yaml:"titles"`
	Layout   LayoutConfig     `yaml:"layout"`
	Frontend FrontendSettings `yaml:"frontend"`
}

// LayoutConfig defines layout settings like the number of columns.
type LayoutConfig struct {
	Columns int `yaml:"columns" json:"columns"`
}

// RefreshSlot defines a time period and its corresponding refresh interval.
type RefreshSlot struct {
	StartHour  int `yaml:"start_hour" json:"start_hour"`
	EndHour    int `yaml:"end_hour" json:"end_hour"`
	IntervalMs int `yaml:"interval_ms" json:"interval_ms"`
}

// FrontendSettings holds configurations specific to the frontend.
type FrontendSettings struct {
	LoadBalancerIP    string        `yaml:"load_balancer_ip" json:"load_balancer_ip"`
	RefreshIntervals  []RefreshSlot `yaml:"refresh_intervals" json:"refresh_intervals"`
	DefaultIntervalMs int           `yaml:"default_interval_ms" json:"default_interval_ms"`
}

// TitlesConfig holds the titles for the UI.
type TitlesConfig struct {
	MainTitle      string `yaml:"main_title" json:"main_title"`
	ProdDataCenter string `yaml:"prod_data_center" json:"prod_data_center"`
	DRDataCenter   string `yaml:"dr_data_center" json:"dr_data_center"`
}

// ServerConfig holds server-related settings.
type ServerConfig struct {
	Port            string `yaml:"port"`
	StaticDir       string `yaml:"static_dir"`
	RefreshInterval int    `yaml:"refresh_interval"`
	PublicBasePath  string `yaml:"public_base_path"` // Public base path for reverse proxy setups
}

// LoggingConfig holds logging settings.
type LoggingConfig struct {
	Level      string `yaml:"level"`
	Filename   string `yaml:"filename"`
	MaxSize    int    `yaml:"max_size_mb"`
	MaxBackups int    `yaml:"max_backups"`
	MaxAge     int    `yaml:"max_age_days"`
}

// DatabaseConfig holds the configuration for a single database to monitor.
type DatabaseConfig struct {
	Name        string `yaml:"name"`
	LBIP        string `yaml:"lb_ip"`
	ProdIP      string `yaml:"prod_ip"`
	DRIP        string `yaml:"dr_ip"`
	Port        int    `yaml:"port"`
	ServiceName string `yaml:"service_name"`
	Username    string `yaml:"username"`
	Password    string `yaml:"password"`
} 