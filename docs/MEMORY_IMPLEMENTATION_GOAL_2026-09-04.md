/goal Finish Scry memory graph quality without regressing ingestion, durable remember, or recall.

## Goal

Finish the two failed clauses of the Scry memory-solid run without weakening the four
that already pass. The live shared graph should contain real identities rather than
status values, fused projects, or hollow remnants. Exact names and aliases should
resolve to the entity that actually owns their facts. Entity repair should be a safe,
reviewable operation that transfers the complete identity—not merely its current fact
edges—and the normal write path should stop recreating the corruption after cleanup.

The critical correction from the last reviewer is architectural: `reattach` is not an
entity merge. Moving facts while leaving the loser's name, aliases, type and index
claims creates a hollow entity and can turn a split recall into an exact-lookup
blackout. Do not apply the old collision proposal with the tools that exist today.

## House rules

These hold no matter how you get to the goal:

- Facts are never discarded. Repairs may relocate facts while preserving text,
  validity, timestamps, confidence and provenance; every live write starts with a real
  Badger backup and is reversible.
- No store-scale rule may infer an entity owner from a name or sentence. Entity merges,
  alias drops, alias rehomes and ambiguous value conversions come from explicit,
  reviewed manifests whose exact inputs are rechecked immediately before writing.
- Do not resume the suffix/prefix word-list treadmill for status values. Use the
  episode's context, keep the resolver's established-identity and artifact vetoes, and
  measure both false negatives and false rejections on names chosen outside the store.
- Ordinary entity writes may not steal an alias-index entry. Index ownership changes
  only through an explicit, validated claim or merge operation, and one episode may
  never merge existing identities across types.
- The Scry constraints stand: Go, no CGO, one static binary, JSON by default, local
  only, no telemetry, no hosted embeddings, and no new transcript retention.
- Work in small commits on `main`, keep `go test ./...` green, restore a live backup
  into a replica before every store-shape change, append measurements to the audit,
  and record architectural decisions with their reasoning in `docs/DECISIONS.md`.

Before anything is pushed, deployed, applied to the live store, or declared final,
hand a fresh-context sub-agent one job: disprove that the change obeys these house
rules. Fix every violation it proves before proceeding.

## Done means

All of the following hold at once against the live shared store on the Mini, not just a
fixture:

1. The four previously passing items remain passing: ingestion is alive and loud,
   remember remains durable and sub-second at p95, recall still meets its genuinely
   held-out bar and 24 KB cap, and Kimi/OpenCode coverage still surfaces through
   orient.
2. All five existing benchmark files match or exceed the immediate pre-prune baseline:
   `heldout-2026-09-03` 53/62, `heldout-b` 34/66, probes 7/7,
   `tuning-strict` 44/50 and `tuning` 47/50. A fresh recall grader also writes at least
   fifty new questions after the last ranking or graph-shape change and gets at least
   90% into the top twenty, with every response below 24 KB.
3. The closed relation vocabulary still contains exactly the documented 39 current
   relations. No current entity is a bare number, measurement, git branch, or status
   value. The resolver prevents new examples using context-bearing tests, including the
   ambiguous pairs `validation_failed` / `user_login_failed` and
   `DONE_WITH_CONCERNS` / `PYTHON_ARGCOMPLETE_OK`, without regressing the independent
   real-name guard corpus.
4. A reviewed entity-merge operation exists. It defaults to dry-run, validates complete
   entity snapshots and every affected current and invalidated fact, takes a nonempty
   backup before apply, moves both fact endpoints when required, preserves all fact
   content and provenance, transfers only reviewed names/aliases/repo refs/description
   and type, rewrites alias ownership, and removes the loser only after no reference
   remains.
5. The merge operation is all-or-nothing for each group and proves its postconditions:
   the loser no longer exists, approved spellings resolve to the survivor, explicitly
   dropped spellings do not resolve to the wrong thing, no hollow entity remains, no
   dangling endpoint exists, and the predicted collision delta equals the observed one.
6. The half-repaired Qwen pair is a single real identity. Exact `memory facts` lookups
   through every approved old spelling return the surviving model's facts; no empty
   Qwen husk or stale alias-index owner remains.
7. Every fact touching `hermes-ops`, `hermes`, and `mac-mini` has been reviewed under a
   written rubric. Project/code/repository facts stay on `hermes-ops`; gateway behavior
   belongs to `hermes`; hardware and host state belong to `mac-mini`; ambiguous facts
   are explicitly recorded rather than guessed. The three identities remain distinct
   under exact lookup and recall.
