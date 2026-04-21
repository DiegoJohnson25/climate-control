# Device Service — Architecture Reference

Standalone Go binary. No HTTP server. Communicates via MQTT (telemetry in,
commands out) and PostgreSQL (appdb read, metricsdb write). Redis for cache
invalidation stream. api-service and device-service are fully decoupled —
share PostgreSQL and Redis but no direct service-to-service calls.

---

## Startup sequence

1. Load config from env
2. Connect to appdb (GORM), metricsdb (pgx pool), Redis, Mosquitto
3. Warm cache from appdb — `appdb.WarmCache(store)`
4. Start telemetry ingestion — `ingestor.Run(ctx)`, fatal if source fails
5. Create Redis consumer group if not exists (BUSYGROUP ignored on restart)
6. Drain pending stream entries with `0-0` before switching to live reads
7. Start Redis stream consumer goroutine
8. Start per-room control loop ticker goroutines, staggered
9. Start periodic cache refresh ticker
10. Block on SIGTERM/SIGINT → graceful shutdown via context cancel + WaitGroup

---

## Cache architecture

### Store

Top-level container. `sync.RWMutex` protects the two maps. Map-level lock held
only for map reads/writes (inserting/deleting pointers). Field-level access
protected by per-struct mutexes.

```go
type Store struct {
    mu                 sync.RWMutex
    rooms              map[uuid.UUID]*RoomCache  // room_id → cache
    devices            map[string]*DeviceCache   // hw_id   → cache
    assignedPartitions map[int32]struct{}         // Phase 7: Kafka partitions owned by this instance
    numPartitions      int32                      // Phase 7: total Kafka topic partitions
}
```

`assignedPartitions` and `numPartitions` are unused until Phase 7. `OwnsRoom()`
returns true for all rooms until Kafka populates them.

### RoomCache

Full runtime state for one room. `Mu sync.RWMutex` exported so callers can hold
read lock across multi-field reads (control loop tick). Write lock held by
ingestion and stream consumer for field updates.

Pre-computed at warm/reload (never recomputed at tick time):
- `Location *time.Location` — resolved from `UserTimezone` string
- `ActuatorHwIDs map[string][]string` — actuator_type → []hw_id in this room
- `SchedulePeriodCache.StartMinutes/EndMinutes int` — parsed from "HH:MM"
- `SchedulePeriodCache.DaysOfWeek [8]bool` — indexed by ISO day (1-7)

Runtime-only (never persisted, preserved across ReloadRoom):
- `LatestReadings map[string][]TimestampedReading` — sensor_type → readings, trimmed on ingestion
- `ActuatorStates map[string]bool` — last commanded state, initialized false
- `LastActivePeriod *SchedulePeriodCache` — used for grace period logic

### DeviceCache

Device metadata. `mu sync.RWMutex` unexported. `RoomID *uuid.UUID` is the only
mutable field — accessed via `GetRoomID()`/`SetRoomID()` safe concurrent wrappers.

```go
Sensors   map[string]SensorEntry   // measurement_type → entry (O(1) lookup)
Actuators map[string]ActuatorEntry // actuator_type    → entry (O(1) lookup)
```

### Cache warm

`appdb.WarmCache(store)` — called once at startup before any goroutines start.

1. Fetch all room IDs
2. Filter to owned rooms via `store.OwnsRoom()` (always true Phase 3–6, Kafka-filtered Phase 7)
3. Bulk fetch with `IN` clause: rooms+timezone (JOIN users), desired states, active
   schedule periods (JOIN schedules WHERE is_active=true), devices, sensors, actuators
4. Build index maps in Go, assemble per room
5. Apply pre-computations: resolve timezone, parse time strings, build DaysOfWeek
   bitmask, build ActuatorHwIDs from devices

`appdb.ReloadRoom(store, roomID)` — targeted single-room queries. Called by stream
consumer and periodic refresh. Preserves `LatestReadings`, `ActuatorStates`,
`LastActivePeriod` from existing cache entry — runtime fields are never clobbered.

`appdb.ReloadDevice(store, hwID)` — upserts or evicts device cache entry. Called
by stream consumer on device assignment change events.

---

## Transport-agnostic ingestion

```go
type Source interface {
    Start(ctx context.Context, handler func(context.Context, TelemetryMessage)) error
    Stop()
}
```

- `mqtt.Source` — Phases 3–6. Paho wrapper, subscribes to Mosquitto, parses payload.
- `kafka.Source` — Phase 7. franz-go consumer, constructs same `TelemetryMessage`.
- `ingestion.Process` is identical in both cases — transport is fully abstracted.

`TelemetryMessage` carries: `HwID`, `RoomID` (nil if unassigned), `Readings`, `Timestamp`.

Drop conditions (silent):
- Unknown hw_id — not in device cache
- Unassigned device — nil RoomID, no room context

Drop conditions (warning logged):
- Room not owned by this instance — cache inconsistency, should not occur
- Room not in store — cache inconsistency, should not occur

---

## Control loop

One goroutine per room. Staggered at startup: `offset = tickInterval * roomIndex / totalRooms`.

On each tick:

1. Acquire `rc.Mu.RLock()` for duration of evaluation
2. `resolveEffectiveState` — determines mode, targets, and control source:
   - Manual override active and not expired → desired state → source: `manual_override`
   - Active schedule period matches current day/time → period → source: `schedule`
   - Within grace period (1 minute of last period end) → last period → source: `grace_period`
   - None of the above → mode OFF → source: `none`
