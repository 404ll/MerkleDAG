package resolver

import (
	"bytes"
	"fmt"
	"path"
	"strings"

	"merkledag/object"
	"merkledag/store"
)

type Result struct {
	CID  string
	Type object.ObjectType
}

type Entry struct {
	Name string
	CID  string
	Type object.ObjectType
	Size int64
}

func Resolve(rootCID, rawPath string, st store.Store) (Result, error) {
	cleaned := path.Clean("/" + strings.TrimPrefix(rawPath, "/"))
	if rawPath == "" || rawPath == "." || cleaned == "/" {
		obj, err := st.GetObject(rootCID)
		if err != nil {
			return Result{}, err
		}
		return Result{CID: rootCID, Type: obj.Type}, nil
	}

	currentCID := rootCID
	parts := strings.Split(strings.TrimPrefix(cleaned, "/"), "/")
	for _, part := range parts {
		current, err := st.GetObject(currentCID)
		if err != nil {
			return Result{}, err
		}
		if current.Type != object.TreeType {
			return Result{}, fmt.Errorf("not a directory while resolving %q: %s", part, currentCID)
		}

		nextCID := ""
		for _, link := range current.Links {
			if link.Name == part {
				nextCID = link.CID
				break
			}
		}
		if nextCID == "" {
			return Result{}, fmt.Errorf("path not found: %s", part)
		}
		currentCID = nextCID
	}

	target, err := st.GetObject(currentCID)
	if err != nil {
		return Result{}, err
	}
	return Result{CID: currentCID, Type: target.Type}, nil
}

func ReadFile(rootCID, rawPath string, st store.Store) ([]byte, error) {
	result, err := Resolve(rootCID, rawPath, st)
	if err != nil {
		return nil, err
	}
	return readObject(result.CID, st)
}

func readObject(cid string, st store.Store) ([]byte, error) {
	obj, err := st.GetObject(cid)
	if err != nil {
		return nil, err
	}

	switch obj.Type {
	case object.BlobType:
		return obj.Data, nil
	case object.ListType:
		var buf bytes.Buffer
		for _, link := range obj.Links {
			data, err := readObject(link.CID, st)
			if err != nil {
				return nil, err
			}
			buf.Write(data)
		}
		return buf.Bytes(), nil
	case object.TreeType:
		return nil, fmt.Errorf("target is not a file: %s", cid)
	default:
		return nil, fmt.Errorf("unknown object type %q: %s", obj.Type, cid)
	}
}

func List(rootCID, rawPath string, st store.Store) ([]Entry, error) {
	result, err := Resolve(rootCID, rawPath, st)
	if err != nil {
		return nil, err
	}
	obj, err := st.GetObject(result.CID)
	if err != nil {
		return nil, err
	}
	if obj.Type != object.TreeType {
		return nil, fmt.Errorf("target is not a directory: %s", result.CID)
	}

	entries := make([]Entry, 0, len(obj.Links))
	for _, link := range obj.Links {
		child, err := st.GetObject(link.CID)
		if err != nil {
			return nil, err
		}
		entries = append(entries, Entry{
			Name: link.Name,
			CID:  link.CID,
			Type: child.Type,
			Size: link.Size,
		})
	}
	return entries, nil
}
