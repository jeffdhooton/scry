package review

import "time"

type FileState struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
	Mode   string `json:"mode"`
}
type Evidence struct {
	ID      string `json:"id"`
	Kind    string `json:"kind"`
	Path    string `json:"path"`
	Content string `json:"content"`
}
type Snapshot struct {
	ID           string      `json:"id"`
	Repository   string      `json:"repository"`
	Head         string      `json:"head"`
	CapturedAt   time.Time   `json:"captured_at"`
	Files        []FileState `json:"files"`
	ChangedFiles []string    `json:"changed_files"`
	Evidence     []Evidence  `json:"evidence"`
	Warnings     []string    `json:"warnings"`
}
type CaptureOptions struct {
	MaxBytes int      `json:"max_bytes"`
	Exclude  []string `json:"exclude"`
}
