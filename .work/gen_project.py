#!/usr/bin/env python3
import os, sys
BASE = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
files = {}

def add(path, content):
    files[path] = content

add("go.mod", """module windfarm-turbine-monitor-service

go 1.23.0
""")

add("runtime_smoke.json", """{
  "mode": "service",
  "start": "go run ./cmd/server",
  "workdir": ".",
  "env": {
    "PORT": "18080",
    "SITE_CODE": "NORTH-PLAINS"
  },
  "ready_url": "http://127.0.0.1:18080/health",
  "ready_status": [200],
  "startup_timeout": 30
}
""")

# ---------------------------------------------------------------------------
# internal/platform
# ---------------------------------------------------------------------------
add("internal/platform/clock.go", """package platform

import "time"

// Clock abstracts time for deterministic tests.
type Clock interface {
	Now() time.Time
}

type SystemClock struct{}

func (SystemClock) Now() time.Time { return time.Now() }

type FixedClock struct {
	T time.Time
}

func (c FixedClock) Now() time.Time { return c.T }
""")

add("internal/platform/identity.go", """package platform

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"
)

// NewID returns a 16-byte random identifier as a hex string.
func NewID(prefix string) string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%s-%s", prefix, hex.EncodeToString(b))
}

// NewSequenceID builds a short, time-ordered identifier for a given subject.
func NewSequenceID(prefix string, n uint64) string {
	return fmt.Sprintf("%s-%d-%d", prefix, time.Now().UnixNano(), n)
}
""")

add("internal/platform/errors.go", """package platform

import "errors"

var (
	ErrNotFound   = errors.New("not found")
	ErrConflict   = errors.New("conflict")
	ErrInvalid    = errors.New("invalid argument")
	ErrUnavailable = errors.New("service unavailable")
)

// Is reports whether target is in err's error chain.
func Is(err, target error) bool { return errors.Is(err, target) }
""")

# ---------------------------------------------------------------------------
# internal/turbine
# ---------------------------------------------------------------------------
add("internal/turbine/model.go", """package turbine

import "time"

type TurbineState string

const (
	StateRunning      TurbineState = "running"
	StateIdle         TurbineState = "idle"
	StateMaintenance  TurbineState = "maintenance"
	StateDecommission TurbineState = "decommissioned"
)

// Turbine describes a single wind turbine in the fleet.
type Turbine struct {
	ID            string       `json:"id"`
	Name          string       `json:"name"`
	Model         string       `json:"model"`
	Site          string       `json:"site"`
	RatedPowerKW  float64      `json:"rated_power_kw"`
	CutInWindMPS  float64      `json:"cut_in_wind_mps"`
	CutOutWindMPS float64      `json:"cut_out_wind_mps"`
	State         TurbineState `json:"state"`
	Commissioned  time.Time    `json:"commissioned"`
	Firmware      string       `json:"firmware"`
}

func (t Turbine) Active() bool {
	return t.State == StateRunning || t.State == StateIdle
}
""")

add("internal/turbine/registry.go", """package turbine

import (
	"fmt"
	"sync"

	"windfarm-turbine-monitor-service/internal/platform"
)

// Registry is a thread-safe in-memory turbine fleet store.
type Registry struct {
	mu       sync.RWMutex
	turbines map[string]Turbine
	order    []string
}

func NewRegistry(seed []Turbine) *Registry {
	r := &Registry{turbines: make(map[string]Turbine, len(seed))}
	for _, t := range seed {
		r.turbines[t.ID] = t
		r.order = append(r.order, t.ID)
	}
	return r
}

func cloneTurbine(t Turbine) Turbine { return t }

func (r *Registry) Put(t Turbine) error {
	if t.ID == "" {
		return fmt.Errorf("turbine id required: %w", platform.ErrInvalid)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.turbines[t.ID]; !exists {
		r.order = append(r.order, t.ID)
	}
	r.turbines[t.ID] = t
	return nil
}

func (r *Registry) Get(id string) (Turbine, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	t, ok := r.turbines[id]
	if !ok {
		return Turbine{}, fmt.Errorf("turbine %s: %w", id, platform.ErrNotFound)
	}
	return cloneTurbine(t), nil
}

func (r *Registry) List() []Turbine {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]Turbine, 0, len(r.order))
	for _, id := range r.order {
		out = append(out, cloneTurbine(r.turbines[id]))
	}
	return out
}

func (r *Registry) ListActive() []Turbine {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]Turbine, 0, len(r.order))
	for _, id := range r.order {
		t := r.turbines[id]
		if t.Active() {
			out = append(out, cloneTurbine(t))
		}
	}
	return out
}

func (r *Registry) UpdateState(id string, state TurbineState) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	t, ok := r.turbines[id]
	if !ok {
		return fmt.Errorf("turbine %s: %w", id, platform.ErrNotFound)
	}
	t.State = state
	r.turbines[id] = t
	return nil
}

func (r *Registry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.turbines)
}
""")

add("internal/turbine/config.go", """package turbine

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"windfarm-turbine-monitor-service/internal/platform"
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
""")

add("internal/turbine/fleet.go", """package turbine

import "sort"

// FleetStats is a small aggregate over the current fleet.
type FleetStats struct {
	Total       int                 `json:"total"`
	Active      int                 `json:"active"`
	Maintenance int                 `json:"maintenance"`
	BySite      map[string]int      `json:"by_site"`
	Models      map[string]int      `json:"models"`
}

func (r *Registry) Stats() FleetStats {
	list := r.List()
	s := FleetStats{Total: len(list), BySite: map[string]int{}, Models: map[string]int{}}
	for _, t := range list {
		s.BySite[t.Site]++
		s.Models[t.Model]++
		switch t.State {
		case StateRunning, StateIdle:
			s.Active++
		case StateMaintenance:
			s.Maintenance++
		}
	}
	return s
}

// Sites returns the distinct site codes in the fleet, sorted.
func (r *Registry) Sites() []string {
	list := r.List()
	seen := map[string]struct{}{}
	for _, t := range list {
		seen[t.Site] = struct{}{}
	}
	out := make([]string, 0, len(seen))
	for s := range seen {
		out = append(out, s)
	}
	sort.Strings(out)
	return out
}
""")

# ---------------------------------------------------------------------------
# internal/telemetry
# ---------------------------------------------------------------------------
add("internal/telemetry/model.go", """package telemetry

import "time"

// Sample is one SCADA reading for a turbine.
type Sample struct {
	TurbineID  string    `json:"turbine_id"`
	Timestamp  time.Time `json:"timestamp"`
	WindSpeed  float64   `json:"wind_speed_mps"`
	RotorRPM   float64   `json:"rotor_rpm"`
	GearboxTemp float64  `json:"gearbox_temp_c"`
	GenTemp    float64   `json:"gen_temp_c"`
	Vibration  float64   `json:"vibration_mm_s"`
	PowerOutput float64  `json:"power_output_kw"`
	YawAngle   float64   `json:"yaw_angle_deg"`
	PitchAngle float64   `json:"pitch_angle_deg"`
}

// SampleBatch is a group of samples ingested together.
type SampleBatch struct {
	Source  string   `json:"source"`
	Samples []Sample `json:"samples"`
}
""")

