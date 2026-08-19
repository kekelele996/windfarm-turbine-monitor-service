package fault

import (
	"sync"
	"time"
)

// HistoryEntry records one state change for audit and diagnostics.
type HistoryEntry struct {
	FaultID string
	From    State
	To      State
	At      time.Time
	Note    string
}

// History stores fault transition history in memory.
type History struct {
	mu      sync.RWMutex
	entries []HistoryEntry
}

func NewHistory() *History { return &History{} }

func (h *History) Append(e HistoryEntry) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.entries = append(h.entries, e)
}

func (h *History) ForFault(id string) []HistoryEntry {
	h.mu.RLock()
	defer h.mu.RUnlock()
	var out []HistoryEntry
	for _, e := range h.entries {
		if e.FaultID == id {
			out = append(out, e)
		}
	}
	return out
}

// Timeline returns a copy of all transitions, oldest first.
func (h *History) Timeline() []HistoryEntry {
	h.mu.RLock()
	defer h.mu.RUnlock()
	out := make([]HistoryEntry, len(h.entries))
	copy(out, h.entries)
	return out
}
