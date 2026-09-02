package distill

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

// opencodeSource is the Source value stamped on episodes produced by
// OpenCodeSession.
const opencodeSource = "opencode-session"

// OpenCodeSession is one row of OpenCode's session table, enough for the
// sweep to decide whether it changed.
type OpenCodeSession struct {
	ID          string
	Directory   string
	Title       string
	TimeUpdated time.Time
}

// sqliteTimeout bounds one sqlite3 invocation. The database is local and
// small; anything slower than this is a wedged process, not a slow query.
const sqliteTimeout = 30 * time.Second

// opencodeIDRE is the only shape a session id may take before it is spliced
// into SQL. OpenCode ids are "ses_" plus base62; anything else is refused
// rather than quoted.
var opencodeIDRE = regexp.MustCompile(`^[A-Za-z0-9_]+$`)

// OpenCodeSessions lists every session in an OpenCode database, oldest
// update first. OpenCode stores sessions in SQLite (WAL mode); reading it
// goes through the sqlite3 CLI that ships with macOS so scry stays a
// CGO-free static binary. A missing database is not an error: it yields
// no sessions.
func OpenCodeSessions(dbPath string) ([]OpenCodeSession, error) {
	if _, err := os.Stat(dbPath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	var rows []struct {
		ID          string `json:"id"`
		Directory   string `json:"directory"`
		Title       string `json:"title"`
		TimeUpdated int64  `json:"time_updated"`
	}
	if err := sqliteJSON(dbPath, `select id, directory, title, time_updated from session order by time_updated, id`, &rows); err != nil {
		return nil, err
	}
	out := make([]OpenCodeSession, 0, len(rows))
	for _, r := range rows {
		out = append(out, OpenCodeSession{ID: r.ID, Directory: r.Directory, Title: r.Title, TimeUpdated: time.UnixMilli(r.TimeUpdated)})
	}
	return out, nil
}

// OpenCodeSessionEpisodes distills one session into episodes and reports
// the session's last-updated time. User and assistant text parts become
// turns; tool parts become "[tool: <name>]" breadcrumbs; reasoning parts
// and tool output never reach an episode. Source refs use message
// ordinals rather than byte offsets: "opencode:<db>:<session>#<i>-<j>".
func OpenCodeSessionEpisodes(dbPath, sessionID string) ([]RawEpisode, time.Time, error) {
	if !opencodeIDRE.MatchString(sessionID) {
		return nil, time.Time{}, fmt.Errorf("opencode: refusing session id %q", sessionID)
	}
	var sess []struct {
		Directory   string `json:"directory"`
		TimeUpdated int64  `json:"time_updated"`
	}
	if err := sqliteJSON(dbPath, fmt.Sprintf(`select directory, time_updated from session where id = '%s'`, sessionID), &sess); err != nil {
		return nil, time.Time{}, err
	}
	if len(sess) == 0 {
		return nil, time.Time{}, fmt.Errorf("opencode: session %s not found", sessionID)
	}
	updated := time.UnixMilli(sess[0].TimeUpdated)

	var msgs []struct {
		ID      string `json:"id"`
		Created int64  `json:"time_created"`
		Role    string `json:"role"`
		Cwd     string `json:"cwd"`
	}
	if err := sqliteJSON(dbPath, fmt.Sprintf(`select m.id, m.time_created, json_extract(m.data,'$.role') role, json_extract(m.data,'$.path.cwd') cwd from message m where m.session_id = '%s' order by m.time_created, m.id`, sessionID), &msgs); err != nil {
		return nil, updated, err
	}
	var parts []struct {
		MessageID string `json:"message_id"`
		Type      string `json:"type"`
		Text      string `json:"text"`
		Tool      string `json:"tool"`
	}
	if err := sqliteJSON(dbPath, fmt.Sprintf(`select p.message_id, json_extract(p.data,'$.type') type, json_extract(p.data,'$.text') text, json_extract(p.data,'$.tool') tool from part p where p.session_id = '%s' order by p.id`, sessionID), &parts); err != nil {
		return nil, updated, err
	}
	byMsg := map[string][]string{}
	for _, p := range parts {
		switch p.Type {
		case "text":
			if t := strings.TrimSpace(p.Text); t != "" {
				byMsg[p.MessageID] = append(byMsg[p.MessageID], t)
			}
		case "tool":
			if p.Tool != "" {
				byMsg[p.MessageID] = append(byMsg[p.MessageID], "[tool: "+p.Tool+"]")
			}
		}
	}

	var turns []turn
	for i, m := range msgs {
		if m.Role != "user" && m.Role != "assistant" {
			continue
		}
		text := strings.Join(byMsg[m.ID], "\n")
		if strings.TrimSpace(text) == "" {
			continue
		}
		cwd := m.Cwd
		if cwd == "" {
			cwd = sess[0].Directory
		}
		turns = append(turns, turn{role: m.Role, text: text, start: int64(i), end: int64(i + 1), ts: time.UnixMilli(m.Created), cwd: cwd})
	}
	if countSubstantive(turns) < minSubstantiveTurns {
		return nil, updated, nil
	}
	return chunkTurns(opencodeSource, OpenCodeRef(dbPath, sessionID), turns), updated, nil
}

// OpenCodeRef is the cursor/source-ref key for one session:
// "opencode:<db>:<session>". ParseOpenCodeRef inverts it.
func OpenCodeRef(dbPath, sessionID string) string {
	return "opencode:" + dbPath + ":" + sessionID
}

// ParseOpenCodeRef splits an OpenCodeRef into its database path and
// session id. The id never contains ':', so the split is from the right.
func ParseOpenCodeRef(ref string) (dbPath, sessionID string, ok bool) {
	rest, ok := strings.CutPrefix(ref, "opencode:")
	if !ok {
		return "", "", false
	}
	i := strings.LastIndex(rest, ":")
	if i <= 0 || i == len(rest)-1 {
		return "", "", false
	}
	return rest[:i], rest[i+1:], true
}

// sqliteJSON runs one read-only query through the sqlite3 CLI and decodes
// its JSON rows into out. No rows leaves out untouched (sqlite3 prints
// nothing rather than "[]").
func sqliteJSON(dbPath, query string, out any) error {
	ctx, cancel := context.WithTimeout(context.Background(), sqliteTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "sqlite3", "-json", "-readonly", dbPath, query)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			return fmt.Errorf("opencode: sqlite3 not found on PATH; it ships with macOS and most Linux distributions: %w", err)
		}
		return fmt.Errorf("opencode: sqlite3 %s: %v: %s", dbPath, err, strings.TrimSpace(stderr.String()))
	}
	if strings.TrimSpace(stdout.String()) == "" {
		return nil
	}
	return json.Unmarshal(stdout.Bytes(), out)
}