add("internal/telemetry/validator.go", """package telemetry

import (
	"fmt"

	"windfarm-turbine-monitor-service/internal/platform"
)

// Validator applies physical plausibility checks to incoming samples.
type Validator struct {
	MaxWindSpeed  float64
	MaxRotorRPM   float64
	MaxGearboxTemp float64
	MaxGenTemp    float64
	MaxVibration  float64
}

func DefaultValidator() Validator {
	return Validator{
		MaxWindSpeed:  40.0,
		MaxRotorRPM:   22.0,
		MaxGearboxTemp: 120.0,
		MaxGenTemp:    140.0,
		MaxVibration:  25.0,
	}
}

// Validate checks one sample and returns a descriptive error for the first
// violation encountered.
func (v Validator) Validate(s Sample) error {
	if s.TurbineID == "" {
		return fmt.Errorf("turbine id missing: %w", platform.ErrInvalid)
	}
	if s.Timestamp.IsZero() {
		return fmt.Errorf("sample timestamp zero: %w", platform.ErrInvalid)
	}
	if s.WindSpeed < 0 || s.WindSpeed > v.MaxWindSpeed {
		return fmt.Errorf("wind speed %.2f out of range: %w", s.WindSpeed, platform.ErrInvalid)
	}
	if s.RotorRPM < 0 || s.RotorRPM > v.MaxRotorRPM {
		return fmt.Errorf("rotor rpm %.2f out of range: %w", s.RotorRPM, platform.ErrInvalid)
	}
	if s.GearboxTemp < -40 || s.GearboxTemp > v.MaxGearboxTemp {
		return fmt.Errorf("gearbox temp %.2f out of range: %w", s.GearboxTemp, platform.ErrInvalid)
	}
	if s.GenTemp < -40 || s.GenTemp > v.MaxGenTemp {
		return fmt.Errorf("gen temp %.2f out of range: %w", s.GenTemp, platform.ErrInvalid)
	}
	if s.Vibration < 0 || s.Vibration > v.MaxVibration {
		return fmt.Errorf("vibration %.2f out of range: %w", s.Vibration, platform.ErrInvalid)
	}
	return nil
}

// ValidateBatch returns the subset of valid samples and a list of errors keyed
// by sample index.
func (v Validator) ValidateBatch(batch SampleBatch) ([]Sample, []error) {
	valid := make([]Sample, 0, len(batch.Samples))
	var errs []error
	for i, s := range batch.Samples {
		if err := v.Validate(s); err != nil {
			errs = append(errs, fmt.Errorf("sample %d: %w", i, err))
			continue
		}
		valid = append(valid, s)
	}
	return valid, errs
}
""")

add("internal/telemetry/normalizer.go", """package telemetry

// Normalizer converts raw SCADA readings into consistent engineering units and
// clamps obvious sensor glitches.
type Normalizer struct{}

// Normalize clamps a sample into physically sane bounds and rounds values to a
// stable precision.
func (Normalizer) Normalize(s Sample) Sample {
	if s.WindSpeed < 0 {
		s.WindSpeed = 0
	}
	if s.RotorRPM < 0 {
		s.RotorRPM = 0
	}
	if s.PowerOutput < 0 {
		s.PowerOutput = 0
	}
	return s
}

// NormalizeBatch normalizes every sample in place and returns the count.
func (n Normalizer) NormalizeBatch(samples []Sample) int {
	for i := range samples {
		samples[i] = n.Normalize(samples[i])
	}
	return len(samples)
}
""")

add("internal/telemetry/ingest.go", """package telemetry

import (
	"context"
	"fmt"
	"sync"
)

// Sink receives normalized samples and persists them.
type Sink interface {
	Append(s Sample) error
}

// Ingestor fans out batches to a bounded pool of workers.
type Ingestor struct {
	sink   Sink
	workers int
}

func NewIngestor(sink Sink, workers int) *Ingestor {
	if workers <= 0 {
		workers = 4
	}
	return &Ingestor{sink: sink, workers: workers}
}

// Ingest processes the given samples concurrently and waits for completion.
// It returns the number of samples successfully appended.
func (in *Ingestor) Ingest(ctx context.Context, samples []Sample) (int, error) {
	if len(samples) == 0 {
		return 0, nil
	}
	jobs := make(chan Sample, len(samples))
	var wg sync.WaitGroup
	wg.Add(in.workers)
	for w := 0; w < in.workers; w++ {
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case s, ok := <-jobs:
					if !ok {
						return
					}
					if err := in.sink.Append(s); err != nil {
						return
					}
				}
			}
		}()
	}
	for _, s := range samples {
		select {
		case <-ctx.Done():
			close(jobs)
			wg.Wait()
			return 0, fmt.Errorf("ingest cancelled: %w", ctx.Err())
		case jobs <- s:
		}
	}
	close(jobs)
	wg.Wait()
	return len(samples), nil
}
""")

add("internal/telemetry/registry.go", """package telemetry

import (
	"sync"
	"time"

	"windfarm-turbine-monitor-service/internal/platform"
)

// SampleRegistry keeps a bounded, per-turbine ring of recent samples.
type SampleRegistry struct {
	mu       sync.RWMutex
	capacity int
	bufs     map[string][]Sample
}

func NewSampleRegistry(capacity int) *SampleRegistry {
	if capacity <= 0 {
		capacity = 512
	}
	return &SampleRegistry{capacity: capacity, bufs: map[string][]Sample{}}
}

func (r *SampleRegistry) Append(s Sample) error {
	if s.TurbineID == "" {
		return platform.ErrInvalid
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	buf := r.bufs[s.TurbineID]
	buf = append(buf, s)
	if len(buf) > r.capacity {
		buf = buf[len(buf)-r.capacity:]
	}
	r.bufs[s.TurbineID] = buf
	return nil
}

// Recent returns a copy of the most recent n samples for a turbine.
func (r *SampleRegistry) Recent(turbineID string, n int) []Sample {
	r.mu.RLock()
	defer r.mu.RUnlock()
	buf := r.bufs[turbineID]
	if len(buf) < n {
		n = len(buf)
	}
	out := make([]Sample, n)
	copy(out, buf[len(buf)-n:])
	return out
}

// Latest returns the most recent sample for a turbine and whether one exists.
func (r *SampleRegistry) Latest(turbineID string) (Sample, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	buf := r.bufs[turbineID]
	if len(buf) == 0 {
		return Sample{}, false
	}
	return buf[len(buf)-1], true
}

// Since returns samples with a timestamp at or after cutoff, oldest first.
func (r *SampleRegistry) Since(turbineID string, cutoff time.Time) []Sample {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var out []Sample
	for _, s := range r.bufs[turbineID] {
		if !s.Timestamp.Before(cutoff) {
			out = append(out, s)
		}
	}
	return out
}
""")

# ---------------------------------------------------------------------------
# internal/ruleengine
# ---------------------------------------------------------------------------
add("internal/ruleengine/model.go", """package ruleengine

import "strings"

type Severity string

const (
	SeverityInfo     Severity = "info"
	SeverityWarning  Severity = "warning"
	SeverityCritical Severity = "critical"
)

// Metric is the name of a telemetry field a rule evaluates.
type Metric string

const (
	MetricWindSpeed   Metric = "wind_speed"
	MetricRotorRPM    Metric = "rotor_rpm"
	MetricGearboxTemp Metric = "gearbox_temp"
	MetricGenTemp     Metric = "gen_temp"
	MetricVibration   Metric = "vibration"
	MetricPowerOutput Metric = "power_output"
)

// Operator is a comparison operator.
type Operator string

const (
	OpGTE Operator = ">="
	OpLTE Operator = "<="
)

// Rule maps a telemetry metric to a severity when a threshold is crossed.
type Rule struct {
	ID        string    `json:"id"`
	TurbineID string    `json:"turbine_id"`
	Metric    Metric    `json:"metric"`
	Operator  Operator  `json:"operator"`
	Threshold float64   `json:"threshold"`
	Severity  Severity  `json:"severity"`
}

// AppliesTo reports whether the rule targets the given turbine (or all).
func (r Rule) AppliesTo(turbineID string) bool {
	return r.TurbineID == "" || r.TurbineID == turbineID
}

func ParseMetric(s string) (Metric, bool) {
	switch Metric(strings.ToLower(s)) {
	case MetricWindSpeed, MetricRotorRPM, MetricGearboxTemp, MetricGenTemp, MetricVibration, MetricPowerOutput:
		return Metric(strings.ToLower(s)), true
	}
	return "", false
}
""")

