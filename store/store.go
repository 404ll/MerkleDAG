package store

import "merkledag/object"

// Store 定义 Merkle DAG 对象存储需要提供的读写能力。
type Store interface {
	PutObject(obj object.Object) (string, error)
	GetObject(cid string) (object.Object, error)
}
