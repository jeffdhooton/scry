package doctor

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/jeffdhooton/scry/internal/daemon"
)

// Split-brain detection. docs/DAEMON_SPLIT_BRAIN_DIAGNOSIS.md: three
// `scry start --foreground` processes survived a takeover race — one on the
// RPC socket, one on the memory UI port, one holding watchers — and the only
// visible symptom was the UI returning 500. `scry doctor` said 0 failed.
// These checks make that state loud.

var foregroundDaemonRe = regexp.MustCompile(`^\s*(\d+)\s+(?:\S*/)?scry start --foreground\s*$`)

// parseForegroundDaemonPIDs extracts daemon PIDs from `ps -axo pid=,args=`
// output. Only a process whose own command is `scry start --foreground`
// counts: not `scry mcp`, not the shell wrapper launchd runs before exec.
func parseForegroundDaemonPIDs(ps string) []int {
	var pids []int
	for _, line := range strings.Split(ps, "\n") {
		m := foregroundDaemonRe.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		pid, err := strconv.Atoi(m[1])
		if err != nil {
			continue
		}
		pids = append(pids, pid)
	}
	sort.Ints(pids)
	return pids
}

func listForegroundDaemonPIDs(timeout time.Duration) ([]int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	out, err := exec.CommandContext(ctx, "ps", "-axo", "pid=,args=").Output()
	if err != nil {
		return nil, err
	}
	return parseForegroundDaemonPIDs(string(out)), nil
}

func evalDaemonInstances(canonical int, pids []int) Check {
	c := Check{ID: "daemon.instances", Category: CategoryDaemon, Name: "single daemon"}
	var orphans []string
	for _, pid := range pids {
		if pid != canonical {
			orphans = append(orphans, strconv.Itoa(pid))
		}
	}
	switch {
	case len(pids) == 0 && canonical == 0:
		c.Status = StatusPass
		c.Detail = "no daemon process"
	case len(orphans) == 0:
		c.Status = StatusPass
		c.Detail = fmt.Sprintf("exactly one foreground daemon (pid %d)", canonical)
	case canonical == 0:
		c.Status = StatusFail
		c.Detail = fmt.Sprintf("%d foreground daemon process(es) running (pid %s) but none answers on the socket",
			len(pids), strings.Join(orphans, ", "))
		c.Remedy = "scry daemon restart (retires the unresponsive daemon and waits for it to exit)"
	default:
		c.Status = StatusFail
		c.Detail = fmt.Sprintf("%d foreground daemons: canonical pid %d plus orphan pid %s holding watchers, stores, or the UI port",
			len(pids), canonical, strings.Join(orphans, ", "))
		c.Remedy = "follow docs/DAEMON_SPLIT_BRAIN_DIAGNOSIS.md: stop launchd KeepAlive, `scry stop`, kill -TERM the orphan pids, then start one daemon"
	}
	return c
}

func checkDaemonInstances(scryHome string, timeout time.Duration) Check {
	_, canonical := daemon.AliveDaemon(daemon.LayoutFor(scryHome))
	pids, err := listForegroundDaemonPIDs(timeout)
	if err != nil {
		return Check{
			ID: "daemon.instances", Category: CategoryDaemon, Name: "single daemon",
			Status: StatusSkip, Detail: "could not list processes: " + err.Error(),
		}
	}
	return evalDaemonInstances(canonical, pids)
}

// uiProbe is the outcome of GET /health on the memory UI.
type uiProbe struct {
	Err         error
	StatusCode  int
	PID         int
	MemoryOK    bool
	MemoryError string
}

func probeMemoryUI(addr string, timeout time.Duration) uiProbe {
	client := &http.Client{Timeout: timeout}
	resp, err := client.Get("http://" + addr + "/health")
	if err != nil {
		return uiProbe{Err: err}
	}
	defer resp.Body.Close()
	p := uiProbe{StatusCode: resp.StatusCode}
	if resp.StatusCode != http.StatusOK {
		return p
	}
	var h daemon.MemoryUIHealth
	if err := json.NewDecoder(resp.Body).Decode(&h); err != nil {
		return p // older daemon without /health (index page served instead)
	}
	p.PID, p.MemoryOK, p.MemoryError = h.PID, h.MemoryOK, h.MemoryError
	return p
}

func evalMemoryUIHealth(canonical int, addr string, p uiProbe) Check {
	c := Check{ID: "daemon.memory_ui", Category: CategoryDaemon, Name: "memory UI ownership"}
	switch {
	case canonical == 0 && p.Err != nil:
		c.Status = StatusSkip
		c.Detail = "daemon not running"
	case p.Err != nil:
		c.Status = StatusFail
		c.Detail = fmt.Sprintf("daemon pid %d is running but nothing answers on %s — the daemon could not bind the UI port, so another process (possibly an orphan daemon) holds it: %v",
			canonical, addr, p.Err)
		c.Remedy = fmt.Sprintf("lsof -nP -iTCP:%s -sTCP:LISTEN", portOf(addr))
	case p.StatusCode != http.StatusOK:
		c.Status = StatusFail
		c.Detail = fmt.Sprintf("memory UI at %s returned HTTP %d", addr, p.StatusCode)
		c.Remedy = "scry daemon restart"
	case p.PID == 0:
		c.Status = StatusWarn
		c.Detail = "running daemon does not report UI health (older build)"
		c.Remedy = "scry daemon restart"
	case p.PID != canonical:
		c.Status = StatusFail
		c.Detail = fmt.Sprintf("memory UI at %s is served by orphan daemon pid %d, not the canonical daemon pid %d", addr, p.PID, canonical)
		if !p.MemoryOK {
			c.Detail += "; it cannot open the memory store: " + p.MemoryError
		}
		c.Remedy = fmt.Sprintf("kill -TERM %d after confirming it is a `scry start --foreground` process, then `scry daemon restart`", p.PID)
	case !p.MemoryOK:
		c.Status = StatusFail
		c.Detail = fmt.Sprintf("memory UI process pid %d cannot open the memory store: %s", p.PID, p.MemoryError)
		c.Remedy = "lsof -nP +D ~/.scry/memory — another process holds the Badger lock"
	default:
		c.Status = StatusPass
		c.Detail = fmt.Sprintf("served by the canonical daemon (pid %d) at http://%s", canonical, addr)
	}
	return c
}

func portOf(addr string) string {
	if i := strings.LastIndex(addr, ":"); i >= 0 {
		return addr[i+1:]
	}
	return addr
}

func checkMemoryUIHealth(scryHome string, timeout time.Duration) Check {
	addr := os.Getenv("SCRY_MEMORY_UI_ADDR")
	if addr == "" {
		addr = "127.0.0.1:7279"
	}
	if addr == "off" {
		return Check{
			ID: "daemon.memory_ui", Category: CategoryDaemon, Name: "memory UI ownership",
			Status: StatusSkip, Detail: "memory UI disabled (SCRY_MEMORY_UI_ADDR=off)",
		}
	}
	_, canonical := daemon.AliveDaemon(daemon.LayoutFor(scryHome))
	return evalMemoryUIHealth(canonical, addr, probeMemoryUI(addr, timeout))
}
