package scan

import (
	"path/filepath"
	"strings"
	"sync/atomic"
	"terrasync/object"
)

// Stats stores scan statistics
type Stats struct {
	fileCount        int64
	dirCount         int64
	totalSize        int64
	totalSymlink     int64
	totalRegularFile int64
	totalNameLength  int64 // 总文件名长度
	maxNameLength    int   // 最大文件名长度
	totalDirDepth    int64 // 总目录深度
	maxDirDepth      int   // 最大目录深度
}

// NewStats creates a new stats instance
func NewStats() *Stats {
	return &Stats{}
}

// Update updates statistics based on file information
func (s *Stats) Update(fileInfo object.FileInfo) {
	// Get file path
	key := fileInfo.Key()

	if fileInfo.IsDir() {
		atomic.AddInt64(&s.dirCount, 1)
	} else {
		// Use filepath to get filename and calculate length
		name := filepath.Base(key)
		nameLength := len(name)

		path := filepath.Dir(key)
		// Count directory separators as depth
		depth := strings.Count(path, string(filepath.Separator))
		// For root directory, depth is 0
		if path == string(filepath.Separator) || path == "." {
			depth = 0
		}

		atomic.AddInt64(&s.fileCount, 1)
		atomic.AddInt64(&s.totalSize, fileInfo.Size())
		atomic.AddInt64(&s.totalNameLength, int64(nameLength)) // Accumulate total filename length
		atomic.AddInt64(&s.totalDirDepth, int64(depth))        // Accumulate total directory depth

		// Update maximum filename length
		if nameLength > s.maxNameLength {
			s.maxNameLength = nameLength
		}

		// Update maximum directory depth
		if depth > s.maxDirDepth {
			s.maxDirDepth = depth
		}

		if fileInfo.IsRegular() {
			atomic.AddInt64(&s.totalRegularFile, 1)
		}
	}
	// Count symlinks and regular files
	if fileInfo.IsSymlink() {
		atomic.AddInt64(&s.totalSymlink, 1)
	}
}

// GetFileCount returns the number of files
func (s *Stats) GetFileCount() int64 {
	return atomic.LoadInt64(&s.fileCount)
}

// GetDirCount returns the number of directories
func (s *Stats) GetDirCount() int64 {
	return atomic.LoadInt64(&s.dirCount)
}

// GetTotalSize returns the total size
func (s *Stats) GetTotalSize() int64 {
	return atomic.LoadInt64(&s.totalSize)
}

// GetTotalSymlink returns the number of symlinks
func (s *Stats) GetTotalSymlink() int64 {
	return atomic.LoadInt64(&s.totalSymlink)
}

// GetTotalRegularFile returns the number of regular files
func (s *Stats) GetTotalRegularFile() int64 {
	return atomic.LoadInt64(&s.totalRegularFile)
}

// GetAvgNameLength returns the average filename length
func (s *Stats) GetAvgNameLength() int {
	count := atomic.LoadInt64(&s.fileCount)
	if count == 0 {
		return 0
	}
	return int(atomic.LoadInt64(&s.totalNameLength) / count)
}

// GetMaxNameLength returns the maximum filename length
func (s *Stats) GetMaxNameLength() int {
	return s.maxNameLength
}

// GetAvgDirDepth returns the average directory depth
func (s *Stats) GetAvgDirDepth() int {
	count := atomic.LoadInt64(&s.fileCount)
	if count == 0 {
		return 0
	}
	return int(atomic.LoadInt64(&s.totalDirDepth) / count)
}

// GetMaxDirDepth returns the maximum directory depth
func (s *Stats) GetMaxDirDepth() int {
	return s.maxDirDepth
}

// Print prints the statistics
func (s *Stats) Print() {
	// Print another separator
	printToConsoleAndLog("\n------------------------- Sanned Count -------------------------\n\n")

	// File count statistics
	fileCount := s.GetFileCount()
	dirCount := s.GetDirCount()
	printToConsoleAndLog("  Total:            %30d\n", fileCount+dirCount)
	printToConsoleAndLog("  Files:            %30d\n", fileCount)
	printToConsoleAndLog("  Directories:      %30d\n", dirCount)

	// Print another separator
	printToConsoleAndLog("\n--------------------------- Capacity ---------------------------\n\n")

	// Format total size using the utility function
	totalSize := s.GetTotalSize()
	averageSizeBytes := totalSize / int64(fileCount)
	printToConsoleAndLog("  Total:            %30s\n", FormatFileSize(totalSize))
	printToConsoleAndLog("  Average:          %30s\n", FormatFileSize(averageSizeBytes))

	// Print another separator
	printToConsoleAndLog("\n------------------------ Filename Length ------------------------\n\n")

	// Filename length statistics
	printToConsoleAndLog("  Avg :             %30d\n", s.GetAvgNameLength())
	printToConsoleAndLog("  Max :             %30d\n", s.GetMaxNameLength())

	// Print another separator
	printToConsoleAndLog("\n------------------------ Directory Depth ------------------------\n\n")

	// Directory depth statistics
	printToConsoleAndLog("  Avg dir depth:    %30d\n", s.GetAvgDirDepth())
	printToConsoleAndLog("  Max dir depth:    %30d\n", s.GetMaxDirDepth())

	// Print final separator
	printToConsoleAndLog("\n-------------------------------------------------------------\n\n")
}
