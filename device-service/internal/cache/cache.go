// Package cache provides the in-memory store for device-service.
// Store.mu protects the rooms and devices maps; per-object fields are protected
// by each object's own mutex. RoomCache exposes its Mu for caller-managed
// locking across multi-field reads; DeviceCache uses an unexported mu and
// exposes GetRoomID and SetRoomID for safe access to its only mutable field.
package cache

import (
	"sync"
	"time"

	"github.com/google/uuid"
)

// TimestampedReading holds a sensor value and the time it was received.
// Stored as a slice per sensor type to support averaging across multiple
// sensors of the same type in a room.
type TimestampedReading struct {
	Value     float64
	Timestamp time.Time
}

// DesiredStateCache mirrors the relevant fields from the desired_states table.
// Targets is keyed by sensor type: "temperature", "humidity".
type DesiredStateCache struct {
	Mode                string
	Targets             map[string]*float64 // sensor_type → target value
	ManualOverrideUntil *time.Time
}

// SchedulePeriodCache mirrors the relevant fields from the schedule_periods
// table with hot-path pre-computations applied at warm/reload time.
// Targets is keyed by sensor type: "temperature", "humidity".
type SchedulePeriodCache struct {
	ID           uuid.UUID
	DaysOfWeek   [8]bool // index 1-7, Monday=1, Sunday=7
	StartMinutes int     // pre-parsed: hour*60 + minute
	EndMinutes   int     // pre-parsed: hour*60 + minute
	Mode         string
	Targets      map[string]*float64 // sensor_type → target value
}

// RoomCache holds the complete runtime state for a single room.
// Mu protects all fields — the control loop holds Mu.RLock for the duration
// of its tick evaluation; ingestion and stream consumer hold Mu.Lock for
// updates. Exported so callers can hold the lock across multi-field reads.
//
// Pre-computed fields (derived at warm/reload, never updated at tick time):
//   - Location: resolved from UserTimezone string
//   - ActuatorHwIDs: derived from devices assigned to this room
//   - SchedulePeriodCache fields: StartMinutes, EndMinutes, DaysOfWeek [8]bool
type RoomCache struct {
	Mu               sync.RWMutex
	RoomID           uuid.UUID
	UserTimezone     string         // kept for reference/reload
	Location         *time.Location // pre-resolved from UserTimezone
	DeadbandTemp     float64
	DeadbandHumidity float64
	DesiredState     DesiredStateCache
	ActivePeriods    []SchedulePeriodCache
	ActuatorHwIDs    map[string][]string             // actuator_type → []hw_id
	ActuatorStates   map[string]bool                 // actuator_type → last commanded state
	LatestReadings   map[string][]TimestampedReading // sensor_type   → readings
	LastActivePeriod *SchedulePeriodCache
}

// SensorEntry holds immutable metadata for a single sensor.
type SensorEntry struct {
	SensorID        uuid.UUID
	MeasurementType string // "temperature", "humidity", "air_quality"
}

// ActuatorEntry holds immutable metadata for a single actuator.
type ActuatorEntry struct {
	ActuatorID   uuid.UUID
	ActuatorType string // "heater", "humidifier"
}

// DeviceCache holds metadata for a single device including its mutable room
// assignment. mu protects RoomID which changes on device assignment/unassignment.
// Sensors and Actuators are immutable after creation and keyed by type for O(1)
// lookup at ingestion and control loop evaluation time.
type DeviceCache struct {
	mu        sync.RWMutex
	DeviceID  uuid.UUID
	HwID      string
	RoomID    *uuid.UUID
	Sensors   map[string]SensorEntry   // measurement_type → entry
	Actuators map[string]ActuatorEntry // actuator_type    → entry
}

// GetRoomID returns the device's current room assignment.
// This method is safe for concurrent use.
func (dc *DeviceCache) GetRoomID() *uuid.UUID {
	dc.mu.RLock()
	defer dc.mu.RUnlock()
	return dc.RoomID
}

