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

	// 缓存重复调用的函数结果
	fileCount := stats.GetFileCount()
	dirCount := stats.GetDirCount()
	symlinkCount := stats.GetTotalSymlink()
	totalSize := stats.GetTotalSize()
	averageSizeBytes := totalSize / int64(fileCount)
	averageNameLength := stats.GetAvgNameLength()
	maxNameLength := stats.GetMaxNameLength()
	averageDirDepth := stats.GetAvgDirDepth()
	maxDirDepth := stats.GetMaxDirDepth()

	// Get unique extension count with error handling
	extCount, err := dbInstance.GetUniqueExtCount()
	if err != nil {
		log.Errorf("Failed to get file type count: %v\n", err)
		extCount = 0 // Set default value in case of error
	}

	// 计算总文件数
	totalFiles := fileCount + dirCount

	// 打印空行
	fmt.Println()

	// 同时输出到控制台和日志
	printToConsoleAndLog(" Command    : %s\n", reportConfig.CmdLine)
	printToConsoleAndLog(" Statistic  : count: total(%d) - files(%d), directories(%d), symlinks(%d), file type(%d)\n", totalFiles, fileCount, dirCount, symlinkCount, extCount)
	printToConsoleAndLog("            : capacity: total(%s), average(%s)\n", FormatFileSize(totalSize), FormatFileSize(averageSizeBytes))
	printToConsoleAndLog("            : filename length: average(%d), max(%d)\n", averageNameLength, maxNameLength)
	printToConsoleAndLog("            : dir depth: average(%d), max(%d)\n", averageDirDepth, maxDirDepth)

	printToConsoleAndLog(" Total Time : %s \n", totalTime.Round(time.Second))
	printToConsoleAndLog(" Job ID     : %s \n", reportConfig.JobID)
	printToConsoleAndLog(" Log Path   : %s \n", reportConfig.LogPath)
}
