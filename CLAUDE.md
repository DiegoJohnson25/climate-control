# Climate Control Project — CLAUDE.md

## Project overview

A distributed IoT climate control system backend in Go. Two independent services
communicate via PostgreSQL and MQTT to manage room climate via ESP32 relay devices.
Fully demonstrable without hardware via a simulator service.

---

## Current state

**Completed:**
- Phase 1 ✅ — repo scaffold, Docker Compose infrastructure, full DB schema migrated
- `feat/api-service-scaffold` ✅ — merged to main
- `feat/api-service-auth` ✅ — merged to main
- `feat/api-service-rooms-devices` ✅ — merged to main
  - rooms domain: CRUD + desired state (7 endpoints)
  - devices domain: registration, assignment, capability conflict enforcement (6 endpoints)
- `feat/api-service-schedules` ✅ — merged to main
  - schedules domain: CRUD + activate/deactivate + periods (11 endpoints)
- Postman test collections committed to `tests/postman/` ✅
- `feat/simulator-scaffold` ✅ — merged to main
  - simulator-service: config loading, provisioning, MQTT client, publish loop
  - Mosquitto: auth enabled, ACL configured, password file committed
  - Makefile: refactored with prefix groups, simulator commands, MQTT subscriptions
- `feat/device-service-scaffold` ✅ — merged to main
  - config, connections, in-memory cache, appdb repository, cache warm, logging package
  - Cache warm verified against `cache-test` simulation with schedules and manual overrides
  - Bug fix: capability checks now use EXISTS + EXISTS pattern (sensor and actuator
    may be on separate devices) — affected `HasTemperatureCapability`,
    `HasHumidityCapability`, `activeSchedulePeriodsHaveConflict`

**Active branch:** `feat/device-service-ingestion`

**Planned branches:**
- `feat/device-service-ingestion` — MQTT subscription, telemetry parsing, TimescaleDB writes
- `feat/device-service-control` — control loop, bang-bang logic, command publishing
- `feat/device-service-stream` — Redis stream consumer, cache invalidation
- `feat/device-service-refresh` — periodic cache refresh, scheduler goroutine management

---

## Repo structure

