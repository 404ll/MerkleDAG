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

// Resolve 从根 CID 出发解析路径，返回目标对象的 CID 和类型。
func Resolve(rootCID, rawPath string, st store.Store) (Result, error) {
	//统一规范化路径，确保以 / 开头，去除多余的 . 和 .. 等
	cleaned := path.Clean("/" + strings.TrimPrefix(rawPath, "/"))
	//如果路径为空、为当前目录或为根目录，则直接返回根对象
	if rawPath == "" || rawPath == "." || cleaned == "/" {
		obj, err := st.GetObject(rootCID)
		if err != nil {
			return Result{}, err
		}
		return Result{CID: rootCID, Type: obj.Type}, nil
	}

	currentCID := rootCID
	//路径分割
	parts := strings.Split(strings.TrimPrefix(cleaned, "/"), "/")

	for _, part := range parts {
		//一层一层找
		current, err := st.GetObject(currentCID)
		if err != nil {
			return Result{}, err
		}
		//当前对象必须是目录
		if current.Type != object.TreeType {
			return Result{}, fmt.Errorf("解析 %q 时当前对象不是目录: %s", part, currentCID)
		}

		nextCID := ""
		for _, link := range current.Links {
			if link.Name == part {
				nextCID = link.CID
				break
			}
		}
		if nextCID == "" {
			return Result{}, fmt.Errorf("路径不存在: %s", part)
		}
		currentCID = nextCID
	}

	target, err := st.GetObject(currentCID)
	if err != nil {
		return Result{}, err
	}
	return Result{CID: currentCID, Type: target.Type}, nil
}

// ReadFile 解析路径并读取目标文件内容，支持直接 Blob 和分块 List 文件。
func ReadFile(rootCID, rawPath string, st store.Store) ([]byte, error) {
	result, err := Resolve(rootCID, rawPath, st)
	if err != nil {
		return nil, err
	}
	return readObject(result.CID, st)
}

// readObject 按 CID 读取文件对象；List 会递归拼接所有分块内容。
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
		return nil, fmt.Errorf("目标不是文件: %s", cid)
	default:
		return nil, fmt.Errorf("未知对象类型 %q: %s", obj.Type, cid)
	}
}

// List 解析目录路径，并返回该目录下所有直接子项的名称、CID、类型和大小。
func List(rootCID, rawPath string, st store.Store) ([]Entry, error) {
	//判断路径是否存在并返回目标对象的 CID 和类型
	result, err := Resolve(rootCID, rawPath, st)
	if err != nil {
		return nil, err
	}
	// 根据 CID 读取并反序列化目标对象
	obj, err := st.GetObject(result.CID)
	if err != nil {
		return nil, err
	}
	if obj.Type != object.TreeType {
		return nil, fmt.Errorf("目标不是目录: %s", result.CID)
	}

	//遍历目录对象的链接，获取每个子对象的详细信息
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
