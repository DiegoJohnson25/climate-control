// Package appdb provides cache warm and reload access to the application
// PostgreSQL database for device-service. All queries use GORM's Raw+Scan
// pattern into unexported scan structs — the cache types are the public
// contract; scan types are never exposed outside this package.
package appdb

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/DiegoJohnson25/climate-control/device-service/internal/cache"
	"github.com/google/uuid"
	"github.com/lib/pq"
	"gorm.io/gorm"
)

// ---------------------------------------------------------------------------
// Scan types
// ---------------------------------------------------------------------------

type roomRow struct {
	ID           uuid.UUID
	UserID       uuid.UUID
	DeadbandTemp float64
	DeadbandHum  float64
	Timezone     string
}

type desiredStateRow struct {
	RoomID              uuid.UUID
	Mode                string
	TargetTemp          *float64
	TargetHum           *float64
	ManualOverrideUntil *time.Time
}

type activePeriodRow struct {
	ID         uuid.UUID
	RoomID     uuid.UUID
	DaysOfWeek pq.Int64Array `gorm:"type:integer[]"`
	StartTime  string
	EndTime    string
	Mode       string
	TargetTemp *float64
	TargetHum  *float64
}

type deviceRow struct {
	ID     uuid.UUID
	HwID   string
	RoomID *uuid.UUID
}

type sensorRow struct {
	ID              uuid.UUID
	DeviceID        uuid.UUID
	MeasurementType string
}

type actuatorRow struct {
	ID           uuid.UUID
	DeviceID     uuid.UUID
	ActuatorType string
}

// actuatorSensorTypes maps actuator types to the sensor type they depend on.
// Used when building ActuatorSensorMap per room.
var actuatorSensorTypes = map[string]string{
	"heater":     "temperature",
	"humidifier": "humidity",
}

// ---------------------------------------------------------------------------
// Repository
// ---------------------------------------------------------------------------

// Repository provides cache warm and reload operations against appdb.
type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