8. Every remaining alias on `childscribe-laravel` and every alias changed during the
   collision repair has been reviewed. Aliases naming another product, project,
   workspace, machine, generic role, or generic surface are dropped or explicitly
   rehomed. Removing a stolen alias cannot leave a rightful entity listing the spelling
   while the alias index resolves nowhere.
9. `scry memory hygiene` reports zero cross-type collisions. A second dry run is a
   complete no-op. An independent audit also reports zero hollow entities, zero dangling
   fact endpoints, and zero aliases that are listed on an entity but indexed to an
   unexplained owner or to nobody.
10. The final code and store state survive two subsequent real sweeps without recreating
    a status-value entity, cross-type collision, stale alias claim, self-loop, or
    proposed hygiene repair. Laptop and Mini run byte-identical binaries, each retaining
    its previous binary, and the final backup, deploy hashes, measurements and reviewer
    verdicts are appended to the audit and handoff.

## Required safety dependencies

- Do not apply the reviewed 41-group/115-move collision subset. The reviewer measured
  that it would reduce 322 collisions only to 281 while leaving 36 hollow entities with
  89 spellings. `scratchpad/rev_subset.json` is not present in the current checkout or
  the searched Mini locations, so regenerate all proposals from a fresh snapshot.
- Close the two normal-write admission holes before the final cleanup: `PutEntity`
  currently writes every name and alias index entry unconditionally, and
  `AdmitAlias` still has existing-owner shortcuts. Prove the replacement against a
  restored live replica before deploying it into the draining queue.
- Build the entity-merge capability before moving another collision group's facts.
  Use the Qwen split as the first fixture and first live repair.
- Extend reviewed alias removal with an explicit optional rehome target. Never choose a
  recipient automatically; verify that the intended target already owns or is approved
  to own the spelling.
- Do not apply the current broad hygiene dry run merely because it reports changes. Its
  proposed drops, splits, reattachments and self-loop invalidations must be reviewed
  against the snapshot that produced them.
- The store is still changing while the extraction backlog drains. Develop and test on
  snapshots, deploy prevention first, then take a fresh backup and regenerate every
  live repair manifest after the queue becomes stable.

## Implementation direction

Use this as the expected shape, while retaining freedom to improve the mechanics:

1. Make alias/name ownership explicit in the write path and add the two missing
   admission regression tests. Ordinary `PutEntity` updates must preserve an index key
   claimed by another entity; only a reviewed resolver action may transfer it.
2. Add `scry memory merge-entities --file <json> [--apply]`. A manifest should identify
   survivor and loser, expected entity hashes, the chosen survivor metadata, explicit
   alias dispositions, the complete affected fact set, and expected postconditions.
3. Make each merge group transactional after full preflight. Handle current and
   invalidated facts, both endpoints, exact-key collisions and would-be self-loops
   explicitly; never silently nudge, discard or orphan content.
4. Repair the Qwen pair, then regenerate the collision inventory from the stable live
   snapshot. Review each group as same identity, distinct homonyms, or a bad fold. Do
   not use fact count as the semantic decision; the prior reviewer proved it preserves
   wrong types in several groups.
5. Harden `unalias`, finish the `childscribe-laravel` alias review, and finish the
   remaining Hermes/Mini fact review in small independently reviewed batches.
6. For status values, first run a bounded go/no-go experiment. Test a forced binary,
   context-bearing identity-versus-value classification on both configured models,
   applied only to suspicious new concepts and guarded by established-entity and
   artifact vetoes. If neither model meets the precision bar, do not disguise that with
   more lexical exceptions: record the evidence and bring the remaining design choice
   to Jeff.
7. Convert legacy status nodes only through an explicit reviewed retirement manifest,
   preserving every fact as an attribute or relocating it to a reviewed entity. Then
   execute the complete replica and live grading sequence.

## Grading

The builder never grades its own work. Use separate fresh-context graders for:

- the status/value boundary and external-name false-rejection corpus;
- write-path alias ownership and one-episode merge resistance;
- entity-merge safety, historical fact preservation and exact lookup;
- the live identity/alias/collision audit;
- recall with a newly written held-out question set;
- the original items 1, 2 and 6 as regression checks;
- the house rules and backup/deployment discipline.

