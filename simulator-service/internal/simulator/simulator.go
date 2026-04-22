// Package simulator runs the per-device publish loop and manages per-room
// environmental state. Each device gets its own goroutine — sensor goroutines
// publish telemetry on a staggered tick interval, actuator goroutines subscribe
// to command topics and update room heat input. A watchdog goroutine per
// actuator clears contributions if device-service goes silent.
package simulator

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand/v2"
	"sync"
	"time"

	"github.com/DiegoJohnson25/climate-control/simulator-service/internal/config"
	"github.com/DiegoJohnson25/climate-control/simulator-service/internal/mqtt"
	"github.com/DiegoJohnson25/climate-control/simulator-service/internal/provisioning"
)

// ---------------------------------------------------------------------------
// MQTT payload types
// ---------------------------------------------------------------------------

type telemetryPayload struct {
	HwID     string    `json:"hw_id"`
	Readings []reading `json:"readings"`
}

type reading struct {
	Type  string  `json:"type"`
	Value float64 `json:"value"`
}

// ---------------------------------------------------------------------------
// Run
// ---------------------------------------------------------------------------

// Run spawns goroutines for every provisioned device and blocks until ctx is
// cancelled. Sensor device goroutines publish telemetry on a staggered tick.
// Actuator device goroutines subscribe to command topics and update room state.
// A watchdog goroutine per actuator clears contributions if device-service goes
// silent beyond the watchdog timeout.
func Run(ctx context.Context, cfg *config.Config, users []provisioning.ProvisionedUser, mqttClient *mqtt.Client) error {
	type roomEntry struct {
		state *RoomState
		model RoomModel
	}

	type deviceEntry struct {
		device provisioning.ProvisionedDevice
		room   *roomEntry
	}

	var entries []deviceEntry
	var rooms []*roomEntry

	publishInterval := cfg.EffectivePublishInterval
	simulatedTickSeconds := cfg.SimulatedTickSeconds
	watchdogTimeout := time.Duration(cfg.WatchdogTimeoutSeconds()) * time.Second

	for _, user := range users {
		for _, room := range user.Rooms {
			ambient := make(map[string]float64, len(room.Config.Measurements))
			for typ, m := range room.Config.Measurements {
				ambient[typ] = m.Base
			}

			re := &roomEntry{
				state: newRoomState(ambient),
				model: newEnvironmentModel(room.Config.Measurements),
			}
			rooms = append(rooms, re)

			for _, dev := range room.Devices {
				entries = append(entries, deviceEntry{device: dev, room: re})
			}
		}
	}

	totalDevices := len(entries)
	if totalDevices == 0 {
		log.Println("no devices to simulate")
		return nil
	}

	var wg sync.WaitGroup

	for i, entry := range entries {
		staggerOffset := time.Duration(float64(i) / float64(totalDevices) * float64(publishInterval))

		if len(entry.device.Config.Sensors) > 0 {
			wg.Add(1)
			go func(dev provisioning.ProvisionedDevice, re *roomEntry, offset time.Duration) {
				defer wg.Done()
				runSensor(ctx, dev, re.state, re.model, mqttClient, publishInterval, simulatedTickSeconds, offset)
			}(entry.device, entry.room, staggerOffset)
		}

		for _, act := range entry.device.Config.Actuators {
			wg.Add(1)
			go func(dev provisioning.ProvisionedDevice, act config.ActuatorConfig, re *roomEntry) {
				defer wg.Done()
				runActuator(ctx, dev, act, re.state, mqttClient, watchdogTimeout)
			}(entry.device, act, entry.room)
		}
	}

	log.Printf("simulator running — %d device(s) across %d room(s)\n\ttime scale: %.0fx | base tick: %ds | publish interval: %s | simulated tick: %gs",
		totalDevices, len(rooms), cfg.Simulation.TimeScale, cfg.BaseTickSeconds, publishInterval, simulatedTickSeconds)

	wg.Wait()
	return nil
}

// ---------------------------------------------------------------------------
// Sensor goroutine
// ---------------------------------------------------------------------------

// runSensor publishes telemetry for a single device on each tick, advancing
// the room model before reading Current values.
func runSensor(ctx context.Context, dev provisioning.ProvisionedDevice, state *RoomState, model RoomModel, mqttClient *mqtt.Client, publishInterval time.Duration, simulatedTickSeconds float64, offset time.Duration) {
	select {
	case <-time.After(offset):
	case <-ctx.Done():
		return
	}

	ticker := time.NewTicker(publishInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			advanceRoom(state, model, simulatedTickSeconds)
			if err := publishTelemetry(dev, state, mqttClient); err != nil {
				log.Printf("[%s] publish error: %v", dev.HwID, err)
			}
		case <-ctx.Done():
			return
		}
	}
}

