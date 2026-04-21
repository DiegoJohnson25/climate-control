package ingestion

import (
	"time"

	"github.com/google/uuid"
)

// TelemetryMessage is the normalised representation of a telemetry payload
// after transport-specific concerns (MQTT topic parsing, Kafka record
// deserialisation) have been resolved by the adapter layer.
//
// Phase 3: constructed by the MQTT adapter in mqtt/client.go after resolving
// hw_id → RoomID from the device cache.
// TODO Phase 7b: constructed identically by the Kafka consumer adapter —
// ingestion.Process is unchanged. The bridge stamps RoomID onto the Kafka
// message payload; the adapter reads it directly rather than resolving from cache.
type TelemetryMessage struct {
	HwID      string
	RoomID    *uuid.UUID // nil if device is unassigned
	Readings  []Reading
	Timestamp time.Time
}

// Reading is a single sensor observation within a TelemetryMessage.
// Type matches the measurement_type values in the sensors table:
// "temperature", "humidity", "air_quality".
type Reading struct {
	Type  string
	Value float64
}
