package distill

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// makeOpenCodeDB builds a small OpenCode-shaped SQLite database with the
// sqlite3 CLI. Tests skip when the binary is missing, mirroring what the
// distiller itself does at runtime.
func makeOpenCodeDB(t *testing.T) string {
	t.Helper()
	if _, err := exec.LookPath("sqlite3"); err != nil {
		t.Skip("sqlite3 not on PATH")
	}
	db := filepath.Join(t.TempDir(), "opencode.db")
	sql := `
create table session (id text primary key, project_id text not null, directory text not null, title text not null, time_created integer not null, time_updated integer not null);
create table message (id text primary key, session_id text not null, time_created integer not null, time_updated integer not null, data text not null);
create table part (id text primary key, message_id text not null, session_id text not null, time_created integer not null, time_updated integer not null, data text not null);
insert into session values ('ses_aaa', 'proj', '/Users/jeff/workspace/cleaning-company', 'Hiring audit', 1788371694995, 1788381345276);
insert into session values ('ses_bbb', 'proj', '/Users/jeff/dotfiles', 'Greeting', 1788296261645, 1788296261777);
insert into message values ('msg_01', 'ses_aaa', 1788371695011, 1788371695011, '{"role":"user","time":{"created":1788371695011},"agent":"build"}');
insert into message values ('msg_02', 'ses_aaa', 1788371700000, 1788371700000, '{"role":"assistant","path":{"cwd":"/Users/jeff/workspace/cleaning-company"},"time":{"created":1788371700000}}');
insert into message values ('msg_03', 'ses_aaa', 1788371800000, 1788371800000, '{"role":"user","time":{"created":1788371800000}}');
insert into message values ('msg_04', 'ses_aaa', 1788371900000, 1788371900000, '{"role":"assistant","path":{"cwd":"/Users/jeff/workspace/cleaning-company"},"time":{"created":1788371900000}}');
insert into message values ('msg_05', 'ses_bbb', 1788296261700, 1788296261700, '{"role":"user"}');
insert into part values ('prt_01', 'msg_01', 'ses_aaa', 0, 0, '{"type":"text","text":"Audit the recruiting pages and tell me which tests cover them."}');
insert into part values ('prt_02', 'msg_02', 'ses_aaa', 0, 0, '{"type":"reasoning","text":"secret reasoning about the audit"}');
insert into part values ('prt_03', 'msg_02', 'ses_aaa', 0, 0, '{"type":"tool","tool":"grep","state":{"output":"tool output sk-abcdefghijklmnopqrstuvwxyz123456"}}');
insert into part values ('prt_04', 'msg_02', 'ses_aaa', 0, 0, '{"type":"text","text":"The recruiting pages live under web/src/recruiting and have no tests."}');
insert into part values ('prt_05', 'msg_03', 'ses_aaa', 0, 0, '{"type":"text","text":"Write the missing tests."}');
insert into part values ('prt_06', 'msg_04', 'ses_aaa', 0, 0, '{"type":"text","text":"Added recruiting.test.ts covering the three pages."}');
insert into part values ('prt_07', 'msg_05', 'ses_bbb', 0, 0, '{"type":"text","text":"hello"}');
`
	cmd := exec.Command("sqlite3", db)
	cmd.Stdin = strings.NewReader(sql)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("sqlite3 create: %v: %s", err, out)
	}
	return db
}

func TestOpenCodeSessionsListsOldestUpdateFirst(t *testing.T) {
	db := makeOpenCodeDB(t)
	sessions, err := OpenCodeSessions(db)
	if err != nil {
		t.Fatal(err)
	}
	if len(sessions) != 2 || sessions[0].ID != "ses_bbb" || sessions[1].ID != "ses_aaa" {
		t.Fatalf("sessions = %+v", sessions)
	}
	if sessions[1].Directory != "/Users/jeff/workspace/cleaning-company" || !sessions[1].TimeUpdated.Equal(time.UnixMilli(1788381345276)) {
		t.Errorf("session = %+v", sessions[1])
	}
}

func TestOpenCodeSessionsMissingDBYieldsNothing(t *testing.T) {
	sessions, err := OpenCodeSessions(filepath.Join(t.TempDir(), "nope.db"))
	if err != nil || len(sessions) != 0 {
		t.Errorf("missing db: %v, %v", sessions, err)
	}
}

func TestOpenCodeSessionEpisodesDistillsTextAndBreadcrumbs(t *testing.T) {
	db := makeOpenCodeDB(t)
	episodes, updated, err := OpenCodeSessionEpisodes(db, "ses_aaa")
	if err != nil {
		t.Fatal(err)
	}
	if !updated.Equal(time.UnixMilli(1788381345276)) {
		t.Errorf("updated = %v", updated)
	}
	if len(episodes) != 1 {
		t.Fatalf("episodes = %d, want 1", len(episodes))
	}
	ep := episodes[0]
	if ep.Source != "opencode-session" || ep.Cwd != "/Users/jeff/workspace/cleaning-company" {
		t.Errorf("episode = %+v", ep)
	}
	if ep.SourceRef != OpenCodeRef(db, "ses_aaa")+"#0-4" {
		t.Errorf("SourceRef = %q", ep.SourceRef)
	}
	if ep.OccurredAt.Year() != 2026 {
		t.Errorf("OccurredAt = %v", ep.OccurredAt)
	}
	for _, want := range []string{"User: Audit the recruiting pages", "Assistant: [tool: grep]", "no tests", "User: Write the missing tests.", "Added recruiting.test.ts"} {
		if !strings.Contains(ep.Text, want) {
			t.Errorf("text missing %q:\n%s", want, ep.Text)
		}
	}
	for _, never := range []string{"secret reasoning", "tool output", "sk-abcdefghijklmnopqrstuvwxyz123456"} {
		if strings.Contains(ep.Text, never) {
			t.Errorf("text must not contain %q:\n%s", never, ep.Text)
		}
	}
}

func TestOpenCodeSessionEpisodesShortSessionYieldsNothing(t *testing.T) {
	db := makeOpenCodeDB(t)
	episodes, _, err := OpenCodeSessionEpisodes(db, "ses_bbb")
	if err != nil || len(episodes) != 0 {
		t.Errorf("short session: %v, %v", episodes, err)
	}
	if _, _, err := OpenCodeSessionEpisodes(db, "ses_zzz"); err == nil {
		t.Error("unknown session must error")
	}
	if _, _, err := OpenCodeSessionEpisodes(db, "x' or 1=1 --"); err == nil {
		t.Error("an id with SQL in it must be refused")
	}
}

func TestOpenCodeRefRoundTrip(t *testing.T) {
	ref := OpenCodeRef("/Users/jeff/.local/share/opencode/opencode.db", "ses_aaa")
	db, id, ok := ParseOpenCodeRef(ref)
	if !ok || db != "/Users/jeff/.local/share/opencode/opencode.db" || id != "ses_aaa" {
		t.Errorf("parse = %q %q %v", db, id, ok)
	}
	if _, _, ok := ParseOpenCodeRef("/some/file.jsonl"); ok {
		t.Error("a plain path is not an opencode ref")
	}
	_ = os.Remove
}
