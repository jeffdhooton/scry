# Background change review

User-approved intent: extend Scry with background understanding of changes, starting with one independent, read-only reviewer. Share captured evidence across future readers, attach findings to exact snapshots, and never admit suspicions into durable memory automatically.

## Scope

An optional local daemon service polls explicitly configured repositories after a quiet period, captures their HEAD plus tracked and nonignored untracked state, assembles bounded evidence, and runs one review at a time. Polling makes operation independent of indexer success and watcher coverage. Existing code indexes enrich evidence where available; missing indexes are reported. Original sources remain retrievable in the evidence record. The first consumer returns regression findings, test gaps, and a concise change explanation in one bounded call. Separate agents can consume the same evidence later.

## State and boundaries

- Capture includes staged, unstaged, untracked, deleted, and renamed changes. File content hashes, HEAD, and modes determine identity. Symlinks are never followed. Git commands disable external diff/text conversion.
- Fingerprint the repository before and after evidence assembly; changed state aborts dispatch. Store reviewed input, provider/model identity, timestamps, result, and error. Fetching results recomputes freshness conservatively against the full repository snapshot.
- Secret-like files are omitted from outgoing evidence; explicit path exclusions supplement defaults. Repository text is untrusted data, not instructions. The model has no tools or write capability.
- Records live under the local Scry home, separate from memory. Provisional findings never call memory.remember or change code. Invalid/truncated model output is a failure, not a clean review.
- Optional settings are independently validated. Default off. Explicit repository allowlist, model, endpoint, credential variable, per-day request cap, per-call output/input bounds, timeout, and quiet period. Reserve requests durably before dispatch. Failed calls count; SDK retries are disabled. Restart cannot replenish daily allowance. One service-wide model call at a time.
- Retain bounded recent records; persist daily allowance independently of result retention. On interruption retain a visible interrupted state. No automatic retry of the same snapshot.

## Integration and operation

CLI: `scry review status`, `preview`, `run`, `list`, and `get ID`. `run` queues asynchronous work; `preview` captures locally without inference. All calls route to the local daemon, independent of remote memory. MCP exposes matching query/queue capabilities in local/all profiles only. Results always carry snapshot identity and freshness; errors never imply no issues.

Add a documented setup guide and agent routing hint. Install the tested binary on the laptop using its existing launchd service, preserving the shared memory configuration and rollback binary. Enable only the approved repositories/provider/usage settings. Verify the real daemon and MCP discovery; a model smoke run requires an available configured provider and records actual outcomes.

## Validation

Real temporary Git repositories verify snapshot identity, stable capture, exclusions, deletions, untracked files and symlink handling. Service tests exercise debounce, deduplication, bounded reservations across restart, concurrency, errors, cancellation, and stale retrieval. HTTP test servers exercise provider output validation and bounded requests without paid inference. CLI/MCP and daemon tests verify routing and profile boundaries. Run targeted tests, race tests where supported, full tests and vet before installation.
