package simulator

import (
	"math/rand/v2"

	"github.com/DiegoJohnson25/climate-control/simulator-service/internal/config"
)

// ---------------------------------------------------------------------------
// EnvironmentModel
// ---------------------------------------------------------------------------

// EnvironmentModel advances a room's environmental state each tick using a
// thermal equation applied uniformly across all measurement types. The same
// equation governs both temperature and humidity — only the per-type
// parameters differ.
//
// For each measurement type on each tick:
//
//	effectiveAmbient = Ambient + roomNoise (zero if room is not noisy)
//	energyInput      = heatInput * simulatedTickSeconds
//	passiveLoss      = conductance * (Current - effectiveAmbient) * simulatedTickSeconds
//	delta            = (energyInput - passiveLoss) / thermalMass
//
// Static rooms (no actuators) always have zero heatInput, so Current tracks
// ambient with only the passive loss term active. Reactive rooms accumulate
// heatInput from actuator contributions via RoomState.SetContribution.
type EnvironmentModel struct {
	thermalMass map[string]float64 // resistance to change per measurement type
	conductance map[string]float64 // rate of return toward ambient per measurement type
	noise       map[string]float64 // room-level noise std dev; zero if not noisy
}

// newEnvironmentModel constructs an EnvironmentModel from resolved room
// measurement config. Noise values are already zeroed by config resolution
// if the room is not noisy — the model has no knowledge of the noisy flag.
func newEnvironmentModel(measurements map[string]config.MeasurementConfig) *EnvironmentModel {
	thermalMass := make(map[string]float64, len(measurements))
	conductance := make(map[string]float64, len(measurements))
	noise := make(map[string]float64, len(measurements))

	for typ, m := range measurements {
		thermalMass[typ] = m.ThermalMass
		conductance[typ] = m.Conductance
		noise[typ] = m.Noise
	}

	return &EnvironmentModel{
		thermalMass: thermalMass,
		conductance: conductance,
		noise:       noise,
	}
}

// Advance computes the delta to apply to each measurement type in Current for
// one tick of simulated time. heatInput is a pre-snapshotted copy from
// RoomState.HeatInput — the caller acquires Mu.Lock after snapshotting and
// holds it for the duration of this call and the subsequent delta application.
// The caller is responsible for clamping bounds after applying deltas.
//
// If a measurement type present in Current has no corresponding model
// parameters (e.g. air_quality on a sensor-only device), its delta is zero.
func (m *EnvironmentModel) Advance(state *RoomState, heatInput map[string]float64, simulatedTickSeconds float64) map[string]float64 {
	deltas := make(map[string]float64, len(state.Current))

	for typ, current := range state.Current {
		thermalMass, ok := m.thermalMass[typ]
		if !ok || thermalMass == 0 {
			deltas[typ] = 0
			continue
		}

		effectiveAmbient := state.Ambient[typ] + rand.NormFloat64()*m.noise[typ]
		energyInput := heatInput[typ] * simulatedTickSeconds
		passiveLoss := m.conductance[typ] * (current - effectiveAmbient) * simulatedTickSeconds
		deltas[typ] = (energyInput - passiveLoss) / thermalMass
	}

	return deltas
}