// WarmCache loads all owned rooms and their associated data from appdb and
// populates the store. Called once at startup before any goroutines start.
func (r *Repository) WarmCache(ctx context.Context, store *cache.Store) error {
	var allRoomIDs []uuid.UUID
	if err := r.db.WithContext(ctx).Raw(`SELECT id FROM rooms`).Scan(&allRoomIDs).Error; err != nil {
		return fmt.Errorf("fetch room ids: %w", err)
	}

	// filter to owned rooms (Phase 3: all rooms, Phase 5: hash-filtered)
	ownedIDs := make([]uuid.UUID, 0, len(allRoomIDs))
	for _, id := range allRoomIDs {
		if store.OwnsRoom(id) {
			ownedIDs = append(ownedIDs, id)
		}
	}
	if len(ownedIDs) == 0 {
		return nil
	}

	rooms, err := r.fetchRooms(ctx, ownedIDs)
	if err != nil {
		return fmt.Errorf("fetch rooms: %w", err)
	}

	desiredStates, err := r.fetchDesiredStates(ctx, ownedIDs)
	if err != nil {
		return fmt.Errorf("fetch desired states: %w", err)
	}

	periods, err := r.fetchActivePeriods(ctx, ownedIDs)
	if err != nil {
		return fmt.Errorf("fetch active periods: %w", err)
	}

	devices, err := r.fetchDevicesByRoom(ctx, ownedIDs)
	if err != nil {
		return fmt.Errorf("fetch devices: %w", err)
	}

	// collect device IDs for sensor/actuator bulk fetch
	allDeviceIDs := make([]uuid.UUID, 0, len(devices))
	for _, dev := range devices {
		allDeviceIDs = append(allDeviceIDs, dev.ID)
	}

	var sensors []sensorRow
	var actuators []actuatorRow
	if len(allDeviceIDs) > 0 {
		sensors, err = r.fetchSensors(ctx, allDeviceIDs)
		if err != nil {
			return fmt.Errorf("fetch sensors: %w", err)
		}
		actuators, err = r.fetchActuators(ctx, allDeviceIDs)
		if err != nil {
			return fmt.Errorf("fetch actuators: %w", err)
		}
	}

	dsMap := make(map[uuid.UUID]desiredStateRow, len(desiredStates))
	for _, ds := range desiredStates {
		dsMap[ds.RoomID] = ds
	}

	periodsMap := make(map[uuid.UUID][]activePeriodRow)
	for _, p := range periods {
		periodsMap[p.RoomID] = append(periodsMap[p.RoomID], p)
	}

	devicesByRoom := make(map[uuid.UUID][]deviceRow)
	for _, dev := range devices {
		if dev.RoomID != nil {
			devicesByRoom[*dev.RoomID] = append(devicesByRoom[*dev.RoomID], dev)
		}
	}

	sensorsByDevice := make(map[uuid.UUID][]sensorRow)
	for _, s := range sensors {
		sensorsByDevice[s.DeviceID] = append(sensorsByDevice[s.DeviceID], s)
	}

	actuatorsByDevice := make(map[uuid.UUID][]actuatorRow)
	for _, a := range actuators {
		actuatorsByDevice[a.DeviceID] = append(actuatorsByDevice[a.DeviceID], a)
	}

	for _, rm := range rooms {
		rc, err := buildRoomCache(
			rm,
			dsMap[rm.ID],
			periodsMap[rm.ID],
			devicesByRoom[rm.ID],
			actuatorsByDevice,
		)
		if err != nil {
			return fmt.Errorf("build room cache for %s: %w", rm.ID, err)
		}
		store.AddRoom(rm.ID, rc)
	}

	for _, dev := range devices {
		dc := buildDeviceCache(dev, sensorsByDevice[dev.ID], actuatorsByDevice[dev.ID])
		store.AddDevice(dev.HwID, dc)
	}

	log.Printf("appdb: cache warm complete — rooms: %d  devices: %d", len(rooms), len(devices))
	return nil
}

// ReloadRoom refreshes the cache entry for a single room. Called by the stream
// consumer on room-scoped events and by the periodic refresh ticker.
// Preserves runtime-only fields from the existing cache entry.
func (r *Repository) ReloadRoom(ctx context.Context, store *cache.Store, roomID uuid.UUID) error {
	rm, err := r.fetchRoom(ctx, roomID)
	if err != nil {
		return fmt.Errorf("fetch room %s: %w", roomID, err)
	}
	if rm == nil {
		store.DeleteRoom(roomID)
		return nil
	}

	ds, err := r.fetchDesiredState(ctx, roomID)
	if err != nil {
		return fmt.Errorf("fetch desired state for room %s: %w", roomID, err)
	}

	periods, err := r.fetchActivePeriodsForRoom(ctx, roomID)
	if err != nil {
		return fmt.Errorf("fetch active periods for room %s: %w", roomID, err)
	}

	devices, err := r.fetchDevicesForRoom(ctx, roomID)
	if err != nil {
		return fmt.Errorf("fetch devices for room %s: %w", roomID, err)
	}

	deviceIDs := make([]uuid.UUID, 0, len(devices))
	for _, dev := range devices {
		deviceIDs = append(deviceIDs, dev.ID)
	}

	actuatorsByDevice := make(map[uuid.UUID][]actuatorRow)
	if len(deviceIDs) > 0 {
		acts, err := r.fetchActuators(ctx, deviceIDs)
		if err != nil {
			return fmt.Errorf("fetch actuators for room %s: %w", roomID, err)
		}
		for _, a := range acts {
			actuatorsByDevice[a.DeviceID] = append(actuatorsByDevice[a.DeviceID], a)
		}
	}

	rc, err := buildRoomCache(
		*rm,
		ds,
		periods,
		devices,
		actuatorsByDevice,
	)
	if err != nil {
		return fmt.Errorf("build room cache for %s: %w", roomID, err)
	}

	// preserve runtime-only fields from existing cache entry
	existing := store.Room(roomID)
	if existing != nil {
		existing.Mu.RLock()
		rc.ActuatorStates = existing.ActuatorStates
		rc.LatestReadings = existing.LatestReadings
		rc.LastActivePeriod = existing.LastActivePeriod
		existing.Mu.RUnlock()
	}

	store.AddRoom(roomID, rc)
	return nil
}

