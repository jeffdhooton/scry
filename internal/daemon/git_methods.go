package daemon

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/jeffdhooton/scry/internal/git"
	gitindex "github.com/jeffdhooton/scry/internal/git/index"
	gitstore "github.com/jeffdhooton/scry/internal/git/store"
	"github.com/jeffdhooton/scry/internal/rpc"
)

func (d *Daemon) registerGitMethods() {
	d.server.Register("git.init", d.handleGitInit)
	d.server.Register("git.blame", d.handleGitBlame)
	d.server.Register("git.history", d.handleGitHistory)
	d.server.Register("git.cochange", d.handleGitCochange)
	d.server.Register("git.hotspots", d.handleGitHotspots)
	d.server.Register("git.contributors", d.handleGitContributors)
	d.server.Register("git.intent", d.handleGitIntent)
}

// --- git.init ---

type GitInitParams struct {
	Repo  string `json:"repo"`
	Depth int    `json:"depth,omitempty"`
}

type GitInitResult struct {
	Repo      string          `json:"repo"`
	Status    string          `json:"status"`
	Stats     gitindex.Stats  `json:"stats"`
	ElapsedMs int64           `json:"elapsed_ms"`
}

func (d *Daemon) handleGitInit(ctx context.Context, raw json.RawMessage) (any, error) {
	var p GitInitParams
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, &rpc.Error{Code: rpc.CodeInvalidParams, Message: err.Error()}
	}
	if p.Repo == "" {
		return nil, &rpc.Error{Code: rpc.CodeInvalidParams, Message: "repo is required"}
	}
	abs, err := filepath.Abs(p.Repo)
	if err != nil {
		return nil, err
	}

	d.gitRegistry.Evict(abs)

	start := time.Now()
	manifest, err := gitindex.Build(ctx, d.scryHome(), abs, p.Depth)
	if err != nil {
		return nil, fmt.Errorf("git index build: %w", err)
	}
	elapsed := time.Since(start)

	layout := gitindex.Layout(d.scryHome(), abs)
	st, err := gitstore.Open(layout.BadgerDir)
	if err != nil {
		return nil, fmt.Errorf("reopen git store after init: %w", err)
	}
	d.gitRegistry.Put(&git.Entry{RepoPath: abs, Layout: layout, Store: st})

	return &GitInitResult{
		Repo:      manifest.RepoPath,
		Status:    manifest.Status,
		Stats:     manifest.Stats,
		ElapsedMs: elapsed.Milliseconds(),
	}, nil
}

// --- git.blame ---

type GitBlameParams struct {
	Repo      string `json:"repo"`
	File      string `json:"file"`
	StartLine int    `json:"start_line,omitempty"`
	EndLine   int    `json:"end_line,omitempty"`
}

func (d *Daemon) handleGitBlame(_ context.Context, raw json.RawMessage) (any, error) {
	var p GitBlameParams
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, &rpc.Error{Code: rpc.CodeInvalidParams, Message: err.Error()}
	}
	if p.Repo == "" || p.File == "" {
		return nil, &rpc.Error{Code: rpc.CodeInvalidParams, Message: "repo and file are required"}
	}
	entry, err := d.gitRegistry.Get(d.scryHome(), p.Repo)
	if err != nil {
		return nil, err
	}
	recs, err := entry.Store.GetBlame(p.File, p.StartLine, p.EndLine)
	if err != nil {
		return nil, err
	}
	if recs == nil {
		recs = []gitstore.BlameRecord{}
	}
	return recs, nil
}

// --- git.history ---

type GitHistoryParams struct {
	Repo  string `json:"repo"`
	File  string `json:"file,omitempty"`
	Limit int    `json:"limit,omitempty"`
}

func (d *Daemon) handleGitHistory(_ context.Context, raw json.RawMessage) (any, error) {
	var p GitHistoryParams
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, &rpc.Error{Code: rpc.CodeInvalidParams, Message: err.Error()}
	}
	if p.Repo == "" {
		return nil, &rpc.Error{Code: rpc.CodeInvalidParams, Message: "repo is required"}
	}
	if p.Limit <= 0 {
		p.Limit = 20
	}
	entry, err := d.gitRegistry.Get(d.scryHome(), p.Repo)
	if err != nil {
		return nil, err
	}
	var recs []gitstore.CommitRecord
	if p.File != "" {
		recs, err = entry.Store.GetFileCommits(p.File, p.Limit)
	} else {
		recs, err = entry.Store.GetRecentCommits(p.Limit)
	}
	if err != nil {
		return nil, err
	}
	if recs == nil {
		recs = []gitstore.CommitRecord{}
	}
	return recs, nil
}

// --- git.cochange ---

type GitCochangeParams struct {
	Repo  string `json:"repo"`
	File  string `json:"file"`
	Limit int    `json:"limit,omitempty"`
}