```
climate-control/
├── api-service/
│   ├── cmd/
│   │   └── main.go
│   ├── internal/
│   │   ├── user/
│   │   │   ├── handler.go        # POST /auth/register, GET /users/me, DELETE /users/me
│   │   │   ├── service.go
│   │   │   ├── repository.go     # GORM — users table
│   │   │   └── errors.go
│   │   ├── auth/
│   │   │   ├── handler.go        # POST /auth/login, /auth/refresh, /auth/logout
│   │   │   ├── middleware.go     # JWT validation middleware
│   │   │   ├── service.go        # login, refresh rotation, logout
│   │   │   ├── repository.go     # Redis — refresh token storage
│   │   │   └── errors.go
│   │   ├── room/
│   │   │   ├── handler.go        # room CRUD + desired state endpoints
│   │   │   ├── service.go
│   │   │   ├── repository.go     # capability queries live here (HasTemperatureCapability etc.)
│   │   │   ├── pg_errors.go
│   │   │   └── errors.go
│   │   ├── device/
│   │   │   ├── handler.go        # device CRUD + list by room
│   │   │   ├── service.go
│   │   │   ├── repository.go     # activeSchedulePeriodsHaveConflict lives here permanently
│   │   │   ├── pg_errors.go
│   │   │   └── errors.go
│   │   ├── schedule/
│   │   │   ├── handler.go        # schedule + period endpoints
│   │   │   ├── service.go
│   │   │   ├── repository.go
│   │   │   ├── pg_errors.go
│   │   │   └── errors.go
│   │   ├── router/
│   │   │   └── router.go         # route registration, middleware chain
│   │   ├── health/
│   │   │   └── health.go         # plain function, no struct
│   │   ├── ctxkeys/
│   │   │   └── keys.go           # context key constants — prevents circular imports
│   │   ├── config/
│   │   │   └── config.go         # typed config struct, Load()
│   │   └── initializers/
│   │       └── redis.go          # Redis connection — cleanup task: move to connect/
│   ├── Dockerfile
│   └── go.mod
├── device-service/
│   ├── cmd/
│   │   └── main.go               # connections, cache warm, signal handling
│   ├── internal/
│   │   ├── config/
│   │   │   └── config.go         # env loading — no ports (hardcoded internal Docker)
│   │   ├── connect/
│   │   │   ├── postgres.go       # ConnectPostgres() → *gorm.DB
│   │   │   ├── timescale.go      # ConnectTimescale() → *pgxpool.Pool
│   │   │   └── redis.go          # ConnectRedis() → *redis.Client
│   │   ├── cache/
│   │   │   └── cache.go          # Store, RoomCache, DeviceCache + entry types
│   │   ├── repository/
│   │   │   ├── appdb/
│   │   │   │   └── repository.go # WarmCache, ReloadRoom, ReloadDevice
│   │   │   └── metricsdb/        # Phase 3c — TimescaleDB writes
│   │   ├── logging/
│   │   │   └── logging.go        # cache inspection — LogSummary, LogStore, LogFullStore etc.
│   │   ├── mqtt/                 # Phase 3c — Paho wrapper
│   │   ├── ingestion/            # Phase 3c — telemetry parsing, TimescaleDB writes
│   │   ├── control/              # Phase 3d — bang-bang logic, command publishing
│   │   ├── scheduler/            # Phase 3d — per-room ticker goroutines, lifecycle management
│   │   └── stream/               # Phase 3e — Redis stream consumer, cache invalidation
│   ├── Dockerfile
│   └── go.mod
├── simulator-service/
│   ├── cmd/
│   │   └── main.go               # flag parsing, run/teardown modes, signal handling
│   ├── config/
│   │   ├── templates/
│   │   │   ├── rooms.yaml        # room templates (behaviour type, base values)
│   │   │   └── devices.yaml      # device templates (sensors, actuators, noise, offset)
│   │   ├── simulations/
│   │   │   ├── default.yaml      # single user, full capability room
│   │   │   ├── multi-room.yaml
│   │   │   ├── multi-user.yaml
│   │   │   ├── sensor-only.yaml
│   │   │   ├── multi-sensor.yaml
│   │   │   └── cache-test.yaml   # 5 rooms covering all capability combinations, interactive
│   │   └── credentials/          # gitignored, written at runtime for interactive groups
│   │       └── .gitkeep
│   ├── internal/
│   │   ├── api/
│   │   │   └── client.go         # HTTP client for api-service
│   │   ├── config/
│   │   │   └── config.go         # env + YAML loading, template resolution, validation
│   │   ├── mqtt/
│   │   │   └── client.go         # Paho wrapper — Publish, Subscribe, Disconnect
│   │   ├── provisioning/
│   │   │   └── provisioning.go   # bootstrap sequence, identity generation, credentials file
│   │   └── simulator/
│   │       └── simulator.go      # publish loop, staggered goroutines per device
│   ├── Dockerfile
│   └── go.mod
├── shared/                       # cleanup task: fold into api-service/internal/
│   ├── models/                   # GORM structs — schema contract
│   └── db/                       # postgres + timescale connection helpers
├── firmware/
│   └── esp32/
├── deployments/
│   ├── docker-compose.services.yml
│   ├── docker-compose.prod.yml
│   ├── mosquitto/
│   │   ├── mosquitto.conf        # auth enabled, passwd + acl file paths
│   │   ├── passwd                # generated via make mosquitto-passwd, committed
│   │   └── acl                   # topic permissions per username
│   └── nginx/nginx.conf
├── migrations/
│   ├── appdb/
│   └── metricsdb/
├── docs/
├── tests/
│   └── postman/
│       ├── climate-control-integration.collection.json
│       ├── climate-control-smoke.collection.json
│       ├── climate-control-manual.collection.json
│       ├── integration.environment.json
│       ├── smoke.environment.json
│       └── manual.environment.json
├── docker-compose.yml
├── go.work
├── Makefile
├── .env
├── CLAUDE.md
└── .github/
    ├── workflows/ci.yml
    └── pull_request_template.md
```

---

## Package naming rules (locked in)

- Package name is the domain: `user`, `auth`, `room`, `device`, `schedule`
- File name encodes the layer: `handler.go`, `service.go`, `repository.go`
- Types drop the domain prefix: `auth.Service` not `auth.AuthService`
- Constructors: `NewService`, `NewHandler`, `NewRepository`
- File names are snake_case, no suffix (e.g. `handler.go` not `user_handler.go`)
- Package/folder names are singular
- Repository constructor always `NewRepository` regardless of backing store
- Postgres unique violation detection in a private `pg_errors.go` per package
  (unexported helper — not extracted to shared utility, intentionally duplicated)

---

## Naming conventions (locked in)

- `models.User` variables → `usr`
- `models.Room` variables → `rm`
- `models.Device` variables → `dev`
- `models.Schedule` variables → `sched`
- `models.SchedulePeriod` variables → `period`
- `user` package imported as `user` — no alias needed since variables use `usr`
- Service struct fields named by concept: `users`, `tokens`, `rooms`, `devices`, `schedules`
- Single-repo services use plain `repo`
- Method receivers use single letter: `(s *Service)`, `(h *Handler)`, `(r *Repository)`
- `List` prefix for slice-returning methods
- No `ID` suffix on method names like `ListByRoom` (not `ListByRoomID`)
- Service layer parameters use `input` naming (not `req`, reserved for handler request structs)