add("internal/ruleengine/registry.go", """package ruleengine

import (
	"fmt"
	"sync"

	"windfarm-turbine-monitor-service/internal/platform"
)

// Registry stores alarm rules in memory.
type Registry struct {
	mu    sync.RWMutex
	rules map[string]Rule
	order []string
}

func NewRegistry(seed []Rule) *Registry {
	r := &Registry{rules: map[string]Rule{}}
	for _, rule := range seed {
		r.rules[rule.ID] = rule
		r.order = append(r.order, rule.ID)
	}
	return r
}

func (r *Registry) Put(rule Rule) error {
	if rule.ID == "" {
		return fmt.Errorf("rule id required: %w", platform.ErrInvalid)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.rules[rule.ID]; !exists {
		r.order = append(r.order, rule.ID)
	}
	r.rules[rule.ID] = rule
	return nil
}

func (r *Registry) Get(id string) (Rule, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	rule, ok := r.rules[id]
	if !ok {
		return Rule{}, fmt.Errorf("rule %s: %w", id, platform.ErrNotFound)
	}
	return rule, nil
}

func (r *Registry) List() []Rule {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]Rule, 0, len(r.order))
	for _, id := range r.order {
		out = append(out, r.rules[id])
	}
	return out
}
""")

add("internal/ruleengine/evaluator.go", """package ruleengine

import (
	"windfarm-turbine-monitor-service/internal/telemetry"
)

// Evaluation is the outcome of applying one rule to one sample.
type Evaluation struct {
	Rule      Rule
	Sample    telemetry.Sample
	Triggered bool
	Value     float64
}

// Evaluator applies rules to samples and returns every triggered evaluation.
type Evaluator struct {
	rules *Registry
}

func NewEvaluator(rules *Registry) *Evaluator { return &Evaluator{rules: rules} }

// Evaluate applies all rules to a sample, returning triggered evaluations in
// rule registration order.
func (e *Evaluator) Evaluate(s telemetry.Sample) []Evaluation {
	rules := e.rules.List()
	var out []Evaluation
	for _, rule := range rules {
		if !rule.AppliesTo(s.TurbineID) {
			continue
		}
		value := metricValue(s, rule.Metric)
		triggered := compare(value, rule.Operator, rule.Threshold)
		if triggered {
			out = append(out, Evaluation{Rule: rule, Sample: s, Triggered: true, Value: value})
		}
	}
	return out
}

func metricValue(s telemetry.Sample, m Metric) float64 {
	switch m {
	case MetricWindSpeed:
		return s.WindSpeed
	case MetricRotorRPM:
		return s.RotorRPM
	case MetricGearboxTemp:
		return s.GearboxTemp
	case MetricGenTemp:
		return s.GenTemp
	case MetricVibration:
		return s.Vibration
	case MetricPowerOutput:
		return s.PowerOutput
	}
	return 0
}

func compare(v float64, op Operator, threshold float64) bool {
	switch op {
	case OpGTE:
		return v >= threshold
	case OpLTE:
		return v <= threshold
	}
	return false
}
""")

add("internal/ruleengine/threshold.go", """package ruleengine

import "windfarm-turbine-monitor-service/internal/turbine"

// ThresholdSet builds default rules for a turbine using fleet config defaults.
func ThresholdSet(t turbine.Turbine, cfg turbine.Config) []Rule {
	return []Rule{
		{ID: "gear-" + t.ID, TurbineID: t.ID, Metric: MetricGearboxTemp, Operator: OpGTE, Threshold: cfg.ThresholdFor("gearbox_temp", 85), Severity: SeverityWarning},
		{ID: "gen-" + t.ID, TurbineID: t.ID, Metric: MetricGenTemp, Operator: OpGTE, Threshold: cfg.ThresholdFor("gen_temp", 95), Severity: SeverityCritical},
		{ID: "vib-" + t.ID, TurbineID: t.ID, Metric: MetricVibration, Operator: OpGTE, Threshold: 12.0, Severity: SeverityWarning},
	}
}
""")

# ---------------------------------------------------------------------------
# internal/fault
# ---------------------------------------------------------------------------
add("internal/fault/model.go", """package fault

import "time"

type State string

const (
	StateNormal      State = "normal"
	StateWarning     State = "warning"
	StateFault       State = "fault"
	StateMaintenance State = "maintenance"
	StateRetrying    State = "retrying"
)

// Fault is one diagnosed issue attached to a turbine.
type Fault struct {
	ID         string    `json:"id"`
	TurbineID  string    `json:"turbine_id"`
	Code       string    `json:"code"`
	State      State     `json:"state"`
	Severity   string    `json:"severity"`
	DetectedAt time.Time `json:"detected_at"`
	UpdatedAt  time.Time `json:"updated_at"`
	Notes      string    `json:"notes"`
}
""")

add("internal/fault/transitions.go", """package fault

// allowed maps each state to the states reachable from it.
var allowed = map[State][]State{
	StateNormal:      {StateWarning, StateFault, StateMaintenance},
	StateWarning:     {StateNormal, StateFault, StateMaintenance},
	StateFault:       {StateMaintenance, StateRetrying},
	StateMaintenance: {StateNormal},
	StateRetrying:    {StateNormal, StateMaintenance},
}

// CanTransition reports whether a move from -> to is legal.
func CanTransition(from, to State) bool {
	if from == to {
		return true
	}
	for _, s := range allowed[from] {
		if s == to {
			return true
		}
	}
	return false
}

// TerminalState reports whether the state is a stable end state for reporting.
func TerminalState(s State) bool { return s == StateNormal || s == StateMaintenance }
""")

add("internal/fault/store.go", """package fault

import (
	"fmt"
	"sync"
	"time"

	"windfarm-turbine-monitor-service/internal/platform"
)

// Store persists faults in memory with a monotonic revision.
type Store struct {
	mu       sync.RWMutex
	faults   map[string]Fault
	order    []string
	revision int
}

func NewStore() *Store { return &Store{faults: map[string]Fault{}, revision: 1} }

func (s *Store) Create(f Fault) (Fault, error) {
	if f.TurbineID == "" {
		return Fault{}, fmt.Errorf("turbine id required: %w", platform.ErrInvalid)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if f.ID == "" {
		f.ID = platform.NewID("fault")
	}
	if f.State == "" {
		f.State = StateWarning
	}
	now := time.Now()
	if f.DetectedAt.IsZero() {
		f.DetectedAt = now
	}
	f.UpdatedAt = now
	s.revision++
	s.faults[f.ID] = f
	s.order = append(s.order, f.ID)
	return f, nil
}

func (s *Store) Get(id string) (Fault, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	f, ok := s.faults[id]
	if !ok {
		return Fault{}, fmt.Errorf("fault %s: %w", id, platform.ErrNotFound)
	}
	return f, nil
}

// Transition moves a fault to the target state if the transition is legal.
func (s *Store) Transition(id string, to State) (Fault, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	f, ok := s.faults[id]
	if !ok {
		return Fault{}, fmt.Errorf("fault %s: %w", id, platform.ErrNotFound)
	}
	if !CanTransition(f.State, to) {
		return Fault{}, fmt.Errorf("illegal transition %s -> %s: %w", f.State, to, platform.ErrConflict)
	}
	f.State = to
	f.UpdatedAt = time.Now()
	s.revision++
	s.faults[id] = f
	return f, nil
}

func (s *Store) ListByTurbine(turbineID string) []Fault {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []Fault
	for _, id := range s.order {
		if s.faults[id].TurbineID == turbineID {
			out = append(out, s.faults[id])
		}
	}
	return out
}

// ListOpen returns faults that are not in a terminal state.
func (s *Store) ListOpen() []Fault {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []Fault
	for _, id := range s.order {
		f := s.faults[id]
		if !TerminalState(f.State) {
			out = append(out, f)
		}
	}
	return out
}

func (s *Store) Revision() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.revision
}
""")

