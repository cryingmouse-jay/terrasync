package db

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"terrasync/log"
	"terrasync/object"
	"time"

	_ "modernc.org/sqlite"
)

// queryFileInfos 执行文件信息查询并返回结果
// sqlQuery: SQL查询语句
// args: 查询参数
// 返回: 文件信息列表和错误
func (s *SQLiteDB) queryFileInfos(sqlQuery string, args ...interface{}) ([]FileInfoData, error) {
	rows, err := s.db.Query(sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	var results []FileInfoData
	for rows.Next() {
		var path string
		var size int64
		var ext string
		var ctime, mtime, atime time.Time
		var perm int
		var isSymlink, isDir, isRegular bool

		err := rows.Scan(&path, &size, &ext, &ctime, &mtime, &atime, &perm, &isSymlink, &isDir, &isRegular)
		if err != nil {
			log.Errorf("failed to scan file row: %w", err)
			continue
		}

		fileInfo := FileInfoData{
			Key:       path,
			Size:      size,
			Ext:       ext,
			CTime:     ctime,
			MTime:     mtime,
			ATime:     atime,
			Perm:      perm,
			IsSymlink: isSymlink,
			IsDir:     isDir,
			IsRegular: isRegular,
		}

		results = append(results, fileInfo)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error during rows iteration: %w", err)
	}

	return results, nil
}

// SQLiteDB SQLite数据库实现
type SQLiteDB struct {
	db   *sql.DB
	path string
}

// FileInfoData 封装processFileInfo函数返回的文件信息数据
type FileInfoData struct {
	Key       string
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

// ProcessFileInfo 处理文件信息，提取公共逻辑
// 返回：封装了文件信息的数据结构
func ProcessFileInfo(fileInfo object.FileInfo) FileInfoData {
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
		Key:       key,
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
func NewSQLiteDB(path string) (*SQLiteDB, error) {
	sqldb := &SQLiteDB{path: path}
	var err error
	sqldb.db, err = sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}

	return sqldb, nil
}

// Init 初始化SQLite数据库连接
func (s *SQLiteDB) CreateTable(name string) error {
	// 创建表结构
	createTableSQL := fmt.Sprintf(`
CREATE TABLE IF NOT EXISTS %s (
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
);`, name)
	_, err := s.db.Exec(createTableSQL)
	return err
}

// Query 执行SQL查询并返回结果行
func (s *SQLiteDB) Query(query string, args ...interface{}) (*sql.Rows, error) {
	return s.db.Query(query, args...)
}

// SaveEntries 批量保存多个文件信息到数据库
func (s *SQLiteDB) SaveEntries(fileInfos []object.FileInfo, tableName string) error {
	if len(fileInfos) == 0 {
		return nil
	}

	// 如果未指定表名，使用默认表名
	if tableName == "" {
		tableName = "file_entries"
	}

	// 准备批量插入语句
	query := `INSERT INTO ` + tableName + ` (
	path, size, ext, ctime, mtime, atime, perm, is_symlink, is_dir, is_regular_file
	) VALUES `

	// 构建参数和值部分
	params := make([]interface{}, 0, len(fileInfos)*10)
	for i, fileInfo := range fileInfos {
		if i > 0 {
			query += ","
		}
		query += "(?, ?, ?, ?, ?, ?, ?, ?, ?, ?)"

		// 调用公共函数处理文件信息
		fileData := ProcessFileInfo(fileInfo)

		params = append(params,
			fileData.Key, fileData.Size, fileData.Ext, fileData.CTime, fileData.MTime, fileData.ATime, fileData.Perm, fileData.IsSymlink, fileData.IsDir, fileData.IsRegular)
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

func (s *SQLiteDB) QueryExactNewFiles(tableName string) []FileInfoData {
	// 构建SQL查询，查找在临时表中但不在file_entries表中的文件
	sqlQuery := fmt.Sprintf(`
        SELECT t.path, t.size, t.ext, t.ctime, t.mtime, t.atime, t.perm, t.is_symlink, t.is_dir, t.is_regular_file
        FROM %s t
        LEFT JOIN file_entries f ON t.path = f.path
        WHERE f.path IS NULL`, tableName)

	results, err := s.queryFileInfos(sqlQuery)
	if err != nil {
		log.Errorf("Failed to query exact new files: %v", err)
		return []FileInfoData{}
	}

	return results
}

func (s *SQLiteDB) QueryChangedFiles(tableName string) []FileInfoData {
	// 查询变更文件：存在于file_entries表中且ctime/mtime与临时表中不同的文件
	sqlQuery := fmt.Sprintf(`
        SELECT t.path, t.size, t.ext, t.ctime, t.mtime, t.atime, t.perm, t.is_symlink, t.is_dir, t.is_regular_file 
        FROM %s t
        JOIN file_entries f ON t.path = f.path
        WHERE t.ctime != f.ctime 
           OR t.mtime != f.mtime`, tableName)

	results, err := s.queryFileInfos(sqlQuery)
	if err != nil {
		log.Errorf("failed to query changed files: %v", err)
		return []FileInfoData{}
	}

	return results
}

// Close 关闭数据库连接
func (s *SQLiteDB) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}