---

## Tech stack

| Layer | Technology | Notes |
|---|---|---|
| Language | Go 1.25.0 | |
| HTTP framework | Gin | `api-service` only |
| ORM | GORM | App DB only — device-service uses Raw+Scan for all queries |
| Time-series queries | pgx raw SQL | TimescaleDB |
| Auth | JWT | golang-jwt library |
| Refresh tokens | Redis | go-redis/v9 (`github.com/redis/go-redis/v9`) |
| Rate limiting | Redis | Applied at router level |
| Event streaming | Redis Streams | `device-service:events` stream — Phase 3e |
| MQTT client | Eclipse Paho Go | aliased as `pahomqtt` to avoid package name collision |
| Broker | Mosquitto 2.x | Auth enabled, ACL per username |
| App DB | PostgreSQL 17 | Internal port 5432, host port 5433 |
| Time-series DB | TimescaleDB 2.25.2-pg17 | Internal port 5432, host port 5434 |
| Cache | In-process memory | `device-service` only — `sync.RWMutex` per struct |
| Migrations | golang-migrate | Auto-run on `docker compose up` |
| Containers | Docker Compose | |
| Testing | Newman + Postman | `make test-api-integration`, `make test-api-smoke` |

---

## Environment variables

```bash
POSTGRES_USER=cc
POSTGRES_PASSWORD=localdev
POSTGRES_DB=appdb
POSTGRES_PORT=5433

TIMESCALE_USER=cc
TIMESCALE_PASSWORD=localdev
TIMESCALE_DB=metricsdb
TIMESCALE_PORT=5434

REDIS_PASSWORD=localdev
REDIS_PORT=6379

MQTT_PORT=1883
MQTT_DEVICE_USERNAME=device
MQTT_DEVICE_PASSWORD=localdev
MQTT_DEVICE_SERVICE_USERNAME=device-service
MQTT_DEVICE_SERVICE_PASSWORD=localdev
# healthcheck user is hardcoded as username=healthcheck password=healthcheck — not in .env

JWT_SECRET=localdev-replace-with-32-plus-chars-in-prod
JWT_ACCESS_TTL_MINUTES=15
JWT_REFRESH_TTL_DAYS=7

API_PORT=8080

SIMULATOR_EMAIL=simulator@local.dev
SIMULATOR_PASSWORD=localdev
```

**Important:** Internal Docker ports are always hardcoded in connection strings.
`host=postgres port=5432`, `host=timescaledb port=5432`, `host=redis port=6379`,
`host=mosquitto port=1883`. The env var ports (5433, 5434 etc.) are host-machine
mappings only — never used inside Docker.

---

## Database schema — appdb (PostgreSQL 17)

```sql
CREATE TABLE users (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email         TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    timezone      TEXT NOT NULL DEFAULT 'UTC',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE rooms (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id           UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name              TEXT NOT NULL,
    deadband_temp     NUMERIC(5,2) NOT NULL DEFAULT 1.0 CHECK (deadband_temp > 0),
    deadband_hum      NUMERIC(5,2) NOT NULL DEFAULT 5.0 CHECK (deadband_hum > 0),
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, name)
);

CREATE TABLE devices (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    room_id     UUID REFERENCES rooms(id) ON DELETE SET NULL,
    name        TEXT NOT NULL,
    hw_id       TEXT NOT NULL UNIQUE,
    device_type TEXT NOT NULL DEFAULT 'physical' CHECK (device_type IN ('physical', 'simulator')),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, name)
);

CREATE TABLE sensors (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    device_id        UUID NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
    measurement_type TEXT NOT NULL CHECK (measurement_type IN ('temperature', 'humidity', 'air_quality')),
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(device_id, measurement_type)
);

CREATE TABLE actuators (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    device_id     UUID NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
    actuator_type TEXT NOT NULL CHECK (actuator_type IN ('heater', 'humidifier')),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(device_id, actuator_type)
);

CREATE TABLE desired_states (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    room_id               UUID NOT NULL REFERENCES rooms(id) ON DELETE CASCADE UNIQUE,
    mode                  TEXT NOT NULL DEFAULT 'OFF' CHECK (mode IN ('OFF', 'AUTO')),
    target_temp           NUMERIC(5,2) CHECK (target_temp BETWEEN 5 AND 40),
    target_hum            NUMERIC(5,2) CHECK (target_hum BETWEEN 0 AND 100),
    manual_override_until TIMESTAMPTZ,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE schedules (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    room_id    UUID NOT NULL REFERENCES rooms(id) ON DELETE CASCADE,
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name       TEXT NOT NULL,
    is_active  BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(room_id, name)
);

CREATE UNIQUE INDEX one_active_schedule_per_room
ON schedules(room_id) WHERE is_active = true;

CREATE TABLE schedule_periods (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    schedule_id     UUID NOT NULL REFERENCES schedules(id) ON DELETE CASCADE,
    name            TEXT,
    days_of_week    INTEGER[] NOT NULL,
    start_time      TEXT NOT NULL,
    end_time        TEXT NOT NULL,
    mode            TEXT NOT NULL DEFAULT 'OFF' CHECK (mode IN ('OFF', 'AUTO')),
    target_temp     NUMERIC(5,2) CHECK (target_temp BETWEEN 5 AND 40),
    target_hum      NUMERIC(5,2) CHECK (target_hum BETWEEN 0 AND 100),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK (end_time > start_time),
    CHECK (mode = 'OFF' OR target_temp IS NOT NULL OR target_hum IS NOT NULL),
    CHECK (array_length(days_of_week, 1) > 0)
);
```

