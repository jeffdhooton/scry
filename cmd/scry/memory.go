package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/jeffdhooton/scry/internal/config"
	"github.com/jeffdhooton/scry/internal/daemon"
	"github.com/jeffdhooton/scry/internal/memory/browse"
	"github.com/jeffdhooton/scry/internal/memory/distill"
	"github.com/jeffdhooton/scry/internal/memory/extract"
	"github.com/jeffdhooton/scry/internal/memory/ingest"
	"github.com/jeffdhooton/scry/internal/memory/migrate"
	"github.com/jeffdhooton/scry/internal/memory/recall"
	"github.com/jeffdhooton/scry/internal/memory/resolve"
	memstore "github.com/jeffdhooton/scry/internal/memory/store"
	"github.com/jeffdhooton/scry/internal/memory/sweep"
)

// dormantNotice is printed (with exit 0, not an error) by ingest/sweep/
// backfill when no API key is set — the memory domain is opt-in, so an
// unconfigured key is a no-op, not a failure.
const dormantNotice = "memory: dormant (no SCRY_MEMORY_API_KEY / DEEPSEEK_API_KEY)"

// memoryProviders resolves the extraction chain the same way the daemon
// does (buildMemoryExtractor in internal/daemon/daemon.go): ~/.scry/
// config.yaml's memory.models first, the environment otherwise. The CLI's
// ingest pipeline and the daemon's memory.remember must agree on which
// models are live and when the domain is dormant.
func memoryProviders() (extract.Providers, error) {
	home, err := scryHome()
	if err != nil {
		return extract.Providers{}, err
	}
	cfg, err := config.Load(home)
	if err != nil {
		return extract.Providers{}, err
	}
	return extract.ResolveProviders(cfg), nil
}

// memoryExtractor builds the CLI's extractor, or returns (nil, nil) after
// printing dormantNotice when no key is configured.
func memoryExtractor() (extract.Providers, extract.Extractor, error) {
	ps, err := memoryProviders()
	if err != nil {
		return ps, nil, err
	}
	if ps.Dormant() {
		fmt.Println(dormantNotice)
		return ps, nil, nil
	}
	if err := ps.Validate(); err != nil {
		return ps, nil, err
	}
	return ps, extract.NewExtractor(ps), nil
}

// memoryCmd is the `scry memory` command tree: global episodic memory graph
// (as opposed to the per-repo code intelligence commands elsewhere in this
// package).
func memoryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "memory",
		Short: "Global episodic memory graph",
		Long: `Global episodic memory graph.

The daemon also permanently serves a live, always-fresh version of the
"browse" visualization at http://127.0.0.1:7279 (loopback only) while it is
running. Set SCRY_MEMORY_UI_ADDR to change the address, or "off" to disable
it.`,
	}
	cmd.AddCommand(memoryIngestCmd(), memorySweepCmd(), memoryBackfillCmd(),
		memoryOrientCmd(), memoryRecallCmd(), memoryRememberCmd(), memoryEntitiesCmd(),
		memoryFactsCmd(), memoryInvalidateCmd(), memoryStatusCmd(), memoryBrowseCmd(),
		memoryHygieneCmd(), memoryDescribeCmd(), memoryQueueCmd(), memoryBackupCmd(), memoryRestoreCmd(), memoryMigrateCmd(), memoryBenchCmd(), memoryRepairReposCmd())
	return cmd
}

// --- migrate ---

func memoryMigrateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "migrate",
		Short: "Apply the current resolver rules to the whole store (closed vocabulary, values as attributes, alias hygiene)",
		Long: `Rewrites every fact's relation onto the closed vocabulary, turns facts
whose endpoint is a value (a status word, a measurement, a branch name)
into attributes and retires those entities, and runs alias hygiene:
reference words dropped, aliases split away from entities of another
type, facts reattached to the entity their text names, self-loops
invalidated.

Defaults to a dry run that prints the report. --apply takes a Badger
backup first (into ~/.scry/backups via the daemon, or into <dir>/../backups
with --dir) and then writes. Runs inside the daemon that owns the store;
--dir runs offline against a store directory instead (the daemon must not
hold it).`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			apply, _ := cmd.Flags().GetBool("apply")
			dir, _ := cmd.Flags().GetString("dir")
			pretty, _ := cmd.Flags().GetBool("pretty")
			if dir != "" {
				st, err := memstore.Open(dir)
				if err != nil {
					return fmt.Errorf("open %s (is a daemon holding it?): %w", dir, err)
				}
				defer st.Close()
				rep, err := migrate.Run(st, migrate.Options{
					DryRun: !apply,
					Backup: func() (string, error) {
						path := filepath.Join(filepath.Dir(dir), "backups", "memory-pre-migrate-"+time.Now().UTC().Format("20060102T150405Z")+".badger")
						if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
							return "", err
						}
						f, err := os.Create(path)
						if err != nil {
							return "", err
						}
						if _, err := st.Backup(f); err != nil {
							f.Close()
							return "", err
						}
						return path, f.Close()
					},
					Logf: func(format string, a ...any) { fmt.Fprintf(os.Stderr, format+"\n", a...) },
				})
				if err != nil {
					return err
				}
				return printJSON(rep, pretty)
			}
			ctx, cancel := context.WithTimeout(context.Background(), 60*time.Minute)
			defer cancel()
			var rep migrate.Report
			if err := callMemoryDaemon(ctx, "memory.migrate", &daemon.MemoryMigrateParams{DryRun: !apply}, &rep); err != nil {
				return err
			}
			return printJSON(rep, pretty)
		},
	}
	cmd.Flags().Bool("apply", false, "write the changes after taking a backup (default is a dry run)")
	cmd.Flags().String("dir", "", "run offline against this store directory instead of the daemon")
	return cmd
}