3. If mode OFF → command all actuators OFF
4. If mode AUTO → for each measurement type with a target:
   - Filter fresh readings from `LatestReadings[sensorType]` (drop older than stale threshold)
   - Average fresh readings
   - Compare against target ± deadband
   - Publish command to `devices/{hw_id}/cmd` QoS 2 for each hw_id in `ActuatorHwIDs`
5. Release read lock
6. Update `ActuatorStates` with commanded values
7. Write `room_control_logs` row via `metricsdb.WriteControlLogEntry`

### Control truth table

| Readings | Mode | Value vs target | Command |
|---|---|---|---|
| Stale/missing | any | — | OFF (safe failure) |
| Fresh | OFF | — | OFF |
| Fresh | AUTO | Below target − deadband | ON |
| Fresh | AUTO | Above target + deadband | OFF |
| Fresh | AUTO | Within deadband | Re-send last commanded state |

### Key design decisions

**Unconditional commands as heartbeat** — every actuator gets a command every tick
regardless of whether the state changed. This doubles as a device connectivity
signal. Device watchdog TTL should be slightly above one tick interval. Foundation
for future device connection status feature.

**Deadband re-sends last state** — within deadband, last commanded state is
re-sent every tick (not silent). Required because `ActuatorStates` tracks the
last *commanded* not *confirmed* state, and devices have watchdog timers that
will turn off if they stop receiving commands.

**Safe failure to OFF** — no fresh readings → command OFF, never hold last state.
Prevents runaway actuators if sensor goes offline.

### Scheduler

Manages goroutine lifecycle — `activeRooms map[uuid.UUID]context.CancelFunc`.
Rooms added and removed dynamically via stream consumer events (room
created/deleted triggers goroutine start/stop).

### Periodic cache refresh

Safety net for missed stream events. One ticker per room, staggered. Default
interval: 5 minutes. Calls `ReloadRoom` — runtime fields are preserved.

---

## Redis stream consumer

**Stream:** `stream:cache_invalidation`

**Consumer group design:** per-instance group `device-service-{hostname}`, not a
shared group. Every instance needs every invalidation event — shared group would
distribute events across instances, leaving some caches stale. Work distribution
via Kafka (Phase 7), not stream consumer.

**Startup:** create consumer group at stream tip (`$`) on first start. On restart:
group already exists (BUSYGROUP ignored). Drain pending entries with `0-0` ID
before switching to live `>` reads.

**Event handling:**
- Unknown event types: acked and skipped — never accumulate in pending
- Failed cache updates: not acked — redelivered on next instance restart
- `blockTimeout = 5s` — bounds shutdown response time, not busy-looping

**Events written by api-service on:**
- Device assigned to room → `ReloadDevice` + `ReloadRoom` (both old and new room)
- Device unassigned from room → `ReloadDevice` + `ReloadRoom`
- Desired state changed → `ReloadRoom`
- Room config changed (deadbands) → `ReloadRoom`
- Schedule activated/deactivated → `ReloadRoom`
- Room created → `ReloadRoom` (warms new room)
- Room deleted → evict from store, stop control loop goroutine

---

## TimescaleDB — metricsdb

Write-only from device-service. Read-only from api-service (Phase 5a climate endpoints).

```sql
-- sensor_readings: one row per sensor per telemetry message
time        TIMESTAMPTZ NOT NULL
sensor_id   UUID NOT NULL
room_id     UUID             -- snapshotted at ingestion time for accurate historical metrics
value       NUMERIC NOT NULL -- calibrated value (= raw_value until calibration implemented)
raw_value   NUMERIC NOT NULL -- pre-calibration value

-- room_control_logs: one row per control loop tick per room
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
reading_count_temp SMALLINT
reading_count_hum  SMALLINT
schedule_period_id UUID        -- set when source is 'schedule' or 'grace_period'
```

`heater_cmd`/`humidifier_cmd` are SMALLINT not BOOLEAN — `AVG()` produces a duty
cycle fraction (0.0–1.0) at any time bucket resolution without casting.

`room_id` on `sensor_readings` is snapshotted at write time. This preserves accurate
historical metrics even after a device is reassigned to a different room.

Both tables are TimescaleDB hypertables partitioned by `time` with 1-day chunks.

---

## MQTT

- Telemetry: `devices/{hw_id}/telemetry` — QoS 1, published by devices/simulator
- Commands: `devices/{hw_id}/cmd` — QoS 2, published by device-service

Multiple device-service instances connect to Mosquitto with the same credentials
but distinct client IDs (`device-service-{HOSTNAME}`). ACL grants publish rights
by username, not client ID. Commands always flow device-service → Mosquitto → device
directly — permanently, even in Phase 7. No benefit routing commands through Kafka.

---

## Timing constants

| Parameter | Env var | Default | Rationale |
|---|---|---|---|
| Simulator tick interval | — | 30s | Matches realistic ESP32 polling rate |
| Stale threshold | `CONTROL_STALE_THRESHOLD_SECONDS` | 90s | 3-reading window — device silent >90s treated as unavailable |
| Control loop tick | `CONTROL_TICK_INTERVAL_SECONDS` | 30s | No benefit evaluating faster than sensor rate |
| Cache refresh interval | `CONTROL_CACHE_REFRESH_MINUTES` | 5min | Safety net for missed stream events |
| Stream block timeout | — | 5s | Bounds graceful shutdown response time |
| Grace period | — | 60s | 1 minute after period end before reverting to OFF |
