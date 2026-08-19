package forecast

import "math"

// ErrorMetrics summarizes forecast accuracy.
type ErrorMetrics struct {
	MAE  float64 `json:"mae"`
	RMSE float64 `json:"rmse"`
	Bias float64 `json:"bias"`
}

// Evaluate compares predictions to observations and returns error metrics.
func Evaluate(predicted, observed []float64) ErrorMetrics {
	var em ErrorMetrics
	n := min(len(predicted), len(observed))
	if n == 0 {
		return em
	}
	var absSum, sqSum, biasSum float64
	for i := 0; i < n; i++ {
		diff := predicted[i] - observed[i]
		absSum += math.Abs(diff)
		sqSum += diff * diff
		biasSum += diff
	}
	em.MAE = absSum / float64(n)
	em.RMSE = math.Sqrt(sqSum / float64(n))
	em.Bias = biasSum / float64(n)
	return em
}

// WithinTolerance reports the fraction of predictions within a tolerance.
func WithinTolerance(predicted, observed []float64, tol float64) float64 {
	n := min(len(predicted), len(observed))
	if n == 0 {
		return 0
	}
	within := 0
	for i := 0; i < n; i++ {
		if math.Abs(predicted[i]-observed[i]) <= tol {
			within++
		}
	}
	return float64(within) / float64(n)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
