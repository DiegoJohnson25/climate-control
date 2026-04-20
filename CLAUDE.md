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
- `feat/device-service-ingestion` ✅ — merged to main
  - Transport-agnostic ingestion via `ingestion.Source` interface
  - `mqtt.Source` implements `ingestion.Source` — Paho wrapper + telemetry adapter
  - `ingestion.Process` — cache `LatestReadings` update + TimescaleDB batch write
  - `metricsdb.Repository` — `WriteSensorReadings` (pgx batch) + `WriteControlLogEntry` stub
  - `cache.Store` — `assignedPartitions`/`numPartitions` stubbed for Phase 6
  - `cache.DeviceCache` — `Sensors` and `Actuators` changed to maps for O(1) lookup
  - `config` — `StaleThreshold` from `CONTROL_STALE_THRESHOLD_SECONDS` env (default 90s)
  - metricsdb migrations: `sensor_readings` updated (added `raw_value`), `room_control_logs` added
  - Comment refactor pass applied across all services per `.claude/COMMENTING_STYLE.md`

**Active branch:** `feat/device-service-control`

**Planned branches:**
- `feat/device-service-control` — control loop, bang-bang logic, command publishing, scheduler
- `feat/device-service-stream` — Redis stream consumer, cache invalidation, periodic refresh

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
│   │   └── main.go               # connections, cache warm, ingestion wiring, signal handling
│   ├── internal/
│   │   ├── config/
│   │   │   └── config.go         # env loading — StaleThreshold, MQTT credentials
│   │   ├── connect/
│   │   │   ├── postgres.go       # Postgres() → *gorm.DB
│   │   │   ├── timescale.go      # Timescale() → *pgxpool.Pool
│   │   │   └── redis.go          # Redis() → *redis.Client
│   │   ├── cache/
│   │   │   └── cache.go          # Store, RoomCache, DeviceCache + entry types
│   │   ├── appdb/
│   │   │   └── repository.go     # WarmCache, ReloadRoom, ReloadDevice
│   │   ├── metricsdb/
│   │   │   └── repository.go     # WriteSensorReadings, WriteControlLogEntry
│   │   ├── logging/
│   │   │   └── logging.go        # LogSummary, LogStore, LogFullStore, LogDevices etc.
│   │   ├── ingestion/
│   │   │   ├── source.go         # Source interface — transport contract
│   │   │   ├── ingestion.go      # Ingestor, Process, Run, Stop, trimStale
│   │   │   └── message.go        # TelemetryMessage, Reading
│   │   ├── mqtt/
│   │   │   ├── client.go         # Paho wrapper — Subscribe, Publish, Disconnect
│   │   │   └── source.go         # mqtt.Source implements ingestion.Source
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
├── .claude/
│   └── COMMENTING_STYLE.md       # Go commenting conventions — used by Claude Code refactor
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

## Commenting style (locked in)

All Go code follows `.claude/COMMENTING_STYLE.md`. Key rules:
- Package comments in the primary file of every package, starting with "Package <n>."
- Godoc comments on exported symbols that need them — start with symbol name, end with period.
  Trivial constructors, self-documenting DTOs, and error sentinels with descriptive messages
  do not need doc comments.
- File-level section breaks use three-line 75-dash bars with a title case label.
- No structural dividers inside functions — plain label comment only when genuinely non-obvious.
- `// TODO Phase N:` for phase-tagged items, `// TODO:` for open-ended items, always with explanation.
- Inline comments only for struct field annotations.

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
| Event streaming | Redis Streams | `stream:device_assignments` — Phase 3e |
| MQTT client | Eclipse Paho Go | aliased as `pahomqtt` to avoid package name collision |
| Broker | Mosquitto 2.x | Auth enabled, ACL per username |
| App DB | PostgreSQL 17 | Internal port 5432, host port 5433 |
| Time-series DB | TimescaleDB 2.25.2-pg17 | Internal port 5432, host port 5434 |
| Cache | In-process memory | `device-service` only — `sync.RWMutex` per struct |
| Migrations | golang-migrate | Auto-run on `docker compose up` |
| Containers | Docker Compose | |
| Testing | Newman + Postman | `make test-api-integration`, `make test-api-smoke` |
| Kafka | Apache Kafka (KRaft) | Phase 6 — telemetry pipeline, franz-go client, 24 partitions |
| MQTT bridge | New Go service | Phase 6 — routes Mosquitto telemetry to Kafka, owns `hw_id → room_id` resolution |

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

