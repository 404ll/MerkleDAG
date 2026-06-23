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
