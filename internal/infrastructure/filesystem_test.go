package infrastructure

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestResolveWithinRejectsTraversalAndSymlink(t *testing.T) {
	parent := t.TempDir()
	root := filepath.Join(parent, "data")
	sibling := filepath.Join(parent, "data-private")
	if err := os.Mkdir(root, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(sibling, 0700); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(sibling, "secret.mp4")
	if err := os.WriteFile(outside, []byte("private"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := ResolveWithin(root, "../data-private/secret.mp4"); !errors.Is(err, os.ErrPermission) {
		t.Fatalf("traversal was accepted: %v", err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "link.mp4")); err != nil {
		t.Fatal(err)
	}
	if _, err := ResolveWithin(root, "link.mp4"); !errors.Is(err, os.ErrPermission) {
		t.Fatalf("symlink escape was accepted: %v", err)
	}
}
