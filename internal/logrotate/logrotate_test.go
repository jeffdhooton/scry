package logrotate

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func write(t *testing.T, path string, n int) {
	t.Helper()
	if err := os.WriteFile(path, make([]byte, n), 0o644); err != nil {
		t.Fatal(err)
	}
}

// TestOpenAppendRotatesAtLimit is the regression test for the 296 MB
// scryd.log: nothing bounded an append-only log, so it grew until someone
// noticed.
func TestOpenAppendRotatesAtLimit(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "calls.jsonl")
	write(t, path, 100)

	f, err := OpenAppend(path, 100, 3, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Size() != 0 {
		t.Errorf("live log should restart empty after rotation, got %d bytes", info.Size())
	}
	rotated, err := os.Stat(path + ".1")
	if err != nil {
		t.Fatalf("expected rotated generation at %s.1: %v", path, err)
	}
	if rotated.Size() != 100 {
		t.Errorf("rotated generation should hold the old content, got %d bytes", rotated.Size())
	}
}

func TestOpenAppendLeavesSmallLogAlone(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "calls.jsonl")
	write(t, path, 10)

	f, err := OpenAppend(path, 100, 3, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	if _, err := os.Stat(path + ".1"); !os.IsNotExist(err) {
		t.Error("a log under the limit must not rotate")
	}
	info, _ := os.Stat(path)
	if info.Size() != 10 {
		t.Errorf("existing content must be preserved, got %d bytes", info.Size())
	}
}

// TestOpenAppendDropsOldestGeneration keeps rotation from becoming its own
// disk leak.
func TestOpenAppendDropsOldestGeneration(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "calls.jsonl")

	for range 5 {
		write(t, path, 100)
		f, err := OpenAppend(path, 100, 2, 0o644)
		if err != nil {
			t.Fatal(err)
		}
		f.Close()
	}

	if _, err := os.Stat(path + ".3"); !os.IsNotExist(err) {
		t.Error("keep=2 must not leave a third generation behind")
	}
	for i := 1; i <= 2; i++ {
		if _, err := os.Stat(fmt.Sprintf("%s.%d", path, i)); err != nil {
			t.Errorf("expected generation %d to exist: %v", i, err)
		}
	}
}
