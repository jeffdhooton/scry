// jev-memory-eval exports synthetic evaluation packets by default. Only --live
// permits provider requests; it never reads the daemon or personal memory store.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/jeffdhooton/scry/internal/memory/assess"
	"github.com/jeffdhooton/scry/internal/memory/assesseval"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	if e := run(); e != nil {
		fmt.Fprintln(os.Stderr, e)
		os.Exit(1)
	}
}
func run() error {
	corpus := flag.String("corpus", "docs/memory-assess/integration-v1/corpus.json", "frozen synthetic corpus")
	out := flag.String("out", "", "new report path (required, refuses overwrite)")
	live := flag.Bool("live", false, "explicitly authorize real paid inference for selected synthetic packets")
	mock := flag.Bool("mock", false, "scripted mock; no quality or billing claim")
	split := flag.String("split", "development", "development, sealed, or all; sealed final results must not tune labels")
	calibrate := flag.String("calibrate-from", "", "offline calibration from frozen context-experiment directory")
	omit := flag.Bool("omit-packets", false, "omit packet content in report (hash and manifest retained)")
	flag.Parse()
	if *out == "" {
		return fmt.Errorf("--out is required")
	}
	if *live && *mock {
		return fmt.Errorf("--live and --mock are mutually exclusive")
	}
	if *calibrate != "" {
		if *live || *mock {
			return fmt.Errorf("calibration cannot run inference")
		}
		c, e := assesseval.CalibrateFrozen(*calibrate)
		if e != nil {
			return e
		}
		f, e := os.OpenFile(*out, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
		if e != nil {
			return e
		}
		defer f.Close()
		enc := json.NewEncoder(f)
		enc.SetIndent("", "  ")
		return enc.Encode(c)
	}
	c, e := assesseval.Load(*corpus)
	if e != nil {
		return e
	}
	if *split != "development" && *split != "sealed" && *split != "all" {
		return fmt.Errorf("invalid --split")
	}
	if *split != "all" {
		cases := c.Cases[:0]
		for _, v := range c.Cases {
			if v.Split == *split {
				cases = append(cases, v)
			}
		}
		c.Cases = cases
		episodes := c.Episodes[:0]
		for _, v := range c.Episodes {
			if v.Split == *split {
				episodes = append(episodes, v)
			}
		}
		c.Episodes = episodes
	}
	// Reserve the destination before paid work so a path/overwrite failure cannot
	// discard completed results. The report is written even when a run fails.
	f, e := os.OpenFile(*out, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if e != nil {
		return e
	}
	defer f.Close()
	o := assesseval.Options{Mode: "preview"}
	if *mock {
		o.Mode = "mock"
	}
	if *live {
		o.Mode = "live"
		o.Client, e = assess.NewClient(os.Getenv("TYPESAFE_API_KEY"))
		if e != nil {
			return e
		}
	}
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	report, runErr := assesseval.Run(ctx, c, o)
	if *omit {
		for i := range report.Results {
			report.Results[i].Packet = nil
		}
	}
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	if e = enc.Encode(report); e != nil {
		return e
	}
	if e = f.Sync(); e != nil {
		return e
	}
	fmt.Fprintf(os.Stderr, "%s: %d requests, %v; synthetic only; report %s\n", o.Mode, report.Requested, report.Counts, *out)
	return runErr
}
