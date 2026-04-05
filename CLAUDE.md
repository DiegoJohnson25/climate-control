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

**Active branch:** `feat/api-service-schedules`

**Last thing actually done:** Rooms and devices domains complete and Postman verified.
All endpoints working. About to start schedules and schedule periods.

---

## Repo structure (domain-first — decided and locked in)

```
climate-control/
├── api-service/
│   ├── cmd/
│   │   └── main.go
│   ├── internal/
│   │   ├── user/
│   │   │   ├── handler.go        # POST /auth/register, GET /users/me
│   │   │   ├── service.go        # registration, password hashing
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
│   │   │   ├── repository.go
│   │   │   ├── pg_errors.go
│   │   │   └── errors.go
│   │   ├── device/
│   │   │   ├── handler.go        # device CRUD + list by room
│   │   │   ├── service.go
│   │   │   ├── repository.go     # includes DeviceWithCapabilities type
│   │   │   ├── pg_errors.go
│   │   │   └── errors.go
│   │   ├── schedule/             # to be written
│   │   │   ├── handler.go
│   │   │   ├── service.go
│   │   │   ├── repository.go
│   │   │   ├── pg_errors.go
│   │   │   └── errors.go
│   │   ├── router/
│   │   │   └── router.go         # route registration, middleware chain
│   │   ├── health/
│   │   │   └── health.go         # plain function, no struct
│   │   ├── ctxkeys/
│   │   │   └── keys.go           # shared Gin context key constants
│   │   ├── config/
│   │   │   └── config.go         # typed config struct, Load() function
│   │   └── initializers/
│   │       └── redis.go          # Redis connection init
│   ├── Dockerfile
│   └── go.mod
├── device-service/
│   ├── cmd/main.go
│   ├── internal/
│   │   ├── cache/
│   │   ├── control/
│   │   ├── mqtt/
│   │   ├── repository/
│   │   └── scheduler/
│   ├── Dockerfile
│   └── go.mod
├── simulator-service/
│   ├── cmd/main.go
│   ├── internal/
│   │   ├── mqtt/
│   │   └── scenario/
│   ├── Dockerfile
│   └── go.mod
├── shared/
│   ├── models/           # GORM structs — schema contract for both services
│   │   ├── user.go
│   │   ├── room.go
│   │   ├── device.go
│   │   ├── sensor.go
│   │   ├── actuator.go
│   │   ├── desired_state.go
│   │   ├── schedule.go
│   │   └── schedule_period.go
│   └── db/
│       ├── postgres.go   # GORM appdb connection
│       └── timescale.go  # pgx TimescaleDB connection pool
├── firmware/
│   └── esp32/
├── deployments/
│   ├── docker-compose.services.yml
│   ├── docker-compose.prod.yml
│   ├── mosquitto/mosquitto.conf
│   └── nginx/nginx.conf
├── migrations/
│   ├── appdb/
│   └── metricsdb/
├── docs/
├── tests/
│   └── postman/
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
- Service struct fields named by concept: `users`, `tokens`, `rooms`, `devices`
- Method receivers use single letter: `(s *Service)`, `(h *Handler)`, `(r *Repository)`

---

## Tech stack

| Layer | Technology | Notes |
|---|---|---|
| Language | Go 1.25.0 | |
| HTTP framework | Gin | `api-service` only |
| ORM | GORM | App DB only |
| Time-series queries | pgx raw SQL | TimescaleDB |
| Auth | JWT | golang-jwt library |
| Refresh tokens | Redis | go-redis/v9 |
| Rate limiting | Redis | Applied at router level |
| MQTT client | Eclipse Paho Go | `device-service` and `simulator-service` only |
| Broker | Mosquitto | |
| App DB | PostgreSQL 17 | Internal port 5432, host port 5433 |
| Time-series DB | TimescaleDB 2.25.2-pg17 | Internal port 5432, host port 5434 |
| Cache | In-process memory | `device-service` control loop only |
| Migrations | golang-migrate | Auto-run on `docker compose up` |
| Containers | Docker Compose | |

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
MQTT_DEVICE_SERVICE_PASSWORD=localdev
MQTT_DEVICE_PASSWORD=localdev

JWT_SECRET=localdev-replace-with-32-plus-chars-in-prod
JWT_ACCESS_TTL_MINUTES=15
JWT_REFRESH_TTL_DAYS=7

API_PORT=8080

SIMULATOR_EMAIL=simulator@local.dev
SIMULATOR_PASSWORD=localdev
```

**Important:** Internal Docker port for both postgres and timescaledb is `5432`.
Connection strings inside Docker always use `port=5432`, `host=postgres`,
`host=timescaledb`, `host=redis`.

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
    target_humidity       NUMERIC(5,2) CHECK (target_humidity BETWEEN 0 AND 100),
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
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX one_active_schedule_per_room
ON schedules(room_id) WHERE is_active = true;