// ---------------------------------------------------------------------------
// Actuator goroutine
// ---------------------------------------------------------------------------

// runActuator subscribes to the command topic for a single device and updates
// the room's heat input when a command matching this actuator's measurement
// type arrives. A watchdog ticker clears the contribution if no matching
// command arrives within the watchdog timeout, treating device-service as absent.
func runActuator(ctx context.Context, dev provisioning.ProvisionedDevice, act config.ActuatorConfig, state *RoomState, mqttClient *mqtt.Client, watchdogTimeout time.Duration) {
	lastCmd := time.Now()
	var lastCmdMu sync.Mutex

	cmdTopic := fmt.Sprintf("devices/%s/cmd", dev.HwID)
	if err := mqttClient.Subscribe(cmdTopic, 2, func(_ string, payload []byte) {
		var msg struct {
			ActuatorType string `json:"actuator_type"`
			State        bool   `json:"state"`
		}
		if err := json.Unmarshal(payload, &msg); err != nil {
			log.Printf("[%s] malformed command payload: %v", dev.HwID, err)
			return
		}

		// each actuator goroutine filters for its own measurement type
		if config.ActuatorNameToMeasurement[msg.ActuatorType] != act.Type {
			return
		}

		if msg.State {
			state.SetContribution(dev.HwID+"/"+act.Type, map[string]float64{act.Type: act.Rate})
		} else {
			state.ClearContribution(dev.HwID + "/" + act.Type)
		}

		lastCmdMu.Lock()
		lastCmd = time.Now()
		lastCmdMu.Unlock()
	}); err != nil {
		log.Printf("[%s] failed to subscribe to cmd topic: %v", dev.HwID, err)
		return
	}

	watchdog := time.NewTicker(watchdogTimeout)
	defer watchdog.Stop()

	for {
		select {
		case <-watchdog.C:
			lastCmdMu.Lock()
			elapsed := time.Since(lastCmd)
			lastCmdMu.Unlock()
			if elapsed >= watchdogTimeout {
				log.Printf("[%s/%s] watchdog: no command in %s, clearing contribution",
					dev.HwID, act.Type, elapsed.Round(time.Second))
				state.ClearContribution(dev.HwID + "/" + act.Type)
			}
		case <-ctx.Done():
			return
		}
	}
}

// ---------------------------------------------------------------------------
// Room model advance
// ---------------------------------------------------------------------------

// advanceRoom advances the room environment model by one tick, applying the
// returned deltas to Current and clamping to measurement bounds.
func advanceRoom(state *RoomState, model RoomModel, simulatedTickSeconds float64) {
	heatInput := state.HeatInput()

	state.Mu.Lock()
	defer state.Mu.Unlock()

	deltas := model.Advance(state, heatInput, simulatedTickSeconds)
	for typ, delta := range deltas {
		bounds, ok := config.MeasurementBounds[typ]
		if !ok {
			state.Current[typ] += delta
			continue
		}
		v := state.Current[typ] + delta
		if v < bounds[0] {
			v = bounds[0]
		} else if v > bounds[1] {
			v = bounds[1]
		}
		state.Current[typ] = v
	}
}

// ---------------------------------------------------------------------------
// Telemetry publish
// ---------------------------------------------------------------------------

// publishTelemetry reads Current values for each sensor type on the device,
// applies per-device noise and offset, and publishes to the telemetry topic.
func publishTelemetry(dev provisioning.ProvisionedDevice, state *RoomState, mqttClient *mqtt.Client) error {
	readings := make([]reading, 0, len(dev.Config.Sensors))

	state.Mu.RLock()
	for _, sensor := range dev.Config.Sensors {
		base, ok := state.Current[sensor.Type]
		if !ok {
			continue
		}
		value := base + rand.NormFloat64()*sensor.Noise + sensor.Offset
		readings = append(readings, reading{
			Type:  sensor.Type,
			Value: roundTo2DP(value),
		})
	}
	state.Mu.RUnlock()

	if len(readings) == 0 {
		return nil
	}

	payload, err := json.Marshal(telemetryPayload{
		HwID:     dev.HwID,
		Readings: readings,
	})
	if err != nil {
		return fmt.Errorf("marshal telemetry: %w", err)
	}

	topic := fmt.Sprintf("devices/%s/telemetry", dev.HwID)
	return mqttClient.Publish(topic, 1, payload)
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func roundTo2DP(v float64) float64 {
	return float64(int(v*100+0.5)) / 100
}
