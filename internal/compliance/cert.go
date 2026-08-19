package compliance

import "time"

// Certificate is a grid-compliance certificate for a turbine model.
type Certificate struct {
	ID           string
	TurbineModel string
	IssuedAt     time.Time
	ExpiresAt    time.Time
	Authority    string
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