add("internal/fault/service.go", """package fault

import (
	"fmt"

	"windfarm-turbine-monitor-service/internal/platform"
)

// Service coordinates fault lifecycle around a Store.
type Service struct {
	store *Store
}

func NewService(store *Store) *Service { return &Service{store: store} }

// Raise creates a fault if an equivalent open fault does not already exist for
// the same turbine and code.
func (svc *Service) Raise(turbineID, code, severity string) (Fault, error) {
	for _, f := range svc.store.ListOpen() {
		if f.TurbineID == turbineID && f.Code == code {
			return f, nil
		}
	}
	return svc.store.Create(Fault{TurbineID: turbineID, Code: code, Severity: severity, State: StateWarning})
}

// Acknowledge moves a fault toward maintenance.
func (svc *Service) Acknowledge(id string) (Fault, error) {
	f, err := svc.store.Get(id)
	if err != nil {
		return Fault{}, fmt.Errorf("lookup fault: %w", err)
	}
	if f.State == StateNormal {
		return f, nil
	}
	return svc.store.Transition(id, StateMaintenance)
}

// Resolve marks a fault as back to normal after maintenance.
func (svc *Service) Resolve(id string) (Fault, error) {
	f, err := svc.store.Get(id)
	if err != nil {
		return Fault{}, fmt.Errorf("lookup fault: %w", err)
	}
	if !CanTransition(f.State, StateNormal) {
		return Fault{}, fmt.Errorf("cannot resolve fault in state %s: %w", f.State, platform.ErrConflict)
	}
	return svc.store.Transition(id, StateNormal)
}
""")

# ---------------------------------------------------------------------------
# internal/alarm
# ---------------------------------------------------------------------------
add("internal/alarm/model.go", """package alarm

import "time"

type Status string

const (
	StatusOpen       Status = "open"
	StatusAck        Status = "acknowledged"
	StatusResolved   Status = "resolved"
	StatusSuppressed Status = "suppressed"
)

// Alarm is a notification derived from a rule evaluation.
type Alarm struct {
	ID         string    `json:"id"`
	TurbineID  string    `json:"turbine_id"`
	RuleID     string    `json:"rule_id"`
	Metric     string    `json:"metric"`
	Severity   string    `json:"severity"`
	Value      float64   `json:"value"`
	Threshold  float64   `json:"threshold"`
	Status     Status    `json:"status"`
	OccurredAt time.Time `json:"occurred_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// Key is the dedup identity for an alarm.
func (a Alarm) Key() string { return a.TurbineID + ":" + a.RuleID }
""")

add("internal/alarm/dedup.go", """package alarm

import "time"

// Deduplicator decides whether a fresh evaluation should be suppressed because
// an identical alarm is still open.
type Deduplicator struct {
	window time.Duration
}

func NewDeduplicator(window time.Duration) *Deduplicator {
	if window <= 0 {
		window = 5 * time.Minute
	}
	return &Deduplicator{window: window}
}

// Suppress reports whether the candidate duplicates an existing open alarm.
func (d *Deduplicator) Suppress(candidate Alarm, existing []Alarm) bool {
	for _, a := range existing {
		if a.Key() == candidate.Key() && a.Status != StatusResolved {
			return true
		}
	}
	return false
}
""")

add("internal/alarm/dispatcher.go", """package alarm

import (
	"sync"
	"time"

	"windfarm-turbine-monitor-service/internal/platform"
)

// Dispatcher stores alarms and supports ack/resolve transitions.
type Dispatcher struct {
	mu      sync.RWMutex
	alarms  map[string]Alarm
	order   []string
	dedup   *Deduplicator
}

func NewDispatcher(dedup *Deduplicator) *Dispatcher {
	return &Dispatcher{alarms: map[string]Alarm{}, dedup: dedup}
}

func (d *Dispatcher) Dispatch(a Alarm) (Alarm, bool, error) {
	if a.TurbineID == "" || a.RuleID == "" {
		return Alarm{}, false, platform.ErrInvalid
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	open := d.openLocked()
	if d.dedup.Suppress(a, open) {
		return Alarm{}, false, nil
	}
	if a.ID == "" {
		a.ID = platform.NewID("alarm")
	}
	if a.OccurredAt.IsZero() {
		a.OccurredAt = time.Now()
	}
	a.UpdatedAt = time.Now()
	a.Status = StatusOpen
	d.alarms[a.ID] = a
	d.order = append(d.order, a.ID)
	return a, true, nil
}

func (d *Dispatcher) Ack(id string) (Alarm, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	a, ok := d.alarms[id]
	if !ok {
		return Alarm{}, platform.ErrNotFound
	}
	a.Status = StatusAck
	a.UpdatedAt = time.Now()
	d.alarms[id] = a
	return a, nil
}

func (d *Dispatcher) Resolve(id string) (Alarm, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	a, ok := d.alarms[id]
	if !ok {
		return Alarm{}, platform.ErrNotFound
	}
	a.Status = StatusResolved
	a.UpdatedAt = time.Now()
	d.alarms[id] = a
	return a, nil
}

func (d *Dispatcher) ListOpen() []Alarm {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.openLocked()
}

func (d *Dispatcher) openLocked() []Alarm {
	var out []Alarm
	for _, id := range d.order {
		a := d.alarms[id]
		if a.Status == StatusOpen {
			out = append(out, a)
		}
	}
	return out
}
""")

add("internal/alarm/escalation.go", """package alarm

import "time"

// EscalationPolicy defines how an unacknowledged alarm escalates.
type EscalationPolicy struct {
	FirstReminder  time.Duration
	EscalateAfter  time.Duration
	EscalateLevels int
}

func DefaultEscalationPolicy() EscalationPolicy {
	return EscalationPolicy{FirstReminder: 10 * time.Minute, EscalateAfter: 30 * time.Minute, EscalateLevels: 3}
}

// LevelFor computes the current escalation level for an open alarm's age.
func (p EscalationPolicy) LevelFor(age time.Duration) int {
	if age < p.FirstReminder {
		return 0
	}
	if age < p.EscalateAfter {
		return 1
	}
	extra := int((age - p.EscalateAfter) / p.EscalateAfter)
	if extra >= p.EscalateLevels {
		return p.EscalateLevels
	}
	return 1 + extra
}
""")

add("internal/alarm/notifier.go", """package alarm

import "sync"

// Notifier fans alarm notifications out to registered channels.
type Notifier struct {
	mu    sync.RWMutex
	sinks []chan Alarm
}

func NewNotifier() *Notifier { return &Notifier{} }

func (n *Notifier) Register(ch chan Alarm) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.sinks = append(n.sinks, ch)
}

func (n *Notifier) Notify(a Alarm) int {
	n.mu.RLock()
	defer n.mu.RUnlock()
	sent := 0
	for _, ch := range n.sinks {
		select {
		case ch <- a:
			sent++
		default:
		}
	}
	return sent
}

func (n *Notifier) SinkCount() int {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return len(n.sinks)
}
""")

