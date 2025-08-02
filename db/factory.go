package db

import (
	"fmt"
	"sync"
)

// dbFactory 定义数据库工厂函数类型
type dbFactory func(string) (DB, error)

// factories 存储已注册的数据库工厂
var factories = make(map[string]dbFactory)
var factoryMu sync.RWMutex

// RegisterDB 注册数据库工厂
func RegisterDB(dbType string, factory dbFactory) {
	factoryMu.Lock()
	defer factoryMu.Unlock()
	factories[dbType] = factory
}

// NewDB 创建数据库实例
// dbType: 数据库类型
// dsn: 数据源名称
func NewDB(dbType string, dsn string) (DB, error) {
	factoryMu.RLock()
	factory, exists := factories[dbType]
	factoryMu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("unsupported database type: %s", dbType)
	}

	return factory(dsn)
}

// 初始化时注册内置数据库驱动
func init() {
	RegisterDB("sqlite", func(dsn string) (DB, error) {
		return NewSQLiteDB(dsn), nil
	})
}