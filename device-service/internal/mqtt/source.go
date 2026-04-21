package mqtt

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/DiegoJohnson25/climate-control/device-service/internal/ingestion"
)

// telemetryPayload mirrors the JSON structure published by devices on
// devices/{hw_id}/telemetry. hw_id is read from the payload — the topic
// segment is used only for broker routing and is not authoritative.
type telemetryPayload struct {
	HwID     string           `json:"hw_id"`
	Readings []readingPayload `json:"readings"`
}

type readingPayload struct {
	Type  string  `json:"type"`
	Value float64 `json:"value"`
}

// Source implements ingestion.Source for MQTT transport. It subscribes to
// Mosquitto and delivers parsed TelemetryMessages to the ingestion handler.
// Active in Phase 3–6e; replaced by kafka.Source in Phase 7b.
//
// TODO Phase 7b: remove this file. Replace mqtt.NewSource in main.go with
// kafka.NewSource — ingestion.Ingestor and ingestion.Process are unchanged.
type Source struct {
	client *Client
}

func NewSource(client *Client) *Source {
	return &Source{client: client}
}

// Start subscribes to devices/+/telemetry and begins delivering messages to
// handler. Returns promptly — Paho delivers messages via its own goroutines.
// Returns an error if the subscription cannot be established.
func (s *Source) Start(ctx context.Context, handler func(context.Context, ingestion.TelemetryMessage)) error {
	return s.client.Subscribe("devices/+/telemetry", 1, func(_ string, payload []byte) {
		var p telemetryPayload
		if err := json.Unmarshal(payload, &p); err != nil {
			log.Printf("mqtt source: failed to parse telemetry payload: %v", err)
			return
		}

		if p.HwID == "" {
			log.Printf("mqtt source: telemetry payload missing hw_id")
			return
		}

		readings := make([]ingestion.Reading, 0, len(p.Readings))
		for _, r := range p.Readings {
			readings = append(readings, ingestion.Reading{
				Type:  r.Type,
				Value: r.Value,
			})
		}

		msg := ingestion.TelemetryMessage{
			HwID:      p.HwID,
			Readings:  readings,
			Timestamp: time.Now().UTC(),
		}

		handler(ctx, msg)
	})
}

// Stop disconnects the MQTT client cleanly.
func (s *Source) Stop() {
	s.client.Disconnect()
}
