package util

import (
	"fmt"
	"github.com/goodwaysIT/go-oracle-dr-dashboard/models"
	"log"
	"os"
)

var Logger *log.Logger

// InitLogger initializes the logger based on the provided logging configuration.
func InitLogger(loggingConfig models.LoggingConfig) error {
	// Ensure logging configuration is available
	if loggingConfig.Filename == "" {
		// Default to stdout if no filename is provided
		Logger = log.New(os.Stdout, "oracle-dashboard: ", log.Ldate|log.Ltime|log.Lshortfile)
		Logger.Println("日志文件名未在配置中指定，将输出到标准输出。")
		return nil
	}

	// Create or open the log file
	logFile, err := os.OpenFile(
		loggingConfig.Filename,
		os.O_CREATE|os.O_WRONLY|os.O_APPEND,
		0666,
	)
	if err != nil {
		return fmt.Errorf("无法打开或创建日志文件 '%s': %v", loggingConfig.Filename, err)
	}

	// Configure the global logger
	Logger = log.New(logFile, "oracle-dashboard: ", log.Ldate|log.Ltime|log.Lshortfile)
	Logger.Println("日志记录器初始化成功。")

	return nil
} 