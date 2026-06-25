package main

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func captureStdout(t *testing.T, fn func() error) (string, error) {
	t.Helper()

	oldStdout := os.Stdout
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = writer

	runErr := fn()

	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	os.Stdout = oldStdout

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, reader); err != nil {
		t.Fatal(err)
	}
	return buf.String(), runErr
}

func TestRunAddPrintsChineseRootCIDLabel(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	demo := filepath.Join(dir, "demo")
	if err := os.MkdirAll(demo, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(demo, "report.txt"), []byte("report"), 0644); err != nil {
		t.Fatal(err)
	}

	out, err := captureStdout(t, func() error {
		return run([]string{"add", demo})
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "根 CID:") {
		t.Fatalf("expected Chinese root CID label, got %q", out)
	}
}

func TestRunResolvePrintsChineseLabels(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	demo := filepath.Join(dir, "demo")
	if err := os.MkdirAll(demo, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(demo, "report.txt"), []byte("report"), 0644); err != nil {
		t.Fatal(err)
	}

	addOut, err := captureStdout(t, func() error {
		return run([]string{"add", demo})
	})
	if err != nil {
		t.Fatal(err)
	}
	rootCID := strings.TrimSpace(strings.TrimPrefix(addOut, "根 CID:"))

	resolveOut, err := captureStdout(t, func() error {
		return run([]string{"resolve", rootCID, "/report.txt"})
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(resolveOut, "目标 CID:") {
		t.Fatalf("expected Chinese target CID label, got %q", resolveOut)
	}
	if !strings.Contains(resolveOut, "类型:") {
		t.Fatalf("expected Chinese type label, got %q", resolveOut)
	}
}

func TestRunUsageIsChinese(t *testing.T) {
	err := run(nil)
	if err == nil {
		t.Fatal("expected usage error")
	}
	if !strings.Contains(err.Error(), "用法:") {
		t.Fatalf("expected Chinese usage, got %q", err.Error())
	}
}
