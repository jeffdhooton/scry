package index

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	gitstore "github.com/jeffdhooton/scry/internal/git/store"
)

func indexBlame(ctx context.Context, repoPath string, st *gitstore.Store) (int, error) {
	files, err := listTrackedFiles(ctx, repoPath)
	if err != nil {
		return 0, fmt.Errorf("list tracked files: %w", err)
	}

	workers := runtime.NumCPU()
	if workers > 8 {
		workers = 8
	}
	if workers < 1 {
		workers = 1
	}

	var totalLines atomic.Int64
	var firstErr error
	var errOnce sync.Once

	ch := make(chan string, workers)
	var wg sync.WaitGroup

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for file := range ch {
				if ctx.Err() != nil {
					return
				}
				recs, err := blameFile(ctx, repoPath, file)
				if err != nil {
					continue
				}
				if len(recs) == 0 {
					continue
				}
				w := st.NewWriter()
				for _, rec := range recs {
					if err := w.PutBlame(file, &rec); err != nil {
						errOnce.Do(func() { firstErr = err })
						return
					}
				}
				if err := w.Flush(); err != nil {
					errOnce.Do(func() { firstErr = err })
					return
				}
				totalLines.Add(int64(len(recs)))
			}
		}()
	}

	for _, f := range files {
		ch <- f
	}
	close(ch)
	wg.Wait()

	if firstErr != nil {
		return 0, firstErr
	}
	return int(totalLines.Load()), nil
}

func listTrackedFiles(ctx context.Context, repoPath string) ([]string, error) {
	cmd := exec.CommandContext(ctx, "git", "ls-files")
	cmd.Dir = repoPath
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git ls-files: %w", err)
	}
	var files []string
	scanner := bufio.NewScanner(bytes.NewReader(out))
	for scanner.Scan() {
		line := scanner.Text()
		if line != "" {
			files = append(files, line)
		}
	}
	return files, nil
}

func blameFile(ctx context.Context, repoPath, relPath string) ([]gitstore.BlameRecord, error) {
	cmd := exec.CommandContext(ctx, "git", "blame", "--porcelain", "--", relPath)
	cmd.Dir = repoPath
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	return parsePorcelainBlame(out)
}

func parsePorcelainBlame(data []byte) ([]gitstore.BlameRecord, error) {
	type commitInfo struct {
		author  string
		email   string
		date    int64
		summary string
	}
	commits := map[string]*commitInfo{}

	var recs []gitstore.BlameRecord
	scanner := bufio.NewScanner(bytes.NewReader(data))

	var currentHash string
	var currentLine int
	var current *commitInfo
	isNew := false

	for scanner.Scan() {
		line := scanner.Text()

		if len(line) > 0 && line[0] == '\t' {
			if current != nil {
				recs = append(recs, gitstore.BlameRecord{
					Author:  current.author,
					Email:   current.email,
					Commit:  currentHash,
					Date:    current.date,
					Summary: current.summary,
					Line:    currentLine,
					Content: line[1:],
				})
			}
			current = nil
			continue
		}

		if len(line) >= 40 && isHexString(line[:40]) {
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				currentHash = parts[0]
				currentLine, _ = strconv.Atoi(parts[2])

				if ci, ok := commits[currentHash]; ok {
					current = ci
					isNew = false
				} else {
					current = &commitInfo{}
					commits[currentHash] = current
					isNew = true
				}
				continue
			}
		}

		if isNew && current != nil {
			if strings.HasPrefix(line, "author ") {
				current.author = line[7:]
			} else if strings.HasPrefix(line, "author-mail ") {
				email := line[12:]
				email = strings.Trim(email, "<>")
				current.email = email
			} else if strings.HasPrefix(line, "author-time ") {
				current.date, _ = strconv.ParseInt(line[12:], 10, 64)
			} else if strings.HasPrefix(line, "summary ") {
				current.summary = line[8:]
			}
		}
	}

	return recs, nil
}

func isHexString(s string) bool {
	for _, c := range s {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return true
}
