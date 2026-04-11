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

// -----------------------------------------------------------------------------
// Runtime state — separate from config, evolves per tick in Phase 4
// -----------------------------------------------------------------------------

type RoomState struct {
	Provisioned provisioning.ProvisionedRoom
	baseValues  map[string]float64 // sensor type → base value, built once at startup
	// Phase 4: CurrentTemp, CurrentHumidity float64
	// Phase 4: calculator RoomCalculator
}

func newRoomState(room provisioning.ProvisionedRoom) *RoomState {
	return &RoomState{
		Provisioned: room,
		baseValues: map[string]float64{
			"temperature": room.Config.Model.BaseTemp,
			"humidity":    room.Config.Model.BaseHumidity,
			// air_quality has no base value defined yet — defaults to 0
		},
	}
}

// -----------------------------------------------------------------------------
// MQTT payload types
// -----------------------------------------------------------------------------

type telemetryPayload struct {
	HwID     string    `json:"hw_id"`
	Readings []reading `json:"readings"`
}

type reading struct {
	Type  string  `json:"type"`
	Value float64 `json:"value"`
}

// -----------------------------------------------------------------------------
// Run — entry point, blocks until ctx is cancelled
// -----------------------------------------------------------------------------

func Run(ctx context.Context, cfg *config.Config, users []provisioning.ProvisionedUser, mqttClient *mqtt.Client) error {
	// collect all devices across all users and rooms for stagger calculation
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

// -----------------------------------------------------------------------------
// Device goroutine
// -----------------------------------------------------------------------------

func runDevice(ctx context.Context, dev provisioning.ProvisionedDevice, room *RoomState, mqttClient *mqtt.Client, tickInterval, offset time.Duration) {
	// subscribe to cmd topic — log received commands for now
	cmdTopic := fmt.Sprintf("devices/%s/cmd", dev.HwID)
	if err := mqttClient.Subscribe(cmdTopic, 2, func(topic string, payload []byte) {
		log.Printf("[%s] received command: %s", dev.HwID, string(payload))
	}); err != nil {
		log.Printf("[%s] failed to subscribe to cmd topic: %v", dev.HwID, err)
	}

	// only publish telemetry for devices with sensors
	if len(dev.Config.Sensors) == 0 {
		// actuator-only device — no telemetry to publish, just wait for context
		<-ctx.Done()
		return
	}

	// stagger offset — spread publish load across tick window
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

// -----------------------------------------------------------------------------
// Telemetry calculation and publish
// -----------------------------------------------------------------------------

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

// -----------------------------------------------------------------------------
// Helpers
// -----------------------------------------------------------------------------

func roundTo2DP(v float64) float64 {
	return float64(int(v*100+0.5)) / 100
}
