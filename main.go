package main

import (
	"fmt"
	"os"
	"path/filepath"

	"terrasync/command"
	"terrasync/log"

	"github.com/spf13/cobra"
)

// Application version information
const (
	AppVersion = "3.0.0"
	AppName    = "terrasync"
)

// initLogger initializes the logging system
func initLogger(loglevel string) error {
	// Initialize logger configuration
	loggerConfig := log.Config{
		EnableFile: true,
		MaxSize:    100,  // MB
		MaxBackups: 10,   // Maximum number of backups
		MaxAge:     30,   // Maximum retention days
		Compress:   true, // Compress old logs
	}

	// Validate log level
	allowedLevels := map[string]bool{"debug": true, "info": true}
	fileLogLevel := loglevel
	if !allowedLevels[fileLogLevel] {
		fileLogLevel = "info"
		log.Warnf("Invalid log level specified, using default: %s", fileLogLevel)
	}

	loggerConfig.FileLevel = fileLogLevel
	loggerConfig.EnableConsole = fileLogLevel == "debug"
	if loggerConfig.EnableConsole {
		loggerConfig.ConsoleLevel = fileLogLevel
	}

	// Get executable directory for log path
	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %v", err)
	}
	exeDir := filepath.Dir(exePath)
	loggerConfig.FilePath = filepath.Join(exeDir, "terrasync.log")

	// Initialize logger
	if err := log.InitLogger(loggerConfig); err != nil {
		return fmt.Errorf("failed to initialize logger: %v", err)
	}

	return nil
}

func main() {
	// Create root command
	rootCmd := &cobra.Command{
		Use:     AppName,
		Short:   "Terrasync is a synchronization tool",
		Long:    `Terrasync - A powerful tool for synchronizing and migrating data between different storage systems.`,
		Version: AppVersion,
		Args:    cobra.MinimumNArgs(1),
	}

	// Add global parameters
	rootCmd.PersistentFlags().StringP("loglevel", "l", "info", "file log level (debug, info)")

	// Parse command line parameters to get log level
	rootCmd.ParseFlags(os.Args)
	loglevel, _ := rootCmd.PersistentFlags().GetString("loglevel")

	// Initialize logging system
	if err := initLogger(loglevel); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}

	// Set subcommands
	scanCmd := command.NewScanCommand(AppVersion)
	migrateCmd := command.NewMigrateCommand(AppVersion)

	rootCmd.AddCommand(scanCmd, migrateCmd)

	// Execute command
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
