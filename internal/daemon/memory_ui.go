package daemon

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/jeffdhooton/scry/internal/memory/browse"
)

// defaultMemoryUIAddr is where the daemon permanently serves the live memory
// graph UI. 7279 spells SCRY on a phone keypad (S=7, C=2, R=7, Y=9).
// Loopback-only by design: the page renders transcript-derived data and must
// never be reachable from outside this machine.
const defaultMemoryUIAddr = "127.0.0.1:7279"

// memoryUIShutdownGrace bounds how long closeMemoryUI waits for in-flight
// requests to finish before dropping them, mirroring the spirit of
// DefaultShutdownGrace but scoped much smaller since this is a local,
// single-page GET endpoint, not a long-lived proxy.
const memoryUIShutdownGrace = 2 * time.Second

// startMemoryUI starts the live memory UI HTTP server per SCRY_MEMORY_UI_ADDR
// (default 127.0.0.1:7279; "off" disables it entirely). It is best-effort: a
// listen failure (e.g. the port is already in use) is logged and otherwise
// ignored — the daemon must come up regardless of this feature's state. The
// server is shut down when ctx is done.
func (d *Daemon) startMemoryUI(ctx context.Context) {
	addr := os.Getenv("SCRY_MEMORY_UI_ADDR")
	if addr == "" {
		addr = defaultMemoryUIAddr
	}
	if addr == "off" {
		return
	}
	addr = forceMemoryUILoopback(addr)

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		log.Printf("memory-ui: listen %s: %v (live memory UI disabled this run)", addr, err)
		return
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", d.handleMemoryUIIndex)
	mux.HandleFunc("/data.json", d.handleMemoryUIData)
	srv := &http.Server{Handler: mux}

	d.memUIMu.Lock()
	d.memUISrv = srv
	d.memUIMu.Unlock()

	go func() {
		if err := srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("memory-ui: serve: %v", err)
		}
	}()

	go func() {
		<-ctx.Done()
		d.closeMemoryUI()
	}()

	log.Printf("memory-ui: serving live memory graph at http://%s", addr)
}

// forceMemoryUILoopback returns addr unchanged if its host is 127.0.0.1 or
// localhost. Otherwise (or if addr doesn't parse) it logs a warning and
// falls back to the loopback default, port preserved when possible — this
// page contains transcript-derived data and must never bind non-loopback.
func forceMemoryUILoopback(addr string) string {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		log.Printf("memory-ui: invalid SCRY_MEMORY_UI_ADDR %q (%v) — falling back to %s", addr, err, defaultMemoryUIAddr)
		return defaultMemoryUIAddr
	}
	if host == "127.0.0.1" || host == "localhost" {
		return addr
	}
	log.Printf("memory-ui: refusing non-loopback host %q in SCRY_MEMORY_UI_ADDR — forcing 127.0.0.1 (this page serves transcript-derived data)", host)
	return net.JoinHostPort("127.0.0.1", port)
}

// closeMemoryUI shuts down the memory UI HTTP server, if one was ever
// started. Safe to call multiple times (e.g. once from ctx.Done() and once
// from Run's defer chain) — http.Server.Shutdown is idempotent.
func (d *Daemon) closeMemoryUI() {
	d.memUIMu.Lock()
	srv := d.memUISrv
	d.memUIMu.Unlock()
	if srv == nil {
		return
	}
	shutdownCtx, cancel := context.WithTimeout(context.Background(), memoryUIShutdownGrace)
	defer cancel()
	_ = srv.Shutdown(shutdownCtx)
}

// handleMemoryUIIndex serves the rendered memory-browse HTML page, gathering
// fresh export data on every request — the whole point of the daemon-hosted
// UI over the CLI's `scry memory browse` file is that a reload always
// reflects the current graph.
func (d *Daemon) handleMemoryUIIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	export, err := d.memoryExport()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	html, err := browse.Render(export)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write(html)
}

// handleMemoryUIData serves the raw export as JSON, for future tooling that
// wants the data without the HTML shell.
func (d *Daemon) handleMemoryUIData(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/data.json" {
		http.NotFound(w, r)
		return
	}

	export, err := d.memoryExport()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	_ = json.NewEncoder(w).Encode(export)
}
