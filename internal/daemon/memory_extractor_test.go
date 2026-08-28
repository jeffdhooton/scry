package daemon

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jeffdhooton/scry/internal/memory/extract"
)

func writeConfig(t *testing.T, home, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(home, "config.yaml"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestBuildMemoryExtractorUsesConfigChain(t *testing.T) {
	t.Setenv("SCRY_MEMORY_API_KEY", "k")
	t.Setenv("SCRY_MEMORY_MODEL", "env-model")
	home := t.TempDir()
	writeConfig(t, home, "memory:\n  models:\n    - model: deepseek-v4-flash\n    - model: deepseek-v4-pro\n")

	ex := buildMemoryExtractor(home)

	ch, ok := ex.(*extract.Chain)
	if !ok {
		t.Fatalf("buildMemoryExtractor() = %T, want *extract.Chain", ex)
	}
	if got := strings.Join(ch.Names(), ","); got != "deepseek-v4-flash,deepseek-v4-pro" {
		t.Errorf("chain = %s, want the config order with env model ignored", got)
	}
}

func TestBuildMemoryExtractorDormantOnMalformedConfig(t *testing.T) {
	t.Setenv("SCRY_MEMORY_API_KEY", "k")
	home := t.TempDir()
	writeConfig(t, home, "memory: [unclosed")

	if ex := buildMemoryExtractor(home); ex != nil {
		t.Errorf("buildMemoryExtractor() = %T, want nil (dormant) for a malformed config.yaml", ex)
	}
}

func TestBuildMemoryExtractorDormantWithoutKey(t *testing.T) {
	t.Setenv("SCRY_MEMORY_API_KEY", "")
	t.Setenv("DEEPSEEK_API_KEY", "")

	if ex := buildMemoryExtractor(t.TempDir()); ex != nil {
		t.Errorf("buildMemoryExtractor() = %T, want nil (dormant) with no key", ex)
	}
}

func TestMemoryStatusReportsModelChain(t *testing.T) {
	d := newTestMemoryDaemon(t)
	d.memExtractor = extract.NewChain(
		extract.Step{Name: "deepseek-v4-flash", Extractor: &fakeExtractor{}},
		extract.Step{Name: "deepseek-v4-pro", Extractor: &fakeExtractor{}},
	)

	res, err := d.handleMemoryStatus(context.Background(), nil)
	if err != nil {
		t.Fatalf("handleMemoryStatus: %v", err)
	}
	st := res.(*MemoryStatusResult)
	if got := strings.Join(st.Models, ","); got != "deepseek-v4-flash,deepseek-v4-pro" {
		t.Errorf("Models = %v, want the chain in order", st.Models)
	}
	if st.Dormant {
		t.Error("Dormant = true with a chain configured")
	}
}
