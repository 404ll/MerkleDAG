package store

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"merkledag/object"
)

// 测试文件存储可以写入对象并按 CID 读取回来。
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
		t.Fatalf("读取到的内容不符合预期: %q", string(got.Data))
	}
}

// 测试读取不存在的 CID 时会返回中文错误。
func TestFileStoreMissingCID(t *testing.T) {
	dir := t.TempDir()
	st := NewFileStore(filepath.Join(dir, "objects"))

	_, err := st.GetObject("missing")
	if err == nil {
		t.Fatal("读取不存在的 CID 应该返回错误")
	}
	if !strings.Contains(err.Error(), "对象不存在") {
		t.Fatalf("错误信息不符合预期: %v", err)
	}
}

// 测试文件内容被篡改时，读取对象会触发完整性校验错误。
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
		t.Fatal("对象内容被篡改时应该返回完整性错误")
	}
	if !strings.Contains(err.Error(), "完整性复验失败") {
		t.Fatalf("错误信息不符合预期: %v", err)
	}
}
