package energy

// Revenue summarizes expected and actual income for a period.
type Revenue struct {
	Gross       float64
	Penalties   float64
	Net         float64
	Settlements int
}

// SumRevenue aggregates settlements into a revenue summary.
func SumRevenue(settlements []Settlement, contracted map[string]PPA) Revenue {
	var r Revenue
	for _, s := range settlements {
		r.Gross += s.EnergyKWh * s.PricePerKWh
		ppa, ok := contracted[s.Site]
		if !ok {
			ppa.PricePerKWh = s.PricePerKWh
		}
		if s.EnergyKWh < ppa.MinDeliver {
			short := ppa.MinDeliver - s.EnergyKWh
			penalty := short * ppa.Penalty
			r.Penalties += penalty
		}
		r.Settlements++
	}
	r.Net = r.Gross - r.Penalties
	return r
}

// ProjectRevenue extrapolates a daily average over future days.
func ProjectRevenue(r Revenue, remainingDays int) float64 {
	if r.Settlements == 0 || remainingDays <= 0 {
		return r.Net
	}
	daily := r.Net / float64(r.Settlements)
	return r.Net + daily*float64(remainingDays)
}
