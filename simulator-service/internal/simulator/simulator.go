// Package simulator runs the per-device publish loop. Each device gets its own
// goroutine, staggered across the tick interval to spread load, and publishes
// a sensor-type keyed telemetry payload to devices/{hw_id}/telemetry. Room
// state is the evolving unit — Phase 4b will replace the static baseValues with
// a physics or drift model calculator.
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
// Runtime state
// ---------------------------------------------------------------------------

type RoomState struct {
	Provisioned provisioning.ProvisionedRoom
	baseValues  map[string]float64
	// TODO Phase 4b: CurrentTemp, CurrentHumidity, calculator RoomCalculator.
}

func newRoomState(room provisioning.ProvisionedRoom) *RoomState {
	return &RoomState{
		Provisioned: room,
		baseValues: map[string]float64{
			"temperature": room.Config.Model.BaseTemp,
			"humidity":    room.Config.Model.BaseHumidity,
		},
	}
}

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

// Run spawns one goroutine per provisioned device and blocks until ctx is
// cancelled. Device starts are staggered evenly across tickInterval so
// telemetry publishes are spread over the window rather than bursting at once.
func Run(ctx context.Context, cfg *config.Config, users []provisioning.ProvisionedUser, mqttClient *mqtt.Client) error {
	type deviceEntry struct {
		device provisioning.ProvisionedDevice
		room   *RoomState
	}

	var entries []deviceEntry
	var roomStates []*RoomState

	for _, user := range users {
		for _, room := range user.Rooms {
			state := newRoomState(room)
			roomStates = append(roomStates, state)
			for _, dev := range room.Devices {
				entries = append(entries, deviceEntry{device: dev, room: state})
			}
		}
	}

	totalDevices := len(entries)
	if totalDevices == 0 {
		log.Println("no devices to simulate")
		return nil
	}

	tickInterval := time.Duration(cfg.Simulation.TickIntervalSeconds) * time.Second

	var wg sync.WaitGroup

	for i, entry := range entries {
		wg.Add(1)

		staggerOffset := time.Duration(float64(i) / float64(totalDevices) * float64(tickInterval))

		go func(idx int, dev provisioning.ProvisionedDevice, room *RoomState, offset time.Duration) {
			defer wg.Done()
			runDevice(ctx, dev, room, mqttClient, tickInterval, offset)
		}(i, entry.device, entry.room, staggerOffset)
	}

	log.Printf("simulator running — %d device(s) across %d room(s)", totalDevices, len(roomStates))

	wg.Wait()
	return nil
}

// ---------------------------------------------------------------------------
// Device goroutine
// ---------------------------------------------------------------------------

func runDevice(ctx context.Context, dev provisioning.ProvisionedDevice, room *RoomState, mqttClient *mqtt.Client, tickInterval, offset time.Duration) {
	cmdTopic := fmt.Sprintf("devices/%s/cmd", dev.HwID)
	if err := mqttClient.Subscribe(cmdTopic, 2, func(topic string, payload []byte) {
		log.Printf("[%s] received command: %s", dev.HwID, string(payload))
	}); err != nil {
		log.Printf("[%s] failed to subscribe to cmd topic: %v", dev.HwID, err)
	}

	// Actuator-only devices have nothing to publish; block until shutdown.
	if len(dev.Config.Sensors) == 0 {
		<-ctx.Done()
		return
	}

	select {
	case <-time.After(offset):
	case <-ctx.Done():
		return
	}

	ticker := time.NewTicker(tickInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := publishTelemetry(dev, room, mqttClient); err != nil {
				log.Printf("[%s] publish error: %v", dev.HwID, err)
			}
		case <-ctx.Done():
			return
		}
	}
}

// ---------------------------------------------------------------------------
// Telemetry calculation and publish
// ---------------------------------------------------------------------------

func publishTelemetry(dev provisioning.ProvisionedDevice, room *RoomState, mqttClient *mqtt.Client) error {
	readings := make([]reading, 0, len(dev.Config.Sensors))

	for _, sensorType := range dev.Config.Sensors {
		base := room.baseValues[sensorType]
		noise := rand.NormFloat64() * dev.Config.Noise[sensorType]
		offset := dev.Config.Offset[sensorType]
		value := base + noise + offset

		readings = append(readings, reading{
			Type:  sensorType,
			Value: roundTo2DP(value),
		})
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
