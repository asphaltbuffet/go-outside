package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/afero"
)

type configKeyType int

// ConfigKey is the context key for the Config value.
const ConfigKey configKeyType = iota

const (
	// OutputFormatJSON is the JSON output mode value.
	OutputFormatJSON = "json"
	// OutputFormatHuman is the human-readable output mode value.
	OutputFormatHuman = "human"
)

// Config holds the complete application configuration.
type Config struct {
	Output OutputConfig `mapstructure:"output" toml:"output"`
	// Threshold is the Score boundary separating ok from nope. 0 means unset;
	// the App substitutes score.DefaultThreshold. User-tunable (ADR-0004).
	Threshold int `mapstructure:"threshold" toml:"threshold"`
	// TODO: add app-specific config sections (e.g. Database, Logging)
}

// OutputConfig holds output formatting configuration.
type OutputConfig struct {
	DefaultFormat string `mapstructure:"default_format" toml:"default_format"`
	Quiet         int    `mapstructure:"quiet"          toml:"quiet"`
}

// IsJSON reports whether JSON output mode is active.
func (c *Config) IsJSON() bool {
	return c.Output.DefaultFormat == OutputFormatJSON
}

// IsQuiet reports whether quiet mode is active.
func (c *Config) IsQuiet() bool {
	return c.Output.Quiet > 0
}

// New creates a Config from the given optional filepath.
// If filepath is empty, checks GO_OUTSIDE_CONFIG env var, then default XDG locations.
func New(fp string) (*Config, error) {
	return NewWithFS(afero.NewOsFs(), fp)
}

// NewWithFS creates a Config using the given filesystem (useful for testing).
func NewWithFS(fs afero.Fs, fp string) (*Config, error) {
	cfg, err := loadConfig(fs, fp)
	if err != nil {
		return nil, err
	}
	applyDefaults(cfg)
	return cfg, nil
}

// GetDefaults returns a Config populated with default values.
func GetDefaults() *Config {
	cfg := &Config{}
	applyDefaults(cfg)
	return cfg
}

// DefaultConfigPath returns the XDG-compliant default config file path.
func DefaultConfigPath() string {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "go-outside", "go-outside.toml")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "go-outside", "go-outside.toml")
}

func applyDefaults(cfg *Config) {
	if cfg.Output.DefaultFormat == "" {
		cfg.Output.DefaultFormat = OutputFormatHuman
	}
}

// Ensure Config implements the fmt.Stringer interface for debugging.
var _ fmt.Stringer = (*Config)(nil)

func (c *Config) String() string {
	return fmt.Sprintf("Config{output:%s quiet:%d}", c.Output.DefaultFormat, c.Output.Quiet)
}
