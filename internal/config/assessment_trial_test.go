package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestAssessmentTrialConfigurationIsStrictAndIndependent(t *testing.T) {
	for _, tc := range []struct {
		name, trial string
		valid       bool
	}{
		{"valid", "starts_at: 2026-09-20T15:00:00Z\n      expires_at: 2026-09-21T15:00:00Z\n      max_requests: 100", true},
		{"missing cap", "starts_at: 2026-09-20T15:00:00Z\n      expires_at: 2026-09-21T15:00:00Z", false},
		{"reversed", "starts_at: 2026-09-21T15:00:00Z\n      expires_at: 2026-09-20T15:00:00Z\n      max_requests: 100", false},
		{"unknown", "SECRET: value", false},
		{"invalid date", "starts_at: SECRET\n      expires_at: 2026-09-21T15:00:00Z\n      max_requests: 100", false},
		{"duplicate", "starts_at: 2026-09-20T15:00:00Z\n      starts_at: 2026-09-20T15:00:00Z\n      expires_at: 2026-09-21T15:00:00Z\n      max_requests: 100", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			home := t.TempDir()
			raw := "memory:\n  models:\n    - model: existing-extractor\n  assessment:\n    mode: shadow\n    trial:\n      " + tc.trial + "\n"
			if e := os.WriteFile(filepath.Join(home, FileName), []byte(raw), 0600); e != nil {
				t.Fatal(e)
			}
			c, e := Load(home)
			if e != nil {
				t.Fatalf("trial disables extraction: %v", e)
			}
			if len(c.Memory.Models) != 1 {
				t.Fatal("lost extractor")
			}
			a, e := c.Memory.Assessment.Effective()
			if tc.valid {
				if e != nil || a.Trial == nil || a.Trial.MaxRequests != 100 || a.Trial.ExpiresAt.Sub(a.Trial.StartsAt) != 24*time.Hour {
					t.Fatalf("invalid parsed trial: %+v %v", a, e)
				}
			} else if e == nil || !strings.Contains(e.Error(), "assessment.trial") || strings.Contains(e.Error(), "SECRET") {
				t.Fatalf("unsafe/missing diagnostic: %v", e)
			}
		})
	}
}

func TestAssessmentUnknownDuplicateDoesNotEchoSecret(t *testing.T) {
	home := t.TempDir()
	os.WriteFile(filepath.Join(home, FileName), []byte("memory:\n  assessment:\n    SECRET: a\n    SECRET: b\n"), 0600)
	c, e := Load(home)
	if e != nil {
		t.Fatal(e)
	}
	_, e = c.Memory.Assessment.Effective()
	if e == nil || strings.Contains(e.Error(), "SECRET") {
		t.Fatalf("unsafe diagnostic: %v", e)
	}
}
