// Package metricsdb provides write-only access to the TimescaleDB instance.
// Read access is handled exclusively by api-service.
package metricsdb

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// SensorReading is a single row to be written to the sensor_readings hypertable.
// RoomID is snapshotted from the device cache at write time — null if the device
// is unassigned. It is never updated retroactively after device reassignment.
// RawValue holds the pre-offset value; Value holds the calibration-adjusted value.
// Both are identical until sensor calibration offsets are implemented.
type SensorReading struct {
	Time     time.Time
	SensorID uuid.UUID
	RoomID   *uuid.UUID
	Value    float64
	RawValue float64
}

// ControlLogEntry is a single row to be written to the room_control_logs hypertable.
// Written once per control loop tick per room. Captures the effective state used
// for evaluation and the commands issued as a result.
//
// HeaterCmd and HumidifierCmd are nullable — null indicates the room has no device
// with that actuator type. 0 = off, 1 = on.
//
// DeadbandTemp and DeadbandHum are null when no target is set for the corresponding
// type — they are only meaningful alongside a target and are always populated as a
// pair with TargetTemp and TargetHum respectively.
//
// ReadingCountTemp and ReadingCountHum are the number of fresh readings that
// contributed to AvgTemp and AvgHum respectively. Null if no readings were available.
//
// SchedulePeriodID is set only when ControlSource is "schedule" or "grace_period".
type ControlLogEntry struct {
	Time             time.Time
	RoomID           uuid.UUID
	AvgTemp          *float64
	AvgHum           *float64
	Mode             string
	TargetTemp       *float64
	TargetHum        *float64
	ControlSource    string
	HeaterCmd        *int16
	HumidifierCmd    *int16
	DeadbandTemp     *float64
	DeadbandHum      *float64
	ReadingCountTemp *int16
	ReadingCountHum  *int16
	SchedulePeriodID *uuid.UUID
}

// Repository handles write-only access to the metricsdb TimescaleDB instance.
type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// WriteSensorReadings inserts all readings from a single telemetry message in
// one pgx batch — one round trip regardless of reading count.
func (r *Repository) WriteSensorReadings(ctx context.Context, readings []SensorReading) error {
	if len(readings) == 0 {
		return nil
	}

	batch := &pgx.Batch{}
	for _, reading := range readings {
		batch.Queue(
			`INSERT INTO sensor_readings (time, sensor_id, room_id, value, raw_value)
			 VALUES ($1, $2, $3, $4, $5)`,
			reading.Time,
			reading.SensorID,
			reading.RoomID,
			reading.Value,
			reading.RawValue,
		)
	}

	results := r.pool.SendBatch(ctx, batch)
	defer results.Close()

	for i := range readings {
		if _, err := results.Exec(); err != nil {
			return fmt.Errorf("sensor_readings insert [%d]: %w", i, err)
		}
	}

	return nil
}

// WriteControlLogEntry inserts a single control loop tick record into
// room_control_logs. Called once per room per tick by the control loop.
func (r *Repository) WriteControlLogEntry(ctx context.Context, entry ControlLogEntry) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO room_control_logs (
			time, room_id, avg_temp, avg_hum, mode, target_temp, target_hum,
			control_source, heater_cmd, humidifier_cmd,
			deadband_temp, deadband_hum,
			reading_count_temp, reading_count_hum, schedule_period_id
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)`,
		entry.Time,
		entry.RoomID,
		entry.AvgTemp,
		entry.AvgHum,
		entry.Mode,
		entry.TargetTemp,
		entry.TargetHum,
		entry.ControlSource,
		entry.HeaterCmd,
		entry.HumidifierCmd,
		entry.DeadbandTemp,
		entry.DeadbandHum,
		entry.ReadingCountTemp,
		entry.ReadingCountHum,
		entry.SchedulePeriodID,
	)
	if err != nil {
		return fmt.Errorf("room_control_logs insert: %w", err)
	}

	return nil
}
