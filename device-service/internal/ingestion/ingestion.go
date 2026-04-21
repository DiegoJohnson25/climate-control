// Package ingestion provides transport-agnostic telemetry processing for
// device-service. Transports implement Source to deliver messages; Process
// handles cache updates and TimescaleDB writes regardless of origin.
package ingestion

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/DiegoJohnson25/climate-control/device-service/internal/cache"
	"github.com/DiegoJohnson25/climate-control/device-service/internal/debug"
	"github.com/DiegoJohnson25/climate-control/device-service/internal/metricsdb"
)

// Ingestor processes telemetry messages — updates the in-memory store and
// writes sensor readings to TimescaleDB. It receives messages via a Source
// implementation and is completely transport-agnostic.
type Ingestor struct {
	source  Source
	store   *cache.Store
	metrics *metricsdb.Repository
	stale   time.Duration
}

func NewIngestor(source Source, store *cache.Store, metrics *metricsdb.Repository, stale time.Duration) *Ingestor {
	return &Ingestor{
		source:  source,
		store:   store,
		metrics: metrics,
		stale:   stale,
	}
}

// Run starts the telemetry source and begins processing messages. Returns
// promptly — message delivery and processing happen in the source's own
// goroutines. Returns a fatal error if the source fails to start (e.g. broker
// unreachable). Per-message errors from Process are logged and dropped
// internally and do not propagate to the caller.
func (n *Ingestor) Run(ctx context.Context) error {
	return n.source.Start(ctx, func(ctx context.Context, msg TelemetryMessage) {
		if err := n.Process(ctx, msg); err != nil {
			log.Printf("ingestion: process error for hw_id=%s: %v", msg.HwID, err)
		}
	})
}

// Stop performs a clean shutdown of the telemetry source.
func (n *Ingestor) Stop() {
	n.source.Stop()
}

// Process handles a single telemetry message. It updates LatestReadings in the
// store and writes sensor readings to TimescaleDB in one batch.
//
// Drop conditions (silent):
//   - hw_id not in device cache — unknown device
//   - device has no room assignment — unassigned devices have no instance owner
//
// Drop conditions (warning logged):
//   - room not owned by this instance — cache inconsistency, should not occur
//   - room not found in store — cache inconsistency, should not occur
//
// Per-reading skip (silent):
//   - no sensor entry matching the reading type — device/DB config mismatch
//
// TODO Phase 8: add Prometheus instrumentation callsites.
// telemetry_messages_total{outcome}, sensor_readings_written_total{type}
func (n *Ingestor) Process(ctx context.Context, msg TelemetryMessage) error {
	dc := n.store.Device(msg.HwID)
	if dc == nil {
		return nil
	}

	roomID := dc.GetRoomID()
	if roomID == nil {
		return nil
	}

	if !n.store.OwnsRoom(*roomID) {
		log.Printf("ingestion: unexpected message for unowned room %s (hw_id=%s)", *roomID, msg.HwID)
		return nil
	}

	rc := n.store.Room(*roomID)
	if rc == nil {
		log.Printf("ingestion: room %s not in store for hw_id=%s — cache inconsistency", *roomID, msg.HwID)
		return nil
	}

	type resolvedReading struct {
		entry   cache.SensorEntry
		reading Reading
	}

	resolved := make([]resolvedReading, 0, len(msg.Readings))
	for _, r := range msg.Readings {
		entry, ok := dc.Sensors[r.Type]
		if !ok {
			continue
		}
		resolved = append(resolved, resolvedReading{entry: entry, reading: r})
	}

	if len(resolved) == 0 {
		return nil
	}

	if debug.AtIngestion() {
		types := make([]string, len(resolved))
		for i, rr := range resolved {
			types[i] = rr.reading.Type
		}
		debug.LogIngestion(msg.HwID, *roomID, types)
	}

	rc.Mu.Lock()
	for _, rr := range resolved {
		rc.LatestReadings[rr.reading.Type] = append(
			rc.LatestReadings[rr.reading.Type],
			cache.TimestampedReading{
				Value:     rr.reading.Value,
				Timestamp: msg.Timestamp,
			},
		)
		rc.LatestReadings[rr.reading.Type] = trimStale(rc.LatestReadings[rr.reading.Type], msg.Timestamp, n.stale)
	}
	rc.Mu.Unlock()

	dbReadings := make([]metricsdb.SensorReading, 0, len(resolved))
	for _, rr := range resolved {
		dbReadings = append(dbReadings, metricsdb.SensorReading{
			SensorID: rr.entry.SensorID,
			RoomID:   roomID,
			Value:    rr.reading.Value,
			RawValue: rr.reading.Value, // TODO: apply sensor offset when calibration is implemented
			Time:     msg.Timestamp,
		})
	}

	if err := n.metrics.WriteSensorReadings(ctx, dbReadings); err != nil {
		return fmt.Errorf("ingestion: write sensor readings for hw_id=%s: %w", msg.HwID, err)
	}

	return nil
}

// trimStale scans from the front — readings are always appended in chronological
// order so all stale entries are at the head of the slice.
func trimStale(readings []cache.TimestampedReading, now time.Time, threshold time.Duration) []cache.TimestampedReading {
	cutoff := now.Add(-threshold)
	i := 0
	for i < len(readings) && readings[i].Timestamp.Before(cutoff) {
		i++
	}
	return readings[i:]
}
