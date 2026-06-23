# Merkle DAG Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 实现一个 Go 语言本地 Merkle DAG 文件存储系统，支持 `Blob`、`Tree`、`List`、`add`、`resolve`、`cat`、`ls`、持久化对象存储和完整性复验。

**Architecture:** 项目按职责拆分为 `object`、`store`、`importer`、`resolver` 和 `cmd/mdag`。`object` 负责对象结构和 CID，`store` 负责磁盘对象存储，`importer` 负责导入本地文件系统，`resolver` 负责路径解析和读取，CLI 只做命令参数解析和输出。

**Tech Stack:** Go 标准库，JSON，SHA-256，文件系统 API，`go test`。

---

## 文件结构

本计划会创建以下文件：

```text
go.mod
object/object.go
object/codec.go
object/codec_test.go
store/store.go
store/file.go
store/file_test.go
importer/importer.go
importer/importer_test.go
resolver/resolver.go
resolver/resolver_test.go
cmd/mdag/main.go
testdata/demo/README.md
testdata/demo/docs/report.txt
testdata/demo/docs/notes.txt
testdata/demo/data/sample.txt
testdata/demo/big/large.txt
README.md
```

实现顺序遵循“先核心库，后命令行，最后文档和演示”。每个任务完成后都要运行测试，并尽量独立提交。

---

### Task 1: 初始化 Go 模块和对象模型

**Files:**
- Create: `go.mod`
- Create: `object/object.go`
- Create: `object/codec.go`
- Create: `object/codec_test.go`

- [ ] **Step 1: 创建 Go module**

创建 `go.mod`：

```go
module merkledag

go 1.22
```

- [ ] **Step 2: 写对象模型**

创建 `object/object.go`：

```go
package object

type ObjectType string

const (
	BlobType ObjectType = "blob"
	TreeType ObjectType = "tree"
	ListType ObjectType = "list"
)

type Link struct {
	Name string `json:"name,omitempty"`
	CID  string `json:"cid"`
	Size int64  `json:"size,omitempty"`
}

type Object struct {
	Type  ObjectType `json:"type"`
	Data  []byte     `json:"data,omitempty"`
	Links []Link     `json:"links,omitempty"`
}
```

- [ ] **Step 3: 先写 CID 测试**

创建 `object/codec_test.go`：

```go
package object

import "testing"

func TestCIDStableForSameObject(t *testing.T) {
	obj := Object{Type: BlobType, Data: []byte("hello")}

	cid1, err := CID(obj)
	if err != nil {
		t.Fatal(err)
	}
	cid2, err := CID(obj)
	if err != nil {
		t.Fatal(err)
	}

	if cid1 != cid2 {
		t.Fatalf("same object should have same CID: %s != %s", cid1, cid2)
	}
}

func TestCIDChangesWhenContentChanges(t *testing.T) {
	first := Object{Type: BlobType, Data: []byte("hello")}
	second := Object{Type: BlobType, Data: []byte("hello!")}

	cid1, err := CID(first)
	if err != nil {
		t.Fatal(err)
	}
	cid2, err := CID(second)
	if err != nil {
		t.Fatal(err)
	}

	if cid1 == cid2 {
		t.Fatalf("different objects should have different CIDs: %s", cid1)
	}
}

func TestEncodeUsesDeterministicJSON(t *testing.T) {
	obj := Object{
		Type: TreeType,
		Links: []Link{
			{Name: "b.txt", CID: "cid-b", Size: 2},
			{Name: "a.txt", CID: "cid-a", Size: 1},
		},
	}

	first, err := Encode(obj)
	if err != nil {
		t.Fatal(err)
	}
	second, err := Encode(obj)
	if err != nil {
		t.Fatal(err)
	}

	if string(first) != string(second) {
		t.Fatalf("encoding should be deterministic:\n%s\n%s", first, second)
	}
}
```

- [ ] **Step 4: 运行测试，确认失败**

Run:

```bash
go test ./object
```

Expected: FAIL，提示 `CID` 和 `Encode` 未定义。