# Device service control
CONTROL_STALE_THRESHOLD_SECONDS=90   # readings older than this are dropped from LatestReadings
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
    value     NUMERIC NOT NULL,
    raw_value NUMERIC NOT NULL  -- pre-offset value; identical to value until calibration offsets implemented
);

SELECT create_hypertable('sensor_readings', 'time',
    chunk_time_interval => INTERVAL '1 day');

CREATE INDEX idx_sensor_readings_sensor_time ON sensor_readings(sensor_id, time DESC);
CREATE INDEX idx_sensor_readings_room_time   ON sensor_readings(room_id, time DESC);

CREATE TABLE room_control_logs (
    time               TIMESTAMPTZ NOT NULL,
    room_id            UUID NOT NULL,
    avg_temp           NUMERIC,
    avg_hum            NUMERIC,
    mode               TEXT CHECK (mode IN ('OFF', 'AUTO')),
    target_temp        NUMERIC,
    target_hum         NUMERIC,
    control_source     TEXT CHECK (control_source IN ('manual_override', 'schedule', 'grace_period', 'none')),
    heater_cmd         SMALLINT CHECK (heater_cmd IN (0, 1)),      -- null if room has no heater
    humidifier_cmd     SMALLINT CHECK (humidifier_cmd IN (0, 1)),  -- null if room has no humidifier
    reading_count_temp SMALLINT,       -- number of fresh readings that contributed to avg_temp
    reading_count_hum  SMALLINT,       -- number of fresh readings that contributed to avg_hum
    schedule_period_id UUID            -- set when control_source is 'schedule' or 'grace_period'
);

SELECT create_hypertable('room_control_logs', 'time',
    chunk_time_interval => INTERVAL '1 day');

CREATE INDEX idx_room_control_logs_room_time ON room_control_logs(room_id, time DESC);
CREATE INDEX idx_room_control_logs_period    ON room_control_logs(schedule_period_id, time DESC)
    WHERE schedule_period_id IS NOT NULL;
