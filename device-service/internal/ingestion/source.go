package ingestion

import "context"

// Source is the interface that transport implementations must satisfy to
// deliver telemetry messages to the ingestion layer.
//
// Implementations:
//   - mqtt.Source — subscribes to Mosquitto, active Phase 3–5
//   - kafka.Source — consumes from Kafka topic, Phase 6 onwards
//
// TODO Phase 6: replace mqtt.Source construction in main.go with kafka.Source.
// Ingestor, Process, and this interface are unchanged.
type Source interface {
	// Start begins delivering telemetry messages to handler. It must return
	// promptly — message delivery happens asynchronously via the transport's
	// own goroutines. Returns an error if the transport cannot be initialised
	// (e.g. broker unreachable, subscription failed).
	Start(ctx context.Context, handler func(context.Context, TelemetryMessage)) error

	// Stop performs a clean shutdown of the transport, allowing in-flight
	// messages to complete before closing the connection.
	Stop()
}
