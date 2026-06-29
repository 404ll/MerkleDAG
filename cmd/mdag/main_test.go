package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"merkledag/importer"
	"merkledag/store"
)

// 测试缺少命令参数时会返回中文用法提示。
func TestRunUsageIsChinese(t *testing.T) {
	err := run(nil)
	if err == nil {
		t.Fatal("缺少参数时应该返回用法错误")
	}
	if !strings.Contains(err.Error(), "用法:") {
		t.Fatalf("应该返回中文用法提示，实际为 %q", err.Error())
	}
}

func TestGraphHTMLWritesAndOpensFile(t *testing.T) {
	oldOpenGraphHTML := openGraphHTML
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		openGraphHTML = oldOpenGraphHTML
		if err := os.Chdir(oldWd); err != nil {
			t.Fatal(err)
		}
	}()

	dir := t.TempDir()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	demo := filepath.Join(dir, "demo")
	if err := os.MkdirAll(demo, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(demo, "hello.txt"), []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}

	rootCID, err := importer.AddPath(demo, store.NewFileStore(filepath.FromSlash(defaultObjectDir)))
	if err != nil {
		t.Fatal(err)
	}

	openedPath := ""
	openGraphHTML = func(path string) error {
		openedPath = path
		return nil
	}

	if err := run([]string{"graph", rootCID, "html"}); err != nil {
		t.Fatal(err)
	}
	if openedPath != "dag.html" {
		t.Fatalf("expected dag.html to be opened, got %q", openedPath)
	}

	data, err := os.ReadFile("dag.html")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "<svg") {
		t.Fatalf("generated HTML should contain SVG graph:\n%s", string(data))
	}
}
