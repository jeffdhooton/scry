package daemon

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReviewConstructorDoesNotRecoverBeforeOwnership(t *testing.T) {
	home := t.TempDir()
	dir := filepath.Join(home, "reviews")
	if e := os.MkdirAll(dir, 0700); e != nil {
		t.Fatal(e)
	}
	body := []byte(`{"version":1,"records":[{"id":"active","state":"running"}],"daily":{"2026-09-21":1},"reserved":{},"seen":{"active":true}}`)
	p := filepath.Join(dir, "state.json")
	if e := os.WriteFile(p, body, 0600); e != nil {
		t.Fatal(e)
	}
	d := New(LayoutFor(home))
	if d.reviewService != nil {
		t.Fatal("review configured without daemon ownership")
	}
	got, e := os.ReadFile(p)
	if e != nil {
		t.Fatal(e)
	}
	if string(got) != string(body) {
		t.Fatal("competing constructor changed active journal")
	}
}