// --- repair-repos ---

// repairRepoBatch is how many episode→cwd pairs ride in one RPC; the
// daemon does one full fact scan per call, so bigger batches are cheaper.
const repairRepoBatch = 4000

func memoryRepairReposCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "repair-repos",
		Short: "Re-attach repository refs to entities from the transcripts, without asking a model anything",
		Long: `Walks the same roots as the sweep, re-distills each transcript locally
(no extraction, no cost), and tells the daemon which repository each
episode ran in. Entities touched by those episodes gain the ref, so
"scry memory orient" in a repo surfaces what happened there.

Needed once because repo refs used to be recorded only when the working
directory existed on the machine resolving the episode, which since the
store moved to a shared daemon is the wrong machine.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			pretty, _ := cmd.Flags().GetBool("pretty")
			dryRun, _ := cmd.Flags().GetBool("dry-run")
			claudeFiles, codexFiles, loomDirs, kimiFiles, ocRefs, errs := sweep.AllCandidates(sweep.Roots{}, 0)
			refs := map[string]string{}
			add := func(eps []distill.RawEpisode) {
				for _, ep := range eps {
					if ep.CwdIsRepo && ep.Cwd != "" {
						refs[ep.ID] = ep.Cwd
					}
				}
			}
			for _, f := range claudeFiles {
				if eps, _, err := distill.ClaudeSession(f, 0); err == nil {
					add(eps)
				} else {
					errs = append(errs, f+": "+err.Error())
				}
			}
			for _, f := range codexFiles {
				if eps, _, err := distill.CodexRollout(f, 0); err == nil {
					add(eps)
				} else {
					errs = append(errs, f+": "+err.Error())
				}
			}
			for _, f := range kimiFiles {
				if eps, _, err := distill.KimiWire(f, 0); err == nil {
					add(eps)
				} else {
					errs = append(errs, f+": "+err.Error())
				}
			}
			for _, d := range loomDirs {
				if eps, err := distill.LoomRun(d); err == nil {
					add(eps)
				}
			}
			for _, ref := range ocRefs {
				db, id, ok := distill.ParseOpenCodeRef(ref)
				if !ok {
					continue
				}
				if eps, _, err := distill.OpenCodeSessionEpisodes(db, id); err == nil {
					add(eps)
				}
			}
			host, _ := os.Hostname()
			out := map[string]any{"host": host, "episodes_with_repo_cwd": len(refs), "errors": len(errs)}
			if dryRun {
				return printJSON(out, pretty)
			}

			ids := make([]string, 0, len(refs))
			for id := range refs {
				ids = append(ids, id)
			}
			total := daemon.MemoryRepairRepoRefsResult{}
			for start := 0; start < len(ids); start += repairRepoBatch {
				end := min(start+repairRepoBatch, len(ids))
				batch := make(map[string]string, end-start)
				for _, id := range ids[start:end] {
					batch[id] = refs[id]
				}
				ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
				var res daemon.MemoryRepairRepoRefsResult
				err := callMemoryDaemon(ctx, "memory.repairRepoRefs", &daemon.MemoryRepairRepoRefsParams{Refs: batch}, &res)
				cancel()
				if err != nil {
					return err
				}
				total.EpisodesKnown += res.EpisodesKnown
				total.EpisodesUpdated += res.EpisodesUpdated
				total.PendingUpdated += res.PendingUpdated
				total.EntitiesUpdated += res.EntitiesUpdated
				total.RefsAdded += res.RefsAdded
			}
			out["result"] = total
			return printJSON(out, pretty)
		},
	}
	cmd.Flags().Bool("dry-run", false, "report how many episodes have a repository cwd without telling the daemon")
	return cmd
}

// --- bench ---

func memoryBenchCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "bench",
		Short: "Score recall against a questions file: is the answering fact in the top N?",
		Long: `Reads a JSON array of {"question", "expect"} items, where expect names the
answering fact by src/relation/dst/value, a fact_substring, or an episode
id, runs each question through recall, and reports how many answers landed
in the top N along with payload sizes. Runs against the daemon by default;
--dir runs offline against a store directory with a fresh index.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			file, _ := cmd.Flags().GetString("file")
			top, _ := cmd.Flags().GetInt("top")
			dir, _ := cmd.Flags().GetString("dir")
			pretty, _ := cmd.Flags().GetBool("pretty")
			qs, err := recall.LoadQuestions(file)
			if err != nil {
				return err
			}
			var rec recall.Recaller
			if dir != "" {
				st, err := memstore.Open(dir)
				if err != nil {
					return fmt.Errorf("open %s: %w", dir, err)
				}
				defer st.Close()
				if rec, err = recall.OfflineRecaller(st); err != nil {
					return err
				}
			} else {
				rec = func(q string) (recall.Result, error) {
					ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
					defer cancel()
					var res recall.Result
					err := callMemoryDaemon(ctx, "memory.recall", &daemon.MemoryRecallParams{Query: q, Limit: top}, &res)
					return res, err
				}
			}
			res, err := recall.Bench(qs, rec, top)
			if err != nil {
				return err
			}
			return printJSON(res, pretty)
		},
	}
	cmd.Flags().String("file", "", "questions JSON file (required)")
	cmd.Flags().Int("top", recall.DefaultFactLimit, "the answer must be within the first N facts")
	cmd.Flags().String("dir", "", "run offline against this store directory")
	_ = cmd.MarkFlagRequired("file")
	return cmd
}

// --- backup / restore ---

func memoryBackupCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "backup",
		Short: "Stream the live memory store to a Badger backup file on the daemon's machine",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			out, _ := cmd.Flags().GetString("out")
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
			defer cancel()
			var result daemon.MemoryBackupResult
			if err := callMemoryDaemon(ctx, "memory.backup", &daemon.MemoryBackupParams{Path: out}, &result); err != nil {
				return err
			}
			pretty, _ := cmd.Flags().GetBool("pretty")
			return printJSON(result, pretty)
		},
	}
	cmd.Flags().String("out", "", "destination path on the daemon's machine (default ~/.scry/backups/memory-<utc>.badger)")
	return cmd
}

func memoryRestoreCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "restore",
		Short: "Wipe the local memory store and load a backup into it (daemon must be stopped)",
		Long: `Opens ~/.scry/memory directly, drops everything, and loads the backup file.
The daemon holds the store's lock while it runs, so stop it first
(launchctl bootout, or kill the scry start --foreground process) and
start it again afterwards.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			from, _ := cmd.Flags().GetString("from")
			dir, _ := cmd.Flags().GetString("dir")
			if dir == "" {
				home, err := scryHome()
				if err != nil {
					return err
				}
				dir = filepath.Join(home, "memory")
			}
			f, err := os.Open(from)
			if err != nil {
				return err
			}
			defer f.Close()
			st, err := memstore.Open(dir)
			if err != nil {
				return fmt.Errorf("open %s (is the daemon stopped?): %w", dir, err)
			}
			defer st.Close()

			// The store about to be wiped is itself backed up first, so a
			// restore from the wrong file is reversible.
			home, err := scryHome()
			if err != nil {
				return err
			}
			pre := filepath.Join(home, "backups", "memory-pre-restore-"+time.Now().UTC().Format("20060102T150405Z")+".badger")
			if err := os.MkdirAll(filepath.Dir(pre), 0o755); err != nil {
				return err
			}
			pf, err := os.Create(pre)
			if err != nil {
				return err
			}
			if _, err := st.Backup(pf); err != nil {
				pf.Close()
				return fmt.Errorf("backup before restore: %w", err)
			}
			if err := pf.Close(); err != nil {
				return err
			}

			if err := st.Restore(f); err != nil {
				return err
			}
			episodes, entities, facts, _ := st.Counts()
			pretty, _ := cmd.Flags().GetBool("pretty")
			return printJSON(map[string]any{
				"restored_from": from, "dir": dir, "previous_store_backup": pre,
				"episodes": episodes, "entities": entities, "facts": facts,
			}, pretty)
		},
	}
	cmd.Flags().String("from", "", "backup file written by `scry memory backup` (required)")
	cmd.Flags().String("dir", "", "store directory (default ~/.scry/memory)")
	_ = cmd.MarkFlagRequired("from")
	return cmd
}

// --- queue ---

func memoryQueueCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "queue",
		Short: "Show episodes waiting for extraction at the daemon",
		Long: `Every memory write (scry_remember, the sweep, ingest) lands in the daemon's
queue first and is extracted in the background. This lists what is waiting:
ready items, items backing off after a transport failure, and parked items
the models could not parse after three tries.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			var result daemon.MemoryQueueResult
			if err := callMemoryDaemon(ctx, "memory.queue", nil, &result); err != nil {
				return err
			}
			pretty, _ := cmd.Flags().GetBool("pretty")
			return printJSON(result, pretty)
		},
	}
	cmd.AddCommand(&cobra.Command{
		Use:   "retry [episode-id]",
		Short: "Replay a parked or backing-off item now (all of them without an id)",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			var p daemon.MemoryQueueRetryParams
			if len(args) == 1 {
				p.ID = args[0]
			}
			var result map[string]int
			if err := callMemoryDaemon(ctx, "memory.queue.retry", &p, &result); err != nil {
				return err
			}
			pretty, _ := cmd.Flags().GetBool("pretty")
			return printJSON(result, pretty)
		},
	})
	return cmd
}

// --- daemonClient: ingest.Daemon implemented over callDaemon ---

// daemonClient implements ingest.Daemon by calling the daemon's memory.*
// RPCs, using exactly the param/result structs internal/daemon/memory_methods.go
// defines (task 7's RPC contract).
type daemonClient struct {
	// force re-applies episodes the store already holds (ingest --force).
	force bool
}

var _ ingest.Daemon = daemonClient{}

func (c daemonClient) Enqueue(ctx context.Context, eps []distill.RawEpisode) (int, int, error) {
	var result daemon.MemoryEnqueueResult
	err := callMemoryDaemon(ctx, "memory.enqueue", &daemon.MemoryEnqueueParams{Episodes: eps, Force: c.force}, &result)
	return result.Queued, result.Known, err
}

func (daemonClient) SweepReport(ctx context.Context, r sweep.Report) error {
	var result map[string]any
	return callMemoryDaemon(ctx, "memory.sweepReport", &daemon.MemorySweepReport{
		Host: r.Host, FilesScanned: r.FilesScanned, FilesIngested: r.FilesIngested,
		Episodes: r.Episodes, Errors: r.Errors,
	}, &result)
}

func (daemonClient) Glossary(ctx context.Context, limit int) ([]string, error) {
	var result []string
	err := callMemoryDaemon(ctx, "memory.glossary", &daemon.MemoryGlossaryParams{Limit: limit}, &result)
	return result, err
}

func (daemonClient) Commit(ctx context.Context, ep memstore.Episode, cwd string, res extract.Result) (resolve.Stats, error) {
	var stats resolve.Stats
	err := callMemoryDaemon(ctx, "memory.commit", &daemon.MemoryCommitParams{
		Episode: ep, Cwd: cwd, CwdIsRepo: distill.CwdIsRepo(cwd), Result: res,
	}, &stats)
	return stats, err
}

