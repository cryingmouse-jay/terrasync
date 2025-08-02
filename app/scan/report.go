package scan

import (
	"fmt"
	"terrasync/db"
	"terrasync/log"
	"time"
)

type KafkaConfig struct {
	Enabled     bool
	Host        string
	Port        int
	Topic       string
	Concurrency int
}

type ReportConfig struct {
	AppVersion  string
	CmdLine     string
	CsvReport   bool
	HtmlReport  bool
	KafkaConfig KafkaConfig
	JobID       string
	LogPath     string
	StartTime   time.Time
	Quiet       bool
}

func GenerateConsoleReportTitle(reportConfig ReportConfig) {
	// Print stats in console
	fmt.Printf("terrasync %s; (c) 2025 LenovoNetapp, Inc.\n\n", reportConfig.AppVersion)
	// print stats into log
	log.Infof("terrasync %s; (c) 2025 LenovoNetapp, Inc.\n\n", reportConfig.AppVersion)

}

// printToConsoleAndLog 同时输出到控制台和日志
func printToConsoleAndLog(format string, args ...interface{}) {
	fmt.Printf(format, args...)
	log.Infof(format, args...)
}

func GenerateConsoleReportSummary(reportConfig ReportConfig, stats Stats, dbInstance db.DB) {
	totalTime := time.Since(reportConfig.StartTime)

	// Get unique extension count with error handling
	extCount, err := dbInstance.GetUniqueExtCount()
	if err != nil {
		log.Errorf("Failed to get file type count: %v\n", err)
		extCount = 0 // Set default value in case of error
	}

	// 打印空行
	fmt.Println()

	// 同时输出到控制台和日志
	printToConsoleAndLog("==================================================================\n")
	printToConsoleAndLog("                          Scan Statistics                         \n")
	printToConsoleAndLog("==================================================================\n\n")

	printToConsoleAndLog("  Command    :    %s\n", reportConfig.CmdLine)
	printToConsoleAndLog("  Total time :    %s\n", totalTime.Round(time.Second))
	printToConsoleAndLog("  Job ID     :    %s\n", reportConfig.JobID)
	printToConsoleAndLog("  Log Path   :    %s\n", reportConfig.LogPath)

	stats.Print()

	printToConsoleAndLog("  File type:        %30d\n", extCount)

	printToConsoleAndLog("\n=================================================================\n")
}
