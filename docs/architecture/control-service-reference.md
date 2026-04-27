# Control Service — Reference

Low-level reference for the Control Service. Covers data structures, startup
sequences, timing constants, MQTT topics, and cache invalidation event types.

For architecture overview and design decisions see
[`control-service.md`](control-service.md).

---

## Startup sequence (Phases 3–6)

1. Load config from env
2. Connect to appdb (GORM), metricsdb (pgx pool), Redis, Mosquitto
3. Start health server on `:8081` — returns 503 until ready
4. Warm cache from appdb — `appdb.WarmCache(store)`
5. Start telemetry ingestion — `ingestor.Run(ctx)`, fatal if source fails
6. Create Redis consumer group if not exists (BUSYGROUP ignored on restart)
7. Drain pending stream entries with `0-0` before switching to live reads
8. Start Redis stream consumer goroutine
9. Start per-room control loop ticker goroutines, staggered
10. Start periodic cache refresh ticker
11. Mark health server ready — returns 200
12. Block on SIGTERM/SIGINT → graceful shutdown via context cancel + WaitGroup

## Startup sequence (Phase 7)

1. Load config from env
2. Connect to appdb (GORM), metricsdb (pgx pool), Mosquitto (commands only), Kafka
3. Start health server on `:8081` — returns 503 until ready
4. Register `OnPartitionsAssigned` and `OnPartitionsRevoked` callbacks with franz-go
5. Join Kafka consumer group → callbacks fire asynchronously
6. `OnPartitionsAssigned` warms cache for assigned rooms, starts control loops
7. Mark health server ready — returns 200
8. Block on SIGTERM/SIGINT → graceful shutdown

Note: `appdb.WarmCache(store)` call in `main.go` is removed in Phase 7. Warm is
entirely owned by `OnPartitionsAssigned`. No Redis connection in Phase 7.

---

## Cache data structures

### Store

```go
type Store struct {
    mu                 sync.RWMutex
    rooms              map[uuid.UUID]*RoomCache  // room_id → cache
    devices            map[string]*DeviceCache   // hw_id   → cache
    assignedPartitions map[int32]struct{}         // Phase 7: Kafka partitions owned by this instance
    numPartitions      int32                      // Phase 7: total Kafka topic partitions
}
```

`mu` protects the maps only — inserting and deleting pointers. Field-level access
on individual entries is protected by per-struct mutexes.

`assignedPartitions` and `numPartitions` are unused until Phase 7. `OwnsRoom()`
returns true for all rooms in Phases 3–6.

### RoomCache

```go
type RoomCache struct {
    Mu sync.RWMutex  // exported — callers hold across multi-field reads

    // Pre-computed at warm/reload — never recomputed at tick time
    DesiredState        DesiredStateCache
    ActivePeriods       []SchedulePeriodCache
    Location            *time.Location
    ActuatorHwIDs       map[string][]string  // actuator_type → []hw_id

    // Runtime-only — preserved across ReloadRoom
    LatestReadings      map[string][]TimestampedReading  // sensor_type → readings
    ActuatorStates      map[string]bool                  // hw_id → last commanded state
    LastActivePeriod    *SchedulePeriodCache
}
```

`Mu` is exported so the control loop can hold a read lock across the entire tick
evaluation. Write lock held by ingestion and stream/Kafka consumer for field updates.

### SchedulePeriodCache

Pre-computed fields derived from the database representation:

```go
type SchedulePeriodCache struct {
    ID           uuid.UUID
    DaysOfWeek   [8]bool  // indexed by ISO day (1=Mon, 7=Sun), index 0 unused
    StartMinutes int      // parsed from "HH:MM" — minutes since midnight
    EndMinutes   int      // parsed from "HH:MM" — minutes since midnight
    TargetTemp   *float64
    TargetHum    *float64
}
```

### DeviceCache

```go
type DeviceCache struct {
    mu       sync.RWMutex  // unexported
    HwID     string
    RoomID   *uuid.UUID    // only mutable field — use GetRoomID()/SetRoomID()
    Sensors   map[string]SensorEntry   // measurement_type → entry
    Actuators map[string]ActuatorEntry // actuator_type    → entry
}
```

`RoomID` is the only field that changes after initial population. Always access
via `GetRoomID()` / `SetRoomID()` — these acquire/release the mutex internally.

### SensorEntry / ActuatorEntry

