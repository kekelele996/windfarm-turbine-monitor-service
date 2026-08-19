package security

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