# ---------------------------------------------------------------------------
# internal/workorder
# ---------------------------------------------------------------------------
add("internal/workorder/model.go", """package workorder

import "time"

type Status string

const (
	StatusPending    Status = "pending"
	StatusScheduled  Status = "scheduled"
	StatusInProgress Status = "in_progress"
	StatusDone       Status = "done"
	StatusCancelled  Status = "cancelled"
)

// WorkOrder schedules maintenance for a turbine.
type WorkOrder struct {
	ID          string    `json:"id"`
	TurbineID   string    `json:"turbine_id"`
	Title       string    `json:"title"`
	Priority    int       `json:"priority"`
	CrewID      string    `json:"crew_id"`
	Status      Status    `json:"status"`
	PlannedAt   time.Time `json:"planned_at"`
	CreatedAt   time.Time `json:"created_at"`
	CompletedAt time.Time `json:"completed_at"`
}
""")

add("internal/workorder/scheduler.go", """package workorder

import (
	"fmt"
	"time"

	"windfarm-turbine-monitor-service/internal/platform"
)

// Scheduler creates work orders and assigns crews by availability.
type Scheduler struct {
	orders map[string]WorkOrder
	crews  *CrewPool
	now    func() time.Time
}

func NewScheduler(crews *CrewPool, now func() time.Time) *Scheduler {
	if now == nil {
		now = time.Now
	}
	return &Scheduler{orders: map[string]WorkOrder{}, crews: crews, now: now}
}

// Schedule creates a pending work order for a turbine.
func (s *Scheduler) Schedule(turbineID, title string, priority int) WorkOrder {
	wo := WorkOrder{
		ID:        platform.NewID("wo"),
		TurbineID: turbineID,
		Title:     title,
		Priority:  priority,
		Status:    StatusPending,
		CreatedAt: s.now(),
	}
	s.orders[wo.ID] = wo
	return wo
}

// Assign tries to attach a free crew to a pending order.
func (s *Scheduler) Assign(orderID string, preferred string) (WorkOrder, error) {
	wo, ok := s.orders[orderID]
	if !ok {
		return WorkOrder{}, fmt.Errorf("order %s: %w", orderID, platform.ErrNotFound)
	}
	if wo.Status != StatusPending {
		return WorkOrder{}, fmt.Errorf("order not pending: %w", platform.ErrConflict)
	}
	crew, ok := s.crews.Assign(preferred)
	if !ok {
		return WorkOrder{}, fmt.Errorf("no crew available: %w", platform.ErrUnavailable)
	}
	wo.CrewID = crew
	wo.Status = StatusScheduled
	wo.PlannedAt = s.now().Add(24 * time.Hour)
	s.orders[orderID] = wo
	return wo, nil
}

func (s *Scheduler) List() []WorkOrder {
	out := make([]WorkOrder, 0, len(s.orders))
	for _, wo := range s.orders {
		out = append(out, wo)
	}
	return out
}
""")

add("internal/workorder/crew.go", """package workorder

import "sync"

// CrewPool tracks which maintenance crews are free.
type CrewPool struct {
	mu    sync.RWMutex
	free  map[string]bool
	order []string
}

func NewCrewPool(crewIDs []string) *CrewPool {
	p := &CrewPool{free: map[string]bool{}}
	for _, id := range crewIDs {
		p.free[id] = true
		p.order = append(p.order, id)
	}
	return p
}

// Assign marks the first available crew as busy and returns its ID.
func (p *CrewPool) Assign(preferred string) (string, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if preferred != "" && p.free[preferred] {
		p.free[preferred] = false
		return preferred, true
	}
	for _, id := range p.order {
		if p.free[id] {
			p.free[id] = false
			return id, true
		}
	}
	return "", false
}

// Release returns a crew to the pool.
func (p *CrewPool) Release(id string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.free[id] = true
}

func (p *CrewPool) Available() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	n := 0
	for _, free := range p.free {
		if free {
			n++
		}
	}
	return n
}
""")

add("internal/workorder/lifecycle.go", """package workorder

import (
	"time"

	"windfarm-turbine-monitor-service/internal/platform"
)

// Lifecycle tracks work order state transitions.
type Lifecycle struct {
	orders map[string]WorkOrder
	now    func() time.Time
}

func NewLifecycle(orders []WorkOrder, now func() time.Time) *Lifecycle {
	if now == nil {
		now = time.Now
	}
	l := &Lifecycle{orders: map[string]WorkOrder{}, now: now}
	for _, wo := range orders {
		l.orders[wo.ID] = wo
	}
	return l
}

// Start moves a scheduled order into progress and releases its crew on finish.
func (l *Lifecycle) Start(id string) (WorkOrder, error) {
	wo, ok := l.orders[id]
	if !ok {
		return WorkOrder{}, platform.ErrNotFound
	}
	if wo.Status != StatusScheduled {
		return WorkOrder{}, platform.ErrConflict
	}
	wo.Status = StatusInProgress
	l.orders[id] = wo
	return wo, nil
}

func (l *Lifecycle) Complete(id string) (WorkOrder, error) {
	wo, ok := l.orders[id]
	if !ok {
		return WorkOrder{}, platform.ErrNotFound
	}
	if wo.Status != StatusInProgress {
		return WorkOrder{}, platform.ErrConflict
	}
	wo.Status = StatusDone
	wo.CompletedAt = l.now()
	l.orders[id] = wo
	return wo, nil
}
""")

# ---------------------------------------------------------------------------
# internal/analytics
# ---------------------------------------------------------------------------
add("internal/analytics/window.go", """package analytics

import "windfarm-turbine-monitor-service/internal/telemetry"

// SlidingWindow computes aggregate statistics over a bounded series of samples.
type SlidingWindow struct {
	capacity int
	values   []float64
}

func NewSlidingWindow(capacity int) *SlidingWindow {
	if capacity <= 0 {
		capacity = 64
	}
	return &SlidingWindow{capacity: capacity}
}

// Add pushes a value and returns the window mean.
func (w *SlidingWindow) Add(v float64) float64 {
	w.values = append(w.values, v)
	if len(w.values) > w.capacity {
		w.values = w.values[len(w.values)-w.capacity:]
	}
	return w.Mean()
}

func (w *SlidingWindow) Mean() float64 {
	if len(w.values) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range w.values {
		sum += v
	}
	return sum / float64(len(w.values))
}

func (w *SlidingWindow) Count() int { return len(w.values) }
""")

add("internal/analytics/powercurve.go", """package analytics

import (
	"sort"

	"windfarm-turbine-monitor-service/internal/telemetry"
)

// PowerCurvePoint is one binned observation of power vs wind speed.
type PowerCurvePoint struct {
	WindBin     float64 `json:"wind_bin"`
	MeanPower   float64 `json:"mean_power_kw"`
	SampleCount int     `json:"sample_count"`
}

// PowerCurve bins samples by wind speed and computes mean power per bin.
func PowerCurve(samples []telemetry.Sample, binWidth float64) []PowerCurvePoint {
	if binWidth <= 0 {
		binWidth = 1.0
	}
	type acc struct{ sum float64; n int }
	bins := map[int]*acc{}
	for _, s := range samples {
		idx := int(s.WindSpeed / binWidth)
		a := bins[idx]
		if a == nil {
			a = &acc{}
			bins[idx] = a
		}
		a.sum += s.PowerOutput
		a.n++
	}
	keys := make([]int, 0, len(bins))
	for k := range bins {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	out := make([]PowerCurvePoint, 0, len(keys))
	for _, k := range keys {
		a := bins[k]
		out = append(out, PowerCurvePoint{
			WindBin:     float64(k) * binWidth,
			MeanPower:   a.sum / float64(a.n),
			SampleCount: a.n,
		})
	}
	return out
}
""")

