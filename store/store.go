package store

import "merkledag/object"

type Store interface {
	PutObject(obj object.Object) (string, error)
	GetObject(cid string) (object.Object, error)
}
