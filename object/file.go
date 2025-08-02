package object

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"terrasync/log"
	"time"
)

const (
	dirSuffix    = "/"
	listDirLen   = 1024
	listQueueLen = 8192
)

type localStorage struct {
	scanPath string
}

type fileObject struct {
	info  os.FileInfo
	dir   string
	root  *string
	ctime time.Time
	atime time.Time
}

type SectionReaderCloser struct {
	*io.SectionReader
	io.Closer
}

func (o *fileObject) Key() string {
	return filepath.Join(o.dir, o.info.Name())
}

func (o *fileObject) Size() int64 {
	return o.info.Size()
}

func (o *fileObject) MTime() time.Time {
	return o.info.ModTime()
}

func (o *fileObject) CTime() time.Time {
	return o.ctime
}

func (o *fileObject) ATime() time.Time {
	return o.atime
}

func (o *fileObject) Perm() os.FileMode {
	return o.info.Mode().Perm()
}

func (o *fileObject) IsRegular() bool {
	return o.info.Mode().IsRegular()
}

func (o *fileObject) IsDir() bool {
	return o.info.IsDir()
}

func (o *fileObject) IsSymlink() bool {
	return o.info.Mode()&os.ModeSymlink != 0
}

func (o *fileObject) IsSticky() bool {
	return o.info.Mode()&os.ModeSticky != 0
}

func (o *fileObject) Delete() error {
	err := os.Remove(o.fullPath())
	if err != nil && os.IsNotExist(err) {
		err = nil
	}
	return err
}

func (o *fileObject) fullPath() string {
	return filepath.Join(*o.root, o.Key())
}

func (o *fileObject) Get(offset, limit int64) (io.ReadCloser, error) {
	if o.IsDir() || offset > o.Size() {
		return io.NopCloser(bytes.NewBuffer([]byte{})), nil
	}
	f, err := os.Open(o.fullPath())
	if err != nil {
		return nil, fmt.Errorf("open %s fail: %v", o.Key(), err)
	}

	if limit > 0 {
		return &SectionReaderCloser{
			SectionReader: io.NewSectionReader(f, offset, limit),
			Closer:        f,
		}, nil
	}
	return f, nil
}

func (s *localStorage) fullPath(key string) string {
	return filepath.Join(s.scanPath, key)
}

func (s *localStorage) List(dir string) (<-chan FileInfo, error) {
	fp, err := os.Open(s.fullPath(dir))
	if err != nil {
		return nil, fmt.Errorf("open %s fail: %v", s.fullPath(dir), err)
	}
	queue := make(chan FileInfo, listQueueLen)
	go func() {
		defer fp.Close()
		defer close(queue)
		for {
			files, err := fp.Readdir(listDirLen)
			if err != nil {
				if err != io.EOF {
					log.Errorf("read local file fail: %v", err)
				}
				return
			}
			for _, file := range files {
				if file.Name() == "." || file.Name() == ".." {
					continue
				}
				// 创建fileObject
				fileObj := &fileObject{
					info: file,
					dir:  dir,
					root: &s.scanPath,
				}

				// 判断操作系统是否为Windows
				if runtime.GOOS == "windows" {
					if sysInfo, ok := file.Sys().(*syscall.Win32FileAttributeData); ok {
						// 转换Windows文件时间到time.Time
						fileObj.ctime = time.Unix(0, sysInfo.CreationTime.Nanoseconds())
						fileObj.atime = time.Unix(0, sysInfo.LastAccessTime.Nanoseconds())
					} else {
						log.Warnf("failed to get system info for file: %s", file.Name())
					}
				} else {
					fileObj.ctime = file.ModTime()
					fileObj.atime = file.ModTime()
				}

				queue <- fileObj
			}
		}
	}()
	return queue, nil
}

func (s *localStorage) Head(key string) (FileInfo, error) {
	return nil, nil
}

func (s *localStorage) Get(key string) (io.ReadCloser, error) {
	return nil, nil
}

func (s *localStorage) Put(key string, in io.Reader) error {
	p := s.fullPath(key)

	if strings.HasSuffix(key, dirSuffix) || key == "" && strings.HasSuffix(s.scanPath, dirSuffix) {
		return os.MkdirAll(p, os.FileMode(0777))
	}
	f, err := os.OpenFile(p, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0666)
	if err != nil && os.IsNotExist(err) {
		if err = os.MkdirAll(filepath.Dir(p), os.FileMode(0777)); err != nil {
			return err
		}
		f, err = os.OpenFile(p, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0666)
	}
	if err != nil {
		return err
	}

	buf := bufPool.Get().(*[]byte)
	defer bufPool.Put(buf)
	_, err = io.CopyBuffer(f, in, *buf)
	if err != nil {
		_ = f.Close()
		return err
	}
	return f.Close()
}

func (s *localStorage) Delete(key string) error {
	err := os.Remove(s.fullPath(key))
	if err != nil && os.IsNotExist(err) {
		err = nil
	}
	return err
}

func (s *localStorage) Close() error {
	return nil
}

func createLocalStorage(scanPath string) (Storage, error) {
	return &localStorage{scanPath: scanPath}, nil
}