add("internal/analytics/percentile.go", """package analytics

import "sort"

// Percentile computes the p-th percentile (0..100) of a non-empty value set.
func Percentile(values []float64, p float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sorted := make([]float64, len(values))
	copy(sorted, values)
	sort.Float64s(sorted)
	if p <= 0 {
		return sorted[0]
	}
	if p >= 100 {
		return sorted[len(sorted)-1]
	}
	idx := (p / 100) * float64(len(sorted)-1)
	lo := int(idx)
	hi := lo + 1
	if hi >= len(sorted) {
		return sorted[len(sorted)-1]
	}
	frac := idx - float64(lo)
	return sorted[lo]*(1-frac) + sorted[hi]*frac
}
""")

add("internal/analytics/aggregate.go", """package analytics

import "windfarm-turbine-monitor-service/internal/telemetry"

// Aggregate summarizes a series of samples for reporting.
type Aggregate struct {
	Count        int     `json:"count"`
	MeanWind     float64 `json:"mean_wind_mps"`
	MeanPower    float64 `json:"mean_power_kw"`
	MaxGearboxTemp float64 `json:"max_gearbox_temp_c"`
	MaxGenTemp   float64 `json:"max_gen_temp_c"`
	EnergyKWh    float64 `json:"energy_kwh"`
}

func AggregateSamples(samples []telemetry.Sample, intervalSeconds float64) Aggregate {
	a := Aggregate{Count: len(samples)}
	if len(samples) == 0 {
		return a
	}
	var wind, power float64
	for _, s := range samples {
		wind += s.WindSpeed
		power += s.PowerOutput
		if s.GearboxTemp > a.MaxGearboxTemp {
			a.MaxGearboxTemp = s.GearboxTemp
		}
		if s.GenTemp > a.MaxGenTemp {
			a.MaxGenTemp = s.GenTemp
		}
	}
	a.MeanWind = wind / float64(len(samples))
	a.MeanPower = power / float64(len(samples))
	// Average power (kW) over the interval yields kWh.
	a.EnergyKWh = a.MeanPower * (intervalSeconds / 3600.0)
	return a
}
""")

# ---------------------------------------------------------------------------
# internal/report
# ---------------------------------------------------------------------------
add("internal/report/model.go", """package report

import "time"

// Report is a rolled-up operational summary for a site.
type Report struct {
	ID          string    `json:"id"`
	Site        string    `json:"site"`
	PeriodStart time.Time `json:"period_start"`
	PeriodEnd   time.Time `json:"period_end"`
	Sections    []Section `json:"sections"`
}

type Section struct {
	Title  string        `json:"title"`
	Rows   []SectionRow  `json:"rows"`
}

type SectionRow struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}
""")

add("internal/report/builder.go", """package report

import (
	"fmt"
	"time"

	"windfarm-turbine-monitor-service/internal/platform"
	"windfarm-turbine-monitor-service/internal/turbine"
	"windfarm-turbine-monitor-service/internal/analytics"
	"windfarm-turbine-monitor-service/internal/telemetry"
)

// Builder assembles site reports from fleet and telemetry data.
type Builder struct {
	registry  *turbine.Registry
	samples   *telemetry.SampleRegistry
	now       func() time.Time
}

func NewBuilder(registry *turbine.Registry, samples *telemetry.SampleRegistry, now func() time.Time) *Builder {
	if now == nil {
		now = time.Now
	}
	return &Builder{registry: registry, samples: samples, now: now}
}

// Build produces a report for a site over the given period.
func (b *Builder) Build(site string, start, end time.Time) Report {
	rep := Report{
		ID:          platform.NewID("rep"),
		Site:        site,
		PeriodStart: start,
		PeriodEnd:   end,
	}
	fleet := b.registry.Stats()
	rep.Sections = append(rep.Sections, Section{
		Title: "Fleet",
		Rows: []SectionRow{
			{Key: "total", Value: fmt.Sprintf("%d", fleet.Total)},
			{Key: "active", Value: fmt.Sprintf("%d", fleet.Active)},
			{Key: "maintenance", Value: fmt.Sprintf("%d", fleet.Maintenance)},
		},
	})
	rep.Sections = append(rep.Sections, b.energySection(site, start, end))
	return rep
}

func (b *Builder) energySection(site string, start, end time.Time) Section {
	section := Section{Title: "Energy"}
	for _, t := range b.registry.List() {
		if t.Site != site {
			continue
		}
		samples := b.samples.Since(t.ID, start)
		var within []telemetry.Sample
		for _, s := range samples {
			if !s.Timestamp.After(end) {
				within = append(within, s)
			}
		}
		agg := analytics.AggregateSamples(within, end.Sub(start).Seconds())
		section.Rows = append(section.Rows, SectionRow{
			Key:   t.ID,
			Value: fmt.Sprintf("%.2f kWh", agg.EnergyKWh),
		})
	}
	return section
}
""")

add("internal/report/aggregate.go", """package report

// Totals rolls up section rows into a flat string map.
func (r Report) Totals() map[string]string {
	out := map[string]string{}
	for _, sec := range r.Sections {
		for _, row := range sec.Rows {
			out[sec.Title+"."+row.Key] = row.Value
		}
	}
	return out
}
""")

# ---------------------------------------------------------------------------
# internal/audit
# ---------------------------------------------------------------------------
add("internal/audit/model.go", """package audit

import "time"

type EventType string

const (
	EventIngest    EventType = "ingest"
	EventAlarm     EventType = "alarm"
	EventFault     EventType = "fault"
	EventWorkOrder EventType = "work_order"
)

// Event is an immutable audit record.
type Event struct {
	ID        string    `json:"id"`
	Type      EventType `json:"type"`
	TurbineID string    `json:"turbine_id"`
	Payload   string    `json:"payload"`
	At        time.Time `json:"at"`
}
""")

add("internal/audit/stream.go", """package audit

import (
	"sync"

	"windfarm-turbine-monitor-service/internal/platform"
)

// Stream is an in-memory fan-out event stream.
type Stream struct {
	mu      sync.RWMutex
	subs    map[string]chan Event
	history []Event
}

func NewStream() *Stream { return &Stream{subs: map[string]chan Event{}} }

func (s *Stream) Subscribe(id string) chan Event {
	s.mu.Lock()
	defer s.mu.Unlock()
	ch := make(chan Event, 256)
	s.subs[id] = ch
	return ch
}

func (s *Stream) Unsubscribe(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if ch, ok := s.subs[id]; ok {
		delete(s.subs, id)
		close(ch)
	}
}

// Publish appends an event and fans it out to every subscriber without blocking.
func (s *Stream) Publish(typ EventType, turbineID, payload string) Event {
	ev := Event{ID: platform.NewID("evt"), Type: typ, TurbineID: turbineID, Payload: payload}
	s.mu.Lock()
	s.history = append(s.history, ev)
	chans := make([]chan Event, 0, len(s.subs))
	for _, ch := range s.subs {
		chans = append(chans, ch)
	}
	s.mu.Unlock()
	for _, ch := range chans {
		select {
		case ch <- ev:
		default:
		}
	}
	return ev
}

func (s *Stream) History() []Event {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Event, len(s.history))
	copy(out, s.history)
	return out
}
""")

add("internal/audit/checkpoint.go", """package audit

import (
	"fmt"
	"sync"
)

// Checkpoint tracks how far each subscriber has read the stream.
type Checkpoint struct {
	mu      sync.RWMutex
	offsets map[string]int
}

func NewCheckpoint() *Checkpoint { return &Checkpoint{offsets: map[string]int{}} }

func (c *Checkpoint) Get(subID string) int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.offsets[subID]
}

func (c *Checkpoint) Set(subID string, offset int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.offsets[subID] = offset
}

// Advance moves the checkpoint forward only when the new offset is greater.
func (c *Checkpoint) Advance(subID string, offset int) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if offset < c.offsets[subID] {
		return fmt.Errorf("offset %d regressed below %d", offset, c.offsets[subID])
	}
	c.offsets[subID] = offset
	return nil
}
""")

