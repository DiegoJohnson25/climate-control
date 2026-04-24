// Package metricsdb provides read-only access to the TimescaleDB instance for api-service.
// Write access is handled exclusively by device-service.
package metricsdb

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DefaultDensity is the target number of data points returned by ClimateHistory
// when no density override is specified. Pass 0 to ClimateHistory to use it.
const DefaultDensity = 120

// ---------------------------------------------------------------------------
// Response types
// ---------------------------------------------------------------------------

// ClimateReading is the current climate snapshot for a room — sourced from the
// most recent room_control_logs row. HeaterCmd and HumidifierCmd are null when
// the room has no device with that actuator type, not when the actuator is off.
type ClimateReading struct {
	Time          time.Time `json:"time"`
	AvgTemp       *float64  `json:"avg_temp"`
	AvgHum        *float64  `json:"avg_hum"`
	Mode          *string   `json:"mode"`
	TargetTemp    *float64  `json:"target_temp"`
	TargetHum     *float64  `json:"target_hum"`
	ControlSource *string   `json:"control_source"`
	HeaterCmd     *bool     `json:"heater_cmd"`
	HumidifierCmd *bool     `json:"humidifier_cmd"`
	DeadbandTemp  *float64  `json:"deadband_temp"`
	DeadbandHum   *float64  `json:"deadband_hum"`
}

// ClimateHistoryPoint is one time-bucketed row from the history query.
// HeaterDuty and HumidifierDuty are the fraction of control-loop ticks within
// the bucket that the actuator was commanded on (AVG of SMALLINT 0/1), giving
// a 0.0–1.0 duty cycle. Null when the room has no device of that actuator type.
// DeadbandTemp and DeadbandHum are null when no target is set for that type.
type ClimateHistoryPoint struct {
	Time           time.Time `json:"time"`
	AvgTemp        *float64  `json:"avg_temp"`
	AvgHum         *float64  `json:"avg_hum"`
	HeaterDuty     *float64  `json:"heater_duty"`
	HumidifierDuty *float64  `json:"humidifier_duty"`
	TargetTemp     *float64  `json:"target_temp"`
	TargetHum      *float64  `json:"target_hum"`
	DeadbandTemp   *float64  `json:"deadband_temp"`
	DeadbandHum    *float64  `json:"deadband_hum"`
}

// ClimateHistoryResult is returned by ClimateHistory.
// BucketSeconds is the aggregation interval — clients may use it for chart tick formatting.
type ClimateHistoryResult struct {
	BucketSeconds int
	Points        []ClimateHistoryPoint
}

// ---------------------------------------------------------------------------
// Scan types
// ---------------------------------------------------------------------------

// climateReadingScan mirrors the LatestClimate SELECT list. HeaterCmd and
// HumidifierCmd are scanned as *int16 (SMALLINT) and converted to *bool on the
// way out — the DB stores 0/1, the API surface exposes false/true.
type climateReadingScan struct {
	Time          time.Time
	AvgTemp       *float64
	AvgHum        *float64
	Mode          *string
	TargetTemp    *float64
	TargetHum     *float64
	ControlSource *string
	HeaterCmd     *int16
	HumidifierCmd *int16
	DeadbandTemp  *float64
	DeadbandHum   *float64
}

// ---------------------------------------------------------------------------
// Repository
// ---------------------------------------------------------------------------

