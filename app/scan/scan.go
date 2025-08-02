package scan

import (
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"terrasync/db"
	"terrasync/log"
	"terrasync/object"
	"time"

	"github.com/bits-and-blooms/bloom/v3"
	"github.com/google/uuid"
)

const (
	listQueueLen    = 8192
	listDirQueueLen = 1024
)

// ScanConfig 扫描配置选项
type ScanConfig struct {
	IncrementalScan bool
	JobDir          string
	DbType          string
	DBBatchSize     int
	Path            string
	Concurrency     int // 并发worker数量
	Depth           int
	Match           []string
	Exclude         []string
	Timeout         time.Duration // 扫描超时时间
}

func Start(scanConfig ScanConfig, reportConfig ReportConfig) error {
	// 设置默认并发数和超时时间
	if scanConfig.Concurrency <= 0 {
		scanConfig.Concurrency = 5
	}

	storage, err := object.CreateStorage(scanConfig.Path)
	if err != nil {
		return fmt.Errorf("failed to create storage: %w", err)
	}
	defer storage.Close()

	// Create match conditions filter
	matchConditions, err := NewConditionFilter(scanConfig.Match)
	if err != nil {
		return fmt.Errorf("failed to create match conditions: %w", err)
	}

	// Create exclude conditions filter
	excludeConditions, err := NewConditionFilter(scanConfig.Exclude)
	if err != nil {
		return fmt.Errorf("failed to create exclude conditions: %w", err)
	}

	GenerateConsoleReportTitle(reportConfig)

	// 开始扫描并应用过滤
	scannedChan := ListAll(storage, scanConfig.Concurrency, scanConfig.Depth, matchConditions, excludeConditions)

	if scanConfig.IncrementalScan {
		// 增量扫描场景,处理文件统计信息
		ProcessFilesForIncrementalScan(scanConfig, scannedChan, reportConfig)
	} else {
		// 全量扫描场景,处理文件统计信息
		if err := ProcessFilesForFullScan(scanConfig, scannedChan, reportConfig); err != nil {
			return fmt.Errorf("failed to process files: %w", err)
		}
	}

	return nil
}

// ListAll recursively lists all files and directories in the given storage starting
// with the specified concurrency level
// ListAll recursively lists all files and directories in the given storage starting
// with the specified concurrency level and depth limit
func ListAll(storage object.Storage, concurrency int, depth int, matchConditions, excludeConditions *ConditionFilter) <-chan object.FileInfo {
	// 定义包含路径和深度信息的结构体

	type dirInfo struct {
		path  string
		depth int
	}

	dirs := make(chan dirInfo, listDirQueueLen)
	results := make(chan object.FileInfo, listQueueLen)
	var wg sync.WaitGroup
	var pending int64

	// list processes a single directory, sending files to results and subdirectories to dirs
	// currentDepth is the depth of the current directory relative to the root
	list := func(dir string, currentDepth int) error {
		// 检查深度限制
		if depth > 0 && currentDepth > depth {
			return nil
		}

		queue, err := storage.List(dir)
		if err != nil {
			return fmt.Errorf("storage list failed: %w", err)
		}

		var subdirs []dirInfo
		for o := range queue {
			// Apply match and exclude filters
			// 当matchConditions为空时默认匹配，excludeConditions为空时默认不匹配
			matchOk := len(matchConditions.conditions) == 0 || matchConditions.IsSatisfied(o)
			excludeOk := len(excludeConditions.conditions) > 0 && excludeConditions.IsSatisfied(o)
			if matchOk && !excludeOk {
				results <- o
			}
			if o.IsDir() {
				subdirs = append(subdirs, dirInfo{path: o.Key(), depth: currentDepth + 1})
			}
		}

		// Add subdirectories to the queue
		if len(subdirs) > 0 {
			atomic.AddInt64(&pending, int64(len(subdirs)))
			go func() {
				for _, d := range subdirs {
					dirs <- d
				}
			}()
		}

		return nil
	}

	// worker processes directories from the dirs channel
	worker := func() {
		defer wg.Done()
		for dirInfo := range dirs {
			if err := list(dirInfo.path, dirInfo.depth); err != nil {
				log.Errorf("Scan error: %v", err)
			}

			// Decrement pending count and close dirs channel if all done
			if atomic.AddInt64(&pending, -1) == 0 {
				close(dirs)
			}
		}
	}

	// Start worker goroutines
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go worker()
	}

	// Add the initial directory to the queue
	atomic.AddInt64(&pending, 1)
	dirs <- dirInfo{path: "/", depth: 1}

	// Start a goroutine to close channels when done
	go func() {
		wg.Wait()
		close(results)
	}()

	return results
}

// ProcessFilesForFullScan 处理文件统计信息并分发到数据库和Kafka
func ProcessFilesForFullScan(scanConfig ScanConfig, scannedChan <-chan object.FileInfo, reportConfig ReportConfig) error {
	// Initialize database
	dbInstance, err := InitDatabase(scanConfig.DbType, scanConfig.JobDir)
	if err != nil {
		log.Errorf("Failed to initialize database: %v", err)
		return fmt.Errorf("failed to initialize database: %w", err)
	}
	defer func() {
		closeStartTime := time.Now()
		if err = (*dbInstance).Close(); err != nil {
			log.Errorf("Error closing database: %v", err)
		} else {
			log.Infof("Database closed successfully in %v", time.Since(closeStartTime))
		}
	}()

	// 初始化Kafka生产者
	var kafkaProducer *KafkaProducer
	if reportConfig.KafkaConfig.Enabled {
		kafkaProducer, err = InitKafkaProducer(reportConfig.KafkaConfig)
		if err != nil {
			kafkaProducer = nil
		}
		if kafkaProducer != nil {
			defer kafkaProducer.Close()
		}
	}

	// 创建统计信息实例
	stats := NewStats()

	// 创建两个通道: 一个给数据库，一个给Kafka
	// 数据库批量处理相关变量
	batchSize := scanConfig.DBBatchSize
	dbChan := make(chan object.FileInfo, batchSize)

	var kafkaChan chan object.FileInfo
	if kafkaProducer != nil {
		kafkaChan = make(chan object.FileInfo, reportConfig.KafkaConfig.Concurrency)
	}

	// 启动数据库批量处理goroutine
	var dbWg sync.WaitGroup
	dbWg.Add(1)
	go func() {
		defer dbWg.Done()
		var buffer []object.FileInfo
		var totalSaved int // 统计总共保存的记录数
		for fileInfo := range dbChan {
			buffer = append(buffer, fileInfo)
			bufferLen := len(buffer)
			if bufferLen >= batchSize {
				startTime := time.Now()
				if err := (*dbInstance).SaveEntries(buffer, ""); err != nil {
					log.Errorf("Failed to save batch: %v", err)
				} else {
					log.Debugf("Saved batch of %d entries in %v", bufferLen, time.Since(startTime))
					totalSaved += bufferLen
				}
				buffer = make([]object.FileInfo, 0, batchSize)
			}
		}
		// 处理剩余数据
		bufferLen := len(buffer)
		if bufferLen > 0 {
			if err := (*dbInstance).SaveEntries(buffer, ""); err != nil {
				log.Errorf("Failed to save final batch: %v", err)
			} else {
				log.Debugf("Saved final batch of %d entries", bufferLen)
				totalSaved += bufferLen
			}
		}
		// 记录总共保存的记录数
		log.Infof("Successfully saved total %d entries to database", totalSaved)
	}()

	// Kafka处理相关变量
	var kafkaWg sync.WaitGroup

	// 启动Kafka消费者goroutine
	if kafkaProducer != nil {
		kafkaWorkerPool := make(chan struct{}, reportConfig.KafkaConfig.Concurrency)
		kafkaWg.Add(1)
		go func() {
			defer kafkaWg.Done()
			for fileInfo := range kafkaChan {
				kafkaWg.Add(1)
				kafkaWorkerPool <- struct{}{}

				go func(fi object.FileInfo) {
					defer kafkaWg.Done()
					defer func() { <-kafkaWorkerPool }()

					kafkaStartTime := time.Now()
					if err := kafkaProducer.SendMessage(reportConfig.KafkaConfig.Topic, fi); err != nil {
						log.Errorf("Kafka error: %v", err)
					} else {
						log.Debugf("Sent message to Kafka topic %s in %v", reportConfig.KafkaConfig.Topic, time.Since(kafkaStartTime))
					}
				}(fileInfo)
			}
		}()
	}

	// 从fileChan读取数据并分发到两个通道
	var fileWg sync.WaitGroup
	fileWg.Add(1)
	go func() {
		defer fileWg.Done()
		for fileInfo := range scannedChan {
			// 打印文件路径
			fileePath := filepath.Join(scanConfig.Path, fileInfo.Key())
			if reportConfig.Quiet {
				log.Infof("Found: %s\n", fileePath)
			} else {
				fmt.Printf("Found: %s\n", fileePath)
			}

			// 分发到两个通道
			dbChan <- fileInfo
			if kafkaProducer != nil && reportConfig.KafkaConfig.Topic != "" {
				kafkaChan <- fileInfo
			}

			// 更新统计信息
			stats.Update(fileInfo)
		}

		// 关闭通道，通知消费者goroutine结束
		close(dbChan)
		if kafkaProducer != nil {
			close(kafkaChan)
		}
	}()

	// 等待所有goroutine完成
	fileWg.Wait()
	dbWg.Wait()
	if kafkaProducer != nil {
		kafkaWg.Wait()
	}

	GenerateConsoleReportSummary(reportConfig, *stats, dbInstance)

	return nil
}

// ProcessFilesForIncrementalScan 处理文件统计信息并分发到数据库和Kafka
// ProcessFilesForIncrementalScan 处理增量扫描的文件统计信息并分发到数据库
func ProcessFilesForIncrementalScan(scanConfig ScanConfig, scannedChan <-chan object.FileInfo, reportConfig ReportConfig) (<-chan db.FileInfoData, <-chan db.FileInfoData, error) {
	dbInstance, err := NewDB(scanConfig.DbType, scanConfig.JobDir)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create database instance: %w", err)
	}

	bloomNewFiles, candidateChan := bloomPreFilter(scannedChan, dbInstance)

	tempTableName := "temp_files_" + strings.Replace(uuid.New().String(), "-", "_", -1)

	(*dbInstance).CreateTable(tempTableName)
	loadCandidatesToTemp(candidateChan, dbInstance, tempTableName, scanConfig)

	// 阶段3：联合查询识别变更
	exactNewFiles := (*dbInstance).QueryExactNewFiles(tempTableName)
	changedFiles := (*dbInstance).QueryChangedFiles(tempTableName)

	// 创建通道
	newFileChan := make(chan db.FileInfoData, len(bloomNewFiles)+len(exactNewFiles))
	changedFileChan := make(chan db.FileInfoData, len(changedFiles))

	// 发送新文件到通道
	for _, file := range bloomNewFiles {
		newFileChan <- db.ProcessFileInfo(file)
	}
	for _, file := range exactNewFiles {
		newFileChan <- file
	}
	close(newFileChan)

	// 发送变更文件到通道
	for _, file := range changedFiles {
		changedFileChan <- file
	}
	close(changedFileChan)

	return newFileChan, changedFileChan, nil
}

// bloomPreFilter 使用布隆过滤器预筛选文件路径
// scannedChan: 扫描到的文件通道
// dbInstance: 数据库实例
// 返回值: [确认的新文件列表, 需要进一步验证的候选文件通道]
func bloomPreFilter(scannedChan <-chan object.FileInfo, dbInstance *db.DB) ([]object.FileInfo, chan object.FileInfo) {
	// 初始化布隆过滤器 (1亿数据，0.1%误判率约需143MB内存)
	filter := bloom.NewWithEstimates(1e8, 0.001)

	// 预热过滤器：加载数据库现有路径
	rows, _ := (*dbInstance).Query("SELECT path FROM file_entries")
	for rows.Next() {
		var path string
		rows.Scan(&path)
		filter.AddString(path)
	}
	// 检查遍历过程中是否有错误
	if err := rows.Err(); err != nil {
		log.Errorf("Error reading rows: %v", err)
	}
	rows.Close()

	// 双通道输出
	newFilesChan := make(chan object.FileInfo, 10000)
	candidateChan := make(chan object.FileInfo, 100000)

	// 使用sync.WaitGroup等待goroutine完成
	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		for item := range scannedChan {
			if !filter.TestString(item.Key()) {
				// 布隆过滤器确认的新增文件
				newFilesChan <- item
			} else {
				// 需要进一步验证的候选文件
				candidateChan <- item
			}
		}
		close(newFilesChan)
		close(candidateChan)
	}()

	// 收集新文件到切片
	newFiles := make([]object.FileInfo, 0, 10000)
	var collectWg sync.WaitGroup
	collectWg.Add(1)
	go func() {
		defer collectWg.Done()
		for file := range newFilesChan {
			newFiles = append(newFiles, file)
		}
	}()

	// 等待处理完成
	wg.Wait()
	collectWg.Wait()

	return newFiles, candidateChan
}

// loadCandidatesToTemp 将候选文件加载到临时表中
// ctx: 用于控制超时的上下文
// candidateChan: 候选文件通道
// dbInstance: 数据库实例
// tableName: 临时表名称
// scanConfig: 扫描配置
func loadCandidatesToTemp(candidateChan <-chan object.FileInfo, dbInstance *db.DB, tableName string, scanConfig ScanConfig) {
	var buffer []object.FileInfo
	var totalSaved int // 统计总共保存的记录数
	for fileInfo := range candidateChan {
		buffer = append(buffer, fileInfo)
		bufferLen := len(buffer)
		if bufferLen >= scanConfig.DBBatchSize {
			startTime := time.Now()
			if err := (*dbInstance).SaveEntries(buffer, tableName); err != nil {
				log.Errorf("Failed to save batch: %v", err)
			} else {
				log.Debugf("Saved batch of %d entries in %v", bufferLen, time.Since(startTime))
				totalSaved += bufferLen
			}
			buffer = make([]object.FileInfo, 0, scanConfig.DBBatchSize)
		}
	}
	// 处理剩余数据
	bufferLen := len(buffer)
	if bufferLen > 0 {
		if err := (*dbInstance).SaveEntries(buffer, tableName); err != nil {
			log.Errorf("Failed to save final batch: %v", err)
		} else {
			log.Debugf("Saved final batch of %d entries", bufferLen)
			totalSaved += bufferLen
		}
	}
	// 记录总共保存的记录数
	log.Infof("Successfully saved total %d entries to database", totalSaved)

}
