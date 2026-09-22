package review

import (
	"bytes"
	"context"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"hash"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"golang.org/x/sys/unix"
)

const (
	defaultEvidenceBytes = 128 << 10
	maxEvidenceBytes     = 4 << 20
	maxInventoryBytes    = 16 << 20
	maxSnapshotFiles     = 100000
	maxSnapshotFileBytes = 256 << 20
	maxSnapshotBytes     = 1 << 30
)

var errCaptureLimit = errors.New("snapshot resource limit exceeded")

type headFile struct{ mode, object string }
type capturedState struct {
	snapshot Snapshot
	heads    map[string]headFile
	index    map[string]headFile
	blobs    map[string]string
}

// Fingerprint hashes HEAD, paths, file modes and file bytes, including deletions.
// Exclusions only affect evidence; they never weaken freshness checking.
func Fingerprint(ctx context.Context, repo string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	state, err := captureState(ctx, repo)
	if err != nil {
		return "", err
	}
	return state.snapshot.ID, nil
}

// Capture constructs bounded, read-only evidence and refuses a moving repository.
func Capture(ctx context.Context, repo string, opts CaptureOptions) (Snapshot, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	if opts.MaxBytes <= 0 {
		opts.MaxBytes = defaultEvidenceBytes
	}
	if opts.MaxBytes > maxEvidenceBytes {
		return Snapshot{}, fmt.Errorf("evidence budget exceeds %d bytes", maxEvidenceBytes)
	}
	before, err := captureState(ctx, repo)
	if err != nil {
		return Snapshot{}, err
	}
	s := before.snapshot
	root, err := os.OpenRoot(s.Repository)
	if err != nil {
		return Snapshot{}, err
	}
	defer root.Close()
	remaining := opts.MaxBytes
	for _, f := range s.Files {
		old, exists := before.heads[f.Path]
		indexed, stagedExists := before.index[f.Path]
		workingMode := gitComparisonMode(f.Mode)
		workChanged := (!exists && f.Mode != "deleted") || (exists && (old.object != before.blobs[f.Path] || old.mode != workingMode))
		indexChanged := exists != stagedExists || old != indexed
		if !workChanged && !indexChanged {
			continue
		}
		s.ChangedFiles = append(s.ChangedFiles, f.Path)
		if PathExcluded(f.Path, opts.Exclude) {
			s.Warnings = append(s.Warnings, "Evidence excluded: "+f.Path)
			continue
		}
		if indexChanged && (indexed.object != before.blobs[f.Path] || indexed.mode != workingMode) {
			mode := indexed.mode
			if !stagedExists {
				mode = "deleted"
			}
			if err := appendSnapshotEvidence(ctx, root, &s, f.Path, "staged_diff", old, mode, indexed.object, nil, &remaining); err != nil {
				return Snapshot{}, err
			}
		}
		if workChanged {
			if err := appendSnapshotEvidence(ctx, root, &s, f.Path, "diff", old, f.Mode, "", &f, &remaining); err != nil {
				return Snapshot{}, err
			}
		}
	}

	after, err := captureState(ctx, s.Repository)
	if err != nil {
		return Snapshot{}, err
	}
	if s.ID != after.snapshot.ID {
		return Snapshot{}, errors.New("repository changed during snapshot capture")
	}
	s.CapturedAt = time.Now().UTC()
	return s, nil
}