func (daemonClient) GetCursor(ctx context.Context, path string) (memstore.Cursor, bool, error) {
	var result daemon.MemoryCursorGetResult
	err := callMemoryDaemon(ctx, "memory.cursor.get", &daemon.MemoryCursorGetParams{Path: path}, &result)
	return result.Cursor, result.Found, err
}

func (daemonClient) PutCursor(ctx context.Context, c memstore.Cursor) error {
	// memory.cursor.put takes the Cursor itself as params (no wrapper
	// struct) — see handleMemoryCursorPut, which unmarshals raw straight
	// into a memstore.Cursor.
	var result map[string]any
	return callMemoryDaemon(ctx, "memory.cursor.put", &c, &result)
}

// HasEpisodes reports which of ids are NOT yet committed to the store — see
// backfillDaemon and runBackfill's use of it to skip re-paying for
// extraction on episodes a previous run already ingested.
func (daemonClient) HasEpisodes(ctx context.Context, ids []string) ([]string, error) {
	var result daemon.MemoryHasEpisodesResult
	err := callMemoryDaemon(ctx, "memory.hasEpisodes", &daemon.MemoryHasEpisodesParams{IDs: ids}, &result)
	return result.Missing, err
}

// --- ingest ---

func memoryIngestCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ingest",
		Short: "Distill one transcript/run/seed source and queue it for extraction at the daemon",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			source, _ := cmd.Flags().GetString("source")
			switch source {
			case "claude", "codex", "kimi", "opencode", "loom", "seed":
			default:
				return fmt.Errorf("--source must be one of claude|codex|kimi|opencode|loom|seed, got %q", source)
			}
			path, _ := cmd.Flags().GetString("path")
			force, _ := cmd.Flags().GetBool("force")

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
			defer cancel()

			sum, err := ingest.File(ctx, ingest.Options{
				Force:  force,
				Source: source,
				Path:   path,
				Daemon: daemonClient{force: force},
			})
			if err != nil {
				return err
			}
			pretty, _ := cmd.Flags().GetBool("pretty")
			return printJSON(sum, pretty)
		},
	}
	cmd.Flags().String("source", "", "source type: claude|codex|kimi|opencode|loom|seed (required)")
	cmd.Flags().String("path", "", "path to the transcript file, run directory, or seed markdown file; for opencode, opencode:<db>:<session id> (required)")
	cmd.Flags().Bool("force", false, "re-queue episodes the store already holds so they are re-applied under the current rules (repair)")
	_ = cmd.MarkFlagRequired("source")
	_ = cmd.MarkFlagRequired("path")
	return cmd
}

// --- describe ---

func memoryDescribeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "describe <slug> <description>",
		Short: "Set an entity's description deliberately",
		Long: `The write path fills descriptions but never replaces them, so a mention can
no longer overwrite an established identity. That also means a wrong
description can only be corrected on purpose. This is how.`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			var out map[string]any
			if err := callMemoryDaemon(ctx, "memory.describe",
				map[string]any{"slug": args[0], "description": args[1]}, &out); err != nil {
				return err
			}
			fmt.Printf("%s: %s\n", args[0], args[1])
			return nil
		},
	}
	return cmd
}

// --- hygiene ---

func memoryHygieneCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "hygiene",
		Short: "Repair entities polluted by run artifacts (temp worktrees, scratch dirs, dead repo refs)",
		Long: `Cleans entity identity that the write path used to accept:

  - aliases that are temp worktrees, scratch paths, or bare hex ids
  - repo refs pointing at directories that are no longer repositories
  - repo refs beyond the per-entity cap

Entities whose own name is a run artifact are REPORTED, not deleted: facts
reference entities by slug, so removing one would orphan its facts.

Defaults to a dry run — this edits recorded history, so read it first.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			apply, _ := cmd.Flags().GetBool("apply")

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
			defer cancel()
			var rep resolve.HygieneReport
			if err := callMemoryDaemon(ctx, "memory.hygiene",
				map[string]any{"dry_run": !apply}, &rep); err != nil {
				return err
			}

			mode := "dry run"
			if apply {
				mode = "applied"
			}
			fmt.Printf("memory hygiene (%s)\n", mode)
			fmt.Printf("  entities scanned:       %d\n", rep.EntitiesScanned)
			fmt.Printf("  entities changed:       %d\n", rep.EntitiesChanged)
			fmt.Printf("  aliases dropped:        %d\n", rep.AliasesDropped)
			fmt.Printf("  aliases split:          %d\n", rep.AliasesSplit)
			fmt.Printf("  facts reattached:       %d\n", rep.FactsReattached)
			fmt.Printf("  self-loops invalidated: %d\n", rep.SelfLoopsInvalidated)
			fmt.Printf("  repo refs dropped:      %d\n", rep.RepoRefsDropped)
			fmt.Printf("  cross-type collisions:  %d\n", rep.CrossTypeCollisions)
			if len(rep.Conflated) > 0 {
				fmt.Printf("\n  %d entities carry another entity's name as an alias — the same\n"+
					"  thing recorded twice, or two things fused into one. Reported only:\n"+
					"  merging or splitting is a judgement call about which facts belong\n"+
					"  where, and no heuristic should make it silently.\n", len(rep.Conflated))
				for i, c := range rep.Conflated {
					if i == 10 {
						fmt.Printf("    ... and %d more\n", len(rep.Conflated)-10)
						break
					}
					fmt.Printf("    %-34s fused with: %s\n", c.Slug, strings.Join(c.CollidesWith, ", "))
				}
			}
			if len(rep.EphemeralEntities) > 0 {
				fmt.Printf("\n  %d entities are themselves run artifacts (reported, not deleted;\n"+
					"  their facts would be orphaned). Review before removing:\n", len(rep.EphemeralEntities))
				for i, n := range rep.EphemeralEntities {
					if i == 15 {
						fmt.Printf("    ... and %d more\n", len(rep.EphemeralEntities)-15)
						break
					}
					fmt.Printf("    %s\n", n)
				}
			}
			if !apply && (rep.EntitiesChanged > 0) {
				fmt.Printf("\nre-run with --apply to write these changes\n")
			}
			return nil
		},
	}
	cmd.Flags().Bool("apply", false, "write the changes (default is a dry run)")
	return cmd
}

// --- sweep (Task 9) / backfill (stub, filled in by Task 10) ---

func memorySweepCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sweep",
		Short: "Scan default roots (Claude/Codex/Kimi/OpenCode transcripts, loom runs) and queue new episodes at the daemon",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			dryRun, _ := cmd.Flags().GetBool("dry-run")
			if d, _ := cmd.Flags().GetDuration("per-file-timeout"); d > 0 {
				sweep.PerFileTimeout = d
			}

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
			defer cancel()

			result, err := sweep.Run(ctx, sweep.Roots{}, ingest.Options{
				Daemon: daemonClient{},
			}, sweep.DefaultActiveWindow, dryRun)
			if err != nil {
				return err
			}
			pretty, _ := cmd.Flags().GetBool("pretty")
			return printJSON(result, pretty)
		},
	}
	cmd.Flags().Bool("dry-run", false, "report what would be ingested without queueing anything")
	cmd.Flags().Duration("per-file-timeout", sweep.PerFileTimeout, "deadline for one candidate's daemon round trips")
	return cmd
}

func memoryBackfillCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "backfill",
		Short: "Backfill every episode across default roots via the Batch API (50% discount), or serially with --no-batch",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {

			sinceStr, _ := cmd.Flags().GetString("since")
			var since time.Time
			if sinceStr != "" {
				t, err := time.Parse("2006-01-02", sinceStr)
				if err != nil {
					return fmt.Errorf("--since: invalid date %q, want YYYY-MM-DD: %w", sinceStr, err)
				}
				since = t
			}
			noBatch, _ := cmd.Flags().GetBool("no-batch")

			ps, extractor, err := memoryExtractor()
			if err != nil || extractor == nil {
				return err
			}
			// The Batches API is Anthropic-only. A compatible endpoint serves
			// v1/messages but not v1/messages/batches, so a custom base URL
			// forces the serial path (and forfeits the 50% batch discount).
			// Only the chain's primary is batched; the serial path runs the
			// whole chain.
			primary := ps.Providers[0]
			if !primary.Batched() && !noBatch {
				noBatch = true
				fmt.Fprintln(os.Stderr, "backfill: extraction is not pointed at Anthropic — using serial extraction, the Batch API is Anthropic-only")
			}

			// No overall timeout — this is a long-running, potentially
			// hours-spanning job (batches can take up to 24h to end). ctx is
			// still cancellable (Ctrl-C), which both the batch poll loop and
			// the serial fallback respect.
			ctx := context.Background()

			summary, err := runBackfill(ctx, backfillConfig{
				Since:   since,
				NoBatch: noBatch,
				Daemon:  daemonClient{},
				Serial:  extractor,
				Batch:   extract.NewBatchRunner(primary),
			})
			if err != nil {
				return err
			}
			pretty, _ := cmd.Flags().GetBool("pretty")
			return printJSON(summary, pretty)
		},
	}
	cmd.Flags().String("since", "", "only backfill episodes that occurred on/after this date (YYYY-MM-DD); default: everything")
	cmd.Flags().Bool("no-batch", false, "use serial Haiku.Extract calls (200ms between requests) instead of the Batch API")
	return cmd
}

// backfillDaemon is the daemon surface runBackfill needs: ingest.Daemon's
// glossary/commit/cursor RPCs plus a batched already-ingested check
// (memory.hasEpisodes) that lets backfill skip re-paying for extraction on
// episodes a previous ingest/sweep/backfill run already committed.
type backfillDaemon interface {
	ingest.Daemon
	HasEpisodes(ctx context.Context, ids []string) ([]string, error)
	Glossary(ctx context.Context, limit int) ([]string, error)
	Commit(ctx context.Context, ep memstore.Episode, cwd string, res extract.Result) (resolve.Stats, error)
}

// backfillConfig bundles what runBackfill needs beyond ctx/flags.
type backfillConfig struct {
	Since   time.Time // zero means "everything"
	NoBatch bool
	Daemon  backfillDaemon
	Serial  extract.Extractor
	Batch   *extract.BatchRunner
}

// backfillGlossaryLimit mirrors ingest.glossaryLimit / the daemon's own
// default (see defaultGlossaryLimit in internal/daemon/memory_methods.go).
const backfillGlossaryLimit = 200

// backfillSummary is what `memory backfill` prints on success.
type backfillSummary struct {
	FilesScanned         int      `json:"files_scanned"`
	EpisodesFound        int      `json:"episodes_found"`
	EpisodesAlreadyKnown int      `json:"episodes_already_known"`
	EpisodesCommitted    int      `json:"episodes_committed"`
	EpisodesSkipped      int      `json:"episodes_skipped"`
	FilesAdvanced        int      `json:"files_advanced"`
	Errors               []string `json:"errors,omitempty"`
}

// backfillGroup is every episode distilled from one source path/dir, kept
// together so that path's cursor is only advanced once every one of its
// episodes has a definitive outcome (committed or recorded as an error) —
// mirroring ingest.File's "commit failures don't advance the cursor" safety
// property, but at file granularity across a batch that spans many files.
type backfillGroup struct {
	source    string // "claude" | "codex" | "loom"
	path      string
	episodes  []distill.RawEpisode
	newOffset int64 // full-file byte length for claude/codex; unused for loom
	size      int64
	modTime   time.Time
}

// runBackfill discovers every episode across the default roots (ignoring
// cursors entirely — commit-side idempotency dedupes anything already
// ingested), filters by cfg.Since, extracts each episode (via the Batch API
// unless cfg.NoBatch), commits every successfully-extracted episode, and —
// per source file/dir whose every episode resolved — advances that path's
// cursor to reflect it's now fully processed.
func runBackfill(ctx context.Context, cfg backfillConfig) (backfillSummary, error) {
	var summary backfillSummary

	claudeFiles, codexFiles, loomDirs, discoverErrs := sweep.Candidates(sweep.Roots{}, sweep.DefaultActiveWindow)
	summary.Errors = append(summary.Errors, discoverErrs...)
	summary.FilesScanned = len(claudeFiles) + len(codexFiles) + len(loomDirs)

	var groups []backfillGroup
	for _, path := range claudeFiles {
		g, err := backfillClaudeOrCodex(path, "claude", distill.ClaudeSession, cfg.Since)
		if err != nil {
			summary.Errors = append(summary.Errors, fmt.Sprintf("%s: %v", path, err))
			continue
		}
		groups = append(groups, g)
	}
	for _, path := range codexFiles {
		g, err := backfillClaudeOrCodex(path, "codex", distill.CodexRollout, cfg.Since)
		if err != nil {
			summary.Errors = append(summary.Errors, fmt.Sprintf("%s: %v", path, err))
			continue
		}
		groups = append(groups, g)
	}
	for _, dir := range loomDirs {
		episodes, err := distill.LoomRun(dir)
		if err != nil {
			summary.Errors = append(summary.Errors, fmt.Sprintf("%s: %v", dir, err))
			continue
		}
		episodes = filterSince(episodes, cfg.Since)
		info, err := os.Stat(dir)
		if err != nil {
			summary.Errors = append(summary.Errors, fmt.Sprintf("%s: stat: %v", dir, err))
			continue
		}
		groups = append(groups, backfillGroup{source: "loom", path: dir, episodes: episodes, modTime: info.ModTime()})
	}

	var discovered []distill.RawEpisode
	for _, g := range groups {
		discovered = append(discovered, g.episodes...)
	}
	summary.EpisodesFound = len(discovered)

	if len(discovered) == 0 {
		fmt.Println("nothing to backfill")
		return summary, nil
	}

	// F5: skip re-extracting (re-paying for) episodes a previous
	// ingest/sweep/backfill run already committed to the store —
	// resolve.Apply's own idempotency check would no-op them anyway, so
	// extracting them again buys nothing but a wasted API call.
	discoveredIDs := make([]string, len(discovered))
	for i, ep := range discovered {
		discoveredIDs[i] = ep.ID
	}
	missing, err := cfg.Daemon.HasEpisodes(ctx, discoveredIDs)
	if err != nil {
		return summary, fmt.Errorf("memory backfill: hasEpisodes: %w", err)
	}
	summary.EpisodesAlreadyKnown = len(discovered) - len(missing)

	var allEpisodes []distill.RawEpisode
	for _, g := range groups {
		needExtract, _ := partitionKnownEpisodes(g.episodes, missing)
		allEpisodes = append(allEpisodes, needExtract...)
	}

	var results map[string]extract.Result
	var extractErrs map[string]error
	if len(allEpisodes) == 0 {
		fmt.Println("nothing to extract: every discovered episode is already ingested")
	} else {
		glossary, err := cfg.Daemon.Glossary(ctx, backfillGlossaryLimit)
		if err != nil {
			return summary, fmt.Errorf("memory backfill: glossary: %w", err)
		}

		var fatalErr error
		if cfg.NoBatch {
			results, extractErrs, fatalErr = backfillSerial(ctx, cfg.Serial, allEpisodes, glossary)
			if fatalErr != nil {
				summary.Errors = append(summary.Errors, fmt.Sprintf("backfill canceled: %v", fatalErr))
			}
		} else {
			totalBatches := (len(allEpisodes) + extract.MaxBatchSize - 1) / extract.MaxBatchSize
			progress := func(done, total int) {
				batchNum := (done + extract.MaxBatchSize - 1) / extract.MaxBatchSize
				if batchNum < 1 {
					batchNum = 1
				}
				if batchNum > totalBatches {
					batchNum = totalBatches
				}
				fmt.Printf("batch %d/%d: %d/%d done\n", batchNum, totalBatches, done, total)
			}
			results, extractErrs, fatalErr = cfg.Batch.Run(ctx, allEpisodes, glossary, progress)
			if fatalErr != nil {
				summary.Errors = append(summary.Errors, fmt.Sprintf("batch run: %v", fatalErr))
			}
		}
	}

	missingSet := make(map[string]bool, len(missing))
	for _, id := range missing {
		missingSet[id] = true
	}

	for _, g := range groups {
		resolved := true
		for _, ep := range g.episodes {
			if !missingSet[ep.ID] {
				// Already committed by a previous run: nothing left to do
				// for it, and it counts as resolved for cursor purposes.
				continue
			}
			res, ok := results[ep.ID]
			if ok {
				if _, err := cfg.Daemon.Commit(ctx, memstore.Episode{
					ID:         ep.ID,
					Source:     ep.Source,
					SourceRef:  ep.SourceRef,
					Summary:    res.EpisodeSummary,
					OccurredAt: ep.OccurredAt,
					IngestedAt: time.Now(),
				}, ep.Cwd, res); err != nil {
					summary.Errors = append(summary.Errors, fmt.Sprintf("commit %s: %v", ep.ID, err))
					resolved = false
					continue
				}
				summary.EpisodesCommitted++
				continue
			}
			if extractErr, ok := extractErrs[ep.ID]; ok {
				summary.Errors = append(summary.Errors, fmt.Sprintf("extract %s: %v", ep.ID, extractErr))
				if errors.Is(extractErr, extract.ErrParse) {
					// F1: only a content-level parse failure is
					// skip-and-continue. Anything else (a transport-ish
					// error, canceled/expired batch item) leaves the
					// group's cursor unresolved below, same as an
					// unresolved episode.
					summary.EpisodesSkipped++
					continue
				}
				resolved = false
				continue
			}
			// Neither map has this episode ID: its batch never resolved
			// (Run returned a fatal error before this chunk completed). Leave
			// the file's cursor untouched so a re-run retries it.
			resolved = false
		}

		if !resolved {
			continue
		}

		// F5(b): with --since set, episodes that occurred before the cutoff
		// were filtered out before we ever saw them (see filterSince) — the
		// group above only reflects what made it past that filter, not the
		// whole file. Advancing the cursor here would wrongly mark the file
		// as fully processed and hide those earlier episodes from every
		// future sweep/backfill. Only a --since-less (full) backfill is
		// allowed to retire a file's cursor.
		if !cfg.Since.IsZero() {
			continue
		}

		cursor := memstore.Cursor{Path: g.path, ModTime: g.modTime}
		if g.source != "loom" {
			cursor.Size = g.size
			cursor.ProcessedBytes = g.newOffset
		}
		if err := cfg.Daemon.PutCursor(ctx, cursor); err != nil {
			summary.Errors = append(summary.Errors, fmt.Sprintf("put cursor %s: %v", g.path, err))
			continue
		}
		summary.FilesAdvanced++
	}

	return summary, nil
}

// partitionKnownEpisodes splits episodes into those that still need
// extraction (their ID is present in missing) and those already committed
// by a previous run (ID absent from missing). Pure and side-effect-free so
// it's cheaply unit-testable independent of the daemon RPC that produces
// missing.
func partitionKnownEpisodes(episodes []distill.RawEpisode, missing []string) (needExtract, alreadyKnown []distill.RawEpisode) {
	missingSet := make(map[string]bool, len(missing))
	for _, id := range missing {
		missingSet[id] = true
	}
	for _, ep := range episodes {
		if missingSet[ep.ID] {
			needExtract = append(needExtract, ep)
		} else {
			alreadyKnown = append(alreadyKnown, ep)
		}
	}
	return needExtract, alreadyKnown
}

// backfillClaudeOrCodex distills the full contents of an episodic
// (byte-offset-resume) source from offset 0 — deliberately ignoring
// whatever cursor might exist, since backfill wants every episode on disk —
// and filters by since.
func backfillClaudeOrCodex(path, source string, distillFn func(path string, offset int64) ([]distill.RawEpisode, int64, error), since time.Time) (backfillGroup, error) {
	episodes, newOffset, err := distillFn(path, 0)
	if err != nil {
		return backfillGroup{}, fmt.Errorf("distill: %w", err)
	}
	episodes = filterSince(episodes, since)

	info, err := os.Stat(path)
	if err != nil {
		return backfillGroup{}, fmt.Errorf("stat: %w", err)
	}

	return backfillGroup{
		source:    source,
		path:      path,
		episodes:  episodes,
		newOffset: newOffset,
		size:      info.Size(),
		modTime:   info.ModTime(),
	}, nil
}

// filterSince returns only the episodes that occurred at or after since. A
// zero since (the --since flag was omitted) keeps everything.
func filterSince(episodes []distill.RawEpisode, since time.Time) []distill.RawEpisode {
	if since.IsZero() {
		return episodes
	}
	var out []distill.RawEpisode
	for _, ep := range episodes {
		if !ep.OccurredAt.Before(since) {
			out = append(out, ep)
		}
	}
	return out
}

// backfillSerial extracts every episode one at a time via Haiku.Extract,
// sleeping 200ms between calls (the --no-batch fallback: no batch discount,
// but no wait for a batch to end either). A per-episode extraction error is
// recorded in the error map and does not stop the run.
//
// The third return is fatal-only, mirroring BatchRunner.Run's contract: nil
// on a normal (possibly partially-erroring) completion, or ctx's error if
// the run was canceled mid-loop — surfaced so the caller can record it in
// the summary rather than silently returning a truncated result set.
func backfillSerial(ctx context.Context, h extract.Extractor, episodes []distill.RawEpisode, glossary []string) (map[string]extract.Result, map[string]error, error) {
	results := make(map[string]extract.Result, len(episodes))
	errs := make(map[string]error)

	for i, ep := range episodes {
		if err := ctx.Err(); err != nil {
			return results, errs, err
		}

		res, err := h.Extract(ctx, ep, glossary)
		if err != nil {
			errs[ep.ID] = err
		} else {
			results[ep.ID] = res
		}
		fmt.Printf("serial: %d/%d done\n", i+1, len(episodes))

		if i < len(episodes)-1 {
			select {
			case <-ctx.Done():
				return results, errs, ctx.Err()
			case <-time.After(200 * time.Millisecond):
			}
		}
	}

	return results, errs, nil
}

// --- query verbs: thin callDaemon wrappers ---

func memoryOrientCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "orient",
		Short: "Short markdown orientation blurb for a working directory",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cwd, _ := cmd.Flags().GetString("cwd")
			if cwd == "" {
				wd, err := os.Getwd()
				if err != nil {
					return err
				}
				cwd = wd
			}
			budget, _ := cmd.Flags().GetInt("budget")

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			var result struct {
				Markdown string `json:"markdown"`
			}
			if err := callMemoryDaemon(ctx, "memory.orient", &daemon.MemoryOrientParams{
				Cwd: cwd, Budget: budget,
			}, &result); err != nil {
				return err
			}
			// Orient prints markdown raw, not JSON — it's meant to be read
			// directly (e.g. injected into an agent's context), not parsed.
			fmt.Println(result.Markdown)
			return nil
		},
	}
	cmd.Flags().String("cwd", "", "working directory to orient for (default: process cwd)")
	cmd.Flags().Int("budget", 2000, "approximate character budget for the blurb")
	return cmd
}

func memoryRecallCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "recall <query>",
		Short: "Ranked fact search: the facts that answer a question, optionally as-of a point in time",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			asOf, _ := cmd.Flags().GetString("as-of")
			limit, _ := cmd.Flags().GetInt("limit")

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			var result any
			if err := callMemoryDaemon(ctx, "memory.recall", &daemon.MemoryRecallParams{
				Query: args[0], AsOf: asOf, Limit: limit,
			}, &result); err != nil {
				return err
			}
			pretty, _ := cmd.Flags().GetBool("pretty")
			return printJSON(result, pretty)
		},
	}
	cmd.Flags().String("as-of", "", "RFC3339 timestamp; empty means current")
	cmd.Flags().Int("limit", recall.DefaultFactLimit, "max facts (the payload is capped at 24 KB regardless)")
	return cmd
}

func memoryRememberCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "remember <fact>",
		Short: "Store a durable fact in global memory (queued; extracted in the background)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			entities, _ := cmd.Flags().GetStringArray("entity")

			// No dormancy gate here: unlike ingest, the daemon holds the
			// extractor for memory.remember and reports Dormant in the
			// result itself, so the CLI just relays that rather than
			// short-circuiting before the call.
			ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
			defer cancel()
			var result daemon.MemoryRememberResult
			if err := callMemoryDaemon(ctx, "memory.remember", &daemon.MemoryRememberParams{
				Fact: args[0], Entities: entities,
			}, &result); err != nil {
				return err
			}
			if result.Dormant {
				fmt.Fprintln(os.Stderr, "memory: daemon is dormant (no API key in its environment) — the fact is queued and will be extracted once a key is configured")
			}
			pretty, _ := cmd.Flags().GetBool("pretty")
			return printJSON(result, pretty)
		},
	}
	cmd.Flags().StringArray("entity", nil, "hint entity name to bias resolution (repeatable)")
	return cmd
}

func memoryEntitiesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "entities",
		Short: "List entities, optionally filtered by type",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			typ, _ := cmd.Flags().GetString("type")

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			var result any
			if err := callMemoryDaemon(ctx, "memory.entities", &daemon.MemoryEntitiesParams{
				Type: typ,
			}, &result); err != nil {
				return err
			}
			pretty, _ := cmd.Flags().GetBool("pretty")
			return printJSON(result, pretty)
		},
	}
	cmd.Flags().String("type", "", "filter by entity type (project|service|machine|tool|person|decision|runbook|concept)")
	return cmd
}

func memoryFactsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "facts <slug>",
		Short: "Every fact about a single entity",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			all, _ := cmd.Flags().GetBool("all")

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			var result any
			if err := callMemoryDaemon(ctx, "memory.facts", &daemon.MemoryFactsParams{
				Slug: args[0], IncludeInvalid: all,
			}, &result); err != nil {
				return err
			}
			pretty, _ := cmd.Flags().GetBool("pretty")
			return printJSON(result, pretty)
		},
	}
	cmd.Flags().Bool("all", false, "include invalidated (historical) facts")
	return cmd
}

func memoryInvalidateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "invalidate <src> <relation> <dst>",
		Short: "Invalidate every current fact matching the exact triple",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			var result any
			if err := callMemoryDaemon(ctx, "memory.invalidate", &daemon.MemoryInvalidateParams{
				Src: args[0], Relation: args[1], Dst: args[2],
			}, &result); err != nil {
				return err
			}
			pretty, _ := cmd.Flags().GetBool("pretty")
			return printJSON(result, pretty)
		},
	}
}

func memoryStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Memory domain counts and dormancy",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			var result daemon.MemoryStatusResult
			if err := callMemoryDaemon(ctx, "memory.status", nil, &result); err != nil {
				return err
			}
			pretty, _ := cmd.Flags().GetBool("pretty")
			return printJSON(result, pretty)
		},
	}
}

// --- browse ---

func memoryBrowseCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "browse",
		Short: "Render a searchable local HTML visualization of the memory graph and open it",
		Long: `Render a searchable local HTML visualization of the memory graph and open it.

