#!/usr/bin/env python3
import os
BASE = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
files = {}
def add(p, c): files[p] = c

add("internal/security/access.go", """package security

import (
	"crypto/subtle"
	"sync"
	"time"
)

// Role is an access role for API operations.
type Role string

const (
	RoleOperator Role = "operator"
	RoleEngineer Role = "engineer"
	RoleAuditor  Role = "auditor"
)

// Key describes an API key with an expiry and role.
type Key struct {
	ID        string
	Secret    string
	Role      Role
	ExpiresAt time.Time
	Active    bool
}

// KeyStore stores API keys and validates access.
type KeyStore struct {
	mu   sync.RWMutex
	keys map[string]Key
}

func NewKeyStore() *KeyStore { return &KeyStore{keys: map[string]Key{}} }

func (s *KeyStore) Put(k Key) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.keys[k.ID] = k
}

// Validate checks a key id/secret pair and reports the role if valid.
func (s *KeyStore) Validate(id, secret string, now time.Time) (Role, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	k, ok := s.keys[id]
	if !ok || !k.Active {
		return "", false
	}
	if !k.ExpiresAt.IsZero() && now.After(k.ExpiresAt) {
		return "", false
	}
	if subtle.ConstantTimeCompare([]byte(k.Secret), []byte(secret)) != 1 {
		return "", false
	}
	return k.Role, true
}

// Can reports whether a role may perform an operation.
func Can(role Role, operation string) bool {
	switch operation {
	case "read":
		return true
	case "write":
		return role == RoleOperator || role == RoleEngineer
	case "admin":
		return role == RoleEngineer
	default:
		return false
	}
}
""")

add("internal/security/hmac.go", """package security

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

// Signer produces and verifies HMAC signatures for payloads.
type Signer struct {
	secret []byte
}

func NewSigner(secret string) *Signer { return &Signer{secret: []byte(secret)} }

// Sign returns the hex HMAC-SHA256 of payload.
func (s *Signer) Sign(payload []byte) string {
	mac := hmac.New(sha256.New, s.secret)
	_, _ = mac.Write(payload)
	return hex.EncodeToString(mac.Sum(nil))
}

// Verify checks a hex signature against the payload.
func (s *Signer) Verify(payload []byte, signature string) (bool, error) {
	want, err := hex.DecodeString(signature)
	if err != nil {
		return false, fmt.Errorf("decode signature: %w", err)
	}
	got := hmac.New(sha256.New, s.secret)
	_, _ = got.Write(payload)
	return hmac.Equal(got.Sum(nil), want), nil
}
""")

add("internal/compliance/cert.go", """package compliance

import "time"

// Certificate is a grid-compliance certificate for a turbine model.
type Certificate struct {
	ID          string
	TurbineModel string
	IssuedAt    time.Time
	ExpiresAt   time.Time
	Authority   string
}

// Status reports whether a certificate is valid at a time.
func (c Certificate) Status(at time.Time) string {
	switch {
	case c.ExpiresAt.IsZero():
		return "unknown"
	case at.After(c.ExpiresAt):
		return "expired"
	case at.Add(30 * 24 * time.Hour).After(c.ExpiresAt):
		return "expiring"
	default:
		return "valid"
	}
}

// Store tracks certificates by model.
type Store struct {
	certs map[string][]Certificate
}

func NewStore() *Store { return &Store{certs: map[string][]Certificate{}} }

func (s *Store) Add(c Certificate) { s.certs[c.TurbineModel] = append(s.certs[c.TurbineModel], c) }

// ValidFor returns whether a model has a certificate valid at the time.
func (s *Store) ValidFor(model string, at time.Time) bool {
	for _, c := range s.certs[model] {
		if c.Status(at) == "valid" || c.Status(at) == "expiring" {
			return true
		}
	}
	return false
}

// Expiring returns models with certificates expiring soon.
func (s *Store) Expiring(at time.Time, days int) []string {
	var out []string
	for model, certs := range s.certs {
		for _, c := range certs {
			if at.Add(time.Duration(days)*24*time.Hour).After(c.ExpiresAt) && !at.After(c.ExpiresAt) {
				out = append(out, model)
				break
			}
		}
	}
	return out
}
""")