- [ ] **Step 5: 实现编码和 CID**

创建 `object/codec.go`：

```go
package object

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
)

func Encode(obj Object) ([]byte, error) {
	return json.Marshal(obj)
}

func Decode(data []byte) (Object, error) {
	var obj Object
	if err := json.Unmarshal(data, &obj); err != nil {
		return Object{}, err
	}
	return obj, nil
}

func CID(obj Object) (string, error) {
	data, err := Encode(obj)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}
```

- [ ] **Step 6: 运行测试，确认通过**

Run:

```bash
go test ./object
```

Expected: PASS。

- [ ] **Step 7: 提交**

```bash
git add go.mod object/object.go object/codec.go object/codec_test.go
git commit -m "feat: add object model and cid encoding"
```

**本步答辩解释：** CID 是对象 JSON 序列化后的 SHA-256 哈希。对象内容一样，哈希输入一样，所以 CID 一样；对象内容变化，哈希输入变化，所以 CID 变化。

---

### Task 2: 实现文件对象存储和完整性复验

**Files:**
- Create: `store/store.go`
- Create: `store/file.go`
- Create: `store/file_test.go`

- [ ] **Step 1: 写 Store 测试**

创建 `store/file_test.go`：

```go
package store

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"merkledag/object"
)

func TestFileStorePutAndGetObject(t *testing.T) {
	dir := t.TempDir()
	st := NewFileStore(filepath.Join(dir, "objects"))
	obj := object.Object{Type: object.BlobType, Data: []byte("hello")}

	cid, err := st.PutObject(obj)
	if err != nil {
		t.Fatal(err)
	}

	got, err := st.GetObject(cid)
	if err != nil {
		t.Fatal(err)
	}
	if string(got.Data) != "hello" {
		t.Fatalf("unexpected data: %q", string(got.Data))
	}
}

func TestFileStoreMissingCID(t *testing.T) {
	dir := t.TempDir()
	st := NewFileStore(filepath.Join(dir, "objects"))

	_, err := st.GetObject("missing")
	if err == nil {
		t.Fatal("expected error for missing CID")
	}
	if !strings.Contains(err.Error(), "object not found") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestFileStoreDetectsTampering(t *testing.T) {
	dir := t.TempDir()
	st := NewFileStore(filepath.Join(dir, "objects"))
	obj := object.Object{Type: object.BlobType, Data: []byte("hello")}

	cid, err := st.PutObject(obj)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "objects", cid+".json")
	if err := os.WriteFile(path, []byte(`{"type":"blob","data":"dGFtcGVyZWQ="}`), 0644); err != nil {
		t.Fatal(err)
	}

	_, err = st.GetObject(cid)
	if err == nil {
		t.Fatal("expected integrity error")
	}
	if !strings.Contains(err.Error(), "integrity check failed") {
		t.Fatalf("unexpected error: %v", err)
	}
}
```

- [ ] **Step 2: 运行测试，确认失败**

Run:

```bash
go test ./store
```

Expected: FAIL，提示 `NewFileStore` 未定义。

- [ ] **Step 3: 定义 Store 接口**

创建 `store/store.go`：

```go
package store

import "merkledag/object"

type Store interface {
	PutObject(obj object.Object) (string, error)
	GetObject(cid string) (object.Object, error)
}
```

- [ ] **Step 4: 实现 FileStore**

创建 `store/file.go`：

```go
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
```

- [ ] **Step 5: 运行测试，确认通过**

Run:

```bash
go test ./store
```

Expected: PASS。

- [ ] **Step 6: 提交**

```bash
git add store/store.go store/file.go store/file_test.go
git commit -m "feat: add file-backed object store"
```

**本步答辩解释：** 基础 `GetObject` 是按 CID 找对象；提高功能是在读出对象后重新计算 CID，发现对象内容和文件名 CID 不一致就报错。

---

### Task 3: 实现文件和目录导入，支持 Blob、Tree、List

**Files:**
- Create: `importer/importer.go`
- Create: `importer/importer_test.go`

- [ ] **Step 1: 写 importer 测试**

