#!/usr/bin/env python3
import os
BASE = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
files = {}
def add(p, c): files[p] = c

add("internal/forecast/persistence.go", """package forecast

import "windfarm-turbine-monitor-service/internal/telemetry"

// PersistenceModel forecasts the next wind speed as the last observed value.
type PersistenceModel struct{}

func (PersistenceModel) Name() string { return "persistence" }

// Forecast returns the most recent wind speed as the next estimate.
func (PersistenceModel) Forecast(history []telemetry.Sample, horizon int) []float64 {
	out := make([]float64, horizon)
	base := 0.0
	if len(history) > 0 {
		base = history[len(history)-1].WindSpeed
	}
	for i := range out {
		out[i] = base
	}
	return out
}
""")

add("internal/forecast/movingaverage.go", """package forecast

import "windfarm-turbine-monitor-service/internal/telemetry"

// MovingAverageModel forecasts by averaging the most recent window of samples.
type MovingAverageModel struct {
	Window int
}

func NewMovingAverageModel(window int) *MovingAverageModel {
	if window <= 0 {
		window = 10
	}
	return &MovingAverageModel{Window: window}
}

func (m *MovingAverageModel) Name() string { return "moving-average" }

// Forecast averages the last Window samples and repeats the value.
func (m *MovingAverageModel) Forecast(history []telemetry.Sample, horizon int) []float64 {
	out := make([]float64, horizon)
	if len(history) == 0 {
		return out
	}
	start := len(history) - m.Window
	if start < 0 {
		start = 0
	}
	var sum float64
	n := 0
	for _, s := range history[start:] {
		sum += s.WindSpeed
		n++
	}
	avg := sum / float64(n)
	for i := range out {
		out[i] = avg
	}
	return out
}

// WindowError computes the mean absolute error over the fitted window.
func (m *MovingAverageModel) WindowError(history []telemetry.Sample) float64 {
	if len(history) < 2 {
		return 0
	}
	var total float64
	n := 0
	for i := 1; i < len(history); i++ {
		start := i - m.Window
		if start < 0 {
			start = 0
		}
		var sum float64
		c := 0
		for _, s := range history[start:i] {
			sum += s.WindSpeed
			c++
		}
		if c == 0 {
			continue
		}
		pred := sum / float64(c)
		diff := pred - history[i].WindSpeed
		if diff < 0 {
			diff = -diff
		}
		total += diff
		n++
	}
	if n == 0 {
		return 0
	}
	return total / float64(n)
}
""")

add("internal/forecast/error.go", """package forecast

import "math"

// ErrorMetrics summarizes forecast accuracy.
type ErrorMetrics struct {
	MAE  float64 `json:"mae"`
	RMSE float64 `json:"rmse"`
	Bias float64 `json:"bias"`
}

// Evaluate compares predictions to observations and returns error metrics.
func Evaluate(predicted, observed []float64) ErrorMetrics {
	var em ErrorMetrics
	n := min(len(predicted), len(observed))
	if n == 0 {
		return em
	}
	var absSum, sqSum, biasSum float64
	for i := 0; i < n; i++ {
		diff := predicted[i] - observed[i]
		absSum += math.Abs(diff)
		sqSum += diff * diff
		biasSum += diff
	}
	em.MAE = absSum / float64(n)
	em.RMSE = math.Sqrt(sqSum / float64(n))
	em.Bias = biasSum / float64(n)
	return em
}

// WithinTolerance reports the fraction of predictions within a tolerance.
func WithinTolerance(predicted, observed []float64, tol float64) float64 {
	n := min(len(predicted), len(observed))
	if n == 0 {
		return 0
	}
	within := 0
	for i := 0; i < n; i++ {
		if math.Abs(predicted[i]-observed[i]) <= tol {
			within++
		}
	}
	return float64(within) / float64(n)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
""")

add("internal/energy/settlement.go", """package energy

import "time"

// Settlement computes energy delivered for a billing interval.
type Settlement struct {
	Site        string
	From        time.Time
	To          time.Time
	EnergyKWh   float64
	PricePerKWh float64
	Total       float64
}

// PPA describes a power purchase agreement.
type PPA struct {
	Site        string
	PricePerKWh float64
	MinDeliver  float64 // minimum contracted energy in kWh
	Penalty     float64 // per-kWh penalty below minimum
}

// Settle builds a settlement record for an interval.
func Settle(site string, from, to time.Time, energyKWh float64, ppa PPA) Settlement {
	s := Settlement{
		Site:        site,
		From:        from,
		To:          to,
		EnergyKWh:   energyKWh,
		PricePerKWh: ppa.PricePerKWh,
	}
	s.Total = energyKWh * ppa.PricePerKWh
	if energyKWh < ppa.MinDeliver {
		short := ppa.MinDeliver - energyKWh
		s.Total -= short * ppa.Penalty
	}
	return s
}

// AdjustEnergy adds a metering correction to a settlement.
func (s *Settlement) AdjustEnergy(deltaKWh float64) {
	s.EnergyKWh += deltaKWh
	s.Total = s.EnergyKWh * s.PricePerKWh
}
""")

