package workorder

// CostEstimate approximates the labor cost of a work order.
type CostEstimate struct {
	OrderID     string
	LaborHours  float64
	RatePerHour float64
	PartsCost   float64
	Total       float64
}

// Estimator builds cost estimates from order priority and title length.
type Estimator struct {
	RatePerHour float64
}

func NewEstimator(ratePerHour float64) *Estimator {
	if ratePerHour <= 0 {
		ratePerHour = 120.0
	}
	return &Estimator{RatePerHour: ratePerHour}
}

// Estimate derives a rough labor estimate from order priority.
func (e *Estimator) Estimate(wo WorkOrder, partsCost float64) CostEstimate {
	hours := map[int]float64{1: 2, 2: 4, 3: 8}[wo.Priority]
	if hours == 0 {
		hours = 1
	}
	est := CostEstimate{
		OrderID:     wo.ID,
		LaborHours:  hours,
		RatePerHour: e.RatePerHour,
		PartsCost:   partsCost,
	}
	est.Total = hours*e.RatePerHour + partsCost
	return est
}