// ReloadDevice refreshes the cache entry for a single device. Called by the
// stream consumer on device_changed events.
func (r *Repository) ReloadDevice(ctx context.Context, store *cache.Store, hwID string) error {
	dev, err := r.fetchDeviceByHwID(ctx, hwID)
	if err != nil {
		return fmt.Errorf("fetch device %s: %w", hwID, err)
	}
	if dev == nil {
		store.DeleteDevice(hwID)
		return nil
	}

	sensors, err := r.fetchSensors(ctx, []uuid.UUID{dev.ID})
	if err != nil {
		return fmt.Errorf("fetch sensors for device %s: %w", hwID, err)
	}

	actuators, err := r.fetchActuators(ctx, []uuid.UUID{dev.ID})
	if err != nil {
		return fmt.Errorf("fetch actuators for device %s: %w", hwID, err)
	}

	dc := buildDeviceCache(*dev, sensors, actuators)
	store.AddDevice(hwID, dc)
	return nil
}

// ---------------------------------------------------------------------------
// Query helpers
// ---------------------------------------------------------------------------

func (r *Repository) fetchRooms(ctx context.Context, roomIDs []uuid.UUID) ([]roomRow, error) {
	var rows []roomRow
	err := r.db.WithContext(ctx).Raw(`
		SELECT r.id, r.user_id, r.deadband_temp, r.deadband_hum, u.timezone
		FROM rooms r
		JOIN users u ON u.id = r.user_id
		WHERE r.id IN ?
	`, roomIDs).Scan(&rows).Error
	return rows, err
}

func (r *Repository) fetchDesiredStates(ctx context.Context, roomIDs []uuid.UUID) ([]desiredStateRow, error) {
	var rows []desiredStateRow
	err := r.db.WithContext(ctx).Raw(`
		SELECT room_id, mode, target_temp, target_hum, manual_override_until
		FROM desired_states
		WHERE room_id IN ?
	`, roomIDs).Scan(&rows).Error
	return rows, err
}

func (r *Repository) fetchActivePeriods(ctx context.Context, roomIDs []uuid.UUID) ([]activePeriodRow, error) {
	var rows []activePeriodRow
	err := r.db.WithContext(ctx).Raw(`
		SELECT sp.id, s.room_id, sp.days_of_week, sp.start_time, sp.end_time,
		       sp.mode, sp.target_temp, sp.target_hum
		FROM schedule_periods sp
		JOIN schedules s ON s.id = sp.schedule_id
		WHERE s.room_id IN ? AND s.is_active = true
	`, roomIDs).Scan(&rows).Error
	return rows, err
}

func (r *Repository) fetchDevicesByRoom(ctx context.Context, roomIDs []uuid.UUID) ([]deviceRow, error) {
	var rows []deviceRow
	err := r.db.WithContext(ctx).Raw(`
		SELECT id, hw_id, room_id
		FROM devices
		WHERE room_id IN ?
	`, roomIDs).Scan(&rows).Error
	return rows, err
}

func (r *Repository) fetchSensors(ctx context.Context, deviceIDs []uuid.UUID) ([]sensorRow, error) {
	var rows []sensorRow
	err := r.db.WithContext(ctx).Raw(`
		SELECT id, device_id, measurement_type
		FROM sensors
		WHERE device_id IN ?
	`, deviceIDs).Scan(&rows).Error
	return rows, err
}

