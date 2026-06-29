package graph

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"merkledag/importer"
	"merkledag/store"
)

func createGraphDemo(t *testing.T) (string, store.Store) {
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
	if err := os.WriteFile(filepath.Join(demo, "big", "large.txt"), []byte(strings.Repeat("large-", 300)), 0644); err != nil {
		t.Fatal(err)
	}

	st := store.NewFileStore(filepath.Join(dir, "objects"))
	rootCID, err := importer.AddPath(demo, st)
	if err != nil {
		t.Fatal(err)
	}
	return rootCID, st
}

func TestRenderDOTIncludesNodesAndEdges(t *testing.T) {
	rootCID, st := createGraphDemo(t)

	dot, err := RenderDOT(rootCID, st)
	if err != nil {
		t.Fatal(err)
	}

	for _, want := range []string{
		"digraph MerkleDAG",
		"tree",
		"blob",
		"list",
		"docs",
		"report.txt",
		"large.txt",
	} {
		if !strings.Contains(dot, want) {
			t.Fatalf("DOT output should contain %q:\n%s", want, dot)
		}
	}
}

func TestRenderHTMLIncludesReadableGraph(t *testing.T) {
	rootCID, st := createGraphDemo(t)

	html, err := RenderHTML(rootCID, st)
	if err != nil {
		t.Fatal(err)
	}

	for _, want := range []string{
		"<!doctype html>",
		"Merkle DAG",
		"<svg",
		"<line",
		"<rect",
		"<text",
		rootCID,
		"tree",
		"blob",
		"list",
		"docs",
		"report.txt",
		"large.txt",
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("HTML output should contain %q:\n%s", want, html)
		}
	}
}