// Repository provides read-only access to the metricsdb TimescaleDB instance.
type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// LatestClimate returns the most recent room_control_logs row for roomID.
// Returns nil, nil if no rows exist — the room is valid but has no data yet.
func (r *Repository) LatestClimate(ctx context.Context, roomID uuid.UUID) (*ClimateReading, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT time,
		        avg_temp::float8, avg_hum::float8,
		        mode,
		        target_temp::float8, target_hum::float8,
		        control_source,
		        heater_cmd, humidifier_cmd,
		        deadband_temp::float8, deadband_hum::float8
		 FROM room_control_logs
		 WHERE room_id = $1
		 ORDER BY time DESC
		 LIMIT 1`,
		roomID,
	)

	var s climateReadingScan
	if err := row.Scan(
		&s.Time, &s.AvgTemp, &s.AvgHum,
		&s.Mode,
		&s.TargetTemp, &s.TargetHum,
		&s.ControlSource,
		&s.HeaterCmd, &s.HumidifierCmd,
		&s.DeadbandTemp, &s.DeadbandHum,
	); err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("room_control_logs latest: %w", err)
	}

	reading := &ClimateReading{
		Time:          s.Time,
		AvgTemp:       s.AvgTemp,
		AvgHum:        s.AvgHum,
		Mode:          s.Mode,
		TargetTemp:    s.TargetTemp,
		TargetHum:     s.TargetHum,
		ControlSource: s.ControlSource,
		DeadbandTemp:  s.DeadbandTemp,
		DeadbandHum:   s.DeadbandHum,
	}
	if s.HeaterCmd != nil {
		v := *s.HeaterCmd == 1
		reading.HeaterCmd = &v
	}
	if s.HumidifierCmd != nil {
		v := *s.HumidifierCmd == 1
		reading.HumidifierCmd = &v
	}

	return reading, nil
}

// ClimateHistory returns time-bucketed averages from room_control_logs for a
// given room and window. Window values: "1h", "6h", "24h", "7d". Defaults to
// "24h" when window is empty. Density is the target number of data points;
// pass 0 to use DefaultDensity. BucketSeconds in the result is rounded to the
// nearest value on a fixed ladder.
func (r *Repository) ClimateHistory(ctx context.Context, roomID uuid.UUID, window string, density int) (ClimateHistoryResult, error) {
	if window == "" {
		window = "24h"
	}
	if density <= 0 {
		density = DefaultDensity
	}

	windowSecs, bucketSecs := windowParams(window, density)
	bucketInterval := fmt.Sprintf("%d seconds", bucketSecs)
	windowInterval := fmt.Sprintf("%d seconds", windowSecs)

	rows, err := r.pool.Query(ctx,
		`SELECT
		    time_bucket($1::interval, time)  AS bucket,
		    AVG(avg_temp)::float8            AS avg_temp,
		    AVG(avg_hum)::float8             AS avg_hum,
		    AVG(heater_cmd)::float8          AS heater_duty,
		    AVG(humidifier_cmd)::float8      AS humidifier_duty,
		    AVG(target_temp)::float8         AS target_temp,
		    AVG(target_hum)::float8          AS target_hum,
		    AVG(deadband_temp)::float8       AS deadband_temp,
		    AVG(deadband_hum)::float8        AS deadband_hum
		FROM room_control_logs
		WHERE room_id = $2
		  AND time >= NOW() - $3::interval
		GROUP BY bucket
		ORDER BY bucket ASC`,
		bucketInterval,
		roomID,
		windowInterval,
	)
	if err != nil {
		return ClimateHistoryResult{}, fmt.Errorf("room_control_logs history: %w", err)
	}
	defer rows.Close()

	points := make([]ClimateHistoryPoint, 0)
	for rows.Next() {
		var p ClimateHistoryPoint
		if err := rows.Scan(
			&p.Time, &p.AvgTemp, &p.AvgHum,
			&p.HeaterDuty, &p.HumidifierDuty,
			&p.TargetTemp, &p.TargetHum,
			&p.DeadbandTemp, &p.DeadbandHum,
		); err != nil {
			return ClimateHistoryResult{}, fmt.Errorf("room_control_logs history scan: %w", err)
		}
		points = append(points, p)
	}
	if err := rows.Err(); err != nil {
		return ClimateHistoryResult{}, fmt.Errorf("room_control_logs history rows: %w", err)
	}

	return ClimateHistoryResult{BucketSeconds: bucketSecs, Points: points}, nil
}

// ---------------------------------------------------------------------------
// Query helpers
// ---------------------------------------------------------------------------

var bucketLadder = []int{30, 60, 120, 300, 600, 900, 1800, 3600, 7200, 10800, 21600}

// nearestBucket returns the value from bucketLadder closest to n.
func nearestBucket(n int) int {
	best := bucketLadder[0]
	for _, v := range bucketLadder[1:] {
		if abs(v-n) < abs(best-n) {
			best = v
		}
	}
	return best
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

// windowParams returns the window duration and bucket size in seconds.
// Bucket size targets density data points per window, rounded to the nearest ladder value.
func windowParams(window string, density int) (windowSecs, bucketSecs int) {
	switch window {
	case "1h":
		windowSecs = 3600
	case "6h":
		windowSecs = 21600
	case "7d":
		windowSecs = 604800
	default: // "24h"
		windowSecs = 86400
	}
	bucketSecs = nearestBucket(windowSecs / density)
	return
}
