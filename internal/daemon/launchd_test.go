package daemon

import (
	"os"
	"path/filepath"
	"testing"
)

const scrydPlist = `<?xml version="1.0" encoding="UTF-8"?>
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.example.scryd</string>
    <key>ProgramArguments</key>
    <array>
        <string>/bin/zsh</string>
        <string>-lc</string>
        <string>source ~/.secrets.zsh 2&gt;/dev/null; exec /usr/local/bin/scry start --foreground</string>
    </array>
    <key>KeepAlive</key>
    <true/>
</dict>
</plist>
`

const unrelatedPlist = `<?xml version="1.0" encoding="UTF-8"?>
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.example.other</string>
    <key>ProgramArguments</key>
    <array><string>/usr/bin/true</string></array>
</dict>
</plist>
`

// FindLaunchAgentIn locates the LaunchAgent that supervises the daemon by
// its program arguments, so clients can hand daemon starts to launchd
// instead of racing it with a detached spawn.
func TestFindLaunchAgentInMatchesByProgramArguments(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "com.example.other.plist"), []byte(unrelatedPlist), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "com.example.scryd.plist"), []byte(scrydPlist), 0o644); err != nil {
		t.Fatal(err)
	}
	agent, ok := FindLaunchAgentIn(dir)
	if !ok {
		t.Fatal("no LaunchAgent found")
	}
	if agent.Label != "com.example.scryd" {
		t.Errorf("Label = %q, want com.example.scryd", agent.Label)
	}
	if agent.Path != filepath.Join(dir, "com.example.scryd.plist") {
		t.Errorf("Path = %q", agent.Path)
	}
}

func TestFindLaunchAgentInReportsAbsence(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "com.example.other.plist"), []byte(unrelatedPlist), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, ok := FindLaunchAgentIn(dir); ok {
		t.Error("found a LaunchAgent in a directory with none for scry")
	}
	if _, ok := FindLaunchAgentIn(filepath.Join(dir, "missing")); ok {
		t.Error("found a LaunchAgent in a missing directory")
	}
}
