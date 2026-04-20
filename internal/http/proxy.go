// Package http provides the reverse proxy capture engine for runtime HTTP
// visibility. It intercepts requests between a client and a dev server,
// recording full headers, bodies, and timing to a BadgerDB store.
package http

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"sync"
	"time"

	httpstore "github.com/jeffdhooton/scry/internal/http/store"
)

type ProxyConfig struct {
	Port       int    `json:"port"`
	TargetAddr string `json:"target_addr"`
}

func DefaultProxyConfig() ProxyConfig {
	return ProxyConfig{
		Port:       8089,
		TargetAddr: "localhost:8000",
	}
}

type Proxy struct {
	config   ProxyConfig
	store    *httpstore.Store
	server   *http.Server
	listener net.Listener

	mu      sync.Mutex
	running bool
}

func NewProxy(config ProxyConfig, st *httpstore.Store) *Proxy {
	return &Proxy{config: config, store: st}
}

func (p *Proxy) Start() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.running {
		return fmt.Errorf("proxy already running on :%d", p.config.Port)
	}

	target, err := url.Parse(fmt.Sprintf("http://%s", p.config.TargetAddr))
	if err != nil {
		return fmt.Errorf("parse target: %w", err)
	}

	rp := httputil.NewSingleHostReverseProxy(target)
	rp.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		reqID := NewULID()
		p.store.Put(&httpstore.Request{
			ID:             reqID,
			Method:         r.Method,
			URL:            r.URL.String(),
			Path:           r.URL.Path,
			RequestHeaders: r.Header,
			StatusCode:     502,
			Error:          err.Error(),
			StartedAt:      time.Now(),
		})
		w.WriteHeader(http.StatusBadGateway)
		fmt.Fprintf(w, "scry proxy: upstream error: %v", err)
	}

	handler := p.captureHandler(rp)

	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", p.config.Port))
	if err != nil {
		return fmt.Errorf("listen :%d: %w", p.config.Port, err)
	}

	p.server = &http.Server{Handler: handler}
	p.listener = ln
	p.running = true

	go func() {
		if err := p.server.Serve(ln); err != nil && err != http.ErrServerClosed {
			fmt.Fprintf(os.Stderr, "scry proxy: serve error: %v\n", err)
		}
	}()

	return nil
}

func (p *Proxy) Stop() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if !p.running {
		return nil
	}
	p.running = false
	return p.server.Close()
}

func (p *Proxy) Running() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.running
}

func (p *Proxy) Config() ProxyConfig {
	return p.config
}

func (p *Proxy) captureHandler(upstream http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqID := NewULID()
		start := time.Now()

		reqBody, reqTruncated, reqOrigSize := readLimited(r.Body, httpstore.MaxBodySize)
		r.Body = io.NopCloser(bytes.NewReader(reqBody))

		cw := &captureWriter{ResponseWriter: w}
		upstream.ServeHTTP(cw, r)

		dur := time.Since(start)
		rec := &httpstore.Request{
			ID:                    reqID,
			Method:                r.Method,
			URL:                   r.URL.String(),
			Path:                  r.URL.Path,
			RequestHeaders:        r.Header,
			RequestBody:           reqBody,
			RequestBodyTruncated:  reqTruncated,
			RequestBodyOrigSize:   reqOrigSize,
			StatusCode:            cw.statusCode,
			ResponseHeaders:       cw.Header(),
			ResponseBody:          cw.body.Bytes(),
			ResponseBodyTruncated: cw.body.Len() > httpstore.MaxBodySize,
			StartedAt:             start,
			Duration:              dur,
		}
		if rec.ResponseBodyTruncated {
			rec.ResponseBodyOrigSize = cw.body.Len()
			rec.ResponseBody = rec.ResponseBody[:httpstore.MaxBodySize]
		}
		if err := p.store.Put(rec); err != nil {
			fmt.Fprintf(os.Stderr, "scry proxy: store error: %v\n", err)
		}
	})
}

type captureWriter struct {
	http.ResponseWriter
	statusCode int
	body       bytes.Buffer
}

func (cw *captureWriter) WriteHeader(code int) {
	cw.statusCode = code
	cw.ResponseWriter.WriteHeader(code)
}

func (cw *captureWriter) Write(b []byte) (int, error) {
	cw.body.Write(b)
	return cw.ResponseWriter.Write(b)
}

func readLimited(r io.Reader, max int) (data []byte, truncated bool, origSize int) {
	if r == nil {
		return nil, false, 0
	}
	buf := make([]byte, max+1)
	n, _ := io.ReadFull(r, buf)
	if n > max {
		return buf[:max], true, n
	}
	return buf[:n], false, n
}
