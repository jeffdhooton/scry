package config

import (
	"fmt"
	"time"

	"gopkg.in/yaml.v3"
)

// Assessment configures observations only. There is deliberately no enforcement mode.
type Assessment struct {
	Mode              string           `json:"mode"`
	Model             string           `json:"model"`
	TargetInputTokens int              `json:"target_input_tokens"`
	Concurrency       int              `json:"concurrency"`
	Timeout           time.Duration    `json:"timeout"`
	Trial             *AssessmentTrial `json:"trial,omitempty"`
	invalidField      string
}

func DefaultAssessment() Assessment {
	return Assessment{Mode: "off", Model: "jev-1.13.0", TargetInputTokens: 20000, Concurrency: 2, Timeout: 15 * time.Second}
}

// UnmarshalYAML retains invalid assessment settings as a scoped diagnostic.
// YAML errors can contain input values; only the field name is retained.
func (a *Assessment) UnmarshalYAML(n *yaml.Node) error {
	*a = DefaultAssessment()
	if n.Kind != yaml.MappingNode {
		a.invalidField = "configuration"
		return nil
	}
	seen := map[string]bool{}
	for i := 0; i < len(n.Content); i += 2 {
		field, v := n.Content[i].Value, n.Content[i+1]
		var err error
		if seen[field] {
			a.invalidField = "configuration"
			continue
		}
		seen[field] = true
		switch field {
		case "mode":
			err = v.Decode(&a.Mode)
		case "model":
			err = v.Decode(&a.Model)
		case "target_input_tokens":
			err = v.Decode(&a.TargetInputTokens)
		case "concurrency":
			err = v.Decode(&a.Concurrency)
		case "trial":
			a.Trial = &AssessmentTrial{}
			err = v.Decode(a.Trial)
		case "timeout":
			var value string
			err = v.Decode(&value)
			if err == nil {
				a.Timeout, err = time.ParseDuration(value)
			}
		default:
			a.invalidField = "configuration"
		}
		if err != nil {
			a.invalidField = field
		}
	}
	return nil
}

// Effective supplies absent defaults and validates independently of Config.Load.
func (a *Assessment) Effective() (Assessment, error) {
	if a == nil {
		return DefaultAssessment(), nil
	}
	field := a.invalidField
	if field == "" {
		switch {
		case a.Mode != "off" && a.Mode != "shadow":
			field = "mode"
		case a.Model != "jev-1.13.0":
			field = "model"
		case a.TargetInputTokens <= 0 || a.TargetInputTokens > 60000:
			field = "target_input_tokens"
		case a.Concurrency <= 0:
			field = "concurrency"
		case a.Timeout <= 0 || a.Timeout > 15*time.Second:
			field = "timeout"
		case a.Trial != nil && !a.Trial.valid():
			field = "trial"
		}
	}
	if field != "" {
		return *a, fmt.Errorf("config: invalid memory.assessment.%s", field)
	}
	return *a, nil
}
