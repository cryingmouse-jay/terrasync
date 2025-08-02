package db

import (
	"terrasync/object"
)

// DB 定义数据库操作接口
type DB interface {
	// Init 初始化数据库连接
	Init() error

	// SaveObject 保存对象到数据库
	SaveEntry(fileInfo object.FileInfo) error

	// SaveEntries 批量保存多个对象到数据库
	SaveEntries(fileInfos []object.FileInfo) error

	// GetUniqueExtCount 获取数据库中不重复的文件扩展名总数
	GetUniqueExtCount() (int, error)

	// Close 关闭数据库连接
	Close() error
}