add("internal/audit/delivery.go", """package audit

import "sync"

// Delivery replays unread events to a subscriber and advances its checkpoint.
type Delivery struct {
	stream     *Stream
	checkpoint *Checkpoint
	mu         sync.Mutex
}

func NewDelivery(stream *Stream, checkpoint *Checkpoint) *Delivery {
	return &Delivery{stream: stream, checkpoint: checkpoint}
}

// Deliver sends every event after the subscriber's checkpoint and returns how
// many were delivered.
func (d *Delivery) Deliver(subID string, ch chan Event) int {
	d.mu.Lock()
	defer d.mu.Unlock()
	offset := d.checkpoint.Get(subID)
	history := d.stream.History()
	delivered := 0
	for i := offset; i < len(history); i++ {
		select {
		case ch <- history[i]:
			delivered++
			_ = d.checkpoint.Advance(subID, i+1)
		default:
			return delivered
		}
	}
	return delivered
}
""")

# ---------------------------------------------------------------------------
# internal/gateway
# ---------------------------------------------------------------------------
add("internal/gateway/app.go", """package gateway

import (
	"windfarm-turbine-monitor-service/internal/alarm"
	"windfarm-turbine-monitor-service/internal/audit"
	"windfarm-turbine-monitor-service/internal/fault"
	"windfarm-turbine-monitor-service/internal/report"
	"windfarm-turbine-monitor-service/internal/ruleengine"
	"windfarm-turbine-monitor-service/internal/telemetry"
	"windfarm-turbine-monitor-service/internal/turbine"
	"windfarm-turbine-monitor-service/internal/workorder"
)

// App bundles all domain services behind the HTTP layer.
type App struct {
	Turbines  *turbine.Registry
	Samples   *telemetry.SampleRegistry
	Ingestor  *telemetry.Ingestor
	Rules     *ruleengine.Registry
	Evaluator *ruleengine.Evaluator
	Faults    *fault.Service
	Alarms    *alarm.Dispatcher
	Notifier  *alarm.Notifier
	Orders    *workorder.Scheduler
	Audit     *audit.Stream
	Reports   *report.Builder
}
""")

add("internal/gateway/dto.go", """package gateway

import (
	"time"

	"windfarm-turbine-monitor-service/internal/telemetry"
	"windfarm-turbine-monitor-service/internal/turbine"
)

type addTurbineRequest struct {
	ID           string  `json:"id"`
	Name         string  `json:"name"`
	Model        string  `json:"model"`
	Site         string  `json:"site"`
	RatedPowerKW float64 `json:"rated_power_kw"`
}

func (r addTurbineRequest) toTurbine() turbine.Turbine {
	return turbine.Turbine{
		ID:           r.ID,
		Name:         r.Name,
		Model:        r.Model,
		Site:         r.Site,
		RatedPowerKW: r.RatedPowerKW,
		State:        turbine.StateRunning,
		Commissioned: time.Now(),
	}
}

type ingestRequest struct {
	Source  string             `json:"source"`
	Samples []telemetry.Sample `json:"samples"`
}
""")

add("internal/gateway/handlers.go", """package gateway

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"windfarm-turbine-monitor-service/internal/alarm"
	"windfarm-turbine-monitor-service/internal/fault"
	"windfarm-turbine-monitor-service/internal/telemetry"
)

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("content-type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func (a *App) handleListTurbines(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, a.Turbines.List())
}

func (a *App) handleAddTurbine(w http.ResponseWriter, r *http.Request) {
	var req addTurbineRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	t := req.toTurbine()
	if t.ID == "" || t.Name == "" {
		writeError(w, http.StatusBadRequest, "id and name required")
		return
	}
	if err := a.Turbines.Put(t); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	a.Audit.Publish("turbine", t.ID, "registered")
	writeJSON(w, http.StatusCreated, t)
}

func (a *App) handleIngest(w http.ResponseWriter, r *http.Request) {
	var req ingestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	validator := telemetry.DefaultValidator()
	valid, errs := validator.ValidateBatch(telemetry.SampleBatch{Source: req.Source, Samples: req.Samples})
	normalizer := telemetry.Normalizer{}
	normalizer.NormalizeBatch(valid)
	ingested, err := a.Ingestor.Ingest(r.Context(), valid)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	for _, s := range valid {
		for _, ev := range a.Evaluator.Evaluate(s) {
			candidate := alarm.Alarm{
				TurbineID: s.TurbineID,
				RuleID:    ev.Rule.ID,
				Metric:    string(ev.Rule.Metric),
				Severity:  string(ev.Rule.Severity),
				Value:     ev.Value,
				Threshold: ev.Rule.Threshold,
			}
			if created, ok, _ := a.Alarms.Dispatch(candidate); ok {
				a.Notifier.Notify(created)
				a.Audit.Publish("alarm", s.TurbineID, created.RuleID)
			}
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"ingested": ingested, "rejected": len(errs)})
}

func (a *App) handleRecentSamples(w http.ResponseWriter, r *http.Request) {
	turbineID := r.PathValue("turbineID")
	n, _ := strconv.Atoi(r.URL.Query().Get("n"))
	if n <= 0 {
		n = 20
	}
	writeJSON(w, http.StatusOK, a.Samples.Recent(turbineID, n))
}

func (a *App) handleListAlarms(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, a.Alarms.ListOpen())
}

func (a *App) handleAckAlarm(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	updated, err := a.Alarms.Ack(id)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func (a *App) handleResolveAlarm(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	updated, err := a.Alarms.Resolve(id)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func (a *App) handleListFaults(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, a.Faults.Open())
}

func (a *App) handleRaiseFault(w http.ResponseWriter, r *http.Request) {
	var req struct {
		TurbineID string `json:"turbine_id"`
		Code      string `json:"code"`
		Severity  string `json:"severity"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	f, err := a.Faults.Raise(req.TurbineID, req.Code, req.Severity)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, f)
}

func (a *App) handleResolveFault(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	f, err := a.Faults.Resolve(id)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, f)
}

func (a *App) handleScheduleOrder(w http.ResponseWriter, r *http.Request) {
	var req struct {
		TurbineID string `json:"turbine_id"`
		Title     string `json:"title"`
		Priority  int    `json:"priority"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	wo := a.Orders.Schedule(req.TurbineID, req.Title, req.Priority)
	a.Audit.Publish("work_order", req.TurbineID, wo.ID)
	writeJSON(w, http.StatusCreated, wo)
}

func (a *App) handleReport(w http.ResponseWriter, r *http.Request) {
	site := r.PathValue("site")
	now := time.Now()
	rep := a.Reports.Build(site, now.Add(-24*time.Hour), now)
	writeJSON(w, http.StatusOK, rep)
}

func (a *App) handleSummary(w http.ResponseWriter, r *http.Request) {
	stats := a.Turbines.Stats()
	writeJSON(w, http.StatusOK, map[string]any{
		"site":       "",
		"turbines":   stats.Total,
		"active":     stats.Active,
		"open_alarms": len(a.Alarms.ListOpen()),
		"open_faults": len(a.Faults.Open()),
	})
}

// idFromPath extracts the trailing path segment for older routers.
func idFromPath(path, prefix string) string {
	return strings.TrimPrefix(path, prefix)
}
""")