```go
type SensorEntry struct {
    ID              uuid.UUID
    MeasurementType string
}

type ActuatorEntry struct {
    ID           uuid.UUID
    ActuatorType string
}
```

---

## Cache warm and reload

### WarmCache

Called once at startup (Phases 3–6) or inside `OnPartitionsAssigned` (Phase 7).
Performs bulk fetches — one query per data type, all using `IN` clauses:

1. Fetch all room IDs (filtered by `OwnsRoom()` in Phase 7)
2. Fetch rooms + user timezone with `JOIN users`
3. Fetch desired states for all rooms
4. Fetch active schedule periods — `JOIN schedules WHERE is_active = true`
5. Fetch devices for all rooms
6. Fetch sensors for all devices
7. Fetch actuators for all devices
8. Build index maps in Go, assemble per-room `RoomCache` entries
9. Pre-compute: resolve timezone strings, parse HH:MM to minutes, build DaysOfWeek
   bitmask, build ActuatorHwIDs map

### ReloadRoom

Targeted single-room reload. Called by:
- Stream consumer on invalidation event (Phases 3–6)
- Kafka invalidation consumer (Phase 7)
- Periodic cache refresh ticker

Runs the same queries as WarmCache but scoped to a single `room_id`. Explicitly
preserves `LatestReadings`, `ActuatorStates`, and `LastActivePeriod` from the
existing cache entry before overwriting — these runtime fields cannot be recovered
from the database.

### ReloadDevice

Upserts or evicts a single device cache entry. Called on device assignment change
events. If the device no longer exists or is unassigned, the entry is evicted.

---

## Transport-agnostic ingestion

### Source interface

```go
type Source interface {
    Start(ctx context.Context, handler func(context.Context, TelemetryMessage)) error
    Stop()
}
```

### TelemetryMessage

```go
type TelemetryMessage struct {
    HwID      string
    RoomID    *uuid.UUID  // nil if device is unassigned
    Readings  []Reading
    Timestamp time.Time
}

type Reading struct {
    MeasurementType string
    Value           float64
}
```

### Drop conditions

Silent (no log):
- `HwID` not in device cache — unknown device
- `RoomID` is nil — device exists but is unassigned

Warning logged (cache inconsistency, should not occur):
- Room not owned by this instance
- Room not in store

---

## Control loop

### Tick sequence

```
for each room goroutine, every CONTROL_TICK_INTERVAL_SECONDS:

1. rc.Mu.RLock()
2. resolveEffectiveState(rc) → EffectiveState{Mode, Targets, Source, Period}
3. if Mode == OFF:
     for each hw_id in all ActuatorHwIDs: publish OFF
4. if Mode == AUTO:
     for each measurementType with a target:
       freshReadings = filter LatestReadings[type] where age < staleThreshold
       if len(freshReadings) == 0: publish OFF for all actuators of this type
       avg = mean(freshReadings)
       if avg < target - deadband: publish ON
       if avg > target + deadband: publish OFF
       if within deadband: re-send ActuatorStates[hwID]
5. rc.Mu.RUnlock()
6. update ActuatorStates
7. metricsdb.WriteControlLogEntry(...)
```

### Control truth table

| Readings | Mode | Value vs target | Command |
|---|---|---|---|
| Stale / missing | any | — | OFF |
| Fresh | OFF | — | OFF |
| Fresh | AUTO | Below target − deadband | ON |
| Fresh | AUTO | Above target + deadband | OFF |
| Fresh | AUTO | Within deadband | Re-send last commanded state |

### Goroutine stagger

```
offset = tickInterval * roomIndex / totalRooms
```

Prevents all rooms from evaluating simultaneously. Spreads TimescaleDB writes and
MQTT publishes evenly across the tick interval.

### Scheduler

```go
type Scheduler struct {
    activeRooms map[uuid.UUID]context.CancelFunc
}
```

Rooms added on `room_created` invalidation event — starts a new goroutine with a
fresh context. Rooms removed on `room_deleted` — calls `CancelFunc`, goroutine exits
on next tick when it checks `ctx.Done()`.

---

## Cache invalidation events (Phases 3–6)

**Stream key:** `stream:cache_invalidation`

**Consumer group:** `control-service-{hostname}` — one group per instance. Every
instance receives every event independently. Work distribution is via Kafka (Phase 7),
not the stream.

**Startup behaviour:**
- First start: create group at stream tip (`$`) — ignore historical events
- Restart: group already exists (BUSYGROUP error ignored). Drain pending entries
  with `0-0` ID before switching to live `>` reads