Each grader is instructed to prove the relevant bar false and is pointed at the real
store, logs, binaries, manifests and exact command output. A builder's explanation is
not evidence. Only a failed disproof counts as passing, and all clauses must pass on two
consecutive grading rounds.

## Loop

Build, restore a replica, measure, ask a fresh grader to disprove the result, fix the
largest proven gap, and repeat. Apply to the live store only after the corresponding
replica result and manifest survive review. You do not get to decide you are finished.
Stop only when Jeff says done, or when every clause above passes two consecutive
fresh-context grading rounds. In Codex, keep iterating in-session under `/goal` until
that condition is met.

## Build on prior work

- Start with `docs/MEMORY_HANDOFF_2026-09-04.md` at commit `0272583`, especially the
  correction explaining why `reattach` cannot satisfy the collision clause.
- Read `docs/MEMORY_AUDIT_2026-09-02.md` in full. It is append-only evidence for every
  successful, failed and reverted experiment. Do not repeat the discarded store-scale
  ownership rules, alias-transfer passes, pruning pass, ranking experiments or
  transcript-retention experiment.
- Read the memory decisions in `docs/DECISIONS.md`, especially “Facts move from a list,
  not from a rule,” “The write path's alias test cannot be run over stored aliases,”
  “The concept wildcard,” “Episode transcripts stay discarded,” and “The value type is
  inert with this model.”
- Read the original kickoff at
  `~/dotfiles/ai/prompts/2026-09-02-scry-memory-solid.md`; its original six-part bar and
  autonomy grants remain in force except where this continuation makes them stricter.
- The immediate pre-prune backup is
  `/Users/jclaw/.scry/backups/memory-20260904T170026Z.badger` on the Mini. It contains
  122 `childscribe-laravel` aliases. The verified live prune contains 89, with exactly
  33 removed. Scry episode `e407d185a86b29e9625a8b89a83fab868b4df0724648a42f5141a827ba82f40d`
  records the full five-suite comparison.
- The verified pre/post-prune scores are identical: 53/62, 34/66, 7/7, 44/50 and
  47/50. The older `heldout-b` 35/66 headline had already become 34/66 before the prune;
  do not misattribute it.
- Relevant code starts at `internal/memory/store/store.go` (`PutEntity`, `DropAlias`,
  fact relocation), `internal/memory/resolve/aliases.go`,
  `internal/memory/resolve/resolve.go`, `internal/memory/resolve/hygiene.go`, and the
  reviewed CLI handlers in `internal/daemon/memory_reattach.go` and
  `internal/daemon/memory_unalias.go`.
- At kickoff, `go test ./...` was green, the worktree was clean, functional binaries
  were built from `26842de`, and `main` was 41 commits ahead of origin after the
  documentation-only reviewer commits `7db257e` and `0272583`.

## Autonomy

- You may spend up to the original run's remaining $25 Z.ai allowance on extraction or
  classification probes. Do not top up or spend DeepSeek balance.
- `Z_AI_API_KEY` is available through the existing laptop and Mini secret/config paths.
  Never print or move credentials.
- You may take backups, restore replicas, build and deploy Scry to the laptop and Mini,
  and restart the two launchd-managed daemons, provided the previous binary and a
  verified nonempty store backup exist first.
- You may mutate the live store only through reviewed dry-run-first manifests after the
  prevention code is deployed. Use small batches and verify exact lookup, collision
  count, benchmarks and second-pass convergence after each one.
- Do not prune accumulated rollback binaries, change transcript retention, install a
  local embedding model, alter hook configuration, push branches, merge PRs, or deploy
  unrelated projects without explicit approval.
- Post each deploy, live-store apply, rollback and grader verdict to Scry room
  `221c0d69ed04` so Jeff can follow the run remotely.
- Make your own calls within these boundaries. Return only when genuinely blocked by a
  decision that belongs to Jeff, or when the complete bar has passed twice.

## Mode

Run this as one controlled implementation loop. The store, resolver, merge primitive
and repair manifests are tightly coupled, and concurrent writers would invalidate the
snapshots used for review. Delegate bounded fresh-context grading tasks, but keep one
lead responsible for sequencing, integration, live-store writes and the final evidence.

## Repository-local continuation checkpoint — 2026-09-04

This file preserves the existing execution contract above so continuation does not
depend on a private Codex attachment. It does not start, clear, or resume a goal
by itself. Read the whole contract before acting. The historical kickoff and
baseline statements above are not claims about today's running binary.

