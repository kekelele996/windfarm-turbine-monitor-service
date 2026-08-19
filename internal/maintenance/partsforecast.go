package maintenance

// PartsForecast predicts spare part consumption from service history.
type PartsForecast struct {
	UsagePerService float64
}

// ForServices projects parts needed for a number of upcoming services.
func (p PartsForecast) ForServices(serviceCount int) float64 {
	if serviceCount <= 0 {
		return 0
	}
	return p.UsagePerService * float64(serviceCount)
}

// SafetyStock adds a buffer percentage to a projected requirement.
func SafetyStock(projected float64, bufferPercent float64) float64 {
	if projected <= 0 {
		return 0
	}
	return projected * (1 + bufferPercent/100)
}