**Block timeout:** 5s — bounds shutdown response time without busy-looping

**Ack behaviour:**
- Unknown event types: acked immediately and skipped
- Successful cache updates: acked
- Failed cache updates: not acked — redelivered on next restart

### Event types and handlers

| Event type | Handler | Notes |
|---|---|---|
| `device_assigned` | `ReloadDevice` + `ReloadRoom` (new room) | Also `ReloadRoom` for old room if device was previously assigned |
| `device_unassigned` | `ReloadDevice` + `ReloadRoom` | |
| `desired_state_changed` | `ReloadRoom` | |
| `room_config_changed` | `ReloadRoom` | Deadband updates |
| `schedule_activated` | `ReloadRoom` | |
| `schedule_deactivated` | `ReloadRoom` | |
| `room_created` | `ReloadRoom` + start control loop goroutine | |
| `room_deleted` | Evict from store + stop control loop goroutine | |

---

## MQTT topics and QoS

| Topic | Direction | QoS | Publisher |
|---|---|---|---|
| `devices/{hw_id}/telemetry` | device → Control Service | 1 | ESP32 / Simulator (Phases 3–6) |
| `devices/{hw_id}/cmd` | Control Service → device | 2 | Control Service (all phases) |

Commands always flow Control Service → Mosquitto → device directly, even in Phase 7.

**Client ID pattern:** `control-service-{HOSTNAME}` — distinct per instance.
Mosquitto ACL grants publish rights by username, not client ID, so multiple
instances can share credentials.

In Phase 7 the Control Service no longer subscribes to `devices/+/telemetry` —
that subscription moves to the Kafka Bridge. The Mosquitto connection is retained
for command publishing only.

---

## Timing constants

| Parameter | Env var | Default | Notes |
|---|---|---|---|
| Control loop tick | `CONTROL_TICK_INTERVAL_SECONDS` | 10s | Shared with Simulator as system heartbeat |
| Stale threshold | `CONTROL_STALE_THRESHOLD_SECONDS` | 90s | 3-reading window at default tick rate |
| Cache refresh interval | `CONTROL_CACHE_REFRESH_MINUTES` | 5min | Safety net for missed invalidation events |
| Stream block timeout | — | 5s | Hardcoded — bounds shutdown response time (Phases 3–6) |
| Grace period | — | 60s | Hardcoded — 1 min after period end before reverting to OFF |

---

## TimescaleDB writes

### sensor_readings — one row per sensor per telemetry message

```sql
time        TIMESTAMPTZ NOT NULL
sensor_id   UUID NOT NULL
room_id     UUID             -- snapshotted at ingestion time
value       NUMERIC NOT NULL -- calibrated value (= raw_value until calibration implemented)
raw_value   NUMERIC NOT NULL -- pre-calibration value
```

`room_id` is snapshotted at write time from the device cache — preserves accurate
historical attribution after device reassignment.

### room_control_logs — one row per tick per room

```sql
time               TIMESTAMPTZ NOT NULL
room_id            UUID NOT NULL
avg_temp           NUMERIC
avg_hum            NUMERIC
mode               TEXT        -- 'OFF' | 'AUTO'
target_temp        NUMERIC
target_hum         NUMERIC
control_source     TEXT        -- 'manual_override' | 'schedule' | 'grace_period' | 'none'
heater_cmd         SMALLINT    -- 0/1, null if room has no heater
humidifier_cmd     SMALLINT    -- 0/1, null if room has no humidifier
deadband_temp      NUMERIC     -- snapshotted from cache at write time
deadband_hum       NUMERIC     -- snapshotted from cache at write time
reading_count_temp SMALLINT
reading_count_hum  SMALLINT
schedule_period_id UUID        -- set when source is 'schedule' or 'grace_period'
```

`heater_cmd` / `humidifier_cmd` are SMALLINT not BOOLEAN — `AVG()` over a time
bucket produces a duty cycle fraction (0.0–1.0) without casting.

Both tables are TimescaleDB hypertables partitioned by `time` with 1-day chunks.

---

## Health server

Minimal HTTP server on `:8081`. Internal to the Docker network — port not
published on the host.

- Returns `503 Service Unavailable` from startup until `SetReady()` is called
- Returns `200 OK` after `SetReady()` — Docker health check target
- Single endpoint: `GET /health`