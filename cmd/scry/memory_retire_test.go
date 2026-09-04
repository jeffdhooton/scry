package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCreateOfflineRetirementBackupNeverOverwrites(t *testing.T) {
	path := filepath.Join(t.TempDir(), "backups", "memory-pre-retirement-fixed.badger")
	firstPath, first, err := createOfflineRetirementBackup(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := first.Write([]byte("FIRST_ROLLBACK")); err != nil {
		t.Fatal(err)
	}
	if err := first.Close(); err != nil {
		t.Fatal(err)
	}
	secondPath, second, err := createOfflineRetirementBackup(path)
	if err != nil {
		t.Fatal(err)
	}
	if firstPath == secondPath {
		t.Fatalf("offline backup path was reused: %s", firstPath)
	}
	if err := second.Close(); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(firstPath)
	if err != nil || string(got) != "FIRST_ROLLBACK" {
		t.Fatalf("first rollback was overwritten: %q err=%v", got, err)
	}
}