```

`heater_cmd` and `humidifier_cmd` are SMALLINT (0/1) not BOOLEAN so that `AVG()` produces
a duty cycle fraction (0.0–1.0) at any time bucket resolution without casting.

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

## Device service architecture

### Overview

Standalone Go binary. No HTTP server. Communicates via MQTT and PostgreSQL only.
api-service and device-service are decoupled — share PostgreSQL and Redis but no
direct service-to-service calls.

### Startup sequence

1. Load config from env
2. Connect to appdb (GORM), metricsdb (pgx), Redis, Mosquitto
3. Warm cache from appdb
4. Start telemetry ingestion via `ingestor.Run(ctx)` — fatal if source fails to start
5. Create Redis consumer group if not exists (Phase 3e)
6. Drain pending stream entries (Phase 3e)
7. Start Redis stream consumer goroutine (Phase 3e)
8. Start per-room control loop ticker goroutines, staggered (Phase 3d)
9. Start periodic cache refresh ticker, staggered (Phase 3d)
10. Block on SIGTERM/SIGINT → graceful shutdown via context cancel + WaitGroup

### Timing constants

| Parameter | Value | Rationale |
|---|---|---|
| Simulator tick interval | 30s | Matches realistic ESP32 polling rate |
| Stale threshold | 90s | 3-reading window per sensor; device silent >90s = unavailable |
| Control loop tick | 30s | Matches sensor rate — no benefit evaluating faster |
| Cache refresh interval | 5min | Safety net for missed stream events |
| Stream consumer block timeout | 5s | Near real-time cache updates |

### Cache architecture

**`Store`** — top-level container. `sync.RWMutex` protects the two maps.
Map-level lock held only for map reads/writes (inserting/deleting pointers).
Field-level access protected by per-struct mutexes.

```go
type Store struct {
    mu                 sync.RWMutex
    rooms              map[uuid.UUID]*RoomCache  // room_id → cache
    devices            map[string]*DeviceCache   // hw_id   → cache
    assignedPartitions map[int32]struct{}         // Phase 6: Kafka partitions owned by this instance
    numPartitions      int32                      // Phase 6: total Kafka topic partitions
}
```

`assignedPartitions` and `numPartitions` are unused until Phase 6. `OwnsRoom()` returns
true for all rooms until Kafka populates them.

**`RoomCache`** — full runtime state for one room. `Mu sync.RWMutex` exported so
callers can hold read lock across multi-field reads (control loop tick). Write lock
held by ingestion and stream consumer for field updates.

Pre-computed at warm/reload (never recomputed at tick time):
- `Location *time.Location` — resolved from `UserTimezone` string
- `ActuatorHwIDs map[string][]string` — actuator_type → []hw_id in this room
- `SchedulePeriodCache.StartMinutes/EndMinutes int` — parsed from "HH:MM"
- `SchedulePeriodCache.DaysOfWeek [8]bool` — indexed by ISO day (1-7)

Runtime-only (never persisted):
- `LatestReadings map[string][]TimestampedReading` — sensor_type → readings, trimmed on ingestion
- `ActuatorStates map[string]bool` — last commanded state, initialized false
- `LastActivePeriod *SchedulePeriodCache` — used for grace period logic

**`DeviceCache`** — device metadata. `mu sync.RWMutex` unexported. `RoomID *uuid.UUID`
is the only mutable field — accessed via `GetRoomID()`/`SetRoomID()` wrapper methods.
`Sensors` and `Actuators` are maps keyed by type for O(1) lookup:

```go
Sensors   map[string]SensorEntry   // measurement_type → entry
Actuators map[string]ActuatorEntry // actuator_type    → entry
```

### Transport-agnostic ingestion

Telemetry ingestion uses a `Source` interface so the transport can be swapped without
touching ingestion logic:

```go
type Source interface {
    Start(ctx context.Context, handler func(context.Context, TelemetryMessage)) error
    Stop()
}
```

- `mqtt.Source` — active Phase 3–5. Subscribes to Mosquitto, parses payload, calls handler.
- `kafka.Source` — Phase 6. Consumes from Kafka topic, constructs same `TelemetryMessage`.
- `ingestion.Process` is unchanged in both cases.

`TelemetryMessage` carries `HwID`, `RoomID` (nil if unassigned), `Readings`, `Timestamp`.
Messages for unassigned devices (nil `RoomID`) are dropped — no cache update, no DB write.
Room context is required for sensor data to be meaningful.

`ingestion.Process` drop conditions:
- Silent: unknown hw_id, unassigned device, reading type not in sensor map
- Warning logged: room not owned by instance, room not in store (cache inconsistency)

### Cache warm

`appdb.WarmCache(store)` — called once at startup before any goroutines.

1. Fetch all room IDs
2. Filter to owned rooms via `store.OwnsRoom()` (always true Phase 3, Kafka-filtered Phase 6)
3. Bulk fetch with `IN` clause: rooms+timezone (JOIN users), desired states, active periods
   (JOIN schedules WHERE is_active=true), devices, sensors, actuators
4. Build index maps in Go, assemble per room
5. Apply pre-computations: resolve timezone, parse time strings, build DaysOfWeek bitmask,
   build ActuatorHwIDs from devices

`appdb.ReloadRoom(store, roomID)` — single room targeted queries, called by
stream consumer and periodic refresh. Preserves `LatestReadings`, `ActuatorStates`,
`LastActivePeriod` from existing cache entry.

`appdb.ReloadDevice(store, hwID)` — upserts or evicts device cache entry.
Called by stream consumer on device assignment events.

### GORM Raw+Scan pattern (device-service only)

device-service never uses GORM model-based queries (`Find`, `First`, `Save`). All appdb
queries use `.Raw().Scan()` into unexported local scan structs. GORM tag required for
array types: `gorm:"type:integer[]"` on `pq.Int64Array` fields.

### Control loop (Phase 3d — next)

One goroutine per room. Staggered at startup. On each tick:

1. Acquire `rc.Mu.RLock()` for duration of evaluation
2. Determine effective mode and targets via `resolveEffectiveState`:
   - Manual override active and not expired → desired state mode + targets → source: `manual_override`
   - Active schedule period matches current day/time → period mode + targets → source: `schedule`
   - Grace period (within 1 minute of last period end) → last period → source: `grace_period`
   - None → mode OFF, nil targets → source: `none`
3. If mode OFF → command all actuators off
4. If mode AUTO → for each entry in targets map:
   - Filter fresh readings from `LatestReadings[sensorType]` (stale = older than stale threshold)
   - Average fresh readings, compare against target ± deadband
   - If command needed, publish to `devices/{hw_id}/cmd` QoS 2 for each hw_id in `ActuatorHwIDs`
5. Release lock, update `ActuatorStates`
6. Write `room_control_logs` row via `metricsdb.WriteControlLogEntry`

**Scheduler** manages goroutine lifecycle — `activeRooms map[uuid.UUID]context.CancelFunc`.
Staggered at startup. Rooms added/removed dynamically via stream consumer events.

**Periodic cache refresh** — safety net for missed stream events. One ticker per room,
staggered. Default 5 minutes. Calls `ReloadRoom` — preserves runtime fields.

### Redis stream events (Phase 3e — planned)

**Stream:** `stream:device_assignments`
**Consumer group:** `device-service` (one group, all instances share it)
**Consumer name:** per-instance hostname

Events written by api-service on:
- Device assigned to room
- Device unassigned from room
- Desired state changed
- Room config (deadbands) changed
- Schedule activated/deactivated
- Room created/deleted

**Stream consumer startup:** drain pending entries (unacknowledged from previous run)
before switching to live entries (`>`). `XACK` after successful cache update.

**Cleanup needed before Phase 3e:**
- Create `api-service/internal/events/events.go` with `NotifyX` functions
- Inject Redis client into room, device, schedule service constructors in `main.go`
- Replace NOTIFY stubs in repositories with service-layer `events.NotifyX()` calls
- Remove `initializers/` package, move Redis connection to `internal/connect/redis.go`

### MQTT topics

- Telemetry: `devices/{hw_id}/telemetry` — QoS 1, published by devices, subscribed by device-service
- Commands: `devices/{hw_id}/cmd` — QoS 2, published by device-service, subscribed by devices

Commands flow directly from device-service to Mosquitto permanently — bypassing Kafka even
in Phase 6. The Kafka bridge handles telemetry ingestion only; commands are point-to-point
and don't benefit from Kafka's fan-out or durability properties.

### Logging package

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

## api-service Redis streams events injection (deferred to Phase 3e)

When Phase 3e starts, add Redis client to these service constructors in `main.go`:
- `room.NewService(roomRepo, rdb)`
- `device.NewService(deviceRepo, roomRepo, rdb)`
- `schedule.NewService(scheduleRepo, roomRepo, rdb)`

And create `internal/events/events.go` with stream name constant and `NotifyX` functions.

---

## Shared module cleanup (deferred)

Fold `shared/models` into `api-service/internal/models/` and `shared/db` into
`api-service/internal/connect/`. Delete `shared/` module. Update all import paths.
Remove `go.work` shared entry and `replace` directive. Verify simulator-service doesn't
use shared models directly (likely uses HTTP responses only).

---

## Future client (Phase 5)

A Go server-side rendered web app (`web-service`) using `html/template`. Thin
presentation layer that calls api-service internally and serves HTML to the browser.
Accessible to reviewers without terminal or Postman setup. Added as a Docker Compose
service exposing a host port. Static files served from `/static/`.

**Transport:** polling every 15-30 seconds — stateless, no sticky sessions needed.
SSE documented as a future enhancement requiring sticky sessions or a dedicated streaming
service. At 30s tick rate, polling is perceptibly equivalent to SSE.

**Historical graphs:**
- Single graph per room, time range selector: 1h / 6h / 24h / 7d
- Short ranges: raw `sensor_readings` data
- Long ranges: `time_bucket` downsampled aggregates from `room_control_logs`
- `heater_cmd`/`humidifier_cmd` overlaid as duty cycle (AVG produces 0.0–1.0 fraction)
- New endpoints needed: `GET /rooms/:id/climate` (current snapshot) +
  `GET /rooms/:id/climate/history?window=6h` (time series, server picks resolution)

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

Planned (Phase 5):
```
GET    /api/v1/rooms/:id/climate          — current averaged readings + control state
GET    /api/v1/rooms/:id/climate/history  — time series, ?window=1h|6h|24h|7d
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
ingestion → (nothing — transport-agnostic)
mqtt     → ingestion
```

Never reversed. `room` never imports `device` or `schedule`.
`device` never imports `schedule`. `schedule` never imports `device`.
`ingestion` never imports `mqtt` or `kafka`.

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
| 3c | `device-service` ingestion — MQTT + TimescaleDB writes | ✅ Done |
| 3d | `device-service` control loop — bang-bang, commands, scheduler | 🔄 Next |
| 3e | `device-service` stream — Redis stream consumer, cache invalidation | Pending |
| 4 | NGINX — api-service horizontal scaling | Pending |
| 5 | Web client | Pending |
| 6 | Kafka + MQTT bridge + partition-aware device-service scaling | Pending |
| 7 | CI, architecture diagrams, README polish | Pending |
| 8 | Prometheus + Grafana observability | Pending |
| 9 | Cloud deployment (AWS) | Pending |
| 10 | Kubernetes + load testing | Pending |

Phases 1–7 are the primary baseline. Phases 8–10 are independent enhancements.

---

## Phase 6 — Kafka + MQTT bridge architecture

### Why Kafka

device-service currently receives telemetry via a shared MQTT subscription. With multiple
instances this breaks — two instances receiving the same message for the same room produce
incoherent caches and incorrect control decisions. Kafka solves this via partition-based
ownership: each room hashes deterministically to one partition, and each partition is owned
by exactly one device-service instance. The Kafka consumer group protocol manages assignment
automatically.

### Kafka cluster

- KRaft mode — no ZooKeeper required
- Single topic: `telemetry`, 24 partitions
- 24 partitions chosen for headroom — supports up to 24 device-service instances without
  repartitioning
- Partition key: `room_id` bytes via murmur2 hash
- Go client: franz-go (`github.com/twmb/franz-go`)

### MQTT bridge (new Go service)

The bridge is the only service that talks to Mosquitto for telemetry in Phase 6.
device-service no longer subscribes to Mosquitto directly.

Responsibilities:
- Subscribe to `devices/+/telemetry` on Mosquitto
- Maintain local `map[string]{RoomID, DeviceID}` cache keyed on `hw_id`
- Warm cache from DB on startup
- Consume `stream:device_assignments` Redis Stream as consumer group `bridge` — independent
  from the `device-service` consumer group, same stream, separate offset
- Publish to Kafka topic `telemetry` with key = `room_id` bytes
- Stamp `{room_id, device_id}` onto the Kafka message payload for downstream consumers

The bridge is stateless routing logic — no control loop, no DB writes, no business logic.

### device-service Phase 6 changes

`mqtt.Source` in `main.go` is replaced by `kafka.Source`. `ingestion.Process` is unchanged —
`TelemetryMessage` struct defined in Phase 3c is constructed identically by both adapters.

`kafka.Source` implements `ingestion.Source`:
- Runs a franz-go `PollFetches` loop internally
- Constructs `TelemetryMessage` from Kafka record payload
- Calls `handler(ctx, msg)` — same interface as `mqtt.Source`

**Partition ownership callbacks:**

`OnPartitionsAssigned(partitions)`:
1. Update `Store.assignedPartitions` under write lock
2. Query DB for all rooms
3. Filter to rooms where `murmur2(room_id) % numPartitions` is in assigned set
4. Warm only those rooms — existing entries for retained partitions untouched

`OnPartitionsRevoked(partitions)`:
1. Evict rooms belonging to revoked partitions from `Store`
2. Update `Store.assignedPartitions` under write lock

**`OwnsRoom()` becomes real:**
```go
func (s *Store) OwnsRoom(roomID uuid.UUID) bool {
    p := murmur2([]byte(roomID.String())) % uint32(s.numPartitions)
    _, owned := s.assignedPartitions[int32(p)]
    return owned
}
```
Uses franz-go's exported murmur2 — identical function to what the bridge producer uses.
Coupling is real but contained and documented.

**Startup sequence change:**
Cannot warm cache before joining consumer group — partition assignment is asynchronous.
Warm happens inside `OnPartitionsAssigned` callback, before broker delivers messages for
those partitions. Order: connect → register callbacks → join group → callbacks fire → warm
→ begin processing.

### Commands still bypass Kafka

Actuator commands flow device-service → Mosquitto → ESP32 directly, permanently.
The bridge handles telemetry ingestion only. Routing commands through Kafka would add
latency and complexity with no benefit — commands are point-to-point, low-volume, and
don't need Kafka's fan-out or durability properties.

Multiple device-service instances connect to Mosquitto with the same credentials but
distinct client IDs (`device-service-{HOSTNAME}`) — Mosquitto allows this. The ACL grants
publish rights by username, not client ID.

---

## Future features (noted, not yet designed)

- Device connection status — Redis hash keyed by `hw_id`, `online`/`offline` + last-seen
  timestamp. device-service writes on telemetry arrival and watchdog timeout. api-service
  reads for `GET /devices/:id/status` or folds into device response. Scope: devices assigned
  to a room only (unassigned devices have no instance owner and produce no telemetry).
- Sensor/actuator `enabled` boolean — `enabled BOOLEAN NOT NULL DEFAULT true` on both
  `sensors` and `actuators` tables. `PATCH /sensors/:id` and `PATCH /actuators/:id` to
  toggle. Ingestion skips disabled sensors. Control loop skips disabled sensors when
  averaging. Capability checks filter to enabled sensors/actuators only. Cache warm
  loads `enabled` field. Redis stream event on toggle for cache invalidation.
- Sensor calibration offset — nullable `offset NUMERIC(5,2) DEFAULT 0` on `sensors`.
  Applied at ingestion time: `value = raw + offset`, both stored in `sensor_readings`.
  Cache warm loads offset into `SensorEntry`. Current implementation stores identical
  `value` and `raw_value` as a placeholder.
- `room_control_logs` write — `metricsdb.WriteControlLogEntry` stub is in place.
  Wired up in Phase 3d when the control loop is written.
- WiFi provisioning for physical ESP32 devices — SoftAP or BLE provisioning flow.
  Backend-agnostic: device self-generates `hw_id` at first boot, user registers via API.
  Document in README under "Hardware deployment".
- Command acknowledgement — device publishes to `devices/{hw_id}/cmd/ack` after acting
  on a command. device-service subscribes, updates `ActuatorStates` only on confirmed ack.
  Currently fire-and-forget (optimistic state update). Document as production consideration
  in README.
- `activatable` bool field on schedule list responses
- Admin API — `DELETE /admin/devices/:hw_id`, `POST /admin/devices/:hw_id/blacklist`
- Connection pool size configuration via env — `DB_POOL_SIZE` per service
- Android app (Kotlin + Jetpack Compose) — learning sequence established in notes