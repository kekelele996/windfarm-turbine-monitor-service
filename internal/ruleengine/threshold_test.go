package ruleengine

import (
	"os"
	"path/filepath"
	"testing"

	"windfarm-turbine-monitor-service/internal/turbine"
)

func writeCfg(t *testing.T, body string) string {
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, "cfg.json")
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestSetThresholdSafeOnEmptyConfig(t *testing.T) {
	cfg := turbine.Config{Site: "EAST"}
	cfg.SetThreshold("gearbox_temp", 99)
	if got := cfg.ThresholdFor("gearbox_temp", 0); got != 99 {
		t.Fatalf("threshold not persisted: %v", got)
	}
}

func TestLoadConfigInitializesThresholds(t *testing.T) {
	cfg, err := turbine.LoadConfig(writeCfg(t, `{"site":"EAST","default_cut_in_wind_mps":4}`))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.AlarmThreshold == nil {
		t.Fatalf("threshold map not initialized")
	}
}

func TestThresholdSetBuildsRules(t *testing.T) {
	cfg := turbine.Config{Site: "EAST"}
	rules := ThresholdSet(turbine.Turbine{ID: "WTG-1"}, cfg)
	if len(rules) != 3 {
		t.Fatalf("expected 3 rules, got %d", len(rules))
	}
}

func TestNormalizeThresholdsSafe(t *testing.T) {
	r := NewRegistry(nil)
	cfg := turbine.Config{Site: "EAST"}
	r.NormalizeThresholds(turbine.Turbine{ID: "WTG-1"}, &cfg)
	if cfg.ThresholdFor("gearbox_temp", 0) != 85 {
		t.Fatalf("threshold not normalized: %v", cfg.ThresholdFor("gearbox_temp", 0))
	}
}
