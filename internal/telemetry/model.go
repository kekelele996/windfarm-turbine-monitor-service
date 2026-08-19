package telemetry

import "time"

// Sample is one SCADA reading for a turbine.
type Sample struct {
	TurbineID   string    `json:"turbine_id"`
	Timestamp   time.Time `json:"timestamp"`
	WindSpeed   float64   `json:"wind_speed_mps"`
	RotorRPM    float64   `json:"rotor_rpm"`
	GearboxTemp float64   `json:"gearbox_temp_c"`
	GenTemp     float64   `json:"gen_temp_c"`
	Vibration   float64   `json:"vibration_mm_s"`
	PowerOutput float64   `json:"power_output_kw"`
	YawAngle    float64   `json:"yaw_angle_deg"`
	PitchAngle  float64   `json:"pitch_angle_deg"`
}

// SampleBatch is a group of samples ingested together.
type SampleBatch struct {
	Source  string   `json:"source"`
	Samples []Sample `json:"samples"`
}