func captureState(ctx context.Context, repo string) (capturedState, error) {
	var state capturedState
	abs, err := filepath.Abs(repo)
	if err != nil {
		return state, err
	}
	abs, err = filepath.EvalSymlinks(abs)
	if err != nil {
		return state, err
	}
	out, cut, err := gitOutput(ctx, abs, maxInventoryBytes, "rev-parse", "--show-toplevel")
	if err != nil || cut {
		return state, fmt.Errorf("resolve repository: %w", nonNilError(err))
	}
	canonical := strings.TrimSpace(string(out))
	if canonical != abs {
		return state, errors.New("review repository must be the Git worktree root")
	}
	state.snapshot.Repository = abs
	state.heads = map[string]headFile{}
	state.index = map[string]headFile{}
	state.blobs = map[string]string{}
	head, _, headErr := gitOutput(ctx, abs, 256, "rev-parse", "--verify", "HEAD")
	if headErr == nil {
		state.snapshot.Head = strings.TrimSpace(string(head))
	} else {
		// Only an unborn symbolic branch is accepted, never an unreadable HEAD.
		ref, _, e := gitOutput(ctx, abs, 1024, "symbolic-ref", "-q", "HEAD")
		if e != nil {
			return state, headErr
		}
		_, _, e = gitOutput(ctx, abs, 256, "show-ref", "--verify", "--quiet", strings.TrimSpace(string(ref)))
		var exit *exec.ExitError
		if !errors.As(e, &exit) || exit.ExitCode() != 1 {
			return state, headErr
		}
	}
	paths := map[string]bool{}
	if state.snapshot.Head != "" {
		tree, cut, e := gitOutput(ctx, abs, maxInventoryBytes, "ls-tree", "-r", "-z", state.snapshot.Head)
		if e != nil || cut {
			return state, fmt.Errorf("list HEAD files: %w", nonNilError(e))
		}
		for _, record := range strings.Split(string(tree), "\x00") {
			if record == "" {
				continue
			}
			meta, p, ok := strings.Cut(record, "\t")
			fields := strings.Fields(meta)
			if !ok || len(fields) != 3 {
				return state, errors.New("invalid Git tree output")
			}
			if fields[1] != "blob" {
				return state, fmt.Errorf("unsupported repository entry: %s", p)
			}
			state.heads[p] = headFile{fields[0], fields[2]}
			paths[p] = true
		}
	}
	listed, cut, err := gitOutput(ctx, abs, maxInventoryBytes, "ls-files", "--cached", "--others", "--exclude-standard", "-z")
	if err != nil || cut {
		return state, fmt.Errorf("list worktree files: %w", nonNilError(err))
	}
	for _, p := range strings.Split(string(listed), "\x00") {
		if p != "" {
			paths[p] = true
		}
	}
	indexBytes, indexCut, indexErr := gitOutput(ctx, abs, maxInventoryBytes, "ls-files", "--stage", "-z")
	if indexErr != nil || indexCut {
		return state, fmt.Errorf("list index files: %w", nonNilError(indexErr))
	}
	for _, record := range strings.Split(string(indexBytes), "\x00") {
		if record == "" {
			continue
		}
		meta, p, ok := strings.Cut(record, "\t")
		fields := strings.Fields(meta)
		if !ok || len(fields) != 3 {
			return state, errors.New("invalid Git index output")
		}
		if fields[2] != "0" {
			return state, errors.New("unmerged Git index cannot be captured coherently")
		}
		if fields[0] == "160000" {
			return state, fmt.Errorf("unsupported submodule: %s", p)
		}
		state.index[p] = headFile{fields[0], fields[1]}
		paths[p] = true
	}
	if len(paths) > maxSnapshotFiles {
		return state, errCaptureLimit
	}
	sorted := make([]string, 0, len(paths))
	for p := range paths {
		sorted = append(sorted, p)
	}
	sort.Strings(sorted)
	root, err := os.OpenRoot(abs)
	if err != nil {
		return state, err
	}
	defer root.Close()
	objectFormat, _, err := gitOutput(ctx, abs, 32, "rev-parse", "--show-object-format")
	if err != nil {
		return state, err
	}
	total := int64(0)
	for _, p := range sorted {
		if err := ctx.Err(); err != nil {
			return state, err
		}
		f, blob, n, e := hashSnapshotFile(ctx, root, p, strings.TrimSpace(string(objectFormat)))
		if e != nil {
			return state, fmt.Errorf("capture %s: %w", p, e)
		}
		total += n
		if total > maxSnapshotBytes {
			return state, errCaptureLimit
		}
		state.snapshot.Files = append(state.snapshot.Files, f)
		state.blobs[p] = blob
	}
	type indexFile struct{ Path, Mode, Object string }
	indexFiles := make([]indexFile, 0, len(state.index))
	for _, p := range sorted {
		if f, ok := state.index[p]; ok {
			indexFiles = append(indexFiles, indexFile{p, f.mode, f.object})
		}
	}
	encoded, _ := json.Marshal(struct {
		Head  string
		Files []FileState
		Index []indexFile
	}{state.snapshot.Head, state.snapshot.Files, indexFiles})
	digest := sha256.Sum256(encoded)
	state.snapshot.ID = hex.EncodeToString(digest[:])
	return state, nil
}

func nonNilError(err error) error {
	if err == nil {
		return errCaptureLimit
	}
	return err
}

