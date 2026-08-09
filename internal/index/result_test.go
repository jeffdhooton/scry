package index

import (
	"fmt"
	"testing"

	"github.com/jeffdhooton/scry/internal/sources/golang"
	"github.com/jeffdhooton/scry/internal/sources/python"
)

func TestClassify(t *testing.T) {
	tests := []struct {
		name       string
		language   string
		err        error
		wantStatus string
		wantRemedy string
	}{
		{
			name:       "nil error is ok",
			language:   "python",
			err:        nil,
			wantStatus: IndexerOK,
			wantRemedy: "",
		},
		{
			name:       "wrapped not-found sentinel is missing, with remedy",
			language:   "python",
			err:        fmt.Errorf("index python: %w", python.ErrIndexerNotFound),
			wantStatus: IndexerMissing,
			wantRemedy: "npm i -g @sourcegraph/scip-python",
		},
		{
			name:       "go sentinel is missing",
			language:   "go",
			err:        fmt.Errorf("run: %w", golang.ErrIndexerNotFound),
			wantStatus: IndexerMissing,
			wantRemedy: "install scip-go manually: go install github.com/sourcegraph/scip-go/cmd/scip-go@latest",
		},
		{
			name:       "arbitrary error is failed, no remedy",
			language:   "go",
			err:        fmt.Errorf("exit status 2: panic in scip-go"),
			wantStatus: IndexerFailed,
			wantRemedy: "",
		},
		{
			name:       "unknown language with arbitrary error is failed",
			language:   "ruby",
			err:        fmt.Errorf("boom"),
			wantStatus: IndexerFailed,
			wantRemedy: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, errMsg, remedy := classify(tt.language, tt.err)
			if status != tt.wantStatus {
				t.Errorf("status = %q, want %q", status, tt.wantStatus)
			}
			if remedy != tt.wantRemedy {
				t.Errorf("remedy = %q, want %q", remedy, tt.wantRemedy)
			}
			if tt.err == nil {
				if errMsg != "" {
					t.Errorf("errMsg = %q, want empty for nil error", errMsg)
				}
			} else if errMsg != tt.err.Error() {
				t.Errorf("errMsg = %q, want %q", errMsg, tt.err.Error())
			}
		})
	}
}

func TestDeriveStatus(t *testing.T) {
	tests := []struct {
		name    string
		results []IndexerResult
		want    string
	}{
		{
			name: "all primary ok is ready",
			results: []IndexerResult{
				{Language: "php", Tier: TierPrimary, Status: IndexerOK},
				{Language: "typescript", Tier: TierPrimary, Status: IndexerOK},
			},
			want: "ready",
		},
		{
			name: "primary missing is partial",
			results: []IndexerResult{
				{Language: "php", Tier: TierPrimary, Status: IndexerOK},
				{Language: "python", Tier: TierPrimary, Status: IndexerMissing},
			},
			want: "partial",
		},
		{
			name: "primary failed is partial",
			results: []IndexerResult{
				{Language: "php", Tier: TierPrimary, Status: IndexerOK},
				{Language: "go", Tier: TierPrimary, Status: IndexerFailed},
			},
			want: "partial",
		},
		{
			name: "incidental skipped alongside ok primaries is ready",
			results: []IndexerResult{
				{Language: "php", Tier: TierPrimary, Status: IndexerOK},
				{Language: "python", Tier: TierIncidental, Status: IndexerSkipped},
			},
			want: "ready",
		},
		{
			name: "incidental failure never degrades",
			results: []IndexerResult{
				{Language: "php", Tier: TierPrimary, Status: IndexerOK},
				{Language: "python", Tier: TierIncidental, Status: IndexerMissing},
			},
			want: "ready",
		},
		{
			name:    "no results is ready",
			results: nil,
			want:    "ready",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := deriveStatus(tt.results); got != tt.want {
				t.Errorf("deriveStatus = %q, want %q", got, tt.want)
			}
		})
	}
}
