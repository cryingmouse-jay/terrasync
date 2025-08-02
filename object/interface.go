package object

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

// FileInfo represents metadata about a file or directory
type FileInfo interface {
	Key() string
	Size() int64
	MTime() time.Time
	CTime() time.Time
	ATime() time.Time
	Perm() os.FileMode
	IsDir() bool
	IsSymlink() bool
	IsRegular() bool
	IsSticky() bool
	Get(offset, limit int64) (io.ReadCloser, error)
	Delete() error
}

// Storage defines the interface for different storage backends
type Storage interface {
	List(dir string) (<-chan FileInfo, error)
	Head(key string) (FileInfo, error)
	Put(key string, in io.Reader) error
	Delete(key string) error
	Close() error
}

// CreateStorage creates a storage instance based on the provided URI
func CreateStorage(scanPath string) (Storage, error) {
	if strings.HasPrefix(scanPath, "s3://") || strings.HasPrefix(scanPath, "S3://") {
		return createS3(scanPath)
	}

	nfsPattern := `^[a-zA-Z0-9.-]+:\S+$`
	if regexp.MustCompile(nfsPattern).MatchString(scanPath) {
		return createNfs(scanPath)
	}

	fileInfo, err := os.Stat(scanPath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat uri %s: %w", scanPath, err)
	}
	if fileInfo.IsDir() {
		absPath, err := filepath.Abs(scanPath)
		if err != nil {
			return nil, fmt.Errorf("failed to get absolute path: %w", err)
		}
		return createLocalStorage(absPath)
	}

	return nil, fmt.Errorf("unsupported storage type for uri: %s", scanPath)
}

// BufferPoolSize defines the size of buffers in the buffer pool
var BufferPoolSize = 1 << 20 // 1MB - can be adjusted based on workload

var bufPool = sync.Pool{
	New: func() interface{} {
		// Default io.Copy uses 32KB buffer, here we choose a larger one (1MiB io-size increases throughput by ~20%)
		buf := make([]byte, BufferPoolSize)
		return &buf
	},
}
