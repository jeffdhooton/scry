# Friction journal implementation evidence

The native journal meets the original one-run pilot acceptance bar in isolated
tests: three original observations survive service restart exactly; identical
retries do not duplicate; a synthetic second run produces a two-run recurrence;
unresolved outcomes, evidence and authored proposals remain attributed. The
synthetic event also survives process kill. No instruction is activated.

- Contract and usage: [friction-journal.md](../../friction-journal.md).
- Exact delivered source/test/documentation bytes: [delivery-hashes.json](delivery-hashes.json).
- Full test and vet results: [checks.json](checks.json), [test.log.txt](test.log.txt), [vet.log.txt](vet.log.txt).
- Fresh independent verification: [independent-review/REVIEW.md](independent-review/REVIEW.md).

The independent verifier used its own black-box assertions against freshly built
CLI, RPC and MCP clients, including 32 concurrent submissions, pagination and
review bounds. Its before/after evidence retains an extreme numeric-input defect:
two finite but exceptionally large reported costs overflowed the aggregate into
an unencodable value. The final change refuses that review with a clear input
error, preserving both source events. All 46 final black-box checks passed, with
additional independent cancellation and unsupported-schema preservation probes.

The full repository test and vet commands passed with `CGO_ENABLED=0`. Go reused
cached results for unchanged packages; the changed friction and daemon tests ran
again after the overflow correction. An evidence-only verifier bridge existed as
a Go package during those commands; it is now archived as `.go.txt` with the
other review material.

The preceding SCIP graph delivery and initial workflow pilot are packaged
separately at `64c0cde`. Their delivered code/fixture hashes remain unchanged.
Journal testing used temporary homes, sockets, stores and subprocesses. It did
not import events into a production journal, install a binary, restart the live
daemon/MCP, modify hooks/settings or deploy. The original pilot remains the
only real work run represented here; demonstrating prevented recurrence requires
the next three real tasks after rollout.
