package turbine

import "time"

// DecommissionWorkflow tracks the stages of retiring a turbine.
type DecommissionWorkflow struct {
	stages map[string][]Stage
}

type Stage struct {
	TurbineID string
	Name      string
	Done      bool
	At        time.Time
}

func NewDecommissionWorkflow() *DecommissionWorkflow {
	return &DecommissionWorkflow{stages: map[string][]Stage{}}
}

// Start creates the standard decommission stage sequence for a turbine.
func (w *DecommissionWorkflow) Start(turbineID string, now time.Time) {
	names := []string{"grid-disconnect", "blade-removal", "tower-demolition", "site-restore"}
	stages := make([]Stage, 0, len(names))
	for _, n := range names {
		stages = append(stages, Stage{TurbineID: turbineID, Name: n})
	}
	w.stages[turbineID] = stages
}

// CompleteStage marks a stage done.
func (w *DecommissionWorkflow) CompleteStage(turbineID, name string, now time.Time) bool {
	stages := w.stages[turbineID]
	for i := range stages {
		if stages[i].Name == name && !stages[i].Done {
			stages[i].Done = true
			stages[i].At = now
			return true
		}
	}
	return false
}

// Progress returns the fraction of completed stages.
func (w *DecommissionWorkflow) Progress(turbineID string) float64 {
	stages := w.stages[turbineID]
	if len(stages) == 0 {
		return 0
	}
	done := 0
	for _, s := range stages {
		if s.Done {
			done++
		}
	}
	return float64(done) / float64(len(stages))
}