CREATE TABLE schedule_periods (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    schedule_id     UUID NOT NULL REFERENCES schedules(id) ON DELETE CASCADE,
    name            TEXT,
    days_of_week    INTEGER[] NOT NULL,
    start_time      TIME NOT NULL,
    end_time        TIME NOT NULL,
    mode            TEXT NOT NULL DEFAULT 'AUTO' CHECK (mode IN ('OFF', 'AUTO')),
    target_temp     NUMERIC(5,2) CHECK (target_temp BETWEEN 5 AND 40),
    target_humidity NUMERIC(5,2) CHECK (target_humidity BETWEEN 0 AND 100),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK (end_time > start_time),
    CHECK (mode = 'OFF' OR target_temp IS NOT NULL OR target_humidity IS NOT NULL),
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
- `days_of_week` uses `pq.Int64Array \`gorm:"type:integer[]"\``
- No constraint tags — migrations handle all constraints
- `desired_states` table name set via `TableName()` method
- GORM used for appdb only — TimescaleDB uses raw pgx
- Shared models never have GORM association fields (e.g. no `Sensors []Sensor` on Device)
  — enriched types live in the domain package that needs them

---

## Auth architecture (implemented)

`auth` imports `user` — one directional. `user` knows nothing about `auth`.

```go
type Service struct {
    users  *user.Repository
    tokens *Repository
}
```

Wiring in `main.go`:
```go
userRepo    := user.NewRepository(db)
userSvc     := user.NewService(userRepo)
authRepo    := auth.NewRepository(rdb, cfg.JWTRefreshTTLDays)
authSvc     := auth.NewService(userRepo, authRepo, cfg.JWTSecret, cfg.JWTAccessTTLMinutes, cfg.JWTRefreshTTLDays)
userHandler := user.NewHandler(userSvc)
authHandler := auth.NewHandler(authSvc)
```

---

## Room architecture (implemented)

```go
type Service struct {
    rooms *Repository
}
```

- `desired_states` row created atomically with room in same transaction
- Capability queries (HasTemperatureCapability, HasHumidityCapability) live permanently
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
    rooms   *room.Repository  // for room ownership checks — device → room import direction
}
```

- `DeviceWithCapabilities` struct defined in `device/repository.go` — embeds `models.Device`,
  adds `Sensors []models.Sensor` and `Actuators []models.Actuator`
- All list/get methods return `DeviceWithCapabilities` via bulk fetch (3 queries always)
- Sensors and actuators always serialize as `[]string` of type names in responses,
  never as full model objects — clients don't need sensor/actuator UUIDs
- Devices created unassigned, assigned to rooms via `PUT /devices/:id`
- `hw_id` checked via `CheckHwIDAvailability` before insert:
  - same user → `ErrAlreadyOwned`
  - different user → `ErrHwIDTaken`
- Capability conflict check (`HasCapabilityConflictAfterRemoval`) blocks delete/unassign
  if it would leave room's active desired_state or active schedule_periods without
  required capability. Inactive schedules intentionally ignored.
- Admin device management (forced deregistration, blacklisting) deferred to future branch

---

## Dependency directions (non-negotiable)

```
auth    → user       (login needs user lookup)
device  → room       (ownership check for GET /rooms/:id/devices)
schedule → room      (ownership check, capability validation on period create/update)
```

Never reversed. `room` never imports `device`. `room` never imports `schedule`.
`device` never imports `schedule`. `schedule` never imports `device`.

---

## Import circular import resolution

Capability queries that cross domain lines live in the package that owns the
business rule, not the package that owns the table:
- `room.Repository` owns `HasTemperatureCapability` / `HasHumidityCapability`
  (queries device/sensor/actuator tables — permanent home, not temporary)
- `device.Repository` owns `activeSchedulePeriodsHaveConflict`
  (queries schedule/schedule_period tables — moves to `schedule.Repository` when
  that package exists)

---

## MQTT conventions (locked in)

- Telemetry: `devices/{hw_id}/telemetry` — QoS 1
- Commands: `devices/{hw_id}/cmd` — QoS 2
- Topics keyed on `hw_id`, NOT database UUID
- device-service resolves `hw_id → device_id → sensor_id` internally from cache
- Devices never need to know their DB-assigned UUIDs
- Only `device-service`, `simulator-service`, ESP32s connect to Mosquitto

---

## LISTEN/NOTIFY conventions (to be implemented in Phase 3)

Channels:
- `room_config_changed` — payload: room_id string. Fire after rooms deadband update.
- `desired_state_changed` — payload: room_id string. Fire after desired_state update.
- `schedule_changed` — payload: room_id string. Fire after schedule activate/deactivate
  or period create/update/delete.

Rules:
- `api-service` fires NOTIFY only, never LISTEN
- `device-service` LISTEN on a dedicated persistent pgx connection (not pool)
- NOTIFY inside transactions fires only on commit — correct behaviour by design
- TODO stubs already in place in room and device repositories

---

## Sensor readings conventions (locked in)

- `room_id` snapshotted at write time by device-service — accurate historical room
  metrics even after device reassignment
- `room_id` nullable — null when device was unassigned at time of reading
- No FK on `sensor_id` or `room_id` — integrity enforced by device-service
- Two indexes: `(sensor_id, time DESC)` for per-sensor diagnostics,
  `(room_id, time DESC)` for room-level history queries
- Raw values stored always — sensor offset (future feature) applied at query time,
  never baked into stored values

---

## Layering conventions

| Layer | Responsibility |
|---|---|
| `handler.go` | HTTP only — parse request, validate input, call service, write response |
| `service.go` | Business logic — orchestrate repository calls, enforce rules |
| `repository.go` | Data access only — all DB/Redis queries live here |
| `errors.go` | Sentinel errors for the domain |
| `pg_errors.go` | Unexported `isUniqueViolation` helper, one per package |

**Chain:** `handler → service → repository`. Handlers never call repositories directly.

**Context:** `context.Context` is the first parameter on all repository and service
methods. Handlers pass `c.Request.Context()`. GORM calls always use `.WithContext(ctx)`.

**Error handling in handlers:**
- `c.JSON` + `return` — NOT `c.AbortWithStatusJSON` (Abort is for middleware only)
- `ShouldBindJSON` not `BindJSON`
- Never leak internal error details in 500 responses
- Sentinel errors translated to specific HTTP status codes
- `ErrCapabilityConflict` responses include a `hint` field with user-facing guidance

**GORM style:**
- Chain `.Error` directly: `r.db.WithContext(ctx).Where(...).First(&rm).Error`
- No `result` variable — `result.Error` pattern not used (no `RowsAffected` needed)
- Pointer returns: repo methods return `*Model, error` — nil pointer on not found
- `Save` for updates — full replacement, always fetch before mutating
- Empty slices initialized explicitly — never return nil slices in responses

**Responses:**
- Never serialize full model structs — construct response fields explicitly
- No `user_id` in resource responses — implicit from JWT
- Sensors/actuators serialized as `[]string` of type names, not model objects

---

## REST API endpoints

```
POST   /api/v1/auth/register
POST   /api/v1/auth/login
POST   /api/v1/auth/refresh
POST   /api/v1/auth/logout
GET    /api/v1/users/me

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

