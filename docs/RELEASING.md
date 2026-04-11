# Releasing scry

This is the operational checklist for cutting a new scry release. The
infrastructure is all in place — tag a commit, push the tag, and the
GitHub Actions release workflow produces a draft release you can eyeball
and publish.

See also:

- `.goreleaser.yaml` — the build matrix, ldflags, archive template,
  changelog filters. Read it before you change the release pipeline.
- `.github/workflows/release.yml` — the CI workflow that runs
  goreleaser. Reads the Go version from `go.mod` so bumps flow through.
- `scripts/install.sh` — the one-liner installer end users copy-paste.
  It pulls the latest published (non-draft) release from the GitHub API.

## Versioning

scry uses semver tags (`vMAJOR.MINOR.PATCH`). There's no formal policy
yet because there's one user; the informal rule is:

- **Patch (v0.1.0 → v0.1.1)**: bug fixes, doc updates, non-breaking
  internal changes.
- **Minor (v0.1.0 → v0.2.0)**: new features, new commands, new language
  support, anything a user would notice but that doesn't break existing
  queries or on-disk formats.
- **Major (v0.x → v1.0.0)**: breaking changes to the CLI surface, the
  RPC protocol, the BadgerDB schema, or the MCP tool shapes. Bump the
  `store.SchemaVersion` constant on any schema change; the daemon
  already wipes and reindexes on a schema mismatch.

**Pre-1.0 caveat:** we're free to break things within `v0.x` — that's
the whole point of 0-series versions. But document the break in the
release notes.

## The full checklist

### 1. Pre-flight on a clean working tree

```bash
cd ~/workspace/scry
git status                         # working tree must be clean
git log --oneline origin/main..    # anything unpushed?
```

If there's uncommitted work, commit or stash it first. If local main is
ahead of `origin/main`, push it — GoReleaser reads `origin/main` state
to build the changelog.

### 2. Run the full test suite locally

```bash
go build ./...
go test -count=1 ./...
```

The release workflow reruns these before GoReleaser kicks in, so a
local failure here would block the release on CI anyway. Catching it
locally saves a minute.

### 3. Sanity-check scry doctor against your own install

```bash
scry doctor
```

If your dev machine's doctor is healthy, the binary you're about to
ship is healthy. If it's red anywhere, fix before tagging.

### 4. Decide on the version

```bash
# What's the current release?
gh release list --limit 5

# What commits have landed since?
git log --oneline $(git describe --tags --abbrev=0)..HEAD
```

Pick `vX.Y.Z` by the semver rules above. No `v0.0.0`, no leading `v`
missing, no pre-release suffix unless you actually want a draft
pre-release (GoReleaser's `prerelease: auto` flag detects `-rc`,
`-beta`, etc. and marks them).

### 5. Tag and push

```bash
git tag v0.2.0
git push origin v0.2.0
```

That's it — no annotated tag needed unless you want a tag message.
GoReleaser uses the commit message for its changelog filter.

### 6. Watch the release workflow

```bash
# Immediately after pushing the tag:
gh run list --workflow=release.yml --limit 1

# Grab the run ID from the output and watch it:
gh run watch <run-id> --exit-status
```

Expected runtime: **~2 minutes**. The steps are:

1. `Check out source` (~5s)
2. `Set up Go` (~30s — cached on subsequent runs)
3. `Verify build and tests pass before releasing` (~15s)
4. `Run GoReleaser` (~1m — cross-compiles 4 binaries, tars them up,
   uploads to GitHub)

If any step fails, GoReleaser leaves no draft release and you can
re-run the workflow after fixing. Tags are sticky — `git tag -d v0.2.0
&& git push --delete origin v0.2.0` if you need to move the tag,
though that's destructive and should be a last resort.

### 7. Eyeball the draft release

```bash
gh release view v0.2.0
```

The draft URL is in the output. Open it in a browser, or use
`gh release view v0.2.0 --web`.

Check:

- **Changelog** — auto-generated from git log. The filters in
  `.goreleaser.yaml` strip `docs:`, `chore:`, `test:`, `ci:`, `wip:`,
  and typo fixes. If the list looks messy, edit the release body
  manually before publishing.
- **Artifacts** — there should be exactly 5 files:
  - `scry_X.Y.Z_darwin_amd64.tar.gz`
  - `scry_X.Y.Z_darwin_arm64.tar.gz`
  - `scry_X.Y.Z_linux_amd64.tar.gz`
  - `scry_X.Y.Z_linux_arm64.tar.gz`
  - `scry_X.Y.Z_checksums.txt`
  Missing platforms mean a build broke silently — check the workflow
  logs.
