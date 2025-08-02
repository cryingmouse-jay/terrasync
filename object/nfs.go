package object

import (
	"io"
)

type nfsStorage struct {
	scanPath string
}

func (s *nfsStorage) List(dir string) (<-chan FileInfo, error) {
	return nil, nil
}

func (s *nfsStorage) Head(key string) (FileInfo, error) {
	return nil, nil
}

func (s *nfsStorage) Get(key string) (io.ReadCloser, error) {
	return nil, nil
}

func (s *nfsStorage) Put(key string, in io.Reader) error {
	return nil
}

func (s *nfsStorage) Delete(key string) error {
	return nil
}

func (s *nfsStorage) Close() error {
	return nil
}

// TODO:
func createNfs(scanPath string) (Storage, error) {
	return &nfsStorage{scanPath: scanPath}, nil
}
