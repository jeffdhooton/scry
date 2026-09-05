# Independent predeploy disproof — 2026-09-05

Verdict: bounded PASS for code-only deployment of the reviewed artifact, including the refreshed 21:53:17 UTC backup/restore gate below. The refreshed snapshot has 10 ready / 0 backoff / 6 parked; the root subsequently observed 6 ready / 0 backoff / 8 parked under the OLD binary. There is no quiet-queue assertion. Code-only deployment can restart a draining queue, with complete pending preservation proved on candidate store startup and actual subsequent processing to be reconciled explicitly. No code/artifact blocker found. Deployment requires immediate live hash/process/queue and retained-binary checks plus postdeployment backup/raw audit and five benchmark suites. This is not a semantic alias-drop approval, a prior-repair backfill approval, a queue-retry approval, or completion of the memory-quality goal. No live writes, provider calls, deployment, shared edits, or semantic repairs were performed by this reviewer.

## Exact artifact

Exact commit: `d1f0a958608389a385ac9a9f57ec1eb941015a10`. Shared checkout has only the preexisting untracked `docs/MEMORY_WORKFLOW_ASSESSMENT_2026-09-04.md`.

An independent `git archive` of this exact commit compared recursively equal to `/tmp/scry-alias-rejection-deploy-sep05.pPmnn3/code` before the private harness was added. The root later added a technical replica helper to its archive after binary build; the recursive equality claim refers to the earlier verified artifact source, not that subsequently instrumented directory. All seven production file hashes match the fully read independent code review. The implementation plan and complete prior code/replica reviews were read.

Independent build: `CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -trimpath -ldflags '-s -w -X main.Version=d1f0a95' -o .../scry ./cmd/scry` with Go 1.26.2. Its bytes exactly match the proposed artifact. SHA-256 of both: `290a14c04ef0cfa9618db3f1a848bc6f3a343ec28d9c197ea49720eedb30b553`. Proposed artifact reports `scry d1f0a95`; embedded build metadata confirms CGO 0, darwin/arm64, trimpath.

## Independent fresh restores

Mini input: `/tmp/scry-alias-rejection-deploy-sep05.pPmnn3/mini-pre-deploy.badger`, 78,067,790 bytes, SHA-256 `c063d83125a81f096e319c94286958da8f29a388e1e29b73460180a37be4d397`. Parent independently checked this against the Mini's original `/Users/jclaw/.scry/backups/memory-20260905T214436Z.badger`; reviewer independently hashes the complete local copy.

Laptop input: `/Users/jeff/.scry/backups/memory-20260905T214436Z.badger`, 19,445,008 bytes, SHA-256 `194adb090f810c141abdf4f4f06e8106b2bed7afa4d65d0dc04d7e7a21f0ac0d`, independently hashed.

For EACH source, a fresh private raw Badger database was loaded directly from the backup without candidate schema handling. Every key/value was inventoried. Candidate Open on that raw-loaded store then preserved every key/value exactly. A separate newly created candidate store was restored using the actual candidate Store.Restore, compared exactly with the independently raw-loaded source, then closed/reopened and compared exactly again. Full equality includes every historical fact, episode, pending record, metadata byte, claim, cursor, attestation, reverse edge, value evidence record, and retirement marker. No namespace is filtered. Three immutable inputs, including the previous Mini source, passed this procedure. Schema is exactly `1`, and `ar:` count is zero before and after every open/restore/reopen. No startup migration, wipe, or synthesized rejection occurs in these tested store startup paths.

| Input | Raw keys | Facts current / historical | Entities | Episodes | Claims | Queue ready / backoff / parked |
|---|---:|---:|---:|---:|---:|---:|
| Fresh Mini | 241,815 | 72,535 / 7,803 | 30,480 | 9,332 | 51,808 | 0 / 0 / 6 |
| Fresh laptop | 83,378 | 18,495 / 2,509 | 14,200 | 2,689 | 23,159 | 0 / 0 / 0 |
| Previous Mini | 241,717 | 72,508 / 7,800 | 30,472 | 9,330 | 51,798 | 0 / 0 / 6 |

Full prefix counts and sorted length-delimited key/value SHA-256 values are retained in `mini-summary.json`, `laptop-summary.json`, `previous-mini-summary.json`. Fresh Mini state hash is `8fdd3352817c8832f27f9740baf0342f7bc581a83c8f9f2df9549ef45f22771c`; laptop is `8efead71b3128363e95c7f39678e3ec9332ae4a7645af03454ccf4780f02c2c7`.

The laptop's snapshot has only its schema metadata and no queue records, corroborating dormant memory state. Actual live service role/process status remains the root's immediate recheck; an offline backup cannot prove that a daemon stays dormant afterwards.

## Refreshed 21:53:17 UTC Mini backup gate

Input `/tmp/scry-alias-rejection-deploy-sep05.pPmnn3/mini-fresh-pre-deploy.badger`, 78,208,870 bytes, independently verified SHA-256 `86fa9d43e579a421ce152a1d14f89ed6443ec4fff20fb9b23abed9a2f9d4a5a4`. Root independently checked the matching remote original `/Users/jclaw/.scry/backups/memory-20260905T215317Z.badger` before this review.

PASS: fresh private raw load, candidate Open, separate candidate Restore, and reopen all preserve every key/value exactly. Test `TestIndependentRefreshedMiniBackup` passed in 2.22 seconds. Refreshed state has 241,868 keys, 80,349 facts (72,545 current / 7,804 historical), 30,486 entities, 9,334 episodes, 51,814 claims, 55,458 adjacency records, 10,785 attestations, 3,120 cursors, 471 value-evidence records, 5 metadata records, 15 retired spelling records and 15 retired slug records. Schema remains 1, `ar:` remains zero. Full sorted key/value hash: `8d7c3ce653ee98257d4c72e1538b5e053f13e07b9738b048b9fd0d6f5b3de178`.

