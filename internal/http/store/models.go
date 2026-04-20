package store

import "time"

const MaxBodySize = 512 * 1024

type Request struct {
	ID                    string              `json:"id"`
	Method                string              `json:"method"`
	URL                   string              `json:"url"`
	Path                  string              `json:"path"`
	RequestHeaders        map[string][]string `json:"request_headers"`
	RequestBody           []byte              `json:"request_body,omitempty"`
	RequestBodyTruncated  bool                `json:"request_body_truncated,omitempty"`
	RequestBodyOrigSize   int                 `json:"request_body_orig_size,omitempty"`
	StatusCode            int                 `json:"status_code"`
	ResponseHeaders       map[string][]string `json:"response_headers,omitempty"`
	ResponseBody          []byte              `json:"response_body,omitempty"`
	ResponseBodyTruncated bool                `json:"response_body_truncated,omitempty"`
	ResponseBodyOrigSize  int                 `json:"response_body_orig_size,omitempty"`
	Error                 string              `json:"error,omitempty"`
	StartedAt             time.Time           `json:"started_at"`
	Duration              time.Duration       `json:"duration"`
}

func (r *Request) Summary() RequestSummary {
	return RequestSummary{
		ID:         r.ID,
		Method:     r.Method,
		Path:       r.Path,
		StatusCode: r.StatusCode,
		Duration:   r.Duration,
		StartedAt:  r.StartedAt,
		Error:      r.Error,
	}
}

type RequestSummary struct {
	ID         string        `json:"id"`
	Method     string        `json:"method"`
	Path       string        `json:"path"`
	StatusCode int           `json:"status_code"`
	Duration   time.Duration `json:"duration"`
	StartedAt  time.Time     `json:"started_at"`
	Error      string        `json:"error,omitempty"`
}

type ListFilter struct {
	Path      string `json:"path,omitempty"`
	Method    string `json:"method,omitempty"`
	StatusMin int    `json:"status_min,omitempty"`
	StatusMax int    `json:"status_max,omitempty"`
	Limit     int    `json:"limit,omitempty"`
}
