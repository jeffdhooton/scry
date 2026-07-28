package daemon

import (
	"os"
	"path/filepath"
	"strings"
)

// canonicalPath returns the canonical form of p: absolute, symlinks resolved,
// and each component rewritten to its on-disk casing. Case normalization
// matters on case-insensitive filesystems (macOS, Windows) where
// /Users/x/Herd/childscribe and /Users/x/Herd/ChildScribe name the same
// directory but hash to different index keys. If p (or a component) doesn't
// exist, the absolute form is returned unchanged — callers use the result in
// error messages, so it must never fail.
func canonicalPath(p string) string {
	abs, err := filepath.Abs(p)
	if err != nil {
		return p
	}
	resolved, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return abs
	}
	return normalizeCase(resolved)
}

// normalizeCase rewrites each component of an existing absolute path to the
// casing recorded in its parent directory. Components that can't be listed
// (permissions, virtual filesystems) are kept as given.
func normalizeCase(p string) string {
	parts := strings.Split(p, string(filepath.Separator))
	out := string(filepath.Separator)
	for _, part := range parts {
		if part == "" {
			continue
		}
		entries, err := os.ReadDir(out)
		if err != nil {
			out = filepath.Join(out, part)
			continue
		}
		name := part
		exact := false
		for _, e := range entries {
			if e.Name() == part {
				exact = true
				break
			}
		}
		if !exact {
			for _, e := range entries {
				if strings.EqualFold(e.Name(), part) {
					name = e.Name()
					break
				}
			}
		}
		out = filepath.Join(out, name)
	}
	return out
}