创建 `importer/importer_test.go`：

```go
package importer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"merkledag/object"
	"merkledag/store"
)

func TestAddSmallFileCreatesBlob(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "hello.txt")
	if err := os.WriteFile(filePath, []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}
	st := store.NewFileStore(filepath.Join(dir, "objects"))

	cid, err := AddPath(filePath, st)
	if err != nil {
		t.Fatal(err)
	}
	obj, err := st.GetObject(cid)
	if err != nil {
		t.Fatal(err)
	}

	if obj.Type != object.BlobType {
		t.Fatalf("expected blob, got %s", obj.Type)
	}
	if string(obj.Data) != "hello" {
		t.Fatalf("unexpected data: %q", string(obj.Data))
	}
}

func TestAddLargeFileCreatesList(t *testing.T) {
	dir := t.TempDir()
	data := []byte(strings.Repeat("x", ChunkSize+20))
	filePath := filepath.Join(dir, "large.txt")
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		t.Fatal(err)
	}
	st := store.NewFileStore(filepath.Join(dir, "objects"))

	cid, err := AddPath(filePath, st)
	if err != nil {
		t.Fatal(err)
	}
	obj, err := st.GetObject(cid)
	if err != nil {
		t.Fatal(err)
	}

	if obj.Type != object.ListType {
		t.Fatalf("expected list, got %s", obj.Type)
	}
	if len(obj.Links) != 2 {
		t.Fatalf("expected 2 chunks, got %d", len(obj.Links))
	}
}

func TestAddDirectoryCreatesStableTree(t *testing.T) {
	dir := t.TempDir()
	demo := filepath.Join(dir, "demo")
	if err := os.MkdirAll(filepath.Join(demo, "docs"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(demo, "docs", "report.txt"), []byte("report"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(demo, "README.md"), []byte("demo"), 0644); err != nil {
		t.Fatal(err)
	}

	st := store.NewFileStore(filepath.Join(dir, "objects"))
	cid1, err := AddPath(demo, st)
	if err != nil {
		t.Fatal(err)
	}
	cid2, err := AddPath(demo, st)
	if err != nil {
		t.Fatal(err)
	}

	if cid1 != cid2 {
		t.Fatalf("same directory should produce same root CID: %s != %s", cid1, cid2)
	}
	obj, err := st.GetObject(cid1)
	if err != nil {
		t.Fatal(err)
	}
	if obj.Type != object.TreeType {
		t.Fatalf("expected tree, got %s", obj.Type)
	}
	if len(obj.Links) != 2 {
		t.Fatalf("expected 2 direct children, got %d", len(obj.Links))
	}
	if obj.Links[0].Name != "README.md" || obj.Links[1].Name != "docs" {
		t.Fatalf("links should be sorted by name: %+v", obj.Links)
	}
}
```

- [ ] **Step 2: 运行测试，确认失败**

Run:

```bash
go test ./importer
```

Expected: FAIL，提示 `AddPath` 和 `ChunkSize` 未定义。

- [ ] **Step 3: 实现 importer**

创建 `importer/importer.go`：

