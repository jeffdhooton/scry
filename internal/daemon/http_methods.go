package daemon

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	scryhttp "github.com/jeffdhooton/scry/internal/http"
	httpstore "github.com/jeffdhooton/scry/internal/http/store"
	"github.com/jeffdhooton/scry/internal/rpc"
)

func (d *Daemon) registerHTTPMethods() {
	d.server.Register("http.start", d.handleHTTPStart)
	d.server.Register("http.stop", d.handleHTTPStop)
	d.server.Register("http.requests", d.handleHTTPRequests)
	d.server.Register("http.request", d.handleHTTPRequest)
	d.server.Register("http.status", d.handleHTTPStatus)
}

// --- http.start ---

type HTTPStartParams struct {
	Port   int    `json:"port,omitempty"`
	Target string `json:"target,omitempty"`
}

type HTTPStartResult struct {
	Port   int    `json:"port"`
	Target string `json:"target"`
	Status string `json:"status"`
}

func (d *Daemon) handleHTTPStart(_ context.Context, raw json.RawMessage) (any, error) {
	var p HTTPStartParams
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, &rpc.Error{Code: rpc.CodeInvalidParams, Message: err.Error()}
	}

	config := scryhttp.DefaultProxyConfig()
	if p.Port > 0 {
		config.Port = p.Port
	}
	if p.Target != "" {
		config.TargetAddr = p.Target
	}

	d.proxyMu.Lock()
	defer d.proxyMu.Unlock()

	if d.proxy != nil && d.proxy.Running() {
		return nil, &rpc.Error{
			Code:    rpc.CodeInternalError,
			Message: fmt.Sprintf("proxy already running on :%d", d.proxy.Config().Port),
		}
	}

	if d.httpStore == nil {
		dataDir := filepath.Join(d.scryHome(), "http")
		if err := os.MkdirAll(dataDir, 0o755); err != nil {
			return nil, fmt.Errorf("create http data dir: %w", err)
		}
		st, err := httpstore.Open(httpstore.DefaultOptions(dataDir))
		if err != nil {
			return nil, fmt.Errorf("open http store: %w", err)
		}
		d.httpStore = st
	}

	d.proxy = scryhttp.NewProxy(config, d.httpStore)
	if err := d.proxy.Start(); err != nil {
		return nil, err
	}

	return &HTTPStartResult{
		Port:   config.Port,
		Target: config.TargetAddr,
		Status: "running",
	}, nil
}

// --- http.stop ---

func (d *Daemon) handleHTTPStop(_ context.Context, _ json.RawMessage) (any, error) {
	d.proxyMu.Lock()
	defer d.proxyMu.Unlock()

	if d.proxy == nil || !d.proxy.Running() {
		return map[string]any{"status": "not_running"}, nil
	}

	if err := d.proxy.Stop(); err != nil {
		return nil, err
	}
	return map[string]any{"status": "stopped"}, nil
}

// --- http.requests ---

func (d *Daemon) handleHTTPRequests(_ context.Context, raw json.RawMessage) (any, error) {
	var f httpstore.ListFilter
	if err := json.Unmarshal(raw, &f); err != nil {
		return nil, &rpc.Error{Code: rpc.CodeInvalidParams, Message: err.Error()}
	}

	d.proxyMu.Lock()
	st := d.httpStore
	d.proxyMu.Unlock()

	if st == nil {
		return []httpstore.RequestSummary{}, nil
	}

	results, err := st.List(f)
	if err != nil {
		return nil, err
	}
	if results == nil {
		results = []httpstore.RequestSummary{}
	}
	return results, nil
}

// --- http.request ---

type HTTPRequestParams struct {
	ID string `json:"id"`
}

func (d *Daemon) handleHTTPRequest(_ context.Context, raw json.RawMessage) (any, error) {
	var p HTTPRequestParams
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, &rpc.Error{Code: rpc.CodeInvalidParams, Message: err.Error()}
	}
	if p.ID == "" {
		return nil, &rpc.Error{Code: rpc.CodeInvalidParams, Message: "id is required"}
	}

	d.proxyMu.Lock()
	st := d.httpStore
	d.proxyMu.Unlock()

	if st == nil {
		return nil, &rpc.Error{Code: rpc.CodeInternalError, Message: "no HTTP data — start the proxy first"}
	}

	req, err := st.Get(p.ID)
	if err != nil {
		return nil, err
	}
	return req, nil
}

// --- http.status ---

type HTTPStatusResult struct {
	Running      bool   `json:"running"`
	Port         int    `json:"port,omitempty"`
	Target       string `json:"target,omitempty"`
	RequestCount int    `json:"request_count"`
}

func (d *Daemon) handleHTTPStatus(_ context.Context, _ json.RawMessage) (any, error) {
	d.proxyMu.Lock()
	defer d.proxyMu.Unlock()

	result := &HTTPStatusResult{}
	if d.proxy != nil && d.proxy.Running() {
		cfg := d.proxy.Config()
		result.Running = true
		result.Port = cfg.Port
		result.Target = cfg.TargetAddr
	}
	if d.httpStore != nil {
		result.RequestCount = d.httpStore.Count()
	}
	return result, nil
}

func (d *Daemon) httpStatusEntry() map[string]any {
	d.proxyMu.Lock()
	defer d.proxyMu.Unlock()

	entry := map[string]any{"domain": "http"}
	if d.proxy != nil && d.proxy.Running() {
		cfg := d.proxy.Config()
		entry["running"] = true
		entry["port"] = cfg.Port
		entry["target"] = cfg.TargetAddr
	} else {
		entry["running"] = false
	}
	if d.httpStore != nil {
		entry["request_count"] = d.httpStore.Count()
	}
	return entry
}

func (d *Daemon) closeHTTP() {
	d.proxyMu.Lock()
	defer d.proxyMu.Unlock()
	if d.proxy != nil {
		_ = d.proxy.Stop()
	}
	if d.httpStore != nil {
		_ = d.httpStore.Close()
	}
}

// ensureHTTPStoreForStatus lazily opens the store read-only if it exists on
// disk, so `scry status` can report request counts even when the proxy isn't
// running this daemon session.
func (d *Daemon) ensureHTTPStoreForStatus() {
	d.proxyMu.Lock()
	defer d.proxyMu.Unlock()
	if d.httpStore != nil {
		return
	}
	dataDir := filepath.Join(d.scryHome(), "http")
	if _, err := os.Stat(dataDir); err != nil {
		return
	}
	st, err := httpstore.Open(httpstore.DefaultOptions(dataDir))
	if err != nil {
		return
	}
	d.httpStore = st
}

// lastProxyStartedAt returns nil if proxy not running; used by status.
func (d *Daemon) lastProxyStartedAt() *time.Time {
	d.proxyMu.Lock()
	defer d.proxyMu.Unlock()
	if d.proxy != nil && d.proxy.Running() {
		t := time.Now() // approximate
		return &t
	}
	return nil
}