Additional source: `docs/MEMORY_WORKFLOW_ASSESSMENT_2026-09-04.md`, measured at
`3b29ae7`. It is an assessment and proposed roadmap, not permission to bypass
the safety dependencies above. Re-measure its observations before changing code.

### Current checkpoint and what not to repeat

- The initial 33-alias prune has already been verified: 122 to 89 aliases,
  all five pre/post benchmark scores unchanged. Do not redo or blame that prune
  for the older heldout-b decline from 35 to 34.
- Normal-write alias protection, transactional episode application, reviewed
  merge/retirement primitives, and deterministic queue parking exist in code.
  Read the latest audit and git log before rebuilding them.
- At checkpoint HEAD `7ac727e`, retirement leaves durable spelling/value
  markers. A fresh reviewer proved one remaining hole: rehoming the retired
  slug lets a punctuation variant recreate its storage key. The pending fix
  separates permanent slug retirement from rehomeable spelling classification.
  Exact reproduction and legitimate-alias controls are in
  `internal/memory/resolve/retired_slug_test.go`. Do not deploy the failed SHA.
- The two tenth-round reviews passed `ace088c`, not every later change.
  Passing unit tests or an earlier review never certifies a different SHA.
- No 41-group subset, Qwen merge, broad collision cleanup, or status retirement
  has been applied by this continuation. Verify live state, do not assume this
  remains true after a later executor updates this checkpoint.
- The 502-entry status inventory is a reviewed candidate boundary, not an
  apply-ready manifest. The committed source-owner review covers all 657
  touching facts, including 13 invalidated facts. An additional source review
  proposed one fact DROP: reject it because the contract preserves every fact.
  Remaining ambiguous owners require evidence or Jeff's decision, not guessing.
- The workflow assessment is user-provided and was untracked at checkpoint;
  preserve it, and do not silently rewrite its historical measurements.

### Execution order for the existing authorized goal

1. **Close demonstrated prevention defects.** Finish permanent slug protection;
   preserve explicit alias rehomes, successful subsequent value ingestion,
   transactional rollback, reopen/backup persistence, callback safety, and the
   independent real-name corpus. Test queue parking versus genuinely retryable
   errors. Keep fixes bounded to demonstrated failures.
2. **Make the existing workflow diagnosable.** Verify explicit `--cwd .`
   orientation equals an absolute client path even with a remote memory daemon.
   Count delivered recall facts and top score in the existing local call log;
   distinguish malformed responses from genuine empty results. Keep content out
   of logs and leave the response/payload cap unchanged. These checks are not
   proof that orientation now contains the best standing rules.
3. **Prove prevention on a restored live replica, then deploy.** Record the
   backup path/size/hash, exact code SHA, fixture and replica results, fresh
   reviewer verdict, and rollback binary. Build one no-CGO artifact and verify
   the same bytes on laptop and Mini. Inspect queue reasons after restart.
   A parked queue item is preserved work requiring review, not a successful drain.
4. **Take a fresh stable snapshot and repair identities in reviewed batches.**
   Qwen is the first merge. Regenerate its stale manifest and every later
   collision/alias proposal from that snapshot. Audit full historical facts,
   all spellings, explicit drops/rehomes, metadata and indexes. Review every
   Hermes/Mini fact and the remaining ChildScribe aliases. Finish status owner
   adjudication without dropping text, validity, confidence or provenance.
   Snapshot drift invalidates a manifest; never force it through.
5. **Run the full live evidence sequence.** Re-run all five fixed suites; they
   are regression sets, no longer honest held-outs. After the final graph or
   ranking change, have a fresh independent grader author at least 50 unseen
   questions with source-backed answers, then measure the existing 90% top-20
   and <24 KB bar. Preserve the other ten-clause requirements, zero-anomaly
   audit, second-pass no-op, two real sweeps and two consecutive grading rounds.

Verified command surfaces for that loop include:

```sh
go test ./...
go vet ./...
go test -race ./internal/memory/... ./internal/mcp ./cmd/scry
scry memory status --pretty
scry memory queue --pretty
scry memory orient --cwd .
scry memory bench --file docs/memory-bench/heldout-2026-09-03.json --top 20
scry memory bench --file docs/memory-bench/heldout-b.json --top 20
scry memory bench --file docs/memory-bench/probes.json --top 20
scry memory bench --file docs/memory-bench/tuning-strict.json --top 20
scry memory bench --file docs/memory-bench/tuning.json --top 20
```

