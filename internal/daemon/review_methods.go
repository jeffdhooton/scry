package daemon

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"time"

	"github.com/jeffdhooton/scry/internal/config"
	"github.com/jeffdhooton/scry/internal/review"
)

func (d *Daemon) configureReview() {
	cfg, e := config.Load(d.scryHome())
	if e != nil {
		d.reviewError = e.Error()
		return
	}
	c := cfg.Review
	if c.QuietSeconds < 0 || c.PollSeconds < 0 || c.TimeoutSeconds < 0 {
		d.reviewError = "review durations must be positive"
		return
	}
	opts := review.Options{Enabled: c.Enabled, Repos: c.Repos, QuietPeriod: time.Duration(c.QuietSeconds) * time.Second, PollInterval: time.Duration(c.PollSeconds) * time.Second, Timeout: time.Duration(c.TimeoutSeconds) * time.Second, MaxInputBytes: c.MaxInputBytes, MaxOutputTokens: c.MaxOutputTokens, MaxRequestsPerDay: c.MaxRequestsPerDay, Retain: c.Retain, Exclude: c.Exclude, Provider: c.Protocol, Model: c.Model, MaxDailyUSD: c.MaxDailyUSD, InputUSDPerMillion: c.InputUSDPerMillion, OutputUSDPerMillion: c.OutputUSDPerMillion}
	var provider review.Reviewer
	if c.Enabled {
		max := c.MaxOutputTokens
		if max == 0 {
			max = 4096
		}
		provider, e = review.NewProvider(review.ProviderConfig{Protocol: c.Protocol, BaseURL: c.BaseURL, Model: c.Model, APIKeyEnv: c.APIKeyEnv, MaxOutputTokens: max})
		if e != nil {
			d.reviewError = e.Error()
		}
	}
	enrich := d.reviewEnricher(c.IncludeMemory, c.Exclude, cfg.MemorySocket())
	d.reviewService, e = review.NewService(filepath.Join(d.scryHome(), "reviews"), opts, provider, enrich)
	if e != nil {
		d.reviewError = e.Error()
	}
}
func (d *Daemon) registerReviewMethods() {
	for _, action := range []string{"status", "preview", "run", "list", "get", "resume"} {
		d.server.Register("review."+action, func(ctx context.Context, raw json.RawMessage) (any, error) {
			var p struct {
				Repo string `json:"repo"`
				ID   string `json:"id"`
			}
			if len(raw) == 0 {
				raw = []byte("{}")
			}
			dec := json.NewDecoder(bytes.NewReader(raw))
			dec.DisallowUnknownFields()
			if e := dec.Decode(&p); e != nil {
				return nil, invalidParams(e)
			}
			var extra any
			if e := dec.Decode(&extra); e != io.EOF {
				return nil, invalidParams(fmt.Errorf("expected one parameter object"))
			}
			if action == "status" {
				if d.reviewService == nil {
					return map[string]any{"ready": false, "error": d.reviewError}, nil
				}
				return map[string]any{"service": d.reviewService.Status(), "error": d.reviewError}, nil
			}
			if d.reviewService == nil {
				return nil, fmt.Errorf("review unavailable: %s", d.reviewError)
			}
			ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
			defer cancel()
			switch action {
			case "resume":
				if e := d.reviewService.Resume(); e != nil {
					return nil, e
				}
				return d.reviewService.Status(), nil
			case "preview":
				return d.reviewService.Preview(ctx, p.Repo)
			case "run":
				return d.reviewService.Queue(p.Repo)
			case "list":
				return d.reviewService.List(ctx, p.Repo)
			case "get":
				return d.reviewService.Get(ctx, p.ID)
			}
			return nil, fmt.Errorf("unknown review action")
		})
	}
}
