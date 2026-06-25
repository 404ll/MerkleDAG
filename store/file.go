package store

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"merkledag/object"
)

type FileStore struct {
	dir string // 存储对象的目录
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
	//确保目录存在
	if err := os.MkdirAll(s.dir, 0755); err != nil {
		return "", err
	}

	//将对象写入文件
	path := filepath.Join(s.dir, cid+".json")
	if err := os.WriteFile(path, data, 0644); err != nil {
		return "", err
	}
	return cid, nil
}

func (s *FileStore) GetObject(cid string) (object.Object, error) {
	path := filepath.Join(s.dir, cid+".json")
	data, err := os.ReadFile(path) //读取文件
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return object.Object{}, fmt.Errorf("对象不存在: %s", cid)
		}
		return object.Object{}, err
	}

	obj, err := object.Decode(data) //反序列化
	if err != nil {
		return object.Object{}, fmt.Errorf("反序列化对象 %s 失败: %w", cid, err)
	}
	actualCID, err := object.CID(obj) //计算对象的CID
	if err != nil {
		return object.Object{}, err
	}
	//通过对比验证文件是否被篡改
	if actualCID != cid {
		return object.Object{}, fmt.Errorf("完整性复验失败: 请求 CID 为 %s，实际内容 CID 为 %s", cid, actualCID)
	}
	return obj, nil
}