### metricsdb (TimescaleDB)

```sql
CREATE TABLE sensor_readings (
    time      TIMESTAMPTZ NOT NULL,
    sensor_id UUID NOT NULL,
    room_id   UUID,
    value     NUMERIC NOT NULL
);

SELECT create_hypertable('sensor_readings', 'time',
    chunk_time_interval => INTERVAL '7 days');

CREATE INDEX idx_sensor_readings_sensor_time ON sensor_readings(sensor_id, time DESC);
CREATE INDEX idx_sensor_readings_room_time   ON sensor_readings(room_id, time DESC);
```

---

## GORM model conventions

- `ID uuid.UUID \`gorm:"type:uuid;primaryKey;default:gen_random_uuid()"\``
- Foreign key UUIDs: `UserID uuid.UUID \`gorm:"type:uuid"\``
- Nullable foreign keys: `RoomID *uuid.UUID \`gorm:"type:uuid"\``
- Nullable values: pointer types (`*float64`, `*string`, `*time.Time`)
- `CreatedAt`, `UpdatedAt` — `time.Time`, managed automatically by GORM
- String defaults: `\`gorm:"default:UTC"\``
- Numeric defaults: `\`gorm:"default:1.0"\``
- `HwID` needs explicit tag: `\`gorm:"column:hw_id"\``
- `days_of_week` uses `pq.Int64Array \`gorm:"type:integer[]"\`` — tag required for GORM scan
- `schedule_periods.start_time` / `end_time` — `string` on model, `TEXT` in DB
- `TargetHumidity` shortened to `TargetHum` on `models.SchedulePeriod` and `desired_states`
- No constraint tags — migrations handle all constraints
- `desired_states` table name set via `TableName()` method
- GORM used for appdb only — TimescaleDB uses raw pgx
- Shared models never have GORM association fields (e.g. no `Sensors []Sensor` on Device)
  — enriched types live in the domain package that needs them

---

## Capability checks (locked in)

All capability checks use EXISTS + EXISTS pattern — sensor and actuator may be on
**separate devices** in the same room. The old pattern (JOIN sensor + actuator on same
device) was a bug fixed in `feat/device-service-scaffold`.

Correct pattern:
```sql
SELECT (
    EXISTS (
        SELECT 1 FROM devices d
        JOIN sensors s ON s.device_id = d.id
        WHERE d.room_id = ? AND s.measurement_type = 'temperature'
    )
    AND
    EXISTS (
        SELECT 1 FROM devices d
        JOIN actuators a ON a.device_id = d.id
        WHERE d.room_id = ? AND a.actuator_type = 'heater'
    )
) AS has_capability
```

`roomID` must be passed **twice** — once per EXISTS subquery.

Applies to: `HasTemperatureCapability`, `HasHumidityCapability` in `room/repository.go`
and `activeSchedulePeriodsHaveConflict` in `device/repository.go`.

---

## Auth architecture (implemented)

`auth` imports `user` — one directional. `user` knows nothing about `auth`.

```go
type Service struct {
    users  *user.Repository
    tokens *Repository
}
```

---

## Room architecture (implemented)

```go
type Service struct {
    rooms *Repository
}
```

- `desired_states` row created atomically with room in same transaction
- Capability queries (`HasTemperatureCapability`, `HasHumidityCapability`) live permanently
  in `room.Repository` — they query device tables but are a room-level concern.
  Moving them to `device.Repository` would create a circular import.
- `manual_override_until` sentinel: `9999-12-31T23:59:59Z` stored for indefinite overrides
- API accepts `"indefinite"` string, stores sentinel, returns `"indefinite"` in responses
- `desired_states.id` is vestigial — `room_id` is the natural PK but migration is not worth it

---

## Device architecture (implemented)

```go
type Service struct {
    devices *Repository
    rooms   *room.Repository
}
```