```go
package importer

import (
	"io"
	"os"
	"path/filepath"
	"sort"

	"merkledag/object"
	"merkledag/store"
)

const ChunkSize = 1024

func AddPath(localPath string, st store.Store) (string, error) {
	info, err := os.Stat(localPath)
	if err != nil {
		return "", err
	}
	if info.IsDir() {
		return addDirectory(localPath, st)
	}
	return addFile(localPath, info.Size(), st)
}

func addDirectory(localPath string, st store.Store) (string, error) {
	entries, err := os.ReadDir(localPath)
	if err != nil {
		return "", err
	}

	links := make([]object.Link, 0, len(entries))
	for _, entry := range entries {
		childPath := filepath.Join(localPath, entry.Name())
		childInfo, err := os.Stat(childPath)
		if err != nil {
			return "", err
		}
		childCID, err := AddPath(childPath, st)
		if err != nil {
			return "", err
		}
		links = append(links, object.Link{
			Name: entry.Name(),
			CID:  childCID,
			Size: childInfo.Size(),
		})
	}

	sort.Slice(links, func(i, j int) bool {
		return links[i].Name < links[j].Name
	})

	return st.PutObject(object.Object{
		Type:  object.TreeType,
		Links: links,
	})
}

func addFile(localPath string, size int64, st store.Store) (string, error) {
	if size <= ChunkSize {
		data, err := os.ReadFile(localPath)
		if err != nil {
			return "", err
		}
		return st.PutObject(object.Object{
			Type: object.BlobType,
			Data: data,
		})
	}
	return addChunkedFile(localPath, st)
}

func addChunkedFile(localPath string, st store.Store) (string, error) {
	file, err := os.Open(localPath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	var links []object.Link
	buffer := make([]byte, ChunkSize)
	for {
		n, readErr := file.Read(buffer)
		if n > 0 {
			chunk := make([]byte, n)
			copy(chunk, buffer[:n])
			chunkCID, err := st.PutObject(object.Object{
				Type: object.BlobType,
				Data: chunk,
			})
			if err != nil {
				return "", err
			}
			links = append(links, object.Link{
				CID:  chunkCID,
				Size: int64(n),
			})
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return "", readErr
		}
	}

	return st.PutObject(object.Object{
		Type:  object.ListType,
		Links: links,
	})
}
```

- [ ] **Step 4: 运行测试，确认通过**

Run:

```bash
go test ./importer
```

Expected: PASS。

- [ ] **Step 5: 运行所有已有测试**

Run:

```bash
go test ./...
```

Expected: PASS。

- [ ] **Step 6: 提交**

```bash
git add importer/importer.go importer/importer_test.go
git commit -m "feat: add filesystem importer"
```

**本步答辩解释：** 目录导入是递归的。文件变成 Blob 或 List，目录变成 Tree。Tree 保存子项名称和子对象 CID，因此父目录通过 HashLink 指向子对象。

---

### Task 4: 实现路径解析、文件读取和目录列出

**Files:**
- Create: `resolver/resolver.go`
- Create: `resolver/resolver_test.go`

- [ ] **Step 1: 写 resolver 测试**

创建 `resolver/resolver_test.go`：

```go
package resolver

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"merkledag/importer"
	"merkledag/object"
	"merkledag/store"
)

func createDemo(t *testing.T) (string, store.Store, string) {
	t.Helper()
	dir := t.TempDir()
	demo := filepath.Join(dir, "demo")
	if err := os.MkdirAll(filepath.Join(demo, "docs"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(demo, "big"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(demo, "docs", "report.txt"), []byte("report content"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(demo, "docs", "notes.txt"), []byte("notes"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(demo, "big", "large.txt"), []byte(strings.Repeat("large-", 300)), 0644); err != nil {
		t.Fatal(err)
	}

	st := store.NewFileStore(filepath.Join(dir, "objects"))
	rootCID, err := importer.AddPath(demo, st)
	if err != nil {
		t.Fatal(err)
	}
	return rootCID, st, demo
}

func TestResolveNestedFile(t *testing.T) {
	rootCID, st, _ := createDemo(t)

	result, err := Resolve(rootCID, "/docs/report.txt", st)
	if err != nil {
		t.Fatal(err)
	}
	if result.Type != object.BlobType {
		t.Fatalf("expected blob, got %s", result.Type)
	}
}

func TestReadFileBlob(t *testing.T) {
	rootCID, st, _ := createDemo(t)

	data, err := ReadFile(rootCID, "/docs/report.txt", st)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "report content" {
		t.Fatalf("unexpected data: %q", string(data))
	}
}

func TestReadFileList(t *testing.T) {
	rootCID, st, demo := createDemo(t)

	got, err := ReadFile(rootCID, "/big/large.txt", st)
	if err != nil {
		t.Fatal(err)
	}
	want, err := os.ReadFile(filepath.Join(demo, "big", "large.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(want) {
		t.Fatal("list read should reconstruct original file")
	}
}

func TestResolveMissingPath(t *testing.T) {
	rootCID, st, _ := createDemo(t)

	_, err := Resolve(rootCID, "/docs/missing.txt", st)
	if err == nil {
		t.Fatal("expected missing path error")
	}
	if !strings.Contains(err.Error(), "path not found") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestReadFileRejectsTree(t *testing.T) {
	rootCID, st, _ := createDemo(t)

	_, err := ReadFile(rootCID, "/docs", st)
	if err == nil {
		t.Fatal("expected not file error")
	}
	if !strings.Contains(err.Error(), "target is not a file") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestListDirectory(t *testing.T) {
	rootCID, st, _ := createDemo(t)

	entries, err := List(rootCID, "/docs", st)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	if entries[0].Name != "notes.txt" || entries[1].Name != "report.txt" {
		t.Fatalf("unexpected entries: %+v", entries)
	}
}
```

