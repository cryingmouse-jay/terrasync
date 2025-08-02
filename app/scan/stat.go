package scan

import (
	"fmt"
	"path/filepath"
	"strings"
	"sync/atomic"
	"terrasync/object"
)

// Stats 存储扫描统计信息
type Stats struct {
	fileCount        int64
	dirCount         int64
	totalSize        int64
	totalSymlink     int64
	totalRegularFile int64
	avgNameLength    float64 // 平均文件名长度
	maxNameLength    int     // 最大文件名长度
	avgDirDepth      float64 // 平均目录深度
	maxDirDepth      int     // 最大目录深度
}

// NewStats 创建一个新的统计实例
func NewStats() *Stats {
	return &Stats{}
}

// Update 根据文件信息更新统计数据
func (s *Stats) Update(fileInfo object.FileInfo) {
	// 获取文件路径
	key := fileInfo.Key()

	if fileInfo.IsDir() {
		atomic.AddInt64(&s.dirCount, 1)
	} else {
		// 使用filepath获取文件名并计算长度
		name := filepath.Base(key)
		nameLength := len(name)

		path := filepath.Dir(key)
		// 计算目录分隔符数量作为深度
		depth := strings.Count(path, string(filepath.Separator))
		// 对于根目录，深度为0
		if path == string(filepath.Separator) || path == "." {
			depth = 0
		}

		atomic.AddInt64(&s.fileCount, 1)
		atomic.AddInt64(&s.totalSize, fileInfo.Size())

		// 更新文件名统计信息
		currentCount := atomic.LoadInt64(&s.fileCount)
		if currentCount > 1 {
			// 增量更新平均文件名长度
			oldAvg := float64(s.avgNameLength)
			s.avgNameLength = (oldAvg*float64(currentCount-1) + float64(nameLength)) / float64(currentCount)

			// 增量更新平均目录深度
			oldDepthAvg := float64(s.avgDirDepth)
			s.avgDirDepth = (oldDepthAvg*float64(currentCount-1) + float64(depth)) / float64(currentCount)
		} else {
			// 如果是第一个文件，直接设置
			s.avgNameLength = float64(nameLength)
			s.avgDirDepth = float64(depth)
		}

		// 更新最大文件名长度
		if nameLength > s.maxNameLength {
			s.maxNameLength = nameLength
		}

		// 更新最大目录深度
		if depth > s.maxDirDepth {
			s.maxDirDepth = depth
		}

		if fileInfo.IsRegular() {
			atomic.AddInt64(&s.totalRegularFile, 1)
		}
	}
	// 统计符号链接和普通文件
	if fileInfo.IsSymlink() {
		atomic.AddInt64(&s.totalSymlink, 1)
	}
}

// GetFileCount 返回文件数量
func (s *Stats) GetFileCount() int64 {
	return atomic.LoadInt64(&s.fileCount)
}

// GetDirCount 返回目录数量
func (s *Stats) GetDirCount() int64 {
	return atomic.LoadInt64(&s.dirCount)
}

// GetTotalSize 返回总大小
func (s *Stats) GetTotalSize() int64 {
	return atomic.LoadInt64(&s.totalSize)
}

// GetTotalSymlink 返回符号链接数量
func (s *Stats) GetTotalSymlink() int64 {
	return atomic.LoadInt64(&s.totalSymlink)
}

// GetTotalRegularFile 返回普通文件数量
func (s *Stats) GetTotalRegularFile() int64 {
	return atomic.LoadInt64(&s.totalRegularFile)
}

// GetAvgNameLength 返回平均文件名长度
func (s *Stats) GetAvgNameLength() int {
	return int(s.avgNameLength)
}

// GetMaxNameLength 返回最大文件名长度
func (s *Stats) GetMaxNameLength() int {
	return s.maxNameLength
}

// GetAvgDirDepth 返回平均目录深度
func (s *Stats) GetAvgDirDepth() int {
	return int(s.avgDirDepth)
}

// GetMaxDirDepth 返回最大目录深度
func (s *Stats) GetMaxDirDepth() int {
	return s.maxDirDepth
}

// Print 打印统计信息
func (s *Stats) Print() {
	fmt.Printf("Scan completed. Files: %d, Directories: %d, Total size: %d bytes\n",
		s.GetFileCount(), s.GetDirCount(), s.GetTotalSize())
	fmt.Printf("File name statistics: Average length: %d, Max length: %d\n",
		s.GetAvgNameLength(), s.GetMaxNameLength())
	fmt.Printf("Directory depth statistics: Average depth: %d, Max depth: %d\n",
		s.GetAvgDirDepth(), s.GetMaxDirDepth())
}