- `DeviceWithCapabilities` struct defined in `device/repository.go`
- All list/get methods return `DeviceWithCapabilities` via bulk fetch (3 queries always)
- Sensors and actuators always serialize as `[]string` of type names in responses
- Devices created unassigned, assigned to rooms via `PUT /devices/:id`
- `hw_id` checked via `CheckHwIDAvailability` before insert
- `activeSchedulePeriodsHaveConflict` lives permanently in `device.Repository`

---

## Schedule architecture (implemented)

```go
type Service struct {
    schedules *Repository
    rooms     *room.Repository
}
```

- Schedules are inactive by default — `is_active = false` on create
- Atomic activation transaction, partial unique index `one_active_schedule_per_room`
- Capability validation only at activation time
- Period overlap detection uses PostgreSQL `&&` array overlap operator on `days_of_week`
- Midnight crossing NOT supported — `end_time` must be > `start_time`
- `days_of_week` validated 1–7 (ISO 8601: Monday=1, Sunday=7) at handler layer
- NOTIFY stubs in place in `Activate`, `Deactivate`, `CreatePeriod`, `UpdatePeriod`,
  `DeletePeriod` — to be replaced with Redis XADD in Phase 3e
- Period create/update/delete only fires event if parent schedule `is_active = true`

---

## Device service architecture (Phase 3b complete)

### Overview

Standalone Go binary. No HTTP server. Communicates via MQTT and PostgreSQL only.
api-service and device-service are decoupled — share PostgreSQL and Redis but no
direct service-to-service calls.

### Startup sequence

1. Load config from env
2. Connect to appdb (GORM), metricsdb (pgx), Redis, Mosquitto
3. Warm cache from appdb
4. Create Redis consumer group if not exists (Phase 3e)
5. Drain pending stream entries (Phase 3e)
6. Start Redis stream consumer goroutine (Phase 3e)
7. Start MQTT telemetry subscriber (Phase 3c)
8. Start per-room control loop ticker goroutines, staggered (Phase 3d)
9. Start periodic cache refresh ticker, staggered (Phase 3d)
10. Block on SIGTERM/SIGINT → graceful shutdown via context cancel + WaitGroup

### Cache architecture

**`Store`** — top-level container. `sync.RWMutex` protects the two maps.
Map-level lock held only for map reads/writes (inserting/deleting pointers).
Field-level access protected by per-struct mutexes.

```go
type Store struct {
    mu      sync.RWMutex
    rooms   map[uuid.UUID]*RoomCache  // room_id → cache
    devices map[string]*DeviceCache   // hw_id   → cache
}
```

**`RoomCache`** — full runtime state for one room. `Mu sync.RWMutex` exported so
callers can hold read lock across multi-field reads (control loop tick). Write lock
held by ingestion and stream consumer for field updates.

Pre-computed at warm/reload (never recomputed at tick time):
- `Location *time.Location` — resolved from `UserTimezone` string
- `ActuatorHwIDs map[string][]string` — actuator_type → []hw_id in this room
- `SchedulePeriodCache.StartMinutes/EndMinutes int` — parsed from "HH:MM"
- `SchedulePeriodCache.DaysOfWeek [8]bool` — indexed by ISO day (1-7)

Runtime-only (never persisted):
- `LatestReadings map[string][]TimestampedReading` — sensor_type → readings, trimmed on write
- `ActuatorStates map[string]bool` — last commanded state, initialized false
- `LastActivePeriod *SchedulePeriodCache` — used for grace period logic

**`DeviceCache`** — device metadata. `mu sync.RWMutex` unexported. `RoomID *uuid.UUID`
is the only mutable field — accessed via `GetRoomID()`/`SetRoomID()` wrapper methods.
`Sensors` and `Actuators` are immutable after creation.

### Cache warm

`repository/appdb.WarmCache(store)` — called once at startup before any goroutines.

1. Fetch all room IDs
2. Filter to owned rooms via `store.OwnsRoom()` (always true Phase 3, hash-filtered Phase 5)
3. Bulk fetch with `IN` clause: rooms+timezone (JOIN users), desired states, active periods
   (JOIN schedules WHERE is_active=true), devices, sensors, actuators
4. Build index maps in Go, assemble per room
5. Apply pre-computations: resolve timezone, parse time strings, build DaysOfWeek bitmask,
   build ActuatorHwIDs from devices

`repository/appdb.ReloadRoom(store, roomID)` — single room targeted queries, called by
stream consumer and periodic refresh. Preserves `LatestReadings`, `ActuatorStates`,
`LastActivePeriod` from existing cache entry.

`repository/appdb.ReloadDevice(store, hwID)` — upserts or evicts device cache entry.
Called by stream consumer on `device_changed` events.

### GORM Raw+Scan pattern (device-service only)

device-service never uses GORM model-based queries (`Find`, `First`, `Save`). All appdb
queries use `.Raw().Scan()` into unexported local scan structs. GORM tag required for
array types: `gorm:"type:integer[]"` on `pq.Int64Array` fields.

