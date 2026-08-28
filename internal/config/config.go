// Package config loads the global scry configuration from
// ~/.scry/config.yaml. The file is optional: a missing file is the zero
// Config, and every setting has a default that matches today's behaviour
// without it. Secrets never live here — API keys are named by the
// environment variable that holds them, not written into the file.
package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// FileName is the config file's name under the scry home directory.
const FileName = "config.yaml"

// Config is the parsed ~/.scry/config.yaml.
type Config struct {
	Memory Memory `yaml:"memory"`
}

// Memory configures the memory domain's extraction.
type Memory struct {
	// Models is the ordered extraction chain: the first entry is tried for
	// every episode, and each later entry only when the one before it
	// failed. When present it replaces SCRY_MEMORY_MODEL / SCRY_MEMORY_BASE_URL
	// entirely.
	Models []Model `yaml:"models"`
}

// Model is one entry in the extraction chain.
type Model struct {
	// Model is the model id sent on the wire. Required.
	Model string `yaml:"model"`
	// BaseURL overrides the endpoint; empty means the DeepSeek default.
	BaseURL string `yaml:"base_url"`
	// APIKeyEnv names the environment variable holding this entry's key;
	// empty means the usual SCRY_MEMORY_API_KEY / DEEPSEEK_API_KEY lookup.
	APIKeyEnv string `yaml:"api_key_env"`
}

// Path returns the config file path for scryHome.
func Path(scryHome string) string {
	return filepath.Join(scryHome, FileName)
}

// Load reads and validates scryHome/config.yaml. A missing file yields the
// zero Config and no error; a malformed or invalid file is an error naming
// the file so the fix is obvious.
func Load(scryHome string) (Config, error) {
	path := Path(scryHome)
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return Config{}, nil
	}
	if err != nil {
		return Config{}, fmt.Errorf("config: read %s: %w", path, err)
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("config: parse %s: %w", path, err)
	}
	for i, m := range cfg.Memory.Models {
		if m.Model == "" {
			return Config{}, fmt.Errorf("config: %s: memory.models[%d] has no model", path, i)
		}
	}
	return cfg, nil
}