- **Release body header** — the install one-liner block is templated
  from `.goreleaser.yaml`'s `release.header`.

### 8. Publish the draft

**Via web UI** (preferred for big releases): click "Publish release"
at the bottom of the draft page.

**Via CLI** (fast path for patches):

```bash
gh release edit v0.2.0 --draft=false
```

Publishing is what makes the release appear in
`gh release list` and what makes the GitHub API's `releases/latest`
endpoint return it. Until you publish, the install script will say
"no published releases found."

### 9. Smoke-test the install script against the published release

This is the single highest-value post-release check. It verifies the
entire distribution pipeline — GitHub API → tarball download →
checksum verification → extraction → ldflags version injection — not
just the local build.

```bash
rm -rf /tmp/scry-smoke
INSTALL_DIR=/tmp/scry-smoke sh scripts/install.sh
/tmp/scry-smoke/scry version                # prints scry X.Y.Z (not "dev")
/tmp/scry-smoke/scry upgrade --check        # "you're up to date"
/tmp/scry-smoke/scry doctor                 # full green on a healthy machine
```

If any of these fail, the release is broken for every end user and
you should `gh release edit vX.Y.Z --draft=true` to unpublish while
you fix it.

### 10. Upgrade your own install

```bash
scry upgrade              # replaces the running binary in place
scry setup --force        # re-registers the new binary path with Claude Code
```

The `setup --force` is important: if you installed scry at
`~/go/bin/scry` and the upgrade rewrote that binary, your Claude Code
MCP config still points at the old path until you re-register. Run
`scry doctor` after to confirm the MCP server is still `Connected`.

### 11. Announce (optional)

When scry has more than one user: update the project README if
anything user-facing changed, mention the release in the changelog
section, post a note wherever users live.

## Common gotchas

**"go test fails on CI but passes locally"** — the release workflow
uses the Go version declared in `go.mod`. If you bumped `go.mod` but
haven't actually installed that Go version locally, you may be running
tests against an older compiler. `go version` and compare.

**"GoReleaser fails with 'archive not found'"** — usually a
`.goreleaser.yaml` typo in the `builds.binary` or `archives.ids` keys.
Run `goreleaser check` locally to validate the config before pushing a
tag.

**"workflow succeeded but no release appeared"** — check the workflow
log. If it says "release skipped: prerelease auto-detected from tag"
and you didn't mean it to be a prerelease, your tag had a
`-rc` / `-beta` / `-alpha` suffix. Retag.

**"install.sh downloads the tarball but fails SHA256 verification"** —
the checksum file didn't match. Usually a GoReleaser version drift.
Re-run the workflow from the same tag; if that fails, delete the tag
and re-push.

**"scry upgrade says 'already-current' but I just tagged a new
release"** — the release is still a draft. Publish it first; the
GitHub API's `releases/latest` endpoint filters drafts.

**"Node.js 20 deprecation warnings in the workflow output"** — GitHub
Actions is migrating runners to Node 24 by September 2026. When the
`actions/checkout@v4` and `goreleaser/goreleaser-action@v6` actions
publish Node 24 compatible versions, bump them in `release.yml`. The
current versions still work until the deprecation hits.

## Cutting a release with local-only GoReleaser (fallback)

If GitHub Actions is down and you need a release out the door:

```bash
# Install goreleaser locally (one-time)
brew install goreleaser

# Tag locally, don't push yet
git tag v0.2.0

# Run the release with a local GitHub token (needs `repo` scope)
export GITHUB_TOKEN=ghp_...
goreleaser release --clean

# Verify, then push the tag so the release is tied to the right commit
git push origin v0.2.0
```

This should be a last resort — the CI workflow exists so every
release is reproducible from a clean Ubuntu runner, not from whatever
state your laptop happens to be in.

## Changing the release matrix

If you want to add Windows support, or additional architectures, or
NPM/Homebrew/Docker distribution:

1. Edit `.goreleaser.yaml`. GoReleaser docs are excellent:
   https://goreleaser.com/customization/
2. Validate: `goreleaser check`
3. Test without publishing: `goreleaser release --snapshot --clean`
   (produces a `dist/` directory without creating a GitHub release)
4. Commit the change, tag a new release, watch the workflow.

Don't add Homebrew or similar without bumping to at least `v0.1.x`
where `x ≥ 1`, so there's something to upgrade from.