```go
type activePeriodRow struct {
    DaysOfWeek pq.Int64Array `gorm:"type:integer[]"`
    // ...
}
```

### Control loop (Phase 3d — planned)

One goroutine per room. Staggered at startup. On each tick:

1. Acquire `rc.Mu.RLock()` for duration of evaluation
2. Determine effective mode and targets via `resolveEffectiveState`:
   - Manual override active and not expired → desired state mode + targets
   - Active schedule period matches current day/time → period mode + targets
   - Grace period (within 1 minute of last period end, no midnight crossing) → last period
   - None → mode OFF, nil targets
3. If mode OFF → command all actuators off
4. If mode AUTO → for each entry in targets map:
   - Average fresh readings from `LatestReadings[sensorType]` (stale = older than ~3 ticks)
   - Compare against target ± deadband
   - If command needed, publish to `devices/{hw_id}/cmd` for each hw_id in `ActuatorHwIDs`
5. Release lock, update `ActuatorStates`

**Scheduler** manages goroutine lifecycle — `activeRooms map[uuid.UUID]context.CancelFunc`.
New rooms added dynamically (no stagger). Deleted rooms cancelled via cancel func.

### MQTT ingestion (Phase 3c — next)

Subscribe `devices/+/telemetry` — wildcard, every instance receives every message.
Handler: parse hw_id → `store.Device(hwID)` → if nil drop → `store.OwnsRoom(roomID)`
→ if false drop → `rc.Mu.Lock()` → append to `LatestReadings`, trim stale →
`rc.Mu.Unlock()` → write to TimescaleDB via raw pgx with snapshotted `room_id`.

Paho delivers messages via its own internal goroutines. Handler is called per message.
Sequential by default — fine at simulator tick rates. Worker pool pattern deferred to
Phase 5 if high-frequency telemetry is needed.

### Redis stream events (Phase 3e — planned)

**Stream:** `device-service:events`
**Consumer group per instance:** `device-service-group-{instanceID}` (one group per
instance — broadcast model, every instance sees every event, filters on `ownsRoom()`)

**Event payloads:**
```
desired_state_changed  → { type, room_id }
room_config_changed    → { type, room_id }
schedule_changed       → { type, room_id }  # only fired when schedule is_active = true
room_created           → { type, room_id }
room_deleted           → { type, room_id }
device_changed         → { type, hw_id, room_id (nullable), previous_room_id (nullable) }
```

**`device_changed` logic:**
- `previous_room_id` set → instance owning that room removes device
- `room_id` set → instance owning that room adds device
- Both null → device deleted, all instances evict from device store

**api-service events package** (`internal/events/events.go`) — thin helper with one
exported function per event type wrapping `XADD`. Redis client injected into room,
device, and schedule services (not repositories). Services fire events after successful
DB writes. Period create/update/delete only fires `schedule_changed` if `is_active = true`.

**Stream consumer startup:** drain pending entries (unacknowledged from previous run)
before switching to live entries (`>`). `XACK` after successful cache update.

**Stream retention:** `MaxLen ~10000, Approx: true` on every `XADD`.

### Periodic cache refresh (Phase 3d — planned)

Safety net for missed stream events. One staggered ticker per room. Interval configurable
via env (default 60-90 seconds). Calls `ReloadRoom` — preserves runtime fields.

### Phase 5 partitioning design (reference)

- **Unit:** room (natural unit of control loop work)
- **Mechanism:** consistent hashing — `hash(room_id) % num_instances`
- **Instance discovery:** Redis TTL heartbeat keys `device-service:instance:{instanceID}`
- **`OwnsRoom()` stub:** Phase 3 always returns true — Phase 5 replaces with hash check
- **Cache warm:** fetch all room IDs → filter via `OwnsRoom()` → load only owned rooms
- **MQTT:** every instance receives every message, drops unowned via nil Device lookup
- **Stream:** each instance has own consumer group, checks `ownsRoom()` before acting
- **`device_changed`:** both `room_id` and `previous_room_id` in payload so both affected
  instances can update immediately without waiting for periodic refresh

### Logging package (`internal/logging/logging.go`)

- `LogSummary(store)` — startup summary (room + device counts) — runs in main.go
- `LogStore(store)` — summary line per room
- `LogFullStore(store)` — full field detail per room
- `LogDevices(store)` — all devices with sensors and actuators
- `LogRoom(rc)` / `LogFullRoom(rc)` — single room summary / full detail
- `LogDevice(dc)` — single device with sensors and actuators

---

## Simulator architecture (implemented)

### Config system

Two-file separation: infrastructure (env vars) vs simulation definition (YAML).

**Template files:**
- `config/templates/rooms.yaml` — room templates (behaviour type, base values)
- `config/templates/devices.yaml` — device templates (sensors, actuators, noise, offset)

