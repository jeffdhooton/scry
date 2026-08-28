package daemon

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
)

// LaunchAgent is a user LaunchAgent that supervises `scry start
// --foreground`. When one is installed it is the single start authority:
// clients hand starts to launchd rather than spawning a detached daemon
// that would race KeepAlive for the same socket (and would lack the
// secret-bearing environment the agent sources).
type LaunchAgent struct {
	Label string
	Path  string
}

var plistLabelRe = regexp.MustCompile(`(?s)<key>\s*Label\s*</key>\s*<string>\s*([^<]+?)\s*</string>`)

// FindLaunchAgent looks for a scry daemon LaunchAgent in the user's
// ~/Library/LaunchAgents. Only meaningful on macOS.
func FindLaunchAgent() (LaunchAgent, bool) {
	if runtime.GOOS != "darwin" {
		return LaunchAgent{}, false
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return LaunchAgent{}, false
	}
	return FindLaunchAgentIn(filepath.Join(home, "Library", "LaunchAgents"))
}

// FindLaunchAgentIn scans dir for a plist whose program arguments run the
// daemon in the foreground, and returns its Label.
func FindLaunchAgentIn(dir string) (LaunchAgent, bool) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return LaunchAgent{}, false
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".plist") {
			continue
		}
		path := filepath.Join(dir, e.Name())
		b, err := os.ReadFile(path)
		if err != nil || !strings.Contains(string(b), "scry start --foreground") {
			continue
		}
		m := plistLabelRe.FindSubmatch(b)
		if m == nil {
			continue
		}
		return LaunchAgent{Label: string(m[1]), Path: path}, true
	}
	return LaunchAgent{}, false
}

// Target is the launchctl service target for the agent in the current
// user's GUI domain.
func (a LaunchAgent) Target() string {
	return fmt.Sprintf("gui/%d/%s", os.Getuid(), a.Label)
}

// Kickstart asks launchd to start the agent now, bypassing any throttle
// from previous exits. Fails if the agent is not bootstrapped, in which
// case the caller falls back to spawning the daemon directly.
func (a LaunchAgent) Kickstart() error {
	out, err := exec.Command("launchctl", "kickstart", a.Target()).CombinedOutput()
	if err != nil {
		return fmt.Errorf("launchctl kickstart %s: %v: %s", a.Target(), err, strings.TrimSpace(string(out)))
	}
	return nil
}