// SetRoomID updates the device's room assignment.
// This method is safe for concurrent use.
func (dc *DeviceCache) SetRoomID(roomID *uuid.UUID) {
	dc.mu.Lock()
	defer dc.mu.Unlock()
	dc.RoomID = roomID
}

// Store is the in-memory cache for device-service.
// mu protects the rooms and devices maps themselves — held only for the
// duration of map reads/writes, not for operations on the objects within.
// RoomCache and DeviceCache each have their own mutex for field-level access.
//
// assignedPartitions and numPartitions are unused until Phase 6 (Kafka).
// They are stubbed here so the Kafka consumer group callbacks can populate
// them without requiring structural changes to Store.
type Store struct {
	mu      sync.RWMutex
	rooms   map[uuid.UUID]*RoomCache // room_id → cache
	devices map[string]*DeviceCache  // hw_id   → cache

	// TODO Phase 6: populated by OnPartitionsAssigned / OnPartitionsRevoked
	// callbacks from the franz-go Kafka consumer. Until then, assignedPartitions
	// is always empty and OwnsRoom returns true for all rooms.
	assignedPartitions map[int32]struct{}
	numPartitions      int32
}

func NewStore() *Store {
	return &Store{
		rooms:              make(map[uuid.UUID]*RoomCache),
		devices:            make(map[string]*DeviceCache),
		assignedPartitions: make(map[int32]struct{}),
	}
}

// Room returns the RoomCache for the given room ID, or nil if not found.
// Callers must acquire Mu before accessing fields of the returned cache.
func (s *Store) Room(roomID uuid.UUID) *RoomCache {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.rooms[roomID]
}

// Device returns the DeviceCache for the given hw_id, or nil if not found.
// This method is safe for concurrent use.
func (s *Store) Device(hwID string) *DeviceCache {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.devices[hwID]
}

// AddRoom inserts a new RoomCache into the store. Called during cache warm
// and on room_created stream events.
// This method is safe for concurrent use.
func (s *Store) AddRoom(roomID uuid.UUID, rc *RoomCache) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.rooms[roomID] = rc
}

// DeleteRoom removes a RoomCache from the store. Called on room_deleted
// stream events. Any goroutines holding a pointer to the room continue
// using it safely — the object remains in memory until GC'd.
// This method is safe for concurrent use.
func (s *Store) DeleteRoom(roomID uuid.UUID) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.rooms, roomID)
}

// AddDevice inserts a new DeviceCache into the store. Called during cache
// warm and on device_changed stream events.
// This method is safe for concurrent use.
func (s *Store) AddDevice(hwID string, dc *DeviceCache) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.devices[hwID] = dc
}

// DeleteDevice removes a DeviceCache from the store. Called on device_changed
// stream events when a device is deleted.
// This method is safe for concurrent use.
func (s *Store) DeleteDevice(hwID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.devices, hwID)
}

// RoomIDs returns all currently cached room IDs. Called once at startup by
// the scheduler to spin up per-room ticker goroutines.
// This method is safe for concurrent use.
func (s *Store) RoomIDs() []uuid.UUID {
	s.mu.RLock()
	defer s.mu.RUnlock()
	ids := make([]uuid.UUID, 0, len(s.rooms))
	for id := range s.rooms {
		ids = append(ids, id)
	}
	return ids
}

// OwnsRoom reports whether this instance is responsible for the given room.
// Phase 3: always returns true — single instance owns all rooms.
//
// TODO Phase 6: replace with Kafka partition ownership check.
// murmur2(room_id) % numPartitions must be in assignedPartitions.
// assignedPartitions is populated by OnPartitionsAssigned /
// OnPartitionsRevoked callbacks from the franz-go Kafka consumer.
func (s *Store) OwnsRoom(_ uuid.UUID) bool {
	return true
}

// DeviceHwIDs returns all currently cached device hw_ids.
// This method is safe for concurrent use.
func (s *Store) DeviceHwIDs() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	hwIDs := make([]string, 0, len(s.devices))
	for hwID := range s.devices {
		hwIDs = append(hwIDs, hwID)
	}
	return hwIDs
}
