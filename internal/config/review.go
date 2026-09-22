package config

// Review is independent of the memory extraction chain. Its provider and
// repository allowlist are always explicit; credentials remain in the launcher.
type Review struct {
	Enabled             bool     `yaml:"enabled"`
	Repos               []string `yaml:"repos"`
	IncludeMemory       bool     `yaml:"include_memory"`
	QuietSeconds        int      `yaml:"quiet_seconds"`
	PollSeconds         int      `yaml:"poll_seconds"`
	TimeoutSeconds      int      `yaml:"timeout_seconds"`
	MaxInputBytes       int      `yaml:"max_input_bytes"`
	MaxOutputTokens     int      `yaml:"max_output_tokens"`
	MaxRequestsPerDay   int      `yaml:"max_requests_per_day"`
	MaxDailyUSD         float64  `yaml:"max_daily_usd"`
	InputUSDPerMillion  float64  `yaml:"input_usd_per_million"`
	OutputUSDPerMillion float64  `yaml:"output_usd_per_million"`
	Retain              int      `yaml:"retain"`
	Exclude             []string `yaml:"exclude"`
	Protocol            string   `yaml:"protocol"`
	BaseURL             string   `yaml:"base_url"`
	Model               string   `yaml:"model"`
	APIKeyEnv           string   `yaml:"api_key_env"`
}