- [ ] **Step 2: 运行测试，确认失败**

Run:

```bash
go test ./resolver
```

Expected: FAIL，提示 `Resolve`、`ReadFile`、`List` 未定义。

- [ ] **Step 3: 实现 resolver**

创建 `resolver/resolver.go`：

```go
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
```

- [ ] **Step 4: 运行测试，确认通过**

Run:

```bash
go test ./resolver
```

Expected: PASS。

- [ ] **Step 5: 运行全部测试**

Run:

```bash
go test ./...
```

Expected: PASS。

- [ ] **Step 6: 提交**

```bash
git add resolver/resolver.go resolver/resolver_test.go
git commit -m "feat: add path resolver and file reader"
```

**本步答辩解释：** Resolve 只做路径定位，逐层读取 Tree 并查找名称；ReadFile 在 Resolve 后读取 Blob 或 List。这个区分对应课程要求里的“路径解析”和“文件读取”。

---

### Task 5: 实现 mdag 命令行

**Files:**
- Create: `cmd/mdag/main.go`

- [ ] **Step 1: 创建 CLI**

创建 `cmd/mdag/main.go`：

```go
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"merkledag/importer"
	"merkledag/resolver"
	"merkledag/store"
)

const defaultObjectDir = "data/objects"

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		return usage()
	}

	st := store.NewFileStore(filepath.FromSlash(defaultObjectDir))

	switch args[0] {
	case "add":
		if len(args) != 2 {
			return fmt.Errorf("usage: mdag add <local-path>")
		}
		rootCID, err := importer.AddPath(args[1], st)
		if err != nil {
			return err
		}
		fmt.Println("Root CID:", rootCID)
		return nil
	case "resolve":
		if len(args) != 3 {
			return fmt.Errorf("usage: mdag resolve <root-cid> <path>")
		}
		result, err := resolver.Resolve(args[1], args[2], st)
		if err != nil {
			return err
		}
		fmt.Println("Target CID:", result.CID)
		fmt.Println("Type:", result.Type)
		return nil
	case "cat":
		if len(args) != 3 {
			return fmt.Errorf("usage: mdag cat <root-cid> <path>")
		}
		data, err := resolver.ReadFile(args[1], args[2], st)
		if err != nil {
			return err
		}
		fmt.Print(string(data))
		return nil
	case "ls":
		if len(args) != 3 {
			return fmt.Errorf("usage: mdag ls <root-cid> <path>")
		}
		entries, err := resolver.List(args[1], args[2], st)
		if err != nil {
			return err
		}
		for _, entry := range entries {
			fmt.Printf("%s\t%s\t%s\t%d\n", entry.Type, entry.Name, entry.CID, entry.Size)
		}
		return nil
	default:
		return usage()
	}
}

func usage() error {
	return fmt.Errorf("usage: mdag <add|resolve|cat|ls> [args]")
}
```

- [ ] **Step 2: 格式化代码**

Run:

```bash
gofmt -w cmd/mdag/main.go
```

Expected: 命令无输出。

- [ ] **Step 3: 运行测试**

Run:

```bash
go test ./...
```

Expected: PASS。

- [ ] **Step 4: 编译 CLI**

Run:

```bash
go build ./cmd/mdag
```

