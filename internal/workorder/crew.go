package workorder

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