add("internal/energy/revenue.go", """package energy

// Revenue summarizes expected and actual income for a period.
type Revenue struct {
	Gross        float64
	Penalties    float64
	Net          float64
	Settlements  int
}

// SumRevenue aggregates settlements into a revenue summary.
func SumRevenue(settlements []Settlement, contracted map[string]PPA) Revenue {
	var r Revenue
	for _, s := range settlements {
		r.Gross += s.EnergyKWh * s.PricePerKWh
		ppa, ok := contracted[s.Site]
		if !ok {
			ppa.PricePerKWh = s.PricePerKWh
		}
		if s.EnergyKWh < ppa.MinDeliver {
			short := ppa.MinDeliver - s.EnergyKWh
			penalty := short * ppa.Penalty
			r.Penalties += penalty
		}
		r.Settlements++
	}
	r.Net = r.Gross - r.Penalties
	return r
}

// ProjectRevenue extrapolates a daily average over future days.
func ProjectRevenue(r Revenue, remainingDays int) float64 {
	if r.Settlements == 0 || remainingDays <= 0 {
		return r.Net
	}
	daily := r.Net / float64(r.Settlements)
	return r.Net + daily*float64(remainingDays)
}
""")

add("internal/energy/metering.go", """package energy

import (
	"time"

	"windfarm-turbine-monitor-service/internal/telemetry"
)

// MeterReading is a cumulative energy counter at a point in time.
type MeterReading struct {
	TurbineID string
	At        time.Time
	EnergyKWh float64
}

// Meter aggregates sample power into energy over a fixed interval.
type Meter struct {
	interval time.Duration
}

func NewMeter(interval time.Duration) *Meter {
	if interval <= 0 {
		interval = time.Hour
	}
	return &Meter{interval: interval}
}

// ComputeEnergy converts average power over the interval into energy (kWh).
func (m *Meter) ComputeEnergy(samples []telemetry.Sample) float64 {
	if len(samples) == 0 {
		return 0
	}
	var sum float64
	for _, s := range samples {
		sum += s.PowerOutput
	}
	mean := sum / float64(len(samples))
	return mean * m.interval.Hours()
}

// ReadingsFor groups samples by turbine and computes one reading each.
func (m *Meter) ReadingsFor(samples []telemetry.Sample) []MeterReading {
	grouped := map[string][]telemetry.Sample{}
	for _, s := range samples {
		grouped[s.TurbineID] = append(grouped[s.TurbineID], s)
	}
	out := make([]MeterReading, 0, len(grouped))
	for id, ss := range grouped {
		out = append(out, MeterReading{
			TurbineID: id,
			EnergyKWh: m.ComputeEnergy(ss),
		})
	}
	return out
}
""")

add("internal/maintenance/plan.go", """package maintenance

import "time"

// Plan describes a recurring maintenance routine.
type Plan struct {
	ID            string
	TurbineModel  string
	IntervalDays  int
	DurationHours float64
	Checklist     []string
}

// NextDue computes the next due date after the last service.
func (p Plan) NextDue(lastService time.Time) time.Time {
	return lastService.Add(time.Duration(p.IntervalDays) * 24 * time.Hour)
}

// Overdue reports whether a turbine is past its next due date.
func (p Plan) Overdue(lastService, now time.Time) bool {
	return now.After(p.NextDue(lastService))
}

// DaysOverdue returns how many days a turbine is overdue (0 when not).
func (p Plan) DaysOverdue(lastService, now time.Time) int {
	due := p.NextDue(lastService)
	if !now.After(due) {
		return 0
	}
	return int(now.Sub(due).Hours() / 24)
}
""")

