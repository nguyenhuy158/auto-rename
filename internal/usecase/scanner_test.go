package usecase

import (
	"os"
	"path/filepath"
	"testing"

	"auto-rename/internal/config"
)

func TestMoveNoReplacePreservesBothFilesOnConflict(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "source.mp4")
	target := filepath.Join(dir, "target.mp4")
	if err := os.WriteFile(source, []byte("source"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte("target"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := moveNoReplace(source, target); err == nil {
		t.Fatal("expected conflict")
	}
	original, err := os.ReadFile(source)
	if err != nil {
		t.Fatal(err)
	}
	existing, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(original) != "source" || string(existing) != "target" {
		t.Fatal("file content changed on conflict")
	}
}

func TestMoveNoReplaceKeepsContent(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "source.mp4")
	target := filepath.Join(dir, "target.mp4")
	if err := os.WriteFile(source, []byte("video bytes"), 0600); err != nil {
		t.Fatal(err)
	}
	before, err := os.Stat(source)
	if err != nil {
		t.Fatal(err)
	}
	if err := moveNoReplace(source, target); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(source); !os.IsNotExist(err) {
		t.Fatalf("source still exists: %v", err)
	}
	after, err := os.Stat(target)
	if err != nil {
		t.Fatal(err)
	}
	if !os.SameFile(before, after) {
		t.Fatal("file identity changed")
	}
	data, err := os.ReadFile(target)
	if err != nil || string(data) != "video bytes" {
		t.Fatal("file content changed")
	}
}

func TestScannerRejectsQuarantineInsideSource(t *testing.T) {
	root := t.TempDir()
	_, err := NewScanner(config.Config{Dir: root, QuarantineDir: filepath.Join(root, "quarantine")}, nil)
	if err == nil {
		t.Fatal("quarantine inside source was accepted")
	}
}
