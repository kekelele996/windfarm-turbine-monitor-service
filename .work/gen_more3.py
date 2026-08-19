#!/usr/bin/env python3
import os
BASE = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
files = {}
def add(p, c): files[p] = c

# --- report additions ---
add("internal/report/compliance.go", """package report

import (
	"time"

	"windfarm-turbine-monitor-service/internal/turbine"
)

// ComplianceSection summarizes regulatory reporting obligations for a site.
type ComplianceSection struct {
	Site         string
	CheckedAt    time.Time
	Warnings     []string
	Passed       bool
}

// BuildCompliance checks fleet-level obligations such as maintenance recency
// and firmware currency.
func BuildCompliance(site string, registry *turbine.Registry, now time.Time, minFirmware string) ComplianceSection {
	sec := ComplianceSection{Site: site, CheckedAt: now, Passed: true}
	for _, t := range registry.List() {
		if t.Site != site {
			continue
		}
		if t.Firmware < minFirmware {
			sec.Warnings = append(sec.Warnings, t.ID+" firmware outdated")
			sec.Passed = false
		}
		if t.State == turbine.StateMaintenance && now.Sub(t.Commissioned) < 30*24*time.Hour {
			sec.Warnings = append(sec.Warnings, t.ID+" premature maintenance")
		}
	}
	return sec
}
""")

add("internal/report/section.go", """package report

// MergeSections combines two sections by title, appending rows.
func MergeSections(a, b Section) Section {
	if a.Title == "" {
		return b
	}
	if a.Title != b.Title {
		return a
	}
	merged := a
	merged.Rows = append(merged.Rows, b.Rows...)
	return merged
}

// RowMap converts a section's rows into a lookup map.
func (s Section) RowMap() map[string]string {
	out := make(map[string]string, len(s.Rows))
	for _, r := range s.Rows {
		out[r.Key] = r.Value
	}
	return out
}

// SortRows orders section rows by key.
func SortRows(s Section) Section {
	rows := make([]SectionRow, len(s.Rows))
	copy(rows, s.Rows)
	for i := 1; i < len(rows); i++ {
		for j := i; j > 0 && rows[j].Key < rows[j-1].Key; j-- {
			rows[j], rows[j-1] = rows[j-1], rows[j]
		}
	}
	s.Rows = rows
	return s
}
""")

# --- audit additions ---
add("internal/audit/retention.go", """package audit

import "time"

// Retention trims stream history to a time window.
type Retention struct {
	window time.Duration
}

func NewRetention(window time.Duration) *Retention {
	if window <= 0 {
		window = 24 * time.Hour
	}
	return &Retention{window: window}
}

// Trim removes events older than the window from history, preserving order.
func (r *Retention) Trim(history []Event, now time.Time) []Event {
	cutoff := now.Add(-r.window)
	out := history[:0]
	for _, e := range history {
		if !e.At.Before(cutoff) {
			out = append(out, e)
		}
	}
	return out
}
""")

add("internal/audit/projection.go", """package audit

import "sort"

// Projection aggregates events by type and turbine for dashboards.
type Projection struct {
	ByType   map[EventType]int
	ByTurbine map[string]int
}

func Project(events []Event) Projection {
	p := Projection{ByType: map[EventType]int{}, ByTurbine: map[string]int{}}
	for _, e := range events {
		p.ByType[e.Type]++
		p.ByTurbine[e.TurbineID]++
	}
	return p
}

// TopTurbines returns turbine IDs ordered by descending event count.
func (p Projection) TopTurbines(limit int) []string {
	type kv struct {
		id  string
		n   int
	}
	list := make([]kv, 0, len(p.ByTurbine))
	for id, n := range p.ByTurbine {
		list = append(list, kv{id, n})
	}
	sort.Slice(list, func(i, j int) bool {
		if list[i].n != list[j].n {
			return list[i].n > list[j].n
		}
		return list[i].id < list[j].id
	})
	out := make([]string, 0, limit)
	for i, item := range list {
		if i >= limit {
			break
		}
		out = append(out, item.id)
	}
	return out
}
""")

