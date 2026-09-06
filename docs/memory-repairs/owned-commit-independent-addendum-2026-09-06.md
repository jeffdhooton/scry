# Additional owned-commit finding: unattested source cwd becomes RepoRefs

This addendum supplements, and does not replace or edit, REPORT.md (SHA-256 `998fd588b516f3c278c8b0b9d47cea24b8f2f43a10d287b2183bf5d9504f70b9`). The bounded verdict remains **NO-GO**, now with a second independently reproduced defect against the same frozen source. The root supplied this additional hypothesis after the initial report was written; I independently inspected the baseline consumer contract and exercised the actual frozen entry. I have not read or graded the root's subsequent source correction for the first finding.

## Evidence and scope

The baseline `internal/memory/queue/queue.go:413` starts the resolver cwd as empty and assigns the original source path only when `p.CwdIsRepo` is true. It simultaneously retains full `p.Cwd` and `p.CwdIsRepo` in the actual Episode. The baseline `internal/memory/resolve/resolve.go:963` explains that workspace-shaped strings are only a path filter; the originating machine attests whether the path is a repository, and the resolving daemon cannot infer that remotely.

The new fixed owner requires `revision.Cwd == incoming.Cwd` in `identity_assertion_admission_episode.go:50`. It pins the complete episode, including the false repository attestation, but `identity_overlay_declarations.go:84` and `:200` create/append RepoRefs from `revision.Cwd` based only on IsWorkspacePath. This turns preserved source context into repository metadata authority. The old queue's separate empty repo cwd cannot be expressed by simply blanking the new revision cwd because the owned EP validation requires exact equality with the original source cwd.

The assertion contract does not separately spell out the CwdIsRepo field's policy. This finding is an independently observed regression against the preserved baseline metadata consumer behavior and the requirement that the composed entry retain established contextual policy. It is an actual private-entry semantic defect, not merely the acknowledged absence of a production adapter. No new lexical or ownership rule is proposed here.

## Actual transaction reproducer

`TestOwnedIndependentRepoAttestation` has four cases: existing legacy entity/new supported birth crossed with true/false repository attestation. Every case invokes real `applyCompleteAdmission` with the full canonical original input and actual canonical EP, then reads the committed EP, selected complete result and actual entity.

Both false-attestation cases fail on the first run. The actual stored EP contains:

`CwdIsRepo=false`, `Cwd="/Users/test/Herd/unattested-source-directory"`.

The actual entity nevertheless contains:

`RepoRefs=["/Users/test/Herd/unattested-source-directory"]`.

Both true-attestation controls pass. Each case also verifies that the valid supporting assertion committed and that the actual EP retained the original path/attestation. No fixtures were corrected, tests weakened, or source files changed. Only a fourth independent test file was added; the earlier report's three-file inventory describes its original checkpoint.

Command, from the private export:

`CGO_ENABLED=0 TMPDIR=/tmp/scry-owned-independent-sep06.zvCJIZ/tmp go test ./internal/memory/store -run '^TestOwnedIndependentRepoAttestation$' -count=1 -v`

Test: `/tmp/scry-owned-independent-sep06.zvCJIZ/code/internal/memory/store/owned_independent_repo_attestation_test.go`, SHA-256 `30b3e9bc4390470c930c34fee2e8211d719129b8849423e54ebd22a5f77f012a`.

First unchanged failure log: `/tmp/scry-owned-independent-sep06.zvCJIZ/repo-attestation-first.log`, SHA-256 `ae322e095d8459b31a549f918314403f28d3bfd39877bb5f5f2344b92be29442`.

Final manifest verification: `/tmp/scry-owned-independent-sep06.zvCJIZ/manifest-final.log`, SHA-256 `b85ee43ac84b170c8d02e77b90659e525896fd1a657327850e7b0b2d5b8afed2`. All 600 baseline and 99 supplied addition hashes still match in frozen and private exported trees. The report's original passing transaction evidence and all limitations stand. No normal Apply, production, all-writer, live-data, actual-backup benchmark or whole-goal approval is given.
