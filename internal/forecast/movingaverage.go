package forecast

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
