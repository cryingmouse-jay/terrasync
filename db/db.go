package db

import (
	"database/sql"
	"terrasync/object"
)

// DB 定义数据库操作接口
type DB interface {
	// Init 初始化数据库连接
	CreateTable(name string) error

	// SaveEntries 批量保存多个对象到数据库
	SaveEntries(fileInfos []object.FileInfo, tableName string) error

	// GetUniqueExtCount 获取数据库中不重复的文件扩展名总数
	GetUniqueExtCount() (int, error)

	QueryExactNewFiles(tableName string) []FileInfoData

	QueryChangedFiles(tableName string) []FileInfoData

	// Close 关闭数据库连接
	Close() error

	// Query 执行SQL查询并返回结果行
	Query(query string, args ...interface{}) (*sql.Rows, error)
}
