package importer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"merkledag/object"
	"merkledag/store"
)

// 测试添加小文件时会直接创建 Blob 对象。
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
		t.Fatalf("应该生成 Blob 对象，实际为 %s", obj.Type)
	}
	if string(obj.Data) != "hello" {
		t.Fatalf("文件内容不符合预期: %q", string(obj.Data))
	}
}

// 测试添加大文件时会创建 List 对象记录所有分块。
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
		t.Fatalf("应该生成 List 对象，实际为 %s", obj.Type)
	}
	if len(obj.Links) != 2 {
		t.Fatalf("应该生成 2 个分块，实际为 %d 个", len(obj.Links))
	}
}

// 测试添加目录时会创建稳定的 Tree 对象，并按名称排序子链接。
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
		t.Fatalf("同一目录应该生成相同的根 CID: %s != %s", cid1, cid2)
	}
	obj, err := st.GetObject(cid1)
	if err != nil {
		t.Fatal(err)
	}
	if obj.Type != object.TreeType {
		t.Fatalf("应该生成 Tree 对象，实际为 %s", obj.Type)
	}
	if len(obj.Links) != 2 {
		t.Fatalf("应该包含 2 个直接子项，实际为 %d 个", len(obj.Links))
	}
	if obj.Links[0].Name != "README.md" || obj.Links[1].Name != "docs" {
		t.Fatalf("目录链接应该按名称排序: %+v", obj.Links)
	}
}