add("internal/maintenance/interval.go", """package maintenance

import (
	"sort"
	"time"
)

// ServiceRecord is one completed maintenance event.
type ServiceRecord struct {
	TurbineID string
	At        time.Time
	Hours     float64
}

// Scheduler plans service windows for a fleet.
type Scheduler struct {
	records map[string][]ServiceRecord
}

func NewScheduler() *Scheduler { return &Scheduler{records: map[string][]ServiceRecord{}} }

func (s *Scheduler) Record(r ServiceRecord) { s.records[r.TurbineID] = append(s.records[r.TurbineID], r) }

// LastService returns the most recent service record for a turbine.
func (s *Scheduler) LastService(turbineID string) (ServiceRecord, bool) {
	rs := s.records[turbineID]
	if len(rs) == 0 {
		return ServiceRecord{}, false
	}
	latest := rs[0]
	for _, r := range rs[1:] {
		if r.At.After(latest.At) {
			latest = r
		}
	}
	return latest, true
}

// Upcoming returns turbines due for service within the next days.
func (s *Scheduler) Upcoming(plan Plan, now time.Time, days int) []string {
	var out []string
	for id, rs := range s.records {
		last, ok := s.last(rs)
		if !ok {
			out = append(out, id)
			continue
		}
		due := plan.NextDue(last)
		if now.Add(time.Duration(days)*24*time.Hour).After(due) {
			out = append(out, id)
		}
	}
	sort.Strings(out)
	return out
}

func (s *Scheduler) last(rs []ServiceRecord) (time.Time, bool) {
	if len(rs) == 0 {
		return time.Time{}, false
	}
	latest := rs[0].At
	for _, r := range rs[1:] {
		if r.At.After(latest) {
			latest = r.At
		}
	}
	return latest, true
}
""")

add("internal/maintenance/partsforecast.go", """package maintenance

// PartsForecast predicts spare part consumption from service history.
type PartsForecast struct {
	UsagePerService float64
}

// ForServices projects parts needed for a number of upcoming services.
func (p PartsForecast) ForServices(serviceCount int) float64 {
	if serviceCount <= 0 {
		return 0
	}
	return p.UsagePerService * float64(serviceCount)
}

// SafetyStock adds a buffer percentage to a projected requirement.
func SafetyStock(projected float64, bufferPercent float64) float64 {
	if projected <= 0 {
		return 0
	}
	return projected * (1 + bufferPercent/100)
}
""")

add("internal/weather/icing.go", """package weather

import "windfarm-turbine-monitor-service/internal/telemetry"

// IcingRisk evaluates conditions that can cause blade icing.
type IcingRisk struct {
	TemperatureLimit float64
	HumidityLimit    float64
}

func DefaultIcingRisk() IcingRisk {
	return IcingRisk{TemperatureLimit: 2.0, HumidityLimit: 90.0}
}

// AtRisk reports whether a sample indicates icing conditions.
func (r IcingRisk) AtRisk(s telemetry.Sample, humidity float64) bool {
	return s.GenTemp <= r.TemperatureLimit && humidity >= r.HumidityLimit
}

// RiskLevel maps a count of risky samples to a severity level.
func RiskLevel(riskySamples, totalSamples int) string {
	if totalSamples == 0 {
		return "none"
	}
	ratio := float64(riskySamples) / float64(totalSamples)
	switch {
	case ratio >= 0.5:
		return "high"
	case ratio >= 0.2:
		return "medium"
	default:
		return "low"
	}
}
""")

add("internal/weather/station.go", """package weather

import "time"

// Reading is a single weather station observation.
type Reading struct {
	StationID string
	At        time.Time
	AirTemp   float64
	Humidity  float64
	Pressure  float64
	WindSpeed float64
}

// Station aggregates readings for one weather station.
type Station struct {
	ID       string
	readings []Reading
}

func NewStation(id string) *Station { return &Station{ID: id} }

func (s *Station) Add(r Reading) { s.readings = append(s.readings, r) }

func (s *Station) Latest() (Reading, bool) {
	if len(s.readings) == 0 {
		return Reading{}, false
	}
	return s.readings[len(s.readings)-1], true
}

// AverageHumidity computes mean humidity over the station's history.
func (s *Station) AverageHumidity() float64 {
	if len(s.readings) == 0 {
		return 0
	}
	var sum float64
	for _, r := range s.readings {
		sum += r.Humidity
	}
	return sum / float64(len(s.readings))
}
""")

add("internal/weather/forecast.go", """package weather

import "time"

// ForecastPoint is one forecasted weather value.
type ForecastPoint struct {
	At        time.Time
	WindSpeed float64
	AirTemp   float64
}

// Forecast is a time series of forecast points.
type Forecast struct {
	Points []ForecastPoint
}

// WindAt returns the forecasted wind speed nearest to a time.
func (f Forecast) WindAt(t time.Time) (float64, bool) {
	if len(f.Points) == 0 {
		return 0, false
	}
	best := f.Points[0]
	bestDiff := absDur(t.Sub(best.At))
	for _, p := range f.Points[1:] {
		if d := absDur(t.Sub(p.At)); d < bestDiff {
			best = p
			bestDiff = d
		}
	}
	return best.WindSpeed, true
}

// Coldest returns the minimum forecasted temperature and its time.
func (f Forecast) Coldest() (float64, time.Time, bool) {
	if len(f.Points) == 0 {
		return 0, time.Time{}, false
	}
	cold := f.Points[0]
	for _, p := range f.Points[1:] {
		if p.AirTemp < cold.AirTemp {
			cold = p
		}
	}
	return cold.AirTemp, cold.At, true
}

func absDur(d time.Duration) time.Duration {
	if d < 0 {
		return -d
	}
	return d
}
""")