// openSnapshotParent walks each directory using O_NOFOLLOW. It prevents a
// replaced directory symlink from turning a tracked path into a secret read.
func openSnapshotParent(root *os.Root, p string) (*os.File, string, error) {
	if !filepath.IsLocal(p) || strings.Contains(p, "\\") {
		return nil, "", errors.New("unsafe repository path")
	}
	parts := strings.Split(p, "/")
	dir, err := root.Open(".")
	if err != nil {
		return nil, "", err
	}
	for _, part := range parts[:len(parts)-1] {
		fd, e := unix.Openat(int(dir.Fd()), part, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
		dir.Close()
		if e != nil {
			return nil, "", e
		}
		dir = os.NewFile(uintptr(fd), part)
	}
	return dir, parts[len(parts)-1], nil
}

func hashSnapshotFile(ctx context.Context, root *os.Root, p, format string) (FileState, string, int64, error) {
	f := FileState{Path: p, Mode: "deleted"}
	dir, name, err := openSnapshotParent(root, p)
	if os.IsNotExist(err) {
		return f, "", 0, nil
	}
	if err != nil {
		return f, "", 0, err
	}
	defer dir.Close()
	// Root access preserves portable metadata/readlink handling. The parent was
	// checked above; regular-file reads use the pinned directory descriptor.
	info, err := root.Lstat(p)
	if os.IsNotExist(err) {
		return f, "", 0, nil
	}
	if err != nil {
		return f, "", 0, err
	}
	var reader io.Reader
	var file *os.File
	var size int64
	if info.Mode()&os.ModeSymlink != 0 {
		target, e := readSnapshotLink(dir, name)
		if e != nil {
			return f, "", 0, e
		}
		f.Mode = "120000"
		reader = strings.NewReader(target)
		size = int64(len(target))
	} else if info.Mode().IsRegular() {
		fd, e := unix.Openat(int(dir.Fd()), name, unix.O_RDONLY|unix.O_NOFOLLOW|unix.O_NONBLOCK|unix.O_CLOEXEC, 0)
		if e != nil {
			return f, "", 0, e
		}
		file = os.NewFile(uintptr(fd), p)
		defer file.Close()
		opened, e := file.Stat()
		if e != nil {
			return f, "", 0, e
		}
		if !opened.Mode().IsRegular() || !os.SameFile(info, opened) {
			return f, "", 0, errors.New("file changed during capture")
		}
		reader = file
		size = opened.Size()
		f.Mode = fmt.Sprintf("100%03o", opened.Mode().Perm())
	} else {
		return f, "", 0, errors.New("unsupported non-regular file")
	}
	if size > maxSnapshotFileBytes {
		return f, "", 0, errCaptureLimit
	}
	sha := sha256.New()
	var blob hash.Hash = sha1.New()
	if format == "sha256" {
		blob = sha256.New()
	}
	fmt.Fprintf(blob, "blob %d\x00", size)
	n, err := io.Copy(io.MultiWriter(sha, blob), &contextReader{ctx: ctx, reader: io.LimitReader(reader, maxSnapshotFileBytes+1)})
	if err != nil {
		return f, "", 0, err
	}
	if n != size {
		return f, "", 0, errors.New("file changed during capture")
	}
	if file != nil {
		after, e := file.Stat()
		if e != nil {
			return f, "", 0, e
		}
		if after.Size() != info.Size() || after.ModTime() != info.ModTime() || after.Mode() != info.Mode() {
			return f, "", 0, errors.New("file changed during capture")
		}
	}
	f.SHA256 = hex.EncodeToString(sha.Sum(nil))
	return f, hex.EncodeToString(blob.Sum(nil)), n, nil
}

type contextReader struct {
	ctx    context.Context
	reader io.Reader
}

func (r *contextReader) Read(p []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	return r.reader.Read(p)
}

func readEvidence(ctx context.Context, root *os.Root, f FileState, limit int) ([]byte, bool, error) {
	if f.Mode == "120000" {
		dir, name, err := openSnapshotParent(root, f.Path)
		if err != nil {
			return nil, false, err
		}
		defer dir.Close()
		target, err := readSnapshotLink(dir, name)
		data := []byte(target)
		if len(data) > limit {
			return data[:limit], true, err
		}
		return data, false, err
	}
	dir, name, err := openSnapshotParent(root, f.Path)
	if err != nil {
		return nil, false, err
	}
	defer dir.Close()
	fd, err := unix.Openat(int(dir.Fd()), name, unix.O_RDONLY|unix.O_NOFOLLOW|unix.O_NONBLOCK|unix.O_CLOEXEC, 0)
	if err != nil {
		return nil, false, err
	}
	file := os.NewFile(uintptr(fd), f.Path)
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return nil, false, err
	}
	if !info.Mode().IsRegular() {
		return nil, false, errors.New("file changed during capture")
	}
	data, err := io.ReadAll(&contextReader{ctx: ctx, reader: io.LimitReader(file, int64(limit)+1)})
	if len(data) > limit {
		return data[:limit], true, err
	}
	return data, false, err
}

