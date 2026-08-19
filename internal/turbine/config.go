package turbine

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Config holds fleet-wide defaults used when a turbine record is missing
// an explicit field.
type Config struct {
	Site           string             `json:"site"`
	DefaultCutIn   float64            `json:"default_cut_in_wind_mps"`
	DefaultCutOut  float64            `json:"default_cut_out_wind_mps"`
	AlarmThreshold map[string]float64 `json:"alarm_thresholds"`
}

func DefaultConfig() Config {
	return Config{
		Site:           "NORTH-PLAINS",
		DefaultCutIn:   3.0,
		DefaultCutOut:  25.0,
		AlarmThreshold: map[string]float64{"gearbox_temp": 85.0, "gen_temp": 95.0},
	}
}

// LoadConfig reads a JSON config file, falling back to defaults when the
// file is absent or a section is empty.
func LoadConfig(path string) (Config, error) {
	if path == "" {
		return DefaultConfig(), nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return DefaultConfig(), nil
		}
		return Config{}, fmt.Errorf("read turbine config: %w", err)
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("parse turbine config: %w", err)
	}
	if cfg.Site == "" {
		cfg.Site = DefaultConfig().Site
	}
	if cfg.DefaultCutIn <= 0 {
		cfg.DefaultCutIn = DefaultConfig().DefaultCutIn
	}
	if cfg.DefaultCutOut <= 0 {
		cfg.DefaultCutOut = DefaultConfig().DefaultCutOut
	}
	if len(cfg.AlarmThreshold) == 0 {
		cfg.AlarmThreshold = DefaultConfig().AlarmThreshold
	}
	return cfg, nil
}

// ThresholdFor returns the configured threshold for a metric, falling back to
// a caller-provided default.
func (c Config) ThresholdFor(metric string, fallback float64) float64 {
	if v, ok := c.AlarmThreshold[metric]; ok {
		return v
	}
	return fallback
}

// FromEnv builds a Config from environment variables for quick boot.
func FromEnv() Config {
	cfg := DefaultConfig()
	if v := os.Getenv("SITE_CODE"); v != "" {
		cfg.Site = v
	}
	if v := os.Getenv("DEFAULT_CUT_IN"); v != "" {
		if f, err := strconv.ParseFloat(strings.TrimSpace(v), 64); err == nil {
			cfg.DefaultCutIn = f
		}
	}
	if v := os.Getenv("DEFAULT_CUT_OUT"); v != "" {
		if f, err := strconv.ParseFloat(strings.TrimSpace(v), 64); err == nil {
			cfg.DefaultCutOut = f
		}
	}
	return cfg
}
