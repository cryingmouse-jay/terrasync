package db

import (
	"database/sql"
	"path/filepath"
	"terrasync/object"
	"time"

	_ "modernc.org/sqlite"
)

// SQLiteDB SQLite数据库实现
type SQLiteDB struct {
	db   *sql.DB
	path string
}

// FileInfoData 封装processFileInfo函数返回的文件信息数据
type FileInfoData struct {
	Size      int64
	Ext       string
	CTime     time.Time
	MTime     time.Time
	ATime     time.Time
	Perm      int
	IsSymlink bool
	IsDir     bool
	IsRegular bool
}

// processFileInfo 处理文件信息，提取公共逻辑
// 返回：封装了文件信息的数据结构
func processFileInfo(fileInfo object.FileInfo) FileInfoData {
	key := fileInfo.Key()
	isDir := fileInfo.IsDir()

	// 获取文件扩展名，如果是目录则为空
	var ext string
	if !isDir {
		ext = filepath.Ext(key)
	}

	// 提取其他属性
	size := fileInfo.Size()
	mtime := fileInfo.MTime()
	atime := fileInfo.ATime()
	ctime := fileInfo.CTime()
	isSymlink := fileInfo.IsSymlink()
	perm := fileInfo.Perm()
	isRegular := fileInfo.IsRegular()

	return FileInfoData{
		Size:      size,
		Ext:       ext,
		CTime:     ctime,
		MTime:     mtime,
		ATime:     atime,
		Perm:      int(perm),
		IsSymlink: isSymlink,
		IsDir:     isDir,
		IsRegular: isRegular,
	}
}

// NewSQLiteDB 创建SQLite数据库实例
func NewSQLiteDB(path string) *SQLiteDB {
	return &SQLiteDB{path: path}
}

// Init 初始化SQLite数据库连接
func (s *SQLiteDB) Init() error {
	var err error
	s.db, err = sql.Open("sqlite", s.path)
	if err != nil {
		return err
	}

	// 创建表结构
	createTableSQL := `
CREATE TABLE IF NOT EXISTS file_entries (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	path TEXT NOT NULL,
	size INTEGER,
	ext TEXT,
	ctime DATETIME,
	mtime DATETIME,
	atime DATETIME,
	perm INTEGER,
	is_symlink INTEGER,
	is_dir INTEGER,
	is_regular_file INTEGER
);`
	_, err = s.db.Exec(createTableSQL)
	return err
}

// SaveEntry 保存对象到SQLite数据库
func (s *SQLiteDB) SaveEntry(fileInfo object.FileInfo) error {
	key := fileInfo.Key()

	// 调用公共函数处理文件信息
	fileData := processFileInfo(fileInfo)

	_, err := s.db.Exec(`
INSERT INTO file_entries (
	path, size, ext, ctime, mtime, atime, perm, is_symlink, is_dir, is_regular_file
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		key, fileData.Size, fileData.Ext, fileData.CTime, fileData.MTime, fileData.ATime, fileData.Perm, fileData.IsSymlink, fileData.IsDir, fileData.IsRegular)
	return err
}

// SaveEntries 批量保存多个文件信息到数据库
func (s *SQLiteDB) SaveEntries(fileInfos []object.FileInfo) error {
	if len(fileInfos) == 0 {
		return nil
	}

	// 准备批量插入语句
	query := `INSERT INTO file_entries (
	path, size, ext, ctime, mtime, atime, perm, is_symlink, is_dir, is_regular_file
	) VALUES `

	// 构建参数和值部分
	params := make([]interface{}, 0, len(fileInfos)*10)
	for i, fileInfo := range fileInfos {
		if i > 0 {
			query += ","
		}
		query += "(?, ?, ?, ?, ?, ?, ?, ?, ?, ?)"

		key := fileInfo.Key()

		// 调用公共函数处理文件信息
		fileData := processFileInfo(fileInfo)

		params = append(params,
			key, fileData.Size, fileData.Ext, fileData.CTime, fileData.MTime, fileData.ATime, fileData.Perm, fileData.IsSymlink, fileData.IsDir, fileData.IsRegular)
	}

	// 执行批量插入
	_, err := s.db.Exec(query, params...)
	return err
}

// GetUniqueExtCount 获取数据库中不重复的文件扩展名总数
func (s *SQLiteDB) GetUniqueExtCount() (int, error) {
	var count int
	err := s.db.QueryRow("SELECT COUNT(DISTINCT ext) FROM file_entries").Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}

// Close 关闭数据库连接
func (s *SQLiteDB) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}