type boundedGitBuffer struct {
	bytes.Buffer
	max       int
	truncated bool
}

func (b *boundedGitBuffer) Write(p []byte) (int, error) {
	left := b.max - b.Len()
	if len(p) > left {
		b.Buffer.Write(p[:left])
		b.truncated = true
		return left, errCaptureLimit
	}
	return b.Buffer.Write(p)
}
func gitOutput(ctx context.Context, repo string, max int, args ...string) ([]byte, bool, error) {
	base := []string{"--no-optional-locks", "--no-replace-objects", "-c", "core.fsmonitor=false", "-c", "core.untrackedCache=false", "-c", "core.quotePath=false", "-C", repo}
	cmd := exec.CommandContext(ctx, "git", append(base, args...)...)
	cmd.WaitDelay = time.Second
	out := &boundedGitBuffer{max: max}
	stderr := &boundedGitBuffer{max: 4096}
	cmd.Stdout = out
	cmd.Stderr = stderr
	err := cmd.Run()
	if ctx.Err() != nil {
		return nil, false, ctx.Err()
	}
	if out.truncated {
		return out.Bytes(), true, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("git %s: %w", args[0], err)
	}
	return out.Bytes(), false, nil
}
func validPrefix(s string, n int) string {
	if len(s) <= n {
		return s
	}
	s = s[:n]
	for !utf8.ValidString(s) && len(s) > 0 {
		s = s[:len(s)-1]
	}
	return s
}

// PathExcluded applies built-in secret exclusions and additional path globs.
func PathExcluded(p string, patterns []string) bool {
	lower := strings.ToLower(p)
	for _, part := range strings.Split(lower, "/") {
		if part == ".envrc" || part == "auth.json" || part == ".git-credentials" || part == ".docker" || part == ".kube" || part == ".env" || strings.HasPrefix(part, ".env.") || strings.HasPrefix(part, "credentials") || part == ".aws" || part == ".ssh" || part == ".gnupg" || part == ".npmrc" || part == ".netrc" || part == ".pypirc" || strings.Contains(part, "private_key") || strings.Contains(part, "private-key") || strings.HasPrefix(part, "id_rsa") || strings.HasPrefix(part, "id_ed25519") || strings.HasPrefix(part, "id_ecdsa") || strings.HasPrefix(part, "id_dsa") || strings.Contains(part, "service-account") || strings.Contains(part, "service_account") || strings.HasPrefix(part, "secrets.") || part == "secrets" {
			return true
		}
	}
	for _, ext := range []string{".pem", ".key", ".p12", ".pfx", ".keystore", ".tfstate", ".tfvars"} {
		if strings.HasSuffix(lower, ext) {
			return true
		}
	}
	for _, pattern := range patterns {
		pattern = strings.TrimPrefix(filepath.ToSlash(pattern), "./")
		if p == pattern || strings.HasPrefix(p, strings.TrimSuffix(pattern, "/**")+"/") {
			return true
		}
		if matchEvidencePattern(pattern, p) {
			return true
		}
		if !strings.Contains(pattern, "/") {
			if match, _ := path.Match(pattern, path.Base(p)); match {
				return true
			}
		}
	}
	return false
}

// ** matches complete path components, including zero nested directories.
func matchEvidencePattern(pattern, name string) bool {
	parts, names := strings.Split(pattern, "/"), strings.Split(name, "/")
	type position struct{ p, n int }
	seen := make(map[position]bool)
	var match func(int, int) bool
	match = func(p, n int) bool {
		key := position{p, n}
		if seen[key] {
			return false
		}
		seen[key] = true
		if p == len(parts) {
			return n == len(names)
		}
		if parts[p] == "**" {
			return match(p+1, n) || (n < len(names) && match(p, n+1))
		}
		if n == len(names) {
			return false
		}
		ok, _ := path.Match(parts[p], names[n])
		return ok && match(p+1, n+1)
	}
	return match(0, 0)
}

func readSnapshotLink(dir *os.File, name string) (string, error) {
	buf := make([]byte, 4097)
	n, err := unix.Readlinkat(int(dir.Fd()), name, buf)
	if err != nil {
		return "", err
	}
	if n == len(buf) {
		return "", errCaptureLimit
	}
	return string(buf[:n]), nil
}

