package index

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	gitstore "github.com/jeffdhooton/scry/internal/git/store"
)

const commitDelimiter = "---SCRY-GIT-COMMIT-BOUNDARY---"

func indexLog(ctx context.Context, repoPath string, depth int, st *gitstore.Store) (int, string, error) {
	args := []string{
		"log",
		fmt.Sprintf("--format=%s%%n%%H%%n%%an%%n%%ae%%n%%at%%n%%B%%n%s", commitDelimiter, "---SCRY-GIT-MSG-END---"),
		"--numstat",
	}
	if depth > 0 {
		args = append(args, fmt.Sprintf("-n%d", depth))
	}

	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = repoPath
	out, err := cmd.Output()
	if err != nil {
		return 0, "", fmt.Errorf("git log: %w", err)
	}

	commits, err := parseGitLog(out)
	if err != nil {
		return 0, "", err
	}

	if len(commits) == 0 {
		return 0, "", nil
	}

	w := st.NewWriter()
	for _, c := range commits {
		if err := w.PutCommit(&c); err != nil {
			return 0, "", err
		}
	}
	if err := w.Flush(); err != nil {
		return 0, "", err
	}

	return len(commits), commits[0].Hash, nil
}

func parseGitLog(data []byte) ([]gitstore.CommitRecord, error) {
	var commits []gitstore.CommitRecord
	sections := bytes.Split(data, []byte(commitDelimiter+"\n"))

	for _, section := range sections {
		section = bytes.TrimSpace(section)
		if len(section) == 0 {
			continue
		}

		rec, err := parseOneCommit(section)
		if err != nil {
			continue
		}
		commits = append(commits, rec)
	}

	return commits, nil
}

func parseOneCommit(data []byte) (gitstore.CommitRecord, error) {
	lines := strings.Split(string(data), "\n")
	if len(lines) < 4 {
		return gitstore.CommitRecord{}, fmt.Errorf("too few lines: %d", len(lines))
	}

	rec := gitstore.CommitRecord{
		Hash:   lines[0],
		Author: lines[1],
		Email:  lines[2],
	}

	ts, err := strconv.ParseInt(lines[3], 10, 64)
	if err != nil {
		return gitstore.CommitRecord{}, fmt.Errorf("parse timestamp: %w", err)
	}
	rec.Date = ts

	var msgLines []string
	i := 4
	for ; i < len(lines); i++ {
		if lines[i] == "---SCRY-GIT-MSG-END---" {
			i++
			break
		}
		msgLines = append(msgLines, lines[i])
	}
	rec.Message = strings.TrimSpace(strings.Join(msgLines, "\n"))

	for ; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 3)
		if len(parts) != 3 {
			continue
		}
		added, _ := strconv.Atoi(parts[0])
		removed, _ := strconv.Atoi(parts[1])
		path := parts[2]

		if idx := strings.Index(path, " => "); idx >= 0 {
			if braceStart := strings.LastIndex(path[:idx], "{"); braceStart >= 0 {
				if braceEnd := strings.Index(path[idx:], "}"); braceEnd >= 0 {
					prefix := path[:braceStart]
					suffix := path[idx+braceEnd+1:]
					newPart := path[idx+4 : idx+braceEnd]
					path = prefix + newPart + suffix
				}
			} else {
				path = path[idx+4:]
			}
		}

		status := "M"
		if added > 0 && removed == 0 {
			status = "A"
		}

		rec.Files = append(rec.Files, gitstore.FileChange{
			Path:    path,
			Added:   added,
			Removed: removed,
			Status:  status,
		})
	}

	return rec, nil
}
