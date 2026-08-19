package fault

// allowed maps each state to the states reachable from it.
var allowed = map[State][]State{
	StateNormal:      {StateWarning},
	StateWarning:     {StateNormal, StateFault},
	StateFault:       {StateRetrying, StateMaintenance},
	StateMaintenance: {StateNormal},
	// A retry that succeeds clears the fault back to normal; one that fails
	// falls through to maintenance instead of stranding the fault mid-flow.
	StateRetrying: {StateNormal, StateMaintenance},
}

// CanTransition reports whether a move from -> to is legal.
func CanTransition(from, to State) bool {
	for _, s := range allowed[from] {
		if s == to {
			return true
		}
	}
	return false
}

// TerminalState reports whether the state is a stable end state for reporting.
func TerminalState(s State) bool { return s == StateNormal || s == StateMaintenance }
