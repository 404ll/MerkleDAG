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