Expected: 当前目录生成可执行文件 `mdag`。

- [ ] **Step 5: 提交**

```bash
git add cmd/mdag/main.go
git commit -m "feat: add mdag cli"
```

**本步答辩解释：** CLI 本身不保存业务逻辑，只是把命令转发给 importer 和 resolver。这样老师问核心逻辑时，可以直接看对应包。

---

### Task 6: 添加演示数据和 README

**Files:**
- Create: `testdata/demo/README.md`
- Create: `testdata/demo/docs/report.txt`
- Create: `testdata/demo/docs/notes.txt`
- Create: `testdata/demo/data/sample.txt`
- Create: `testdata/demo/big/large.txt`
- Create: `README.md`

- [ ] **Step 1: 创建 testdata 目录**

Run:

```bash
mkdir -p testdata/demo/docs testdata/demo/data testdata/demo/big
```

Expected: 命令无输出。

- [ ] **Step 2: 写演示小文件**

创建 `testdata/demo/README.md`：

```markdown
# Demo Directory

This directory is used to demonstrate the simplified Merkle DAG importer.
```

创建 `testdata/demo/docs/report.txt`：

```text
This is a report for the Merkle DAG course project.
```

创建 `testdata/demo/docs/notes.txt`：

```text
Tree objects store names and child CIDs.
```

创建 `testdata/demo/data/sample.txt`：

```text
Sample data file.
```

- [ ] **Step 3: 创建超过 1024 字节的大文件**

Run:

```bash
perl -e 'print "large file chunk line\n" x 80' > testdata/demo/big/large.txt
```

Expected: `testdata/demo/big/large.txt` 大于 1024 字节。

- [ ] **Step 4: 写 README**

创建 `README.md`：

```markdown
# Merkle DAG 课程项目

这是一个使用 Go 语言实现的简化版 Merkle DAG 文件存储与路径解析系统。

## 已完成功能

- Blob：保存小文件内容。
- Tree：保存目录的直接子项名称和子对象 CID。
- List：将大于 1024 字节的文件切分为多个 Blob，并按顺序链接。
- CID：使用 `hex(SHA256(JSON对象))` 生成内容标识。
- 本地对象存储：对象保存到 `./data/objects/<cid>.json`。
- 完整性复验：读取对象后重新计算 CID，检测对象文件是否被修改。
- 命令行：支持 `add`、`resolve`、`cat`、`ls`。

## 编译

```bash
go build ./cmd/mdag
```

## 运行示例

导入演示目录：

```bash
./mdag add ./testdata/demo
```

输出示例：

```text
Root CID: <root-cid>
```

解析两级路径：

```bash
./mdag resolve <root-cid> /docs/report.txt
```

读取文件：

```bash
./mdag cat <root-cid> /docs/report.txt
```

列出目录：

```bash
./mdag ls <root-cid> /docs
```

演示 List：

```bash
./mdag resolve <root-cid> /big/large.txt
./mdag cat <root-cid> /big/large.txt
```

`large.txt` 大于 1024 字节，导入后会被编码为 List，List 中的 Link 按顺序指向多个 Blob 分块。

## 核心原理

CID 是对象内容的哈希指纹。同一个对象序列化结果相同，因此 CID 相同；对象内容发生变化，CID 也会变化。

Blob 保存文件字节。Tree 表示目录，保存子项名称和子对象 CID。List 表示分块文件，按顺序保存多个 Blob 的 CID。

HashLink 指父对象不直接嵌入子对象内容，而是保存子对象 CID。目录的根 CID 能代表整个目录，是因为每个子对象 CID 都会影响父 Tree 的 CID，并最终影响根 Tree 的 CID。

Resolve 只负责根据路径找到目标对象 CID。Cat 会在 Resolve 的基础上读取 Blob 或 List 的字节内容。

## 测试

```bash
go test ./...
```

## 答辩演示流程

1. 运行 `./mdag add ./testdata/demo`，生成根 CID。
2. 运行 `./mdag resolve <root-cid> /docs/report.txt`，展示路径解析。
3. 运行 `./mdag cat <root-cid> /docs/report.txt`，展示文件读取。
4. 运行 `./mdag resolve <root-cid> /big/large.txt`，展示 List 类型。
5. 修改 `testdata/demo/docs/report.txt` 后重新 add，展示根 CID 变化。

## 小组分工

如为个人完成，可写：本项目由本人独立完成，负责对象模型、CID 生成、对象存储、目录导入、路径解析、命令行和测试。
```

