package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/jeffdhooton/scry/internal/doctor"
)

func TestPrintDoctorReportClipsStderrAndPointsToJSON(t *testing.T) {
	report := &doctor.Report{Checks: []doctor.Check{{
		ID:       "repo.current",
		Category: doctor.CategoryRepo,
		Name:     "index state",
		Status:   doctor.StatusWarn,
		Detail:   "typescript failed",
		Stderr:   "first\nsecond\nthird\nfourth\nfifth",
	}}}
	var out bytes.Buffer
	printDoctorReport(&out, report)
	got := out.String()

	if strings.Contains(got, "first") || strings.Contains(got, "second") {
		t.Fatalf("human output includes clipped stderr lines:\n%s", got)
	}
	for _, want := range []string{"third", "fourth", "fifth", "scry doctor --json"} {
		if !strings.Contains(got, want) {
			t.Fatalf("human output missing %q:\n%s", want, got)
		}
	}
}