func (r *Repository) fetchActuators(ctx context.Context, deviceIDs []uuid.UUID) ([]actuatorRow, error) {
	var rows []actuatorRow
	err := r.db.WithContext(ctx).Raw(`
		SELECT id, device_id, actuator_type
		FROM actuators
		WHERE device_id IN ?
	`, deviceIDs).Scan(&rows).Error
	return rows, err
}

func (r *Repository) fetchRoom(ctx context.Context, roomID uuid.UUID) (*roomRow, error) {
	var row roomRow
	err := r.db.WithContext(ctx).Raw(`
		SELECT r.id, r.user_id, r.deadband_temp, r.deadband_hum, u.timezone
		FROM rooms r
		JOIN users u ON u.id = r.user_id
		WHERE r.id = ?
	`, roomID).Scan(&row).Error
	if err != nil {
		return nil, err
	}
	if row.ID == uuid.Nil {
		return nil, nil
	}
	return &row, nil
}

func (r *Repository) fetchDesiredState(ctx context.Context, roomID uuid.UUID) (desiredStateRow, error) {
	var row desiredStateRow
	err := r.db.WithContext(ctx).Raw(`
		SELECT room_id, mode, target_temp, target_hum, manual_override_until
		FROM desired_states
		WHERE room_id = ?
	`, roomID).Scan(&row).Error
	return row, err
}

func (r *Repository) fetchActivePeriodsForRoom(ctx context.Context, roomID uuid.UUID) ([]activePeriodRow, error) {
	var rows []activePeriodRow
	err := r.db.WithContext(ctx).Raw(`
		SELECT sp.id, s.room_id, sp.days_of_week, sp.start_time, sp.end_time,
		       sp.mode, sp.target_temp, sp.target_hum
		FROM schedule_periods sp
		JOIN schedules s ON s.id = sp.schedule_id
		WHERE s.room_id = ? AND s.is_active = true
	`, roomID).Scan(&rows).Error
	return rows, err
}

func (r *Repository) fetchDevicesForRoom(ctx context.Context, roomID uuid.UUID) ([]deviceRow, error) {
	var rows []deviceRow
	err := r.db.WithContext(ctx).Raw(`
		SELECT id, hw_id, room_id
		FROM devices
		WHERE room_id = ?
	`, roomID).Scan(&rows).Error
	return rows, err
}

func (r *Repository) fetchDeviceByHwID(ctx context.Context, hwID string) (*deviceRow, error) {
	var row deviceRow
	err := r.db.WithContext(ctx).Raw(`
		SELECT id, hw_id, room_id
		FROM devices
		WHERE hw_id = ?
	`, hwID).Scan(&row).Error
	if err != nil {
		return nil, err
	}
	if row.ID == uuid.Nil {
		return nil, nil
	}
	return &row, nil
}

// ---------------------------------------------------------------------------
// Cache assembly
// ---------------------------------------------------------------------------

// buildRoomCache assembles a RoomCache from raw DB rows, applying all
// pre-computations so the control loop hot path does no redundant work.
func buildRoomCache(
	rm roomRow,
	ds desiredStateRow,
	periods []activePeriodRow,
	devices []deviceRow,
	actuatorsByDevice map[uuid.UUID][]actuatorRow,
) (*cache.RoomCache, error) {
	loc, err := time.LoadLocation(rm.Timezone)
	if err != nil {
		return nil, fmt.Errorf("resolve timezone %q: %w", rm.Timezone, err)
	}

	periodCaches, err := buildPeriodCaches(periods)
	if err != nil {
		return nil, err
	}

	actuatorHwIDs := buildActuatorHwIDs(devices, actuatorsByDevice)

	return &cache.RoomCache{
		RoomID:       rm.ID,
		UserTimezone: rm.Timezone,
		Location:     loc,
		DeadbandTemp: rm.DeadbandTemp,
		DeadbandHum:  rm.DeadbandHum,
		DesiredState: cache.DesiredStateCache{
			Mode: ds.Mode,
			Targets: map[string]*float64{
				"temperature": ds.TargetTemp,
				"humidity":    ds.TargetHum,
			},
			ManualOverrideUntil: ds.ManualOverrideUntil,
		},
		ActivePeriods:  periodCaches,
		ActuatorHwIDs:  actuatorHwIDs,
		ActuatorStates: make(map[string]bool),
		LatestReadings: make(map[string][]cache.TimestampedReading),
	}, nil
}

