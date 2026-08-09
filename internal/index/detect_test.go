package index

import (
	"os"
	"path/filepath"
	"testing"
)

// writeRepo materializes a fixture repo: each map key is a repo-relative
// path, each value is that file's contents.
func writeRepo(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	for rel, content := range files {
		full := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", filepath.Dir(full), err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", full, err)
		}
	}
	return root
}

// nFiles generates n files named <prefix>N<ext> under dir.
func nFiles(dir, prefix, ext string, n int) map[string]string {
	out := map[string]string{}
	for i := 0; i < n; i++ {
		out[filepath.Join(dir, prefix+string(rune('a'+i%26))+string(rune('a'+i/26))+ext)] = "x"
	}
	return out
}

func merge(maps ...map[string]string) map[string]string {
	out := map[string]string{}
	for _, m := range maps {
		for k, v := range m {
			out[k] = v
		}
	}
	return out
}

// find returns the DetectedLanguage for lang, or a zero value if absent.
func find(dets []DetectedLanguage, lang string) DetectedLanguage {
	for _, d := range dets {
		if d.Language == lang {
			return d
		}
	}
	return DetectedLanguage{}
}

func TestDetectLanguages_MarkerPromotesLowShare(t *testing.T) {
	// 95 php files, 5 python files (5%), but a pyproject.toml declares
	// Python as a real component.
	root := writeRepo(t, merge(
		nFiles("app", "f", ".php", 95),
		nFiles("scripts", "g", ".py", 5),
		map[string]string{"pyproject.toml": "[project]\nname='x'\n"},
	))

	dets, err := detectLanguages(root)
	if err != nil {
		t.Fatalf("detectLanguages: %v", err)
	}
	py := find(dets, "python")
	if py.Tier != TierPrimary {
		t.Errorf("python tier = %q, want %q (marker file should promote)", py.Tier, TierPrimary)
	}
	if py.Marker != "pyproject.toml" {
		t.Errorf("python marker = %q, want pyproject.toml", py.Marker)
	}
	if py.FileCount != 5 {
		t.Errorf("python file count = %d, want 5", py.FileCount)
	}
}

func TestDetectLanguages_NoMarkerLowShareIsIncidental(t *testing.T) {
	// The childscribe-beta-r4 shape: a dominant PHP app with a handful of
	// stray .py scripts and no Python marker anywhere.
	root := writeRepo(t, merge(
		nFiles("app", "f", ".php", 95),
		nFiles("scripts", "g", ".py", 5),
		map[string]string{"composer.json": "{}"},
	))

	dets, err := detectLanguages(root)
	if err != nil {
		t.Fatalf("detectLanguages: %v", err)
	}
	if got := find(dets, "python").Tier; got != TierIncidental {
		t.Errorf("python tier = %q, want %q", got, TierIncidental)
	}
	if got := find(dets, "php").Tier; got != TierPrimary {
		t.Errorf("php tier = %q, want %q", got, TierPrimary)
	}
}

func TestDetectLanguages_HighShareWithoutMarkerIsPrimary(t *testing.T) {
	// 70 go files, 30 python files (30%), no python marker. Undeclared but
	// substantial code still deserves an index.
	root := writeRepo(t, merge(
		nFiles("cmd", "f", ".go", 70),
		nFiles("tools", "g", ".py", 30),
		map[string]string{"go.mod": "module x\n"},
	))

	dets, err := detectLanguages(root)
	if err != nil {
		t.Fatalf("detectLanguages: %v", err)
	}
	py := find(dets, "python")
	if py.Tier != TierPrimary {
		t.Errorf("python tier = %q, want %q (30%% share should promote)", py.Tier, TierPrimary)
	}
	if py.Marker != "" {
		t.Errorf("python marker = %q, want empty (share-promoted, not marker-promoted)", py.Marker)
	}
}

func TestDetectLanguages_BelowMinShareIsAbsent(t *testing.T) {
	// 1 python file in 200 = 0.5%, below the 1% floor.
	root := writeRepo(t, merge(
		nFiles("app", "f", ".php", 199),
		map[string]string{"one.py": "x", "composer.json": "{}"},
	))

	dets, err := detectLanguages(root)
	if err != nil {
		t.Fatalf("detectLanguages: %v", err)
	}
	if find(dets, "python").Language != "" {
		t.Errorf("python should be absent entirely below the 1%% floor, got %+v", find(dets, "python"))
	}
}

func TestDetectLanguages_MarkerDoesNotResurrectBelowFloor(t *testing.T) {
	// A marker file cannot promote a language with no source files at all.
	// Laravel apps ship a package.json but may have zero .ts/.js of their own.
	root := writeRepo(t, merge(
		nFiles("app", "f", ".php", 100),
		map[string]string{"composer.json": "{}", "package.json": "{}"},
	))

	dets, err := detectLanguages(root)
	if err != nil {
		t.Fatalf("detectLanguages: %v", err)
	}
	if find(dets, "typescript").Language != "" {
		t.Error("typescript should be absent with zero .ts files despite package.json")
	}
	if find(dets, "javascript").Language != "" {
		t.Error("javascript should be absent with zero .js files despite package.json")
	}
}

func TestDetectLanguages_SkipsVendorAndVenv(t *testing.T) {
	root := writeRepo(t, merge(
		nFiles("app", "f", ".php", 100),
		nFiles("vendor/pkg", "v", ".php", 50),
		nFiles(".venv/lib", "w", ".py", 500),
		nFiles("node_modules/dep", "n", ".js", 500),
		map[string]string{"composer.json": "{}"},
	))

	dets, err := detectLanguages(root)
	if err != nil {
		t.Fatalf("detectLanguages: %v", err)
	}
	if find(dets, "python").Language != "" {
		t.Error("python from .venv must not be counted")
	}
	if find(dets, "javascript").Language != "" {
		t.Error("javascript from node_modules must not be counted")
	}
	if got := find(dets, "php").FileCount; got != 100 {
		t.Errorf("php file count = %d, want 100 (vendor/ excluded)", got)
	}
}

func TestPrimaryLanguages(t *testing.T) {
	dets := []DetectedLanguage{
		{Language: "php", Tier: TierPrimary},
		{Language: "python", Tier: TierIncidental},
		{Language: "typescript", Tier: TierPrimary},
	}
	got := primaryLanguages(dets)
	want := []string{"php", "typescript"}
	if len(got) != len(want) {
		t.Fatalf("primaryLanguages = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("primaryLanguages = %v, want %v", got, want)
		}
	}
}
