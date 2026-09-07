package genCURD

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGenerationSessionRollsBackNewAndChangedFiles(t *testing.T) {
	root := t.TempDir()
	oldPath := filepath.Join(root, "internal", "demoService", "data", "z_dao_Demo.gen.go")
	newPath := filepath.Join(root, "proto", "z_demoService.gen.proto")
	if err := os.MkdirAll(filepath.Dir(oldPath), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(oldPath, []byte("old"), 0644); err != nil {
		t.Fatal(err)
	}
	session, err := newGenerationSession(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(oldPath, []byte("changed"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(newPath), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(newPath, []byte("new"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := session.rollback(); err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(oldPath)
	if err != nil || string(content) != "old" {
		t.Fatalf("old generated file was not restored: %q, %v", content, err)
	}
	if _, err = os.Stat(newPath); !os.IsNotExist(err) {
		t.Fatalf("new generated file remains after rollback: %v", err)
	}
}

func TestGenerationSessionCommitRemovesStaleGeneratedFiles(t *testing.T) {
	root := t.TempDir()
	stalePath := filepath.Join(root, "internal", "demoService", "z_stale.gen.go")
	if err := os.MkdirAll(filepath.Dir(stalePath), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(stalePath, []byte("old"), 0644); err != nil {
		t.Fatal(err)
	}
	session, err := newGenerationSession(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(stalePath); err != nil {
		t.Fatal(err)
	}
	if err := session.commit(); err != nil {
		t.Fatal(err)
	}
	if _, err = os.Stat(stalePath); !os.IsNotExist(err) {
		t.Fatalf("stale generated file remains after commit: %v", err)
	}
}
