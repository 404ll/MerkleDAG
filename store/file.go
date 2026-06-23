package store

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"merkledag/object"
)

type FileStore struct {
	dir string
}

func NewFileStore(dir string) *FileStore {
	return &FileStore{dir: dir}
}

func (s *FileStore) PutObject(obj object.Object) (string, error) {
	cid, err := object.CID(obj)
	if err != nil {
		return "", err
	}
	data, err := object.Encode(obj)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(s.dir, 0755); err != nil {
		return "", err
	}
	path := filepath.Join(s.dir, cid+".json")
	if err := os.WriteFile(path, data, 0644); err != nil {
		return "", err
	}
	return cid, nil
}

func (s *FileStore) GetObject(cid string) (object.Object, error) {
	path := filepath.Join(s.dir, cid+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return object.Object{}, fmt.Errorf("object not found: %s", cid)
		}
		return object.Object{}, err
	}

	obj, err := object.Decode(data)
	if err != nil {
		return object.Object{}, fmt.Errorf("decode object %s: %w", cid, err)
	}
	actualCID, err := object.CID(obj)
	if err != nil {
		return object.Object{}, err
	}
	if actualCID != cid {
		return object.Object{}, fmt.Errorf("integrity check failed: requested %s but content is %s", cid, actualCID)
	}
	return obj, nil
}