add("internal/scada/heartbeat.go", """package scada

import "time"

// Heartbeat tracks liveness of SCADA connections.
type Heartbeat struct {
	lastSeen map[string]time.Time
}

func NewHeartbeat() *Heartbeat { return &Heartbeat{lastSeen: map[string]time.Time{}} }

func (h *Heartbeat) Seen(turbineID string, at time.Time) { h.lastSeen[turbineID] = at }

// Stale reports turbines whose last heartbeat is older than the timeout.
func (h *Heartbeat) Stale(timeout time.Duration, now time.Time) []string {
	var out []string
	for id, seen := range h.lastSeen {
		if now.Sub(seen) > timeout {
			out = append(out, id)
		}
	}
	return out
}

// Age returns how long ago a turbine last heartbeated.
func (h *Heartbeat) Age(turbineID string, now time.Time) (time.Duration, bool) {
	seen, ok := h.lastSeen[turbineID]
	if !ok {
		return 0, false
	}
	return now.Sub(seen), true
}
""")

add("internal/scada/framing.go", """package scada

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
)

// Frame is one SCADA protocol frame.
type Frame struct {
	Address uint16
	Command byte
	Payload []byte
}

const (
	frameMagic = 0xA5
	maxPayload = 255
)

// WriteFrame serializes a frame with a simple magic/len/checksum framing.
func WriteFrame(w io.Writer, f Frame) error {
	if len(f.Payload) > maxPayload {
		return fmt.Errorf("payload too large: %d", len(f.Payload))
	}
	header := make([]byte, 6)
	header[0] = frameMagic
	binary.BigEndian.PutUint16(header[1:3], f.Address)
	header[3] = f.Command
	header[4] = byte(len(f.Payload))
	header[5] = checksum(header[:5])
	if _, err := w.Write(header); err != nil {
		return fmt.Errorf("write frame header: %w", err)
	}
	if _, err := w.Write(f.Payload); err != nil {
		return fmt.Errorf("write frame payload: %w", err)
	}
	return nil
}

// ReadFrame parses one frame from a buffered reader.
func ReadFrame(r *bufio.Reader) (Frame, error) {
	header := make([]byte, 6)
	if _, err := io.ReadFull(r, header); err != nil {
		return Frame{}, fmt.Errorf("read header: %w", err)
	}
	if header[0] != frameMagic {
		return Frame{}, fmt.Errorf("bad magic byte 0x%x", header[0])
	}
	if header[5] != checksum(header[:5]) {
		return Frame{}, fmt.Errorf("header checksum mismatch")
	}
	f := Frame{
		Address: binary.BigEndian.Uint16(header[1:3]),
		Command: header[3],
	}
	n := int(header[4])
	f.Payload = make([]byte, n)
	if _, err := io.ReadFull(r, f.Payload); err != nil {
		return Frame{}, fmt.Errorf("read payload: %w", err)
	}
	return f, nil
}

func checksum(b []byte) byte {
	var sum byte
	for _, c := range b {
		sum ^= c
	}
	return sum
}
""")

add("internal/scada/session.go", """package scada

import (
	"sync"
	"time"
)

// Session is an active SCADA channel for one turbine.
type Session struct {
	TurbineID string
	OpenedAt  time.Time
	closed    bool
	mu        sync.Mutex
}

// SessionPool manages open SCADA sessions.
type SessionPool struct {
	mu       sync.RWMutex
	sessions map[string]*Session
}

func NewSessionPool() *SessionPool { return &SessionPool{sessions: map[string]*Session{}} }

// Open creates a session for a turbine, replacing any existing one.
func (p *SessionPool) Open(turbineID string, now time.Time) *Session {
	p.mu.Lock()
	defer p.mu.Unlock()
	s := &Session{TurbineID: turbineID, OpenedAt: now}
	p.sessions[turbineID] = s
	return s
}

// Close marks a session closed and removes it from the pool.
func (p *SessionPool) Close(turbineID string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if s, ok := p.sessions[turbineID]; ok {
		s.mu.Lock()
		s.closed = true
		s.mu.Unlock()
		delete(p.sessions, turbineID)
	}
}

// Get returns the open session for a turbine.
func (p *SessionPool) Get(turbineID string) (*Session, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	s, ok := p.sessions[turbineID]
	return s, ok
}

// Count returns the number of open sessions.
func (p *SessionPool) Count() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.sessions)
}

// IsClosed reports whether a session has been closed.
func (s *Session) IsClosed() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.closed
}
""")

for rel, c in files.items():
    p = os.path.join(BASE, rel)
    os.makedirs(os.path.dirname(p), exist_ok=True)
    with open(p, "w") as f:
        f.write(c)
print(f"wrote {len(files)} files")
