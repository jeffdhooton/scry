package doctor

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/jeffdhooton/scry/internal/config"
	"github.com/jeffdhooton/scry/internal/daemon"
	"github.com/jeffdhooton/scry/internal/rpc"
)

// CategoryMemory groups the memory-domain checks.
const CategoryMemory Category = "Memory"

const (
	// maxIngestAge is how long the store may go without a transcript-derived
	// episode before ingestion counts as dead. Every agent on the machine
	// produces sessions during a working day, and the sweep runs every 30
	// minutes, so six quiet hours means the pipeline stopped, not the user.
	maxIngestAge = 6 * time.Hour
	// maxSweepAge is the longest gap between sweeps before the schedule
	// itself is suspect (the interval is 30 minutes).
	maxSweepAge = 2 * time.Hour
	// maxQueueReady is a queue depth that means the worker is not keeping
	// up, or is not running at all.
	maxQueueReady = 50
	// maxExtractGap is how long a non-empty queue may go without a single
	// successful extraction before the pipeline counts as stopped. The
	// worker can look perfectly healthy while every attempt fails, so
	// liveness is measured by work completed, not by a running goroutine.
	maxExtractGap = 30 * time.Minute
)

// memorySocket resolves the memory daemon's socket the same way the CLI
// does: SCRY_MEMORY_SOCKET, then memory.socket in config.yaml, then the
// local daemon.
func memorySocket(scryHome string) (path, source string) {
	if p := os.Getenv("SCRY_MEMORY_SOCKET"); p != "" {
		return p, "SCRY_MEMORY_SOCKET"
	}
	if cfg, err := config.Load(scryHome); err == nil {
		if p := cfg.MemorySocket(); p != "" {
			return p, "config.yaml memory.socket"
		}
	}
	return daemon.LayoutFor(scryHome).SocketPath, "local daemon"
}

// checkMemory queries memory.status on the memory daemon and evaluates the
// ingestion checks. A daemon that cannot be reached yields one failing
// check rather than four, since every other question is moot.
func checkMemory(scryHome string, timeout time.Duration) []Check {
	socket, source := memorySocket(scryHome)
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	client, err := rpc.Dial(socket)
	if err != nil {
		return []Check{{
			ID: "memory.daemon", Category: CategoryMemory, Name: "memory daemon reachable",
			Status: StatusFail, Detail: fmt.Sprintf("cannot dial %s (%s): %v", socket, source, err),
			Remedy: "check the tunnel (launchctl print gui/$(id -u)/com.jhoot.scry-memory-tunnel) or the daemon on the store's machine",
		}}
	}
	defer client.Close()
	var res daemon.MemoryStatusResult
	if err := client.Call(ctx, "memory.status", nil, &res); err != nil {
		return []Check{{
			ID: "memory.daemon", Category: CategoryMemory, Name: "memory daemon reachable",
			Status: StatusFail, Detail: fmt.Sprintf("memory.status on %s (%s) failed: %v", socket, source, err),
			Remedy: "the daemon answered but has no memory domain; deploy a current build",
		}}
	}
	checks := []Check{{
		ID: "memory.daemon", Category: CategoryMemory, Name: "memory daemon reachable",
		Status: StatusPass, Detail: fmt.Sprintf("%s (%s): %d entities, %d facts, %d episodes", socket, source, res.Entities, res.Facts, res.Episodes),
	}}
	return append(checks, evalMemoryStatus(&res, time.Now())...)
}