add("internal/compliance/report.go", """package compliance

import (
	"time"

	"windfarm-turbine-monitor-service/internal/turbine"
)

// Finding is a single compliance gap.
type Finding struct {
	Severity string
	Turbine  string
	Message  string
}

// Auditor checks fleet-level compliance against certificates.
type Auditor struct {
	certs *Store
}

func NewAuditor(certs *Store) *Auditor { return &Auditor{certs: certs} }

// Audit checks every turbine model has a valid certificate.
func (a *Auditor) Audit(registry *turbine.Registry, at time.Time) []Finding {
	var findings []Finding
	seen := map[string]bool{}
	for _, t := range registry.List() {
		if seen[t.Model] {
			continue
		}
		seen[t.Model] = true
		if !a.certs.ValidFor(t.Model, at) {
			findings = append(findings, Finding{
				Severity: "critical",
				Turbine:  t.ID,
				Message:  "model " + t.Model + " lacks a valid certificate",
			})
		}
	}
	return findings
}

// Summary reduces findings to a pass/fail status.
type Summary struct {
	Passed   bool
	Count    int
	Critical int
}

func Summarize(findings []Finding) Summary {
	s := Summary{Passed: true, Count: len(findings)}
	for _, f := range findings {
		if f.Severity == "critical" {
			s.Critical++
			s.Passed = false
		}
	}
	return s
}
""")

add("internal/projection/alarmproj.go", """package projection

import (
	"sync"

	"windfarm-turbine-monitor-service/internal/alarm"
)

// AlarmProjection keeps per-turbine alarm counters updated incrementally.
type AlarmProjection struct {
	mu     sync.RWMutex
	byTurbine map[string]int
	bySeverity map[string]int
}

func NewAlarmProjection() *AlarmProjection {
	return &AlarmProjection{byTurbine: map[string]int{}, bySeverity: map[string]int{}}
}

// Apply updates counters from a single alarm.
func (p *AlarmProjection) Apply(a alarm.Alarm) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.byTurbine[a.TurbineID]++
	p.bySeverity[a.Severity]++
}

// ByTurbine returns a snapshot of per-turbine counts.
func (p *AlarmProjection) ByTurbine() map[string]int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	out := make(map[string]int, len(p.byTurbine))
	for k, v := range p.byTurbine {
		out[k] = v
	}
	return out
}

// TopSeverity returns the highest-count severity label.
func (p *AlarmProjection) TopSeverity() string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	best := ""
	bestN := -1
	for s, n := range p.bySeverity {
		if n > bestN {
			best = s
			bestN = n
		}
	}
	return best
}
""")

add("internal/projection/fleetproj.go", """package projection

import "windfarm-turbine-monitor-service/internal/turbine"

// FleetProjection maintains derived fleet views for fast reads.
type FleetProjection struct {
	activeBySite map[string]int
}

func NewFleetProjection() *FleetProjection {
	return &FleetProjection{activeBySite: map[string]int{}}
}

// Rebuild recomputes derived views from a registry snapshot.
func (p *FleetProjection) Rebuild(list []turbine.Turbine) {
	p.activeBySite = map[string]int{}
	for _, t := range list {
		if t.Active() {
			p.activeBySite[t.Site]++
		}
	}
}

// ActiveOnSite returns the active turbine count for a site.
func (p *FleetProjection) ActiveOnSite(site string) int { return p.activeBySite[site] }

// Sites returns distinct site codes with at least one active turbine.
func (p *FleetProjection) Sites() []string {
	out := make([]string, 0, len(p.activeBySite))
	for s, n := range p.activeBySite {
		if n > 0 {
			out = append(out, s)
		}
	}
	return out
}
""")

add("internal/storage/snapshot.go", """package storage

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Snapshot captures a serializable state tree to disk.
type Snapshot struct {
	Version int                    `json:"version"`
	Data    map[string]interface{} `json:"data"`
}

// Write persists a snapshot atomically to a file.
func Write(path string, snap Snapshot) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(snap, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// Read loads a snapshot from disk.
func Read(path string) (Snapshot, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Snapshot{}, err
	}
	var snap Snapshot
	if err := json.Unmarshal(data, &snap); err != nil {
		return Snapshot{}, err
	}
	return snap, nil
}

// Merge overlays b onto a, returning a new map.
func Merge(a, b map[string]interface{}) map[string]interface{} {
	out := make(map[string]interface{}, len(a)+len(b))
	for k, v := range a {
		out[k] = v
	}
	for k, v := range b {
		out[k] = v
	}
	return out
}
""")

add("internal/storage/encoding.go", """package storage

import (
	"encoding/json"
	"fmt"
)

// Marshal encodes a value to indented JSON bytes.
func Marshal(v interface{}) ([]byte, error) {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal: %w", err)
	}
	return data, nil
}

// Unmarshal decodes JSON bytes into v.
func Unmarshal(data []byte, v interface{}) error {
	if err := json.Unmarshal(data, v); err != nil {
		return fmt.Errorf("unmarshal: %w", err)
	}
	return nil
}

// Clone deep-copies a value through JSON round-trip.
func Clone(v interface{}) (interface{}, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	var out interface{}
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, err
	}
	return out, nil
}
""")

for rel, c in files.items():
    p = os.path.join(BASE, rel)
    os.makedirs(os.path.dirname(p), exist_ok=True)
    with open(p, "w") as f:
        f.write(c)
print(f"wrote {len(files)} files")
