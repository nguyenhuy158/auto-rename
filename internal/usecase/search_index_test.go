package usecase

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
	"unicode/utf8"
)

func TestSearchIndexRanksFilesAndKeepsPDFSnippetValid(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "résumé.pdf"), []byte("fixture"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "clip.mkv"), []byte("fixture"), 0600); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(t.TempDir(), "outside.pdf")
	if err := os.WriteFile(outside, []byte("private"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "escape.pdf")); err != nil {
		t.Fatal(err)
	}
	index := NewSearchIndex(root)
	if !index.Trigger() || index.Trigger() {
		t.Fatal("index should run only once at a time")
	}
	deadline := time.Now().Add(10 * time.Second)
	for index.Status().Running && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if index.Status().Running {
		t.Fatal("index did not finish")
	}
	if index.Status().TotalFiles != 2 {
		t.Fatalf("unexpected indexed files: %d", index.Status().TotalFiles)
	}
	index.mutex.Lock()
	file := index.files["résumé.pdf"]
	file.PDFText = "Đây là nội dung tài liệu về chăm sóc cây trồng."
	index.files[file.Path] = file
	index.mutex.Unlock()
	results := index.Search("chăm sóc", 10)
	if len(results) != 1 || results[0].Name != "résumé.pdf" || !strings.Contains(results[0].Snippet, "chăm sóc") || !utf8.ValidString(results[0].Snippet) {
		t.Fatalf("invalid PDF search result: %+v", results)
	}
	if results[0].PDFText != "" {
		t.Fatal("raw PDF text leaked in search response")
	}
}
