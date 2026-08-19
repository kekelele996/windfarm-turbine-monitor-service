package turbine

import "time"

type TurbineState string

const (
	StateRunning      TurbineState = "running"
	StateIdle         TurbineState = "idle"
	StateMaintenance  TurbineState = "maintenance"
	StateDecommission TurbineState = "decommissioned"
)

// Turbine describes a single wind turbine in the fleet.
type Turbine struct {
	ID            string       `json:"id"`
	Name          string       `json:"name"`
	Model         string       `json:"model"`
	Site          string       `json:"site"`
	RatedPowerKW  float64      `json:"rated_power_kw"`
	CutInWindMPS  float64      `json:"cut_in_wind_mps"`
	CutOutWindMPS float64      `json:"cut_out_wind_mps"`
	State         TurbineState `json:"state"`
	Commissioned  time.Time    `json:"commissioned"`
	Firmware      string       `json:"firmware"`
}

func (t Turbine) Active() bool {
	return t.State == StateRunning || t.State == StateIdle
}
