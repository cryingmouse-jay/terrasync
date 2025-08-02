package object

import (
	"io"
)

type s3Storage struct {
	uri string
}

func (s *s3Storage) List(dir string) (<-chan FileInfo, error) {
	return nil, nil
}

func (s *s3Storage) Head(key string) (FileInfo, error) {
	return nil, nil
}

func (s *s3Storage) Get(key string) (io.ReadCloser, error) {
	return nil, nil
}

func (s *s3Storage) Put(key string, in io.Reader) error {
	return nil
}

func (s *s3Storage) Delete(key string) error {
	return nil
}

func (s *s3Storage) Close() error {
	return nil
}

// TODO:
func createS3(uri string) (Storage, error) {
	return &s3Storage{uri: uri}, nil
}