add("internal/gateway/router.go", """package gateway

import "net/http"

// NewRouter wires all routes to the app handlers using Go 1.22 path values.
func NewRouter(a *App) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("content-type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})
	mux.HandleFunc("GET /api/summary", a.handleSummary)
	mux.HandleFunc("GET /api/turbines", a.handleListTurbines)
	mux.HandleFunc("POST /api/turbines", a.handleAddTurbine)
	mux.HandleFunc("POST /api/telemetry", a.handleIngest)
	mux.HandleFunc("GET /api/telemetry/{turbineID}/recent", a.handleRecentSamples)
	mux.HandleFunc("GET /api/alarms", a.handleListAlarms)
	mux.HandleFunc("POST /api/alarms/{id}/ack", a.handleAckAlarm)
	mux.HandleFunc("POST /api/alarms/{id}/resolve", a.handleResolveAlarm)
	mux.HandleFunc("GET /api/faults", a.handleListFaults)
	mux.HandleFunc("POST /api/faults", a.handleRaiseFault)
	mux.HandleFunc("POST /api/faults/{id}/resolve", a.handleResolveFault)
	mux.HandleFunc("POST /api/workorders", a.handleScheduleOrder)
	mux.HandleFunc("GET /api/reports/{site}", a.handleReport)
	mux.Handle("/", http.FileServer(http.Dir("web")))
	return mux
}
""")

# ---------------------------------------------------------------------------
# cmd/server
# ---------------------------------------------------------------------------
add("cmd/server/main.go", """package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"windfarm-turbine-monitor-service/internal/alarm"
	"windfarm-turbine-monitor-service/internal/audit"
	"windfarm-turbine-monitor-service/internal/fault"
	"windfarm-turbine-monitor-service/internal/gateway"
	"windfarm-turbine-monitor-service/internal/report"
	"windfarm-turbine-monitor-service/internal/ruleengine"
	"windfarm-turbine-monitor-service/internal/telemetry"
	"windfarm-turbine-monitor-service/internal/turbine"
	"windfarm-turbine-monitor-service/internal/workorder"
)

func main() {
	cfg := turbine.FromEnv()

	seed := []turbine.Turbine{
		{ID: "WTG-001", Name: "Turbine Alpha", Model: "V90", Site: cfg.Site, RatedPowerKW: 2000, CutInWindMPS: cfg.DefaultCutIn, CutOutWindMPS: cfg.DefaultCutOut, State: turbine.StateRunning, Commissioned: time.Now().Add(-700 * 24 * time.Hour), Firmware: "fw-1.4.2"},
		{ID: "WTG-002", Name: "Turbine Beta", Model: "V112", Site: cfg.Site, RatedPowerKW: 3000, CutInWindMPS: cfg.DefaultCutIn, CutOutWindMPS: cfg.DefaultCutOut, State: turbine.StateRunning, Commissioned: time.Now().Add(-400 * 24 * time.Hour), Firmware: "fw-1.4.2"},
		{ID: "WTG-003", Name: "Turbine Gamma", Model: "V90", Site: cfg.Site, RatedPowerKW: 2000, CutInWindMPS: cfg.DefaultCutIn, CutOutWindMPS: cfg.DefaultCutOut, State: turbine.StateMaintenance, Commissioned: time.Now().Add(-900 * 24 * time.Hour), Firmware: "fw-1.3.9"},
	}
	registry := turbine.NewRegistry(seed)

	samples := telemetry.NewSampleRegistry(1024)
	ingestor := telemetry.NewIngestor(samples, 4)

	rules := ruleengine.NewRegistry(nil)
	for _, t := range registry.List() {
		for _, rule := range ruleengine.ThresholdSet(t, cfg) {
			_ = rules.Put(rule)
		}
	}
	evaluator := ruleengine.NewEvaluator(rules)

	faultStore := fault.NewStore()
	faultSvc := fault.NewService(faultStore)

	dedup := alarm.NewDeduplicator(5 * time.Minute)
	dispatcher := alarm.NewDispatcher(dedup)
	notifier := alarm.NewNotifier()

	crews := workorder.NewCrewPool([]string{"crew-a", "crew-b", "crew-c"})
	orders := workorder.NewScheduler(crews, nil)

	stream := audit.NewStream()
	builder := report.NewBuilder(registry, samples, nil)

	app := &gateway.App{
		Turbines:  registry,
		Samples:   samples,
		Ingestor:  ingestor,
		Rules:     rules,
		Evaluator: evaluator,
		Faults:    faultSvc,
		Alarms:    dispatcher,
		Notifier:  notifier,
		Orders:    orders,
		Audit:     stream,
		Reports:   builder,
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "18080"
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           gateway.NewRouter(app),
		ReadHeaderTimeout: 5 * time.Second,
	}
	go func() {
		<-ctx.Done()
		_ = srv.Close()
	}()
	log.Printf("windfarm turbine monitor listening on %s", srv.Addr)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}
""")

# ---------------------------------------------------------------------------
# web
# ---------------------------------------------------------------------------
add("web/index.html", """<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Wind Farm Turbine Monitor</title>
  <style>
    body { font-family: system-ui, sans-serif; margin: 2rem; color: #1f2937; }
    h1 { font-size: 1.4rem; }
    table { border-collapse: collapse; margin-top: 1rem; width: 100%; max-width: 720px; }
    th, td { border: 1px solid #e5e7eb; padding: 0.4rem 0.6rem; text-align: left; font-size: 0.9rem; }
    th { background: #f3f4f6; }
    .muted { color: #6b7280; }
  </style>
</head>
<body>
  <h1>Wind Farm Turbine Monitor</h1>
  <p class="muted">Fleet status, telemetry and alarm summary for on-site operators.</p>
  <table id="summary"></table>
  <script>
    fetch('/api/summary').then(r => r.json()).then(d => {
      const rows = Object.entries(d).map(([k, v]) => `<tr><th>${k}</th><td>${v}</td></tr>`).join('');
      document.getElementById('summary').innerHTML = `<tr><th>Field</th><th>Value</th></tr>` + rows;
    });
  </script>
</body>
</html>
""")

# ---------------------------------------------------------------------------
# README
# ---------------------------------------------------------------------------
add("README.md", """# Wind Farm Turbine Monitor Service

A Go service for wind-farm operations. It ingests SCADA telemetry, validates and
normalizes samples, evaluates alarm rules, drives a fault state machine,
dispatches maintenance work orders, computes power-curve analytics and produces
site reports. All domain state is memory-backed; a small HTTP API and a status
page are served by the same process.

## Structure

- `cmd/server`: HTTP service entrypoint and dependency wiring.
- `internal/turbine`: fleet registry and turbine configuration.
- `internal/telemetry`: SCADA sample validation, normalization, ingestion and storage.
- `internal/ruleengine`: threshold alarm rules and evaluation.
- `internal/fault`: fault state machine and lifecycle.
- `internal/alarm`: alarm dedup, dispatch, escalation and notification.
- `internal/workorder`: maintenance scheduling and crew assignment.
- `internal/analytics`: sliding windows, power curve and aggregation.
- `internal/report`: site report building.
- `internal/audit`: event stream, checkpoint and delivery.
- `internal/gateway`: HTTP routing and handlers.
- `web`: a small operations status page served by the Go process.

## Run

```bash
go run ./cmd/server
```

The service listens on `PORT` (default `18080`). Health is at `/health`; a
summary is at `/api/summary`; telemetry is ingested at `POST /api/telemetry`.

## Test

```bash
go test ./...
go test -race ./...
```

## Environment

- `PORT`: HTTP port, defaults to `18080`.
- `SITE_CODE`: logical site identifier, defaults to `NORTH-PLAINS`.
- `DEFAULT_CUT_IN` / `DEFAULT_CUT_OUT`: fleet default wind-speed bounds.
""")

# ---------------------------------------------------------------------------
# write out
# ---------------------------------------------------------------------------
for rel, content in files.items():
    path = os.path.join(BASE, rel)
    os.makedirs(os.path.dirname(path), exist_ok=True)
    with open(path, "w") as f:
        f.write(content)

print(f"wrote {len(files)} files")
for rel in sorted(files):
    print(" ", rel)