Available device templates: `climate-sensor`, `temp-sensor`, `humidity-sensor`,
`air-quality-sensor`, `heater`, `humidifier`, `temp-heater`, `humidity-humidifier`,
`climate-controller`

**Simulation files** — reference templates by id. Selected via `--simulation=name` flag.

**Identity generation (deterministic):**
```
email:   sim-{simulation_name}-user-{000}@{domain}
hw_ids:  sim-{simulation_name}-{user_idx}-{room_idx}-{device_idx}
```

### Provisioning

Login-first auth (register only on 401). Idempotent — handles 409 via lookup maps.
Credentials file written to `/app/config/credentials/{sim-name}.txt` for interactive groups.

### Schedule provisioning (future — Phase 4)

Schedules not yet provisioned by simulator. Currently added manually via Postman.
Phase 4 will add schedule definition to simulation YAML and provisioning sequence.
`cache-test.yaml` simulation has reference Postman bodies in `notes/cache-test-postman.md`.

---

## Redis stream (api-service side — Phase 3e)

**Cleanup needed before Phase 3e:**
- Create `api-service/internal/events/events.go` with `NotifyX` functions
- Inject Redis client into room, device, schedule service constructors in `main.go`
- Replace NOTIFY stubs in repositories with service-layer `events.NotifyX()` calls
- Remove `initializers/` package, move Redis connection to `internal/connect/redis.go`

---

## Shared module cleanup (deferred)

Fold `shared/models` into `api-service/internal/models/` and `shared/db` into
`api-service/internal/connect/`. Delete `shared/` module. Update all import paths.
Remove `go.work` shared entry and `replace` directive. Verify simulator-service doesn't
use shared models directly (likely uses HTTP responses only).

---

## api-service Redis streams events injection (deferred to Phase 3e)

When Phase 3e starts, add Redis client to these service constructors in `main.go`:
- `room.NewService(roomRepo, rdb)`
- `device.NewService(deviceRepo, roomRepo, rdb)`
- `schedule.NewService(scheduleRepo, roomRepo, rdb)`

And create `internal/events/events.go` with stream name constant and `NotifyX` functions.

---

## Future client (Phase 5)

A Go server-side rendered web app (`web-service`) using `html/template`. Thin
presentation layer that calls api-service internally and serves HTML to the browser.
No JS framework required — minimal JS for auto-refreshing sensor readings only.
Accessible to reviewers without terminal or Postman setup. Added as a Docker Compose
service exposing a host port. Static files served from `/static/`.

**Alternative considered:** CLI tool (`ccctl`) with interactive navigation. Web app
preferred for reviewer accessibility and portfolio impact.

---

## Connection conventions (locked in)

All services use `internal/connect/` package with separate files per connection type.
Package name `connect` — reads naturally as `connect.Postgres(...)`, `connect.Redis(...)`.
Internal Docker hostnames and ports always hardcoded in connect functions:
- `host=postgres port=5432`
- `host=timescaledb port=5432`
- `host=redis port=6379`
- `host=mosquitto port=1883`
- `http://api-service:8080`

Env vars for ports are host-machine mappings only.

---

## Docker Compose

- `docker-compose.yml` — infrastructure only
- `deployments/docker-compose.services.yml` — application services overlay
- All services have healthchecks, `depends_on` uses `condition: service_healthy`
- `device-service` has no HTTP healthcheck — add in Phase 3c when MQTT is running
- Dockerfile `context: ..` points to repo root so `shared/` is accessible
- `replace` directive in `go.mod` for local `shared` module resolution

## Makefile

Key targets:
```
up / down / down-hard / rebuild        — project lifecycle
infra- prefix                          — infrastructure only
rebuild- prefix                        — individual service rebuild
restart-device                         — restart device-service without rebuild (cache re-warm)
logs- prefix                           — service logs
shell- prefix                          — container shell access
go- prefix                             — go vet, go build
test-api- prefix                       — Newman test suites
simulator- prefix                      — simulator lifecycle
mqtt- prefix                           — mosquitto_sub subscriptions for debugging
docker- prefix                         — raw Docker utilities
mosquitto-passwd                       — regenerate passwd file from .env
```

---

## REST API endpoints (all implemented)