# --- new package: internal/grid ---
add("internal/grid/curtailment.go", """package grid

import "time"

// Curtailment is a grid-operator instruction to reduce output.
type Curtailment struct {
	ID          string
	Site        string
	StartAt     time.Time
	EndAt       time.Time
	LimitKW     float64
	Active      bool
}

// Schedule tracks active and upcoming curtailment instructions.
type Schedule struct {
	items []Curtailment
}

func NewSchedule() *Schedule { return &Schedule{} }

func (s *Schedule) Add(c Curtailment) { s.items = append(s.items, c) }

// ActiveFor returns the active curtailment limit for a site at a time.
func (s *Schedule) ActiveFor(site string, at time.Time) (float64, bool) {
	for _, c := range s.items {
		if c.Site == site && c.Active && !at.Before(c.StartAt) && at.Before(c.EndAt) {
			return c.LimitKW, true
		}
	}
	return 0, false
}

// Expire marks any curtailment whose window has passed as inactive.
func (s *Schedule) Expire(at time.Time) int {
	expired := 0
	for i := range s.items {
		if s.items[i].Active && !at.Before(s.items[i].EndAt) {
			s.items[i].Active = false
			expired++
		}
	}
	return expired
}
""")

add("internal/grid/connection.go", """package grid

// ConnectionState describes the grid interconnection status.
type ConnectionState string

const (
	Connected    ConnectionState = "connected"
	Islanded     ConnectionState = "islanded"
	Disconnected ConnectionState = "disconnected"
)

// Connection tracks grid link health per site.
type Connection struct {
	Site    string
	State   ConnectionState
	Voltage float64
}

// AllowExport reports whether power may be exported under the current state.
func AllowExport(c Connection) bool {
	return c.State == Connected && c.Voltage >= 0.9
}

// ConnRegistry stores connection state by site.
type ConnRegistry struct {
	conns map[string]Connection
}

func NewConnRegistry() *ConnRegistry { return &ConnRegistry{conns: map[string]Connection{}} }

func (r *ConnRegistry) Set(c Connection) { r.conns[c.Site] = c }

func (r *ConnRegistry) Get(site string) (Connection, bool) {
	c, ok := r.conns[site]
	return c, ok
}
""")

# --- new package: internal/site ---
add("internal/site/metmast.go", """package site

import "time"

// MetMast is a meteorological reference mast measurement.
type MetMast struct {
	ID        string
	Site      string
	Height    float64
	WindSpeed float64
	WindDir   float64
	AirTemp   float64
	Recorded  time.Time
}

// MetMastStore keeps recent met mast readings.
type MetMastStore struct {
	readings map[string][]MetMast
}

func NewMetMastStore() *MetMastStore { return &MetMastStore{readings: map[string][]MetMast{}} }

func (s *MetMastStore) Add(m MetMast) { s.readings[m.ID] = append(s.readings[m.ID], m) }

func (s *MetMastStore) Latest(id string) (MetMast, bool) {
	rs := s.readings[id]
	if len(rs) == 0 {
		return MetMast{}, false
	}
	return rs[len(rs)-1], true
}

// MeanWind returns the average wind across all masts for a site.
func (s *MetMastStore) MeanWind(site string) float64 {
	var sum float64
	n := 0
	for _, rs := range s.readings {
		if len(rs) == 0 || rs[0].Site != site {
			continue
		}
		for _, m := range rs {
			sum += m.WindSpeed
			n++
		}
	}
	if n == 0 {
		return 0
	}
	return sum / float64(n)
}
""")

add("internal/site/topology.go", """package site

// Topology describes the layout of turbines within a site.
type Topology struct {
	Site     string
	Clusters map[string][]string // cluster id -> turbine ids
}

func NewTopology(site string) *Topology {
	return &Topology{Site: site, Clusters: map[string][]string{}}
}

func (t *Topology) Assign(cluster string, turbineID string) {
	t.Clusters[cluster] = append(t.Clusters[cluster], turbineID)
}

// ClusterOf returns the cluster containing a turbine.
func (t *Topology) ClusterOf(turbineID string) string {
	for cluster, ids := range t.Clusters {
		for _, id := range ids {
			if id == turbineID {
				return cluster
			}
		}
	}
	return ""
}

// ClusterSizes returns a map of cluster id to turbine count.
func (t *Topology) ClusterSizes() map[string]int {
	out := map[string]int{}
	for c, ids := range t.Clusters {
		out[c] = len(ids)
	}
	return out
}
""")

for rel, c in files.items():
    p = os.path.join(BASE, rel)
    os.makedirs(os.path.dirname(p), exist_ok=True)
    with open(p, "w") as f:
        f.write(c)
print(f"wrote {len(files)} files")
