package coverage

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// parseGocover parses a Go coverprofile (produced by go test -coverprofile).
//
// Format: each line after the "mode:" header is:
//
//	<import_path>/<file>:<startLine>.<startCol>,<endLine>.<endCol> <numStmt> <count>
//
// modulePath is the Go module path (e.g. "github.com/jeffdhooton/scry") used
// to convert import paths to repo-relative file paths.
func parseGocover(path, repoPath string) ([]CoveredRange, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	modulePath := detectGoModule(repoPath)

	var ranges []CoveredRange
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "mode:") {
			continue
		}

		// Parse: github.com/foo/bar/pkg/file.go:10.2,15.30 1 1
		colonIdx := strings.LastIndex(line, ":")
		if colonIdx < 0 {
			continue
		}
		filePart := line[:colonIdx]
		rest := line[colonIdx+1:]

		// Convert import path to repo-relative path
		relFile := filePart
		if modulePath != "" && strings.HasPrefix(filePart, modulePath) {
			relFile = strings.TrimPrefix(filePart, modulePath)
			relFile = strings.TrimPrefix(relFile, "/")
		}

		// Parse startLine.startCol,endLine.endCol numStmt count
		parts := strings.Fields(rest)
		if len(parts) < 3 {
			continue
		}
		rangePart := parts[0]
		count, err := strconv.Atoi(parts[2])
		if err != nil {
			continue
		}

		// Parse startLine.startCol,endLine.endCol
		commaIdx := strings.Index(rangePart, ",")
		if commaIdx < 0 {
			continue
		}
		startLine, _ := parseLineCol(rangePart[:commaIdx])
		endLine, _ := parseLineCol(rangePart[commaIdx+1:])

		if startLine > 0 && endLine > 0 {
			ranges = append(ranges, CoveredRange{
				File:    relFile,
				Line:    startLine,
				EndLine: endLine,
				Count:   count,
			})
		}
	}
	return ranges, scanner.Err()
}

// parseLineCol parses "line.col" and returns (line, col).
func parseLineCol(s string) (int, int) {
	dotIdx := strings.Index(s, ".")
	if dotIdx < 0 {
		n, _ := strconv.Atoi(s)
		return n, 0
	}
	line, _ := strconv.Atoi(s[:dotIdx])
	col, _ := strconv.Atoi(s[dotIdx+1:])
	return line, col
}

// detectGoModule reads the module path from go.mod in the repo root.
func detectGoModule(repoPath string) string {
	data, err := os.ReadFile(filepath.Join(repoPath, "go.mod"))
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module"))
		}
	}
	return ""
}

func init() {
	// Register parser so the orchestrator can dispatch by format name.
	registerParser("gocover", func(path, repoPath string) ([]CoveredRange, error) {
		return parseGocover(path, repoPath)
	})
}

type parserFunc func(path, repoPath string) ([]CoveredRange, error)

var parsers = map[string]parserFunc{}

func registerParser(name string, fn parserFunc) {
	if _, exists := parsers[name]; exists {
		panic(fmt.Sprintf("coverage: duplicate parser %q", name))
	}
	parsers[name] = fn
}
