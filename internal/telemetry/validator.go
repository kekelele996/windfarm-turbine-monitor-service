package telemetry

import (
	"fmt"

	"windfarm-turbine-monitor-service/internal/platform"
)

// Validator applies physical plausibility checks to incoming samples.
type Validator struct {
	MaxWindSpeed   float64
	MaxRotorRPM    float64
	MaxGearboxTemp float64
	MaxGenTemp     float64
	MaxVibration   float64
}

func DefaultValidator() Validator {
	return Validator{
		MaxWindSpeed:   40.0,
		MaxRotorRPM:    22.0,
		MaxGearboxTemp: 120.0,
		MaxGenTemp:     140.0,
		MaxVibration:   25.0,
	}
}

// Validate checks one sample and returns a descriptive error for the first
// violation encountered.
func (v Validator) Validate(s Sample) error {
	if s.TurbineID == "" {
		return fmt.Errorf("turbine id missing: %w", platform.ErrInvalid)
	}
	if s.Timestamp.IsZero() {
		return fmt.Errorf("sample timestamp zero: %w", platform.ErrInvalid)
	}
	if s.WindSpeed < 0 || s.WindSpeed > v.MaxWindSpeed {
		return fmt.Errorf("wind speed %.2f out of range: %w", s.WindSpeed, platform.ErrInvalid)
	}
	if s.RotorRPM < 0 || s.RotorRPM > v.MaxRotorRPM {
		return fmt.Errorf("rotor rpm %.2f out of range: %w", s.RotorRPM, platform.ErrInvalid)
	}
	if s.GearboxTemp < -40 || s.GearboxTemp > v.MaxGearboxTemp {
		return fmt.Errorf("gearbox temp %.2f out of range: %w", s.GearboxTemp, platform.ErrInvalid)
	}
	if s.GenTemp < -40 || s.GenTemp > v.MaxGenTemp {
		return fmt.Errorf("gen temp %.2f out of range: %w", s.GenTemp, platform.ErrInvalid)
	}
	if s.Vibration < 0 || s.Vibration > v.MaxVibration {
		return fmt.Errorf("vibration %.2f out of range: %w", s.Vibration, platform.ErrInvalid)
	}
	return nil
}

// ValidateBatch returns the subset of valid samples and a list of errors keyed
// by sample index.
func (v Validator) ValidateBatch(batch SampleBatch) ([]Sample, []error) {
	valid := make([]Sample, 0, len(batch.Samples))
	var errs []error
	for i, s := range batch.Samples {
		if err := v.Validate(s); err != nil {
			errs = append(errs, fmt.Errorf("sample %d: %w", i, err))
			continue
		}
		valid = append(valid, s)
	}
	return valid, errs
}