// buildPeriodCaches converts raw period rows into SchedulePeriodCache structs
// with pre-computed DaysOfWeek bitmask and start/end minutes.
func buildPeriodCaches(periods []activePeriodRow) ([]cache.SchedulePeriodCache, error) {
	result := make([]cache.SchedulePeriodCache, 0, len(periods))
	for _, p := range periods {
		startMinutes, err := parseTimeToMinutes(p.StartTime)
		if err != nil {
			return nil, fmt.Errorf("parse start_time %q for period %s: %w", p.StartTime, p.ID, err)
		}
		endMinutes, err := parseTimeToMinutes(p.EndTime)
		if err != nil {
			return nil, fmt.Errorf("parse end_time %q for period %s: %w", p.EndTime, p.ID, err)
		}

		var dow [8]bool
		for _, d := range p.DaysOfWeek {
			if d >= 1 && d <= 7 {
				dow[int(d)] = true
			}
		}

		result = append(result, cache.SchedulePeriodCache{
			ID:           p.ID,
			DaysOfWeek:   dow,
			StartMinutes: startMinutes,
			EndMinutes:   endMinutes,
			Mode:         p.Mode,
			Targets: map[string]*float64{
				"temperature": p.TargetTemp,
				"humidity":    p.TargetHum,
			},
		})
	}
	return result, nil
}

// buildActuatorHwIDs builds a map of actuator_type → []hw_id for a room
// based on which actuator types its devices actually have.
func buildActuatorHwIDs(devices []deviceRow, actuatorsByDevice map[uuid.UUID][]actuatorRow) map[string][]string {
	result := make(map[string][]string)
	for _, dev := range devices {
		for _, act := range actuatorsByDevice[dev.ID] {
			if _, ok := actuatorSensorTypes[act.ActuatorType]; ok {
				result[act.ActuatorType] = append(result[act.ActuatorType], dev.HwID)
			}
		}
	}
	return result
}

// buildDeviceCache assembles a DeviceCache from raw DB rows.
// Sensors and Actuators are stored as maps keyed by type for O(1) lookup
// at ingestion and control loop evaluation time.
func buildDeviceCache(dev deviceRow, sensors []sensorRow, actuators []actuatorRow) *cache.DeviceCache {
	sensorEntries := make(map[string]cache.SensorEntry, len(sensors))
	for _, s := range sensors {
		sensorEntries[s.MeasurementType] = cache.SensorEntry{
			SensorID:        s.ID,
			MeasurementType: s.MeasurementType,
		}
	}

	actuatorEntries := make(map[string]cache.ActuatorEntry, len(actuators))
	for _, a := range actuators {
		actuatorEntries[a.ActuatorType] = cache.ActuatorEntry{
			ActuatorID:   a.ID,
			ActuatorType: a.ActuatorType,
		}
	}

	return &cache.DeviceCache{
		DeviceID:  dev.ID,
		HwID:      dev.HwID,
		RoomID:    dev.RoomID,
		Sensors:   sensorEntries,
		Actuators: actuatorEntries,
	}
}

// parseTimeToMinutes parses a "HH:MM" string into minutes since midnight.
func parseTimeToMinutes(s string) (int, error) {
	parts := strings.SplitN(s, ":", 2)
	if len(parts) != 2 {
		return 0, fmt.Errorf("invalid time format %q, expected HH:MM", s)
	}
	hour, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, fmt.Errorf("invalid hour in %q: %w", s, err)
	}
	minute, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, fmt.Errorf("invalid minute in %q: %w", s, err)
	}
	return hour*60 + minute, nil
}