Do not run memory migration, hygiene apply, replay-all, or any repair apply as a
routine validation command. Consult each current CLI's help and reviewed manifest
before using its mutation flags. Store inspection and benchmarks use the configured
authority; a local fixture pass is not a live pass.

### Assessment follow-on plan: useful memory, beyond graph cleanup

These are the next implementation checkpoints to propose/execute within separately
confirmed scope. They do not silently enlarge the existing goal's stopping condition
or override protected systems. Code prototypes and test design can proceed locally;
new live ingestion sources need an explicit source inventory and rollout review.

**A. Coverage and standing-rule orientation come before more lexical ranking.**

- Curated notes: start with explicit project/path mappings rather than decoding
  project names heuristically from directory names. Preserve source path,
  project, note kind, content hash and version identity. Test unchanged-file
  no-op, edits, renames, removals, restart, and failed extraction before advancing
  a cursor. A removed note must not silently delete its historical facts.
- Project docs: scoped allowlist, heading-level chunks with stable IDs and blob
  hashes; bound input size and extraction cost. Heading rename/deletion and
  contradictory edits need explicit provenance/supersession semantics. Prove
  “why no CGO” and three other independently selected repo constraints from
  source-backed answers. Do not globally crawl every repository by default.
- Orientation: show standing rules, recent decisions, open blockers and useful
  runbook facts from the correct repository with compact provenance. Test that
  unrelated high-volume projects cannot crowd them out. The current CLI
  `--budget` is characters, not tokens: 500 is not a 500-token budget.
- Room ingestion: preserve room, task, author, sequence and repo provenance;
  chunk boundedly, resume idempotently, and handle reopen/new posts. Start with
  a reviewed closed-room fixture. Never silently promote every worker/task to
  a durable project/person, or retain extra raw transcript copies.

**B. Structured remember needs a precise contract before direct writes.**

- Keep the existing prose interface and durable asynchronous behavior working.
  Validate any optional subject/kind/project/supersedes fields. Reuse alias
  ownership protection and atomic store transactions; typed input does not
  authorize creation or transfer of another identity.
- Scope supersession to explicit fact identities, not all subject/relation
  facts across environments. Preserve staging and production simultaneously.
  Make primary-write idempotency and failure/retry behavior testable. A
  caller-supplied assertion must not be represented as independently verified
  truth merely by assigning confidence 1.0.
- Test immediate primary-fact visibility, primary versus secondary extraction
  failure, duplicate retry and rollback. Update the tool contract only after
  this behavior is real. Changing global agent configuration remains gated.

**C. Descriptive and temporal recall are experiments, not a seventh fitted pass.**

- Measure a two-call discovery protocol separately from single-call recall;
  account for both calls' latency and payload and publish both scores.
- Evaluate current-state presentation against independently authored questions.
  Expose timestamps/provenance and conflicts; do not treat the newest sentence
  as automatically authoritative. Do not widen `DefaultExclusive`.
- Do not train synonyms or ranking constants on the five contaminated suites.
  New outcome thresholds beyond the existing contract require an explicit
  agreed acceptance bar; report measurements rather than inventing one.

**D. Remaining assessment items are distinct, bounded work.**

- Code graph: reproduce unspecified SCIP kinds, then derive only the kind
  supported by the actual descriptor/metadata. Preserve explicit kinds;
  do not label every type an interface or invent function-to-table edges.
  Verify Go/TS/PHP fixtures and real node/edge counts before claiming
  architectural paths work. This is adjacent code-intelligence work, not
  evidence that memory recall improved.
- Agent identities and file paths: preserve actual context and full original
  paths. A worker-shaped name or shared basename is not enough to merge or
  retype an identity. Any legacy repair still needs a per-entity manifest;
  entity-count targets cannot justify deleting people or facts.
- Fleet noise: measure episode-local versus durable evidence before adding an
  admission rule. Retain recoverable provenance for rejected candidates;
  never discard a one-off user decision just because it has one attestation.
- Hooks and local embeddings remain approval-gated. Do not install hooks,
  edit settings/global agent files, install a model, or spend on a new provider
  from this roadmap. The assessment's stale 41-group subset is forbidden,
  not an optional shortcut. Do not change /goal state by editing hooks.

For each follow-on checkpoint, specify source ownership, update semantics,
privacy/cost limits, regression fixtures and an independent acceptance test before
enabling it live. Record outcomes in the audit; record durable architectural
decisions in Scry and in DECISIONS.md as required by the existing contract.

