package simulator

import (
	"sync"
)

// RoomModel is implemented by environment models that advance a room's state
// each tick. Advance receives a pre-snapshotted heatInput map and returns
// deltas to apply to RoomState.Current, keyed by measurement type. The caller
// snapshots heatInput via RoomState.HeatInput before acquiring Mu.Lock, then
// holds Mu.Lock for the duration of the Advance call and delta application.
type RoomModel interface {
	Advance(state *RoomState, heatInput map[string]float64, simulatedTickSeconds float64) map[string]float64
}

// RoomState holds the shared runtime state for a single simulated room.
// Current and Ambient are accessed by the publish loop and the environment
// model under the caller-held Mu lock. Actuator contributions are managed
// via SetContribution and ClearContribution, which are self-locking.
type RoomState struct {
	Mu      sync.RWMutex
	Current map[string]float64 // evolves each tick via model advances
	Ambient map[string]float64 // equilibrium — set at startup, never changes

	contributions map[string]map[string]float64 // hwID → measurementType → watts
	heatInput     map[string]float64            // derived sum of all contributions
}

// newRoomState constructs a RoomState with Current initialised equal to Ambient.
func newRoomState(ambient map[string]float64) *RoomState {
	current := make(map[string]float64, len(ambient))
	for k, v := range ambient {
		current[k] = v
	}
	return &RoomState{
		Current:       current,
		Ambient:       ambient,
		contributions: make(map[string]map[string]float64),
		heatInput:     make(map[string]float64),
	}
}

// SetContribution records an actuator device's contribution by measurement
// type and recomputes the derived heatInput sum. Calling SetContribution with
// the same hwID and rates repeatedly is a no-op beyond the first call.
// This method is safe for concurrent use.
func (rs *RoomState) SetContribution(hwID string, rates map[string]float64) {
	rs.Mu.Lock()
	defer rs.Mu.Unlock()
	rs.contributions[hwID] = rates
	rs.recomputeHeatInput()
}

// ClearContribution removes an actuator device's contribution and recomputes
// the derived heatInput sum. Safe to call if no contribution exists for hwID.
// This method is safe for concurrent use.
func (rs *RoomState) ClearContribution(hwID string) {
	rs.Mu.Lock()
	defer rs.Mu.Unlock()
	delete(rs.contributions, hwID)
	rs.recomputeHeatInput()
}

// HeatInput returns a snapshot of the current derived heat input sum by
// measurement type. The snapshot is safe to read without holding Mu.
// This method is safe for concurrent use.
func (rs *RoomState) HeatInput() map[string]float64 {
	rs.Mu.RLock()
	defer rs.Mu.RUnlock()
	snapshot := make(map[string]float64, len(rs.heatInput))
	for k, v := range rs.heatInput {
		snapshot[k] = v
	}
	return snapshot
}

// recomputeHeatInput derives heatInput from the current contributions map.
// Callers must hold Mu before calling.
func (rs *RoomState) recomputeHeatInput() {
	clear(rs.heatInput)
	for _, rates := range rs.contributions {
		for typ, rate := range rates {
			rs.heatInput[typ] += rate
		}
	}
}
