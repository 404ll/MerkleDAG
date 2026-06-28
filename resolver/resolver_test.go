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

// 测试可以从根 CID 解析到嵌套文件路径。
func TestResolveNestedFile(t *testing.T) {
	rootCID, st, _ := createDemo(t)

	result, err := Resolve(rootCID, "/docs/report.txt", st)
	if err != nil {
		t.Fatal(err)
	}
	if result.Type != object.BlobType {
		t.Fatalf("应该解析到 Blob 对象，实际为 %s", result.Type)
	}
}

// 测试可以读取普通 Blob 文件的内容。
func TestReadFileBlob(t *testing.T) {
	rootCID, st, _ := createDemo(t)

	data, err := ReadFile(rootCID, "/docs/report.txt", st)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "report content" {
		t.Fatalf("读取到的内容不符合预期: %q", string(data))
	}
}

// 测试可以读取 List 分块文件并还原原始内容。
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
		t.Fatal("读取 List 对象时应该重建原始文件内容")
	}
}

// 测试解析不存在的路径时会返回错误。
func TestResolveMissingPath(t *testing.T) {
	rootCID, st, _ := createDemo(t)

	_, err := Resolve(rootCID, "/docs/missing.txt", st)
	if err == nil {
		t.Fatal("解析不存在路径应该返回错误")
	}
	if !strings.Contains(err.Error(), "路径不存在") {
		t.Fatalf("错误信息不符合预期: %v", err)
	}
}

// 测试按文件读取目录时会返回错误。
func TestReadFileRejectsTree(t *testing.T) {
	rootCID, st, _ := createDemo(t)

	_, err := ReadFile(rootCID, "/docs", st)
	if err == nil {
		t.Fatal("按文件读取目录应该返回错误")
	}
	if !strings.Contains(err.Error(), "目标不是文件") {
		t.Fatalf("错误信息不符合预期: %v", err)
	}
}

// 测试可以列出目录下的直接子项。
func TestListDirectory(t *testing.T) {
	rootCID, st, _ := createDemo(t)

	entries, err := List(rootCID, "/docs", st)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Fatalf("应该列出 2 个条目，实际为 %d 个", len(entries))
	}
	if entries[0].Name != "notes.txt" || entries[1].Name != "report.txt" {
		t.Fatalf("目录条目不符合预期: %+v", entries)
	}
}
