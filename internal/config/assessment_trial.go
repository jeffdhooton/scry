package config

import (
	"gopkg.in/yaml.v3"
	"time"
)

// AssessmentTrial bounds a shadow run by source time, wall clock and a durable
// lifetime dispatch count. Changing/restarting the daemon never resets that count.
type AssessmentTrial struct {
	StartsAt    time.Time `json:"starts_at"`
	ExpiresAt   time.Time `json:"expires_at"`
	MaxRequests uint64    `json:"max_requests"`
	invalid     bool
}

func (t *AssessmentTrial) valid() bool {
	return !t.invalid && !t.StartsAt.IsZero() && t.ExpiresAt.After(t.StartsAt) && t.MaxRequests > 0
}

func (t *AssessmentTrial) UnmarshalYAML(n *yaml.Node) error {
	*t = AssessmentTrial{}
	if n.Kind != yaml.MappingNode {
		t.invalid = true
		return nil
	}
	seen := map[string]bool{}
	for i := 0; i < len(n.Content); i += 2 {
		key, v := n.Content[i].Value, n.Content[i+1]
		if seen[key] {
			t.invalid = true
		}
		seen[key] = true
		var err error
		switch key {
		case "starts_at", "expires_at":
			var raw string
			err = v.Decode(&raw)
			if err == nil {
				var parsed time.Time
				parsed, err = time.Parse(time.RFC3339, raw)
				if key == "starts_at" {
					t.StartsAt = parsed
				} else {
					t.ExpiresAt = parsed
				}
			}
		case "max_requests":
			err = v.Decode(&t.MaxRequests)
		default:
			t.invalid = true
		}
		if err != nil {
			t.invalid = true
		}
	}
	return nil
}