func (d *Daemon) handleGitCochange(_ context.Context, raw json.RawMessage) (any, error) {
	var p GitCochangeParams
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, &rpc.Error{Code: rpc.CodeInvalidParams, Message: err.Error()}
	}
	if p.Repo == "" || p.File == "" {
		return nil, &rpc.Error{Code: rpc.CodeInvalidParams, Message: "repo and file are required"}
	}
	if p.Limit <= 0 {
		p.Limit = 10
	}
	entry, err := d.gitRegistry.Get(d.scryHome(), p.Repo)
	if err != nil {
		return nil, err
	}
	recs, err := entry.Store.GetCochange(p.File, p.Limit)
	if err != nil {
		return nil, err
	}
	if recs == nil {
		recs = []gitstore.CochangeRecord{}
	}
	return recs, nil
}

// --- git.hotspots ---

type GitHotspotsParams struct {
	Repo  string `json:"repo"`
	Limit int    `json:"limit,omitempty"`
}

func (d *Daemon) handleGitHotspots(_ context.Context, raw json.RawMessage) (any, error) {
	var p GitHotspotsParams
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, &rpc.Error{Code: rpc.CodeInvalidParams, Message: err.Error()}
	}
	if p.Repo == "" {
		return nil, &rpc.Error{Code: rpc.CodeInvalidParams, Message: "repo is required"}
	}
	if p.Limit <= 0 {
		p.Limit = 20
	}
	entry, err := d.gitRegistry.Get(d.scryHome(), p.Repo)
	if err != nil {
		return nil, err
	}
	recs, err := entry.Store.GetHotspots(p.Limit)
	if err != nil {
		return nil, err
	}
	if recs == nil {
		recs = []gitstore.ChurnRecord{}
	}
	return recs, nil
}

// --- git.contributors ---

type GitContributorsParams struct {
	Repo string `json:"repo"`
	File string `json:"file,omitempty"`
}

func (d *Daemon) handleGitContributors(_ context.Context, raw json.RawMessage) (any, error) {
	var p GitContributorsParams
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, &rpc.Error{Code: rpc.CodeInvalidParams, Message: err.Error()}
	}
	if p.Repo == "" {
		return nil, &rpc.Error{Code: rpc.CodeInvalidParams, Message: "repo is required"}
	}
	entry, err := d.gitRegistry.Get(d.scryHome(), p.Repo)
	if err != nil {
		return nil, err
	}
	recs, err := entry.Store.GetContributors(p.File)
	if err != nil {
		return nil, err
	}
	if recs == nil {
		recs = []gitstore.ContribRecord{}
	}
	return recs, nil
}

// --- git.intent ---

type GitIntentParams struct {
	Repo string `json:"repo"`
	File string `json:"file"`
	Line int    `json:"line"`
}

type GitIntentResult struct {
	Blame  *gitstore.BlameRecord  `json:"blame"`
	Commit *gitstore.CommitRecord `json:"commit,omitempty"`
}

func (d *Daemon) handleGitIntent(_ context.Context, raw json.RawMessage) (any, error) {
	var p GitIntentParams
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, &rpc.Error{Code: rpc.CodeInvalidParams, Message: err.Error()}
	}
	if p.Repo == "" || p.File == "" || p.Line <= 0 {
		return nil, &rpc.Error{Code: rpc.CodeInvalidParams, Message: "repo, file, and line are required"}
	}
	entry, err := d.gitRegistry.Get(d.scryHome(), p.Repo)
	if err != nil {
		return nil, err
	}
	recs, err := entry.Store.GetBlame(p.File, p.Line, p.Line)
	if err != nil {
		return nil, err
	}
	if len(recs) == 0 {
		return nil, &rpc.Error{Code: rpc.CodeInternalError, Message: "no blame data for that line"}
	}
	blame := &recs[0]
	commit, _ := entry.Store.GetCommitByHash(blame.Commit)
	return &GitIntentResult{Blame: blame, Commit: commit}, nil
}

// handleGitStatus returns the git index status for all repos (used internally).
func (d *Daemon) gitStatusEntries() []map[string]any {
	var entries []map[string]any
	for _, e := range d.gitRegistry.Snapshot() {
		entry := map[string]any{"repo": e.RepoPath, "domain": "git"}
		if m, err := gitindex.LoadManifest(e.Layout); err == nil {
			entry["indexed_at"] = m.IndexedAt
			entry["stats"] = m.Stats
		}
		entries = append(entries, entry)
	}

	reposDir := filepath.Join(d.scryHome(), "repos")
	dirs, _ := os.ReadDir(reposDir)
	seen := map[string]bool{}
	for _, e := range d.gitRegistry.Snapshot() {
		seen[e.RepoPath] = true
	}
	for _, dir := range dirs {
		gitManifest := filepath.Join(reposDir, dir.Name(), "git", "manifest.json")
		if m, err := gitindex.LoadManifest(gitindex.RepoLayout{ManifestPath: gitManifest}); err == nil {
			if !seen[m.RepoPath] {
				entries = append(entries, map[string]any{
					"repo":       m.RepoPath,
					"domain":     "git",
					"indexed_at": m.IndexedAt,
					"stats":      m.Stats,
				})
			}
		}
	}
	return entries
}