// ReadEvidenceFile reads at most maxBytes from a repository-relative path.
// Secret paths are refused and symlinks yield their target text, never contents.
// Call PathExcluded first to enforce additional configured exclusions.
func ReadEvidenceFile(ctx context.Context, repo, name string, maxBytes int) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if maxBytes <= 0 || maxBytes > maxEvidenceBytes {
		return "", errors.New("invalid evidence byte limit")
	}
	if PathExcluded(name, nil) {
		return "", errors.New("evidence path is excluded")
	}
	root, err := os.OpenRoot(repo)
	if err != nil {
		return "", err
	}
	defer root.Close()
	dir, _, err := openSnapshotParent(root, name)
	if err != nil {
		return "", err
	}
	dir.Close()
	info, err := root.Lstat(name)
	if err != nil {
		return "", err
	}
	mode := "100644"
	if info.Mode()&os.ModeSymlink != 0 {
		mode = "120000"
	}
	data, cut, err := readEvidence(ctx, root, FileState{Path: name, Mode: mode}, maxBytes)
	if err != nil {
		return "", err
	}
	if bytes.IndexByte(data, 0) >= 0 {
		return "", errors.New("binary evidence omitted")
	}
	text := strings.ToValidUTF8(string(data), "�")
	if cut || len(text) > maxBytes {
		const marker = "\n[source truncated]"
		if maxBytes < len(marker) {
			return "", errors.New("source truncated: evidence budget too small for truncation marker")
		}
		return validPrefix(text, maxBytes-len(marker)) + marker, nil
	}
	return text, nil
}

// Raw blobs plus local byte reads avoid executing repository clean filters,
// which git diff may run even with external diff and textconv disabled.
func appendSnapshotEvidence(ctx context.Context, root *os.Root, s *Snapshot, name, kind string, old headFile, newMode, newObject string, current *FileState, remaining *int) error {
	if *remaining == 0 {
		s.Warnings = append(s.Warnings, "Evidence truncated: "+name)
		return nil
	}
	var oldData, newData []byte
	truncated := false
	perSide := *remaining / 3
	if old.object != "" {
		data, cut, err := gitOutput(ctx, s.Repository, perSide, "cat-file", "blob", old.object)
		if err != nil {
			return err
		}
		oldData = data
		truncated = truncated || cut
	}
	if newMode != "deleted" {
		var data []byte
		var cut bool
		var err error
		if current != nil {
			data, cut, err = readEvidence(ctx, root, *current, perSide)
		} else {
			data, cut, err = gitOutput(ctx, s.Repository, perSide, "cat-file", "blob", newObject)
		}
		if err != nil {
			return err
		}
		newData = data
		truncated = truncated || cut
	}
	if bytes.IndexByte(oldData, 0) >= 0 || bytes.IndexByte(newData, 0) >= 0 {
		s.Warnings = append(s.Warnings, "Binary content omitted: "+name)
		return nil
	}
	label := "working tree compared with HEAD"
	if kind == "staged_diff" {
		label = "staged index compared with HEAD"
	}
	diff := fmt.Sprintf("%s\n--- a/%s (mode %s)\n+++ b/%s (mode %s)\n", label, name, old.mode, name, newMode)
	if len(oldData) > 0 {
		diff += "-" + strings.ReplaceAll(strings.TrimSuffix(string(oldData), "\n"), "\n", "\n-") + "\n"
	}
	if len(newData) > 0 {
		diff += "+" + strings.ReplaceAll(strings.TrimSuffix(string(newData), "\n"), "\n", "\n+") + "\n"
	}
	diff = strings.ToValidUTF8(diff, "�")
	if len(diff) > *remaining {
		diff = validPrefix(diff, *remaining)
		truncated = true
	}
	if diff != "" {
		s.Evidence = append(s.Evidence, Evidence{ID: fmt.Sprintf("e%d", len(s.Evidence)+1), Kind: kind, Path: name, Content: diff})
		*remaining -= len(diff)
	}
	if truncated {
		s.Warnings = append(s.Warnings, "Evidence truncated: "+name)
	}
	return nil
}

// Git records executable status for regular files, not all permission bits.
// The complete mode remains in Files so snapshot freshness still detects chmod.
func gitComparisonMode(mode string) string {
	if !strings.HasPrefix(mode, "100") {
		return mode
	}
	bits, err := strconv.ParseUint(mode, 8, 32)
	if err != nil {
		return mode
	}
	if bits&0111 != 0 {
		return "100755"
	}
	return "100644"
}