The daemon also permanently serves a live version of this same UI at
http://127.0.0.1:7279 (loopback only, env SCRY_MEMORY_UI_ADDR to change or
disable with "off") for whenever a one-off file isn't wanted.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			out, _ := cmd.Flags().GetString("out")
			noOpen, _ := cmd.Flags().GetBool("no-open")

			if out == "" {
				home, err := scryHome()
				if err != nil {
					return err
				}
				out = filepath.Join(home, "memory", "browse.html")
			}

			ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
			defer cancel()
			var export daemon.MemoryExportResult
			if err := callMemoryDaemon(ctx, "memory.export", nil, &export); err != nil {
				return err
			}

			html, err := browse.Render(export)
			if err != nil {
				return err
			}

			if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
				return fmt.Errorf("create output dir: %w", err)
			}
			if err := os.WriteFile(out, html, 0o644); err != nil {
				return fmt.Errorf("write %s: %w", out, err)
			}
			fmt.Println(out)

			if !noOpen {
				if err := exec.Command("open", out).Run(); err != nil {
					fmt.Fprintf(os.Stderr, "memory browse: could not open %s automatically: %v\n", out, err)
				}
			}
			return nil
		},
	}
	cmd.Flags().String("out", "", "output path for the HTML file (default: ~/.scry/memory/browse.html)")
	cmd.Flags().Bool("no-open", false, "write the file but do not open it")
	return cmd
}