```
POST   /api/v1/auth/register
POST   /api/v1/auth/login
POST   /api/v1/auth/refresh
POST   /api/v1/auth/logout
GET    /api/v1/users/me
DELETE /api/v1/users/me

GET    /api/v1/rooms
POST   /api/v1/rooms
GET    /api/v1/rooms/:id
PUT    /api/v1/rooms/:id
DELETE /api/v1/rooms/:id

GET    /api/v1/rooms/:id/desired-state
PUT    /api/v1/rooms/:id/desired-state

GET    /api/v1/devices
POST   /api/v1/devices
GET    /api/v1/devices/:id
PUT    /api/v1/devices/:id
DELETE /api/v1/devices/:id
GET    /api/v1/rooms/:id/devices

GET    /api/v1/rooms/:id/schedules
POST   /api/v1/rooms/:id/schedules
GET    /api/v1/schedules/:id
PUT    /api/v1/schedules/:id
DELETE /api/v1/schedules/:id
PATCH  /api/v1/schedules/:id/activate
PATCH  /api/v1/schedules/:id/deactivate

POST   /api/v1/schedules/:id/periods
GET    /api/v1/schedules/:id/periods
PUT    /api/v1/schedule-periods/:id
DELETE /api/v1/schedule-periods/:id
```

---

## Business logic rules

### desired_states
- Every room always has exactly one row — created in same transaction as room
- `api-service` writes when user makes direct control request — sets `manual_override_until`
- `device-service` writes on schedule period transition — does NOT set `manual_override_until`
- `manual_override_until = NULL` means scheduler controls the room
- `manual_override_until = 9999-12-31T23:59:59Z` means indefinite override
- API contract: client sends `"indefinite"`, timestamp string, or `null`

### Capability checks
- `mode = AUTO` + `target_temp NOT NULL` → room must have temperature sensor + heater
- `mode = AUTO` + `target_hum NOT NULL` → room must have humidity sensor + humidifier
- `mode = AUTO` + both targets NULL → reject with 422
- `mode = OFF` → always valid
- Sensor and actuator may be on **separate devices** — EXISTS + EXISTS pattern

### Device capability conflicts
- DELETE or unassign blocked if device is sole provider of a capability required by
  room's active desired_state or active schedule periods
- Inactive schedules ignored — conflict checked at activation time instead
- Error response includes `hint` field

### Schedule period rules
- `days_of_week` values must be 1–7 (ISO 8601)
- `end_time` must be strictly greater than `start_time` — no midnight crossing
- Overlap rejected per shared day
- Capability validation only at activation

---

## Dependency directions (non-negotiable)

```
auth     → user
device   → room
schedule → room
```

Never reversed. `room` never imports `device` or `schedule`.
`device` never imports `schedule`. `schedule` never imports `device`.

---

## HTTP status code conventions (locked in)

- `400` — malformed request
- `404` — not found or unauthorized (ownership gate — no information leakage)
- `409` — state conflict
- `422` — semantically invalid request
- `500` — internal server error (never leak details)

---

## GORM style (locked in — api-service)

- Chain `.Error` directly
- No `result` variable
- Pointer returns: repo methods return `*Model, error`
- `Save` for updates — full replacement, always fetch before mutating
- Handler owns field filtering via request structs
- Empty slices initialized explicitly

**Responses:**
- Never serialize full model structs — use `gin.H`
- No `user_id` in resource responses
- Sensors/actuators serialized as `[]string` of type names
- Timestamps formatted as RFC3339 UTC
- `start_time`/`end_time` formatted as `"HH:MM"`

---

## Development phases

| Phase | Scope | Status |
|---|---|---|
| 1 | Repo scaffold, Docker Compose, DB schema | ✅ Done |
| 2 | `api-service` — all REST endpoints | ✅ Done |
| 3a | `simulator-service` scaffold | ✅ Done |
| 3b | `device-service` scaffold — cache warm | ✅ Done |
| 3c | `device-service` ingestion — MQTT + TimescaleDB writes | 🔄 Next |
| 3d | `device-service` control loop — bang-bang, commands, scheduler | Pending |
| 3e | `device-service` stream — Redis stream consumer, cache invalidation | Pending |
| 4 | Scenario simulator — drift and physics room models, schedule provisioning | Pending |
| 5 | CI, architecture diagrams, README, NGINX, web client, tests | Pending |
| Later | Cloud deployment | Pending |

---

## Future features (noted, not yet designed)

- Device connection status — Redis hash keyed by `hw_id`, `online`/`offline` + last-seen
  timestamp. device-service writes on telemetry arrival and watchdog timeout. api-service
  reads for `GET /devices/:id/status` or folds into device response.
- Sensor calibration offset — nullable `offset NUMERIC(5,2) DEFAULT 0` on `sensors`,
  applied at query time not write time
- `activatable` bool field on schedule list responses
- Admin API — `DELETE /admin/devices/:hw_id`, `POST /admin/devices/:hw_id/blacklist`
- NGINX load balancing for api-service (Phase 5)
- Multiple device-service instances with room-level partitioning (Phase 5)
- Connection pool size configuration via env — `DB_POOL_SIZE` per service
- Prometheus metrics + Grafana dashboard (Phase 5)
- Android app (Kotlin + Jetpack Compose) — learning sequence established in notes