POST   /api/v1/schedules/:id/periods
GET    /api/v1/schedules/:id/periods
PUT    /api/v1/schedule-periods/:id
DELETE /api/v1/schedule-periods/:id
```

All endpoints except auth register/login/refresh require `Authorization: Bearer <access_token>`.
All responses JSON. All timestamps UTC.

---

## Business logic rules

### desired_states
- Every room always has exactly one row — created in same transaction as room
- `api-service` writes when user makes direct control request — sets `manual_override_until`
- `device-service` writes on schedule period transition — does NOT set `manual_override_until`
- `manual_override_until = NULL` means scheduler controls the room
- `manual_override_until = 9999-12-31T23:59:59Z` means indefinite override
- API contract: client sends `"indefinite"`, timestamp string, or `null`

### Validation for desired_states and schedule_periods
- `mode = AUTO` + `target_temp NOT NULL` → room must have temperature sensor + heater
- `mode = AUTO` + `target_humidity NOT NULL` → room must have humidity sensor + humidifier
- `mode = AUTO` + both targets NULL → reject, use `OFF`
- `mode = OFF` → always valid

### Device capability conflicts
- DELETE or unassign (room_id → null or different room) blocked if device is sole
  provider of a capability required by room's active desired_state or active schedule periods
- Inactive schedules ignored — conflict checked at activation time instead
- Error response includes `hint` field with user-facing guidance
- Future: `activatable` bool field on schedule list responses (Phase 5 polish)

### Device provisioning
- Devices created via API by user (no manufacturer registry in this project)
- `hw_id` self-generated by ESP32 at first boot, stored in flash
- `CheckHwIDAvailability` run before insert — distinguishes owned vs taken
- Future: admin endpoints for forced deregistration and device blacklisting
- Future: `blacklisted_devices` table with `hw_id`, `reason`, `blacklisted_at`

### Schedule period overlap
- Rejected at API layer: `new_start < existing_end AND new_end > existing_start`
- Per-day — only periods sharing at least one day need checking

### Schedule midnight crossing
- API transparently splits into two periods:
  - Period A: original days, `start_time` to `23:59`
  - Period B: days +1, `00:00` to `end_time`
  - Sunday (7) wraps to Monday (1) for Period B

### Schedule activation
- Atomic transaction — deactivates existing active schedule, activates new one
- Partial unique index `one_active_schedule_per_room` enforces at DB level

### PUT vs PATCH convention
- PUT: full replacement of all mutable fields — client always sends all of them
- PATCH: targeted state transitions only — `PATCH /schedules/:id/activate` is the
  only PATCH in this API

### Rooms are user-scoped
- All queries filter by `user_id` from JWT claims

---

## Control loop (device-service — Phase 3)

Event-driven — triggered by incoming MQTT telemetry. Bang-bang control with hysteresis.

```go
type RoomCache struct {
    DeadbandTemp      float64
    DeadbandHumidity  float64
    DesiredState      DesiredState
    ActivePeriods     []SchedulePeriod
    UserTimezone      string
    ActuatorStates    map[string]bool
    LastActivePeriod  *SchedulePeriod
    LastPeriodEndTime time.Time
}
```

Control logic:
```
current_value = avg(all sensors of type in room)
if current_value < target - deadband  →  ON
if current_value > target + deadband  →  OFF
else                                  →  hold (hysteresis)
```

MQTT telemetry payload from device:
```json
{
    "hw_id": "esp32-abc123",
    "readings": [
        {"type": "temperature", "value": 21.5},
        {"type": "humidity", "value": 45.0}
    ]
}
```

device-service resolves: `hw_id → device_id → sensor_id` from cache, writes to
TimescaleDB with both `sensor_id` and `room_id` snapshotted.

---

## Days of week convention

ISO 8601: Monday = 1, Sunday = 7.
Go's `time.Weekday()` uses Sunday = 0 — must convert explicitly.
Write a test case specifically for Sunday.

---

## Measurement unit conventions

| Type | Unit |
|---|---|
| `temperature` | Celsius |
| `humidity` | % relative humidity (0-100) |
| `air_quality` | PPM |

---

## Docker Compose

- `docker-compose.yml` — infrastructure only
- `deployments/docker-compose.services.yml` — application services overlay
- All services have healthchecks, `depends_on` uses `condition: service_healthy`
- Dockerfile `context: ..` points to repo root so `shared/` is accessible
- `replace` directive in `api-service/go.mod` for local `shared` module resolution

## Makefile targets

```
up              — bring all services up (no rebuild)
rebuild         — bring all services up with --build
down            — bring all services down
down-hard       — bring all services down, wipe volumes
infra           — infrastructure only
infra-down      — infrastructure down
infra-down-hard — infrastructure down, wipe volumes
rebuild-api     — rebuild api-service only
rebuild-device  — rebuild device-service only
rebuild-simulator — rebuild simulator-service only
logs            — tail all logs
logs-api        — tail api-service logs
logs-device     — tail device-service logs
logs-postgres   — tail postgres logs
logs-redis      — tail redis logs
logs-mqtt       — tail mosquitto logs
shell-api       — shell into api-service container
shell-postgres  — psql into postgres
shell-timescale — psql into timescaledb
shell-redis     — redis-cli into redis
ps              — show container status
build-api       — build api-service image directly
vet             — go vet all modules
build           — go build all modules
```

---

## Simulator service (Phase 4)

On startup:
1. Register user using `SIMULATOR_EMAIL`/`SIMULATOR_PASSWORD` (or login if exists)
2. POST devices via REST API with `device_type = 'simulator'`
3. Assign devices to rooms via PUT
4. Publish MQTT telemetry using `hw_id` from API response
5. device-service resolves `hw_id → sensor_id` from cache

---

## Development phases

| Phase | Scope | Status |
|---|---|---|
| 1 | Repo scaffold, Docker Compose, DB schema | ✅ Done |
| 2 | `api-service` — all REST endpoints | 🔄 In progress |
| 3 | `device-service` — MQTT, control loop, cache | Pending |
| 4 | `simulator-service` — API registration, MQTT telemetry | Pending |
| 5 | CI, architecture diagrams, README, tests | Pending |
| Later | NGINX load balancing, cloud deployment | Pending |

---

## Future features (noted, not yet designed)

- Sensor calibration offset — nullable `offset NUMERIC(5,2) DEFAULT 0` on `sensors`,
  applied at query time not write time
- `activatable` bool field on schedule list responses — computed from capability check,
  for client UI to show greyed-out/warning schedules without a DB round trip
- Admin API — `DELETE /admin/devices/:hw_id`, `POST /admin/devices/:hw_id/blacklist`,
  `GET /admin/devices` — separate auth, future branch
- Web client and Android app (currently Postman)