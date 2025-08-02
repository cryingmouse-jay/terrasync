package scan

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"terrasync/db"
	"terrasync/log"
	"time"
)

// parseConditions parses and cleans condition lists
func ParseConditions(input string) []string {
	// Split conditions using "and"
	conditions := strings.Split(strings.ToLower(input), "and")
	// Clean whitespace around each condition and filter empty conditions
	filteredConditions := []string{}
	for _, cond := range conditions {
		trimmed := strings.TrimSpace(cond)
		if trimmed != "" {
			filteredConditions = append(filteredConditions, trimmed)
		}
	}
	return filteredConditions
}

// initDatabase initializes the database connection
func InitDatabase(dbType, jobsDir string) (db.DB, error) {
	dbPath := filepath.Join(jobsDir, "index.db")
	// Create database directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		log.Errorf("failed to create database directory: %w", err)
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	dbInstance, err := db.NewDB(dbType, dbPath)
	if err != nil {
		log.Errorf("failed to create database instance: %w", err)
		return nil, fmt.Errorf("failed to create database instance: %w", err)
	}

	if err := dbInstance.Init(); err != nil {
		log.Errorf("failed to initialize database: %w", err)
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	return dbInstance, nil
}

// InitKafkaProducer 初始化Kafka生产者
func InitKafkaProducer(kafkaConfig KafkaConfig) (*KafkaProducer, error) {
	if !kafkaConfig.Enabled {
		log.Info("Kafka is disabled, skipping initialization")
		return nil, nil
	}

	if kafkaConfig.Host == "" || kafkaConfig.Port <= 0 {
		return nil, fmt.Errorf("invalid Kafka configuration: host=%s, port=%d", kafkaConfig.Host, kafkaConfig.Port)
	}

	// 使用传入的host和port构建Kafka brokers地址
	brokerAddr := fmt.Sprintf("%s:%d", kafkaConfig.Host, kafkaConfig.Port)
	brokers := []string{brokerAddr}
	log.Infof("Connecting to Kafka brokers: %v", brokers)

	// 创建Kafka生产者
	startTime := time.Now()
	producer, err := NewKafkaProducer(brokers)
	if err != nil {
		log.Errorf("Failed to create Kafka producer after %v: %v", time.Since(startTime), err)
		return nil, fmt.Errorf("failed to create Kafka producer: %w", err)
	}

	log.Infof("Successfully created Kafka producer in %v", time.Since(startTime))
	return producer, nil
}

// FormatFileSize 将字节数转换为最适合的单位（B、KiB、MiB、GiB或TiB）
func FormatFileSize(bytes int64) string {
	const (
		_   = iota
		KiB = 1 << (10 * iota)
		MiB
		GiB
		TiB
	)

	switch {
	case bytes >= TiB:
		return fmt.Sprintf("%.2f TiB", float64(bytes)/TiB)
	case bytes >= GiB:
		return fmt.Sprintf("%.2f GiB", float64(bytes)/GiB)
	case bytes >= MiB:
		return fmt.Sprintf("%.2f MiB", float64(bytes)/MiB)
	case bytes >= KiB:
		return fmt.Sprintf("%.2f KiB", float64(bytes)/KiB)
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}
