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