- [ ] **Step 5: 编译并手动演示**

Run:

```bash
go build ./cmd/mdag
./mdag add ./testdata/demo
```

Expected: 输出 `Root CID: ...`。

- [ ] **Step 6: 提交**

```bash
git add testdata README.md
git commit -m "docs: add demo data and readme"
```

**本步答辩解释：** `testdata/demo` 是现场演示入口，README 里的命令就是答辩时可以照着运行的脚本。

---

### Task 7: 补充端到端 CLI 验证

**Files:**
- Modify: `README.md`

- [ ] **Step 1: 运行完整测试**

Run:

```bash
go test ./...
```

Expected: PASS。

- [ ] **Step 2: 编译 CLI**

Run:

```bash
go build ./cmd/mdag
```

Expected: 当前目录存在 `mdag` 可执行文件。

- [ ] **Step 3: 清理旧对象存储并导入演示目录**

Run:

```bash
rm -rf data
./mdag add ./testdata/demo
```

Expected: 输出 `Root CID: <root-cid>`。记录这个 CID，用于后续命令。

- [ ] **Step 4: 验证 resolve**

Run:

```bash
./mdag resolve <root-cid> /docs/report.txt
```

Expected:

```text
Target CID: <target-cid>
Type: blob
```

- [ ] **Step 5: 验证 cat**

Run:

```bash
./mdag cat <root-cid> /docs/report.txt
```

Expected:

```text
This is a report for the Merkle DAG course project.
```

- [ ] **Step 6: 验证 ls**

Run:

```bash
./mdag ls <root-cid> /docs
```

Expected: 输出包含 `notes.txt` 和 `report.txt`。

- [ ] **Step 7: 验证 List**

Run:

```bash
./mdag resolve <root-cid> /big/large.txt
```

Expected:

```text
Target CID: <target-cid>
Type: list
```

- [ ] **Step 8: 如果命令输出和 README 有差异，修正 README**

只修改实际不一致的命令或输出说明。不要扩大 README 范围。

- [ ] **Step 9: 提交**

```bash
git add README.md
git commit -m "docs: verify cli examples"
```

如果 README 没有变化，则跳过提交。

**本步答辩解释：** 端到端验证证明项目不只是单元测试通过，还能按老师要求现场演示 `add`、`resolve`、`cat` 和修改文件导致 CID 变化。

---

## Self-Review

### Spec coverage

- Blob 对象：Task 1、Task 3、Task 4 覆盖。
- Tree 对象：Task 1、Task 3、Task 4 覆盖。
- List 分块文件：Task 1、Task 3、Task 4、Task 6、Task 7 覆盖。
- CID 生成：Task 1 覆盖。
- 对象存储：Task 2 覆盖。
- 完整性复验：Task 2 覆盖。
- 文件/目录导入：Task 3 覆盖。
- 路径解析：Task 4 覆盖。
- 文件读取：Task 4 覆盖。
- `add`、`resolve`、`cat`、`ls` 命令：Task 5 覆盖。
- testdata 和 README：Task 6 覆盖。
- 端到端演示：Task 7 覆盖。

### Placeholder scan

计划中没有占位符或“以后再补”的步骤。所有任务都给出了具体文件、代码或命令。

### Type consistency

核心类型保持一致：

- `object.ObjectType`
- `object.Link`
- `object.Object`
- `store.Store`
- `importer.AddPath`
- `resolver.Resolve`
- `resolver.ReadFile`
- `resolver.List`

`resolver.Result` 用于返回目标 CID 和类型，`resolver.Entry` 用于 `ls` 输出。