Snapshot queue is exactly 10 ready / 0 backoff / 6 parked. All original six parked records remain byte-identical. Ten new records all have zero attempts and are not parked: six Codex transcript slices and four Claude CellSaviors transcript slices, as established by their stored source/source_ref fields. Candidate startup does not alter any pending text, provenance, attempts, errors, parked flag, next_attempt, or eligibility classification. This is complete offline state preservation; no provider processing is invoked by the reviewer.

Compared to the 21:44 snapshot, changed raw prefixes are 10 adj, 6 al, 2 att, 7 cur, 8 en, 2 ep, 14 fa, 4 meta, 10 pq, 2 ve. The two new completed episodes concern CellSaviors cell-image/branding brainstorming and a CADFormats admin review. There are no changes to `ar:`, `rt:`, `rs:`, or keys referencing ChildScribe/Envoyer/office-dashboard/driver-core-worktree. Complete delta is retained in `refreshed-mini-source-delta.json`; counts and summary are in the adjacent JSON files. This source drift is preserved and is not independent semantic endorsement.

The root's later approximately 21:54 live read reported 6 ready / 0 backoff / 8 parked BEFORE deployment. It identified newly parked CellSaviors records `479b704974a9b8b4e8002051539ef2fa291b75bbcddc31ef84ba9a17e595e577` and `eca00e8b0f63b924b86afe0a69b716fcb2c0e8d8728c44025493b13c9bd924ab`, caused under the old 62cf binary by an existing gemini-api to gemini claim conflict. Both records are present as ready, attempts-zero records in the verified 21:53 backup; they were not lost. These live observations are parent-provided and distinguish ordinary draining/parking from any future candidate behavior. Actual postdeployment raw audit must reconcile all subsequent completions, new arrivals and parking changes, and must never attribute these already observed old-binary conflicts to the candidate. No retry or semantic drop is authorized by this gate.

## Source drift since reviewed Mini backup

Previous Mini input: `/tmp/scry-canonical-62cf-deploy-sep05.G82GKc/memory-20260905T211821Z.badger`, expected SHA-256 `3f09b0d6622d4612811b83fc6738f845063c4d19b07ced41c7b893e9b4421e38`.

Complete changed raw records are retained in `mini-source-delta.json`, with prefix totals in `mini-source-delta-counts.json`: 11 adj, 10 al, 1 att, 1 cur, 13 en, 2 ep, 36 fa, 4 meta, 39 ve. The diff distinguishes missing keys from present empty adjacency values.

Two new episodes concern the CADFormats workbench checkpoint and an earlier bounded Scry deployment review. Eight entities and ten claims were added. Of five existing entities changed, four changed only last_seen; `feature-flags` added the literal `admin feature flags` plus last_seen. This is ordinary source drift and is not independent confirmation that the new alias has rightful ownership. The graph contains a generic existing feature-flags description from a separate task; any semantic review of that new assignment remains separate.

Thirty fact records were added and six existing records modified: two invalid_at updates, three provenance episode updates, and one confidence/provenance update. Historical facts remain present. No `ar:`, `rt:`, `rs:`, or `pq:` key changed. No changed key references ChildScribe, Envoyer, office-dashboard, or driver-core-worktree. Full saved ChildScribe entity and the three proposed fixture identity records are unchanged; their claims are unchanged. No changed rejection or retirement decision input is present. Existing prior live cleanup is still unprotected by rejection records, as expected; this deploy does not create a backfill.

## Focused checks and limits

PASS: private `TestIndependentPredeployRestores`, initial 3.88 seconds and repeated corrected source-diff accounting run 4.01 seconds. The initial run created every private restore directory fresh. Its raw equality checks already included empty values; only the separate human-inspection source-delta loop needed explicit missing-vs-empty distinction.

PASS: focused store and resolver alias rejection/repair tests and the daemon test proving legacy inferred identity apply endpoints remain disabled. The separate deterministic queue-conflict test verifies rejection conflicts park immediately while preserving the episode. No five-suite offline benchmark is claimed; the root's reported live predeployment baseline remains 52/62, 29/66, 7/7, 45/50, 47/50, max 13,352 and zero over cap. Five postdeployment suites and bounded latency remain mandatory.

The candidate ordinary-path difference is an owner/spelling rejection lookup and deterministic queue classification; both restored inputs have zero rejection records. Existing legacy hygiene/migration apply entrypoints were inspected and remain disabled. The documented dormant merge-helper inheritance bypass remains outside the supported active paths and must not be enabled.

Deploy only the identical artifact above to both hosts after immediately reconfirming prior binary hashes/processes/queues and retaining both exact prior binaries at unique explicit sibling paths. After deployment, prove both installed hashes and restarted processes, validate Mini ingestion and laptop role, take fresh nonempty backups and compare all raw state with explicit accounting for ordinary ingestion, prove no unexpected marker, and run all five suites. First live alias apply/backfill remains held. Old writers ignore `ar:` records: binary-only rollback is permissible only while no marker exists; after the first marker, use a corrected marker-aware binary or a separately reviewed restoration that reconciles later writes. No automatic backup rollback may discard later evidence.

Private evidence root: `/tmp/scry-alias-rejection-predeploy-independent.yeIfid`. Harness: `code/internal/memory/store/independent_predeploy_test.go`. No shared repository files changed.