// evalMemoryStatus turns a memory.status payload into the four ingestion
// checks. Pure, so the thresholds are table-testable.
func evalMemoryStatus(res *daemon.MemoryStatusResult, now time.Time) []Check {
	var out []Check

	extraction := Check{ID: "memory.extraction", Category: CategoryMemory, Name: "extraction chain"}
	switch {
	case res.Dormant:
		extraction.Status = StatusFail
		extraction.Detail = "dormant: no extraction model configured, queued writes never become facts"
		extraction.Remedy = "set memory.models in ~/.scry/config.yaml on the store's machine and export the key in its launchd environment"
	case res.AllModelsRefusing:
		extraction.Status = StatusFail
		extraction.Detail = fmt.Sprintf("every model in the chain %v is being refused over billing or authentication; queued work is held, not lost", res.ModelsRefusing)
		extraction.Remedy = "recharge the provider account, or point memory.models in the store machine's ~/.scry/config.yaml at a chain that has credit"
	case len(res.ModelsRefusing) > 0:
		extraction.Status = StatusWarn
		extraction.Detail = fmt.Sprintf("chain %v, worker running, but %v is being refused over billing or authentication", res.Models, res.ModelsRefusing)
		extraction.Remedy = "recharge that provider before the rest of the chain runs out too"
	case !res.WorkerRunning:
		extraction.Status = StatusFail
		extraction.Detail = fmt.Sprintf("chain %v configured but the queue worker is not running", res.Models)
		extraction.Remedy = "restart the daemon; the worker starts with it"
	default:
		extraction.Status = StatusPass
		extraction.Detail = fmt.Sprintf("chain %v, worker running", res.Models)
		if res.LastExtractOKAt != nil {
			extraction.Detail += fmt.Sprintf(", last success %s ago", ageString(now.Sub(*res.LastExtractOKAt)))
		}
	}
	out = append(out, extraction)

	ingest := Check{ID: "memory.ingest_age", Category: CategoryMemory, Name: "hours since last ingest"}
	switch {
	case res.LastIngestAt == nil:
		ingest.Status = StatusFail
		ingest.Detail = "no transcript has ever been queued into this store"
		ingest.Remedy = "run `scry memory sweep` and check ~/.scry/logs/memory-sweep.log"
	case now.Sub(*res.LastIngestAt) > maxIngestAge:
		ingest.Status = StatusFail
		ingest.Detail = fmt.Sprintf("%.1f hours since the last transcript was queued (limit %.0f)", now.Sub(*res.LastIngestAt).Hours(), maxIngestAge.Hours())
		ingest.Remedy = "run `scry memory sweep` by hand and read its errors; check the sweep LaunchAgent is loaded"
	default:
		ingest.Status = StatusPass
		ingest.Detail = fmt.Sprintf("%.1f hours", now.Sub(*res.LastIngestAt).Hours())
	}
	out = append(out, ingest)

	sweep := Check{ID: "memory.sweep_age", Category: CategoryMemory, Name: "last sweep"}
	switch {
	case res.LastSweepAt == nil:
		sweep.Status = StatusWarn
		sweep.Detail = "no sweep has reported to this store yet"
		sweep.Remedy = "load the sweep LaunchAgent (com.jhoot.scry-memory-sweep on the laptop, ai.jermes.scry-memory-sweep on the mini)"
	case now.Sub(*res.LastSweepAt) > maxSweepAge:
		sweep.Status = StatusWarn
		sweep.Detail = fmt.Sprintf("%s ago (interval is 30m)", ageString(now.Sub(*res.LastSweepAt)))
		sweep.Remedy = "launchctl kickstart gui/$(id -u)/com.jhoot.scry-memory-sweep and read ~/.scry/logs/memory-sweep.log"
	default:
		sweep.Status = StatusPass
		sweep.Detail = fmt.Sprintf("%s ago", ageString(now.Sub(*res.LastSweepAt)))
	}
	out = append(out, sweep)

	queue := Check{ID: "memory.queue", Category: CategoryMemory, Name: "extraction queue"}
	stalled := res.QueueReady > 0 && res.LastExtractOKAt != nil && now.Sub(*res.LastExtractOKAt) > maxExtractGap
	switch {
	case stalled:
		queue.Status = StatusFail
		queue.Detail = fmt.Sprintf("%d items waiting and nothing extracted for %s; the queue is stopped", res.QueueReady, ageString(now.Sub(*res.LastExtractOKAt)))
		queue.Remedy = "read the daemon log on the store's machine for the failure every attempt is hitting"
	case res.QueueParked > 0:
		queue.Status = StatusWarn
		queue.Detail = fmt.Sprintf("%d parked (unparseable after %d tries), %d ready, %d backing off", res.QueueParked, 3, res.QueueReady, res.QueueBackoff)
		queue.Remedy = "scry memory queue, then scry memory queue retry <id> once the cause is fixed"
	case res.QueueReady > maxQueueReady:
		queue.Status = StatusWarn
		queue.Detail = fmt.Sprintf("%d ready items waiting; the worker is not keeping up", res.QueueReady)
		queue.Remedy = "check the daemon log for provider failures"
	default:
		queue.Status = StatusPass
		queue.Detail = fmt.Sprintf("%d ready, %d backing off, 0 parked", res.QueueReady, res.QueueBackoff)
	}
	out = append(out, queue)

	return out
}

func ageString(d time.Duration) string {
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%ds", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	default:
		return fmt.Sprintf("%.1fh", d.Hours())
	}
}
