package energy

import "time"

// Settlement computes energy delivered for a billing interval.
type Settlement struct {
	Site        string
	From        time.Time
	To          time.Time
	EnergyKWh   float64
	PricePerKWh float64
	Total       float64
}

// PPA describes a power purchase agreement.
type PPA struct {
	Site        string
	PricePerKWh float64
	MinDeliver  float64 // minimum contracted energy in kWh
	Penalty     float64 // per-kWh penalty below minimum
}

// Settle builds a settlement record for an interval.
func Settle(site string, from, to time.Time, energyKWh float64, ppa PPA) Settlement {
	s := Settlement{
		Site:        site,
		From:        from,
		To:          to,
		EnergyKWh:   energyKWh,
		PricePerKWh: ppa.PricePerKWh,
	}
	s.Total = energyKWh * ppa.PricePerKWh
	if energyKWh < ppa.MinDeliver {
		short := ppa.MinDeliver - energyKWh
		s.Total -= short * ppa.Penalty
	}
	return s
}

// AdjustEnergy adds a metering correction to a settlement.
func (s *Settlement) AdjustEnergy(deltaKWh float64) {
	s.EnergyKWh += deltaKWh
	s.Total = s.EnergyKWh * s.PricePerKWh
}
