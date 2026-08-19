package scada

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
