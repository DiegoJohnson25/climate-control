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

**Active branch:** `feat/device-service`

**Next up:** `feat/device-service-scaffold` — file structure, config loading, cache warm from DB

**Planned branches:**
- `feat/device-service-scaffold` — file structure, config, cache warm
- `feat/device-service-ingestion` — MQTT subscription, telemetry parsing, TimescaleDB writes
- `feat/device-service-control` — control loop, bang-bang logic, command publishing
- `feat/device-service-listen-notify` — LISTEN/NOTIFY cache invalidation

---

## Repo structure (domain-first — decided and locked in)

```
climate-control/
├── api-service/
│   ├── cmd/
│   │   └── main.go
│   ├── internal/
│   │   ├── user/
│   │   │   ├── handler.go        # POST /auth/register, GET /users/me, DELETE /users/me
│   │   │   ├── service.go        # registration, password hashing, deletion
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
│   ├── cmd/
│   │   └── main.go               # flag parsing, run/teardown modes, signal handling
│   ├── config/
│   │   ├── templates/
│   │   │   ├── rooms.yaml        # room templates (noise models)
│   │   │   └── devices.yaml      # device templates (sensors, actuators, noise, offset)
│   │   ├── simulations/
│   │   │   ├── default.yaml      # single user, full capability room
│   │   │   ├── multi-room.yaml
│   │   │   ├── multi-user.yaml
│   │   │   ├── sensor-only.yaml
│   │   │   └── multi-sensor.yaml
│   │   └── credentials/          # gitignored, written at runtime for interactive groups
│   │       └── .gitkeep
│   ├── internal/
│   │   ├── api/
│   │   │   └── client.go         # HTTP client for api-service — extractable
│   │   ├── config/
│   │   │   └── config.go         # env + YAML loading, template resolution, validation
│   │   ├── mqtt/
│   │   │   └── client.go         # Paho wrapper — Publish, Subscribe, Disconnect
│   │   ├── provisioning/
│   │   │   └── provisioning.go   # bootstrap sequence, identity generation, credentials file
│   │   └── simulator/
│   │       └── simulator.go      # publish loop, staggered goroutines per device, RoomState
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
| ORM | GORM | App DB only |
| Time-series queries | pgx raw SQL | TimescaleDB |
| Auth | JWT | golang-jwt library |
| Refresh tokens | Redis | go-redis/v9 |
| Rate limiting | Redis | Applied at router level |
| MQTT client | Eclipse Paho Go | `device-service` and `simulator-service` only |
| Broker | Mosquitto 2.x | Auth enabled, ACL per username |
| App DB | PostgreSQL 17 | Internal port 5432, host port 5433 |
| Time-series DB | TimescaleDB 2.25.2-pg17 | Internal port 5432, host port 5434 |
| Cache | In-process memory | `device-service` control loop only |
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

**Important:** Internal Docker port for both postgres and timescaledb is `5432`.
Connection strings inside Docker always use `port=5432`, `host=postgres`,
`host=timescaledb`, `host=redis`, `host=mosquitto`.

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
- `days_of_week` uses `pq.Int64Array \`gorm:"type:integer[]"\``
- `schedule_periods.start_time` / `end_time` — `string` on model, `TEXT` in DB
- `TargetHumidity` shortened to `TargetHum` on `models.SchedulePeriod`
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
    rooms   *room.Repository  // for room ownership checks — device → room import direction
}
```

- `DeviceWithCapabilities` struct defined in `device/repository.go` — embeds `models.Device`,
  adds `Sensors []models.Sensor` and `Actuators []models.Actuator`
- All list/get methods return `DeviceWithCapabilities` via bulk fetch (3 queries always)
- Sensors and actuators always serialize as `[]string` of type names in responses
- Devices created unassigned, assigned to rooms via `PUT /devices/:id`
- `hw_id` checked via `CheckHwIDAvailability` before insert
- Capability conflict check (`HasCapabilityConflictAfterRemoval`) blocks delete/unassign
  if it would leave room's active desired_state or active schedule_periods without
  required capability. Inactive schedules intentionally ignored.
- `activeSchedulePeriodsHaveConflict` lives permanently in `device.Repository` —
  moving it to `schedule.Repository` would create a circular import
  (`schedule → room`, `device → room`, `device` must never import `schedule`)

---

## Schedule architecture (implemented)

```go
type Service struct {
    schedules *Repository
    rooms     *room.Repository  // for room ownership checks — schedule → room import direction
}
```

- Schedules are inactive by default — `is_active = false` on create
- `PATCH /schedules/:id/activate` — atomic transaction: deactivates existing active
  schedule for the room, activates the target. DB enforced via partial unique index
  `one_active_schedule_per_room`
- `PATCH /schedules/:id/deactivate` — symmetric with activate. `ErrAlreadyInactive`
  guard prevents no-op. Allows stopping scheduler control without replacing the schedule.
- Capability validation only at activation time — inactive schedules are stored future
  intent. Period create/update does NOT check room capabilities.
- `PeriodsHaveCapability` in `schedule.Repository` — called at activation, checks all
  periods in the schedule against the room's current devices
- Period overlap detection uses PostgreSQL `&&` array overlap operator on `days_of_week`
- Midnight crossing NOT supported — `end_time` must be > `start_time`. Enforced at
  handler (422) and DB level (`CHECK (end_time > start_time)`). Users create two
  periods for overnight windows.
- `start_time`/`end_time` stored as `TEXT` (`"HH:MM"` format). Lexicographic comparison
  works correctly for zero-padded time strings.
- `days_of_week` validated 1–7 (ISO 8601: Monday=1, Sunday=7) at handler layer
- `NOTIFY` stubs in place in `Activate`, `Deactivate`, `CreatePeriod`, `UpdatePeriod`,
  `DeletePeriod` — to be uncommented when device-service LISTEN/NOTIFY is implemented
- `activatable` bool field on schedule list responses deferred to Phase 5 polish

---

## Simulator architecture (implemented)

### Config system

Two-file separation: infrastructure (env vars) vs simulation definition (YAML).

**Env vars** — `SIMULATOR_EMAIL`, `SIMULATOR_PASSWORD`, `MQTT_DEVICE_USERNAME`, `MQTT_DEVICE_PASSWORD`

**YAML config** — volume mounted at `/app/config` from `simulator-service/config/` on host.
Three directories: `templates/`, `simulations/`, `credentials/`

**Template files** (shared, defined once):
- `config/templates/rooms.yaml` — room templates with model type and base values
- `config/templates/devices.yaml` — device templates with sensors, actuators, noise, offset

**Simulation files** (reference templates by id):
- `config/simulations/{name}.yaml` — topology: user groups, room counts, device counts
- Selected via `--simulation=name` flag, defaults to `default`
- Can define `template_overrides` to override shared templates locally

**Config loading flow:** env → load room/device templates → load simulation file → merge overrides → resolve template references → validate (no duplicate name_prefix) → return flat `Config` struct. Template concept is fully dissolved after resolution — downstream packages never see template ids.

### Identity generation (deterministic, simulation-scoped)

```
email:  sim-{simulation_name}-user-{000}@{domain}
rooms:  {name_prefix}-{local_index}
devices: {room_name}-{device_prefix}-{local_index}
hw_ids:  sim-{simulation_name}-{user_idx}-{room_idx}-{device_idx}
```

Simulation name embedded in all identities — different simulations never collide in the DB. Identities are fully deterministic from config so restarts are idempotent without stored state.

### Provisioning (idempotency)

On startup: login-first auth (register only on 401), fetch existing rooms and devices once per user into lookup maps, create rooms/devices handling 409 via map lookup. `AssignDevice` called unconditionally — idempotent. Credentials file written to `/app/config/credentials/{sim-name}.txt` for interactive user groups only.

### Teardown

`--mode=teardown` flag. Regenerates same deterministic credentials from config, logs in as each user, calls `DELETE /api/v1/users/me`. Cascade handles all rooms, devices, sensors, actuators, schedules, desired_states. TimescaleDB sensor_readings rows for those sensor UUIDs become orphaned (no FK by design) — acceptable.

**Workflow:**
- Normal restart → idempotency handles it, no teardown needed
- Config value changes (base_temp, noise, tick interval) → just restart
- Topology changes (sensors, actuators, room structure) → teardown first
- Changed config before teardown → `make down-hard && make up`

### Publish loop

One goroutine per device. Stagger offset = `tickInterval * deviceIndex / totalDevices` — spreads all publishes evenly across tick window, eliminates thundering herd at scale. Actuator-only devices subscribe to cmd topic but publish no telemetry. Signal handling via SIGTERM/SIGINT → context cancel → all goroutines exit cleanly → WaitGroup unblocks → process exits.

### Room model types

**Phase 3a (implemented):** `noise` — Gaussian noise around base values. No evolving room state. Value per tick = `base + rand.NormFloat64()*noise_stddev + offset`. Base values live on room model, noise/offset live on device template.

**Phase 3b (planned):**
- `drift` — noise + drift block with per-measurement `rate` and `target`
- `physics` — noise + physics block with `thermal_mass`, `thermal_conductance`, `external_temp_profile` (type: sinusoidal, mean, amplitude, period_hours)

All types share `base_temp`, `base_humidity`, and noise block. Calculator interface assigned once at startup based on model type. Runtime state (`RoomState`) is separate from config and evolves per tick independently.

### api/client.go

Outbound HTTP client for api-service. Unexported in Phase 3a, designed to be extracted to its own package later. Methods: `Register`, `Login`, `DeleteMe`, `CreateRoom`, `ListRooms`, `CreateDevice`, `ListDevices`, `AssignDevice`. `ErrConflict` sentinel for 409 responses.

---

## Mosquitto configuration (implemented)

Auth enabled — `allow_anonymous false`. Three users:
- `device` — shared by all simulators and ESP32s. Publish `devices/+/telemetry`, subscribe `devices/+/cmd`
- `device-service` — subscribe `devices/+/telemetry`, publish `devices/+/cmd`
- `healthcheck` — hardcoded username and password both `healthcheck`, publish `healthcheck` topic only

Password file at `deployments/mosquitto/passwd` — generated via `make mosquitto-passwd`, committed to repo. Regenerate after any MQTT password change in `.env`.

ACL file at `deployments/mosquitto/acl` — plain text, no extension, no comments (Mosquitto parser sensitive to formatting).

Healthcheck credentials hardcoded in compose healthcheck command — not in `.env`. Avoids leaking that the healthcheck password matches other credentials.

---

## Control loop (device-service — to be implemented)

**Trigger:** time-windowed evaluation, not event-driven per message. Each room has its own ticker goroutine. Staggered across rooms same as simulator publish stagger — avoids thundering herd on DB and MQTT at scale.

**Latest readings cache:** incoming telemetry updates `LatestReadings map[string]TimestampedReading` on the room cache. Control loop ticker reads latest values, filters stale readings (older than ~3 tick intervals), computes average per sensor type, runs bang-bang logic.

**Bang-bang with hysteresis:**
```
current_value = avg(all sensors of type in room from LatestReadings)
if current_value < target - deadband  →  publish cmd state=true
if current_value > target + deadband  →  publish cmd state=false
else                                  →  hold (no command published)
```

**RoomCache struct:**
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
    LatestReadings    map[string]TimestampedReading  // sensor type → {value, time}
}
```

Grace period: if no active period found and time is within 60 seconds of last active period's `end_time`, use last period's targets — prevents relay toggling at period boundaries.

---

## MQTT payload conventions (locked in)

**Telemetry** (device → device-service, QoS 1):
```json
{
    "hw_id": "sim-default-0-0-0",
    "readings": [
        {"type": "temperature", "value": 21.5},
        {"type": "humidity", "value": 45.0}
    ]
}
```

**Command** (device-service → device, QoS 2):
```json
{
    "actuator_type": "heater",
    "state": true
}
```

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

## MQTT conventions (locked in)

- Telemetry: `devices/{hw_id}/telemetry` — QoS 1
- Commands: `devices/{hw_id}/cmd` — QoS 2
- Topics keyed on `hw_id`, NOT database UUID
- device-service resolves `hw_id → device_id → sensor_id` internally from cache
- Devices never need to know their DB-assigned UUIDs
- Only `device-service`, `simulator-service`, ESP32s connect to Mosquitto
- Paho aliased as `pahomqtt` in mqtt packages to avoid package name collision

---

## LISTEN/NOTIFY conventions (to be implemented in device-service)

Channels:
- `room_config_changed` — payload: room_id string. Fire after rooms deadband update.
- `desired_state_changed` — payload: room_id string. Fire after desired_state update.
- `schedule_changed` — payload: room_id string. Fire after schedule activate/deactivate
  or period create/update/delete.

Rules:
- `api-service` fires NOTIFY only, never LISTEN
- `device-service` LISTENs on a dedicated persistent pgx connection (not pool)
- NOTIFY inside transactions fires only on commit — correct behaviour by design
- TODO stubs already in place in room, device, and schedule repositories

---

## Sensor readings conventions (locked in)

- `room_id` snapshotted at write time by device-service
- `room_id` nullable — null when device was unassigned at time of reading
- No FK on `sensor_id` or `room_id` — integrity enforced by device-service
- Two indexes: `(sensor_id, time DESC)` and `(room_id, time DESC)`
- Raw values stored always — sensor offset (future feature) applied at query time
- Multiple readings from one telemetry message written in one transaction (same timestamp)

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
- `ErrCapabilityConflict` responses include a `hint` field

---

## Docker Compose

- `docker-compose.yml` — infrastructure only
- `deployments/docker-compose.services.yml` — application services overlay
- All services have healthchecks, `depends_on` uses `condition: service_healthy`
- Dockerfile `context: ..` points to repo root so `shared/` is accessible
- `replace` directive in `api-service/go.mod` for local `shared` module resolution
- Simulator service: `SIMULATOR_SIMULATION` env var selects simulation file, command in compose overrides Dockerfile CMD

## Makefile

Refactored with consistent prefix groups. Full cheatsheet in personal notes. Key groups:

```
up / down / down-hard / rebuild      — project lifecycle
infra- prefix                        — infrastructure only
rebuild- prefix                      — individual service rebuild
logs- prefix                         — service logs
shell- prefix                        — container shell access
go- prefix                           — go vet, go build
test-api- prefix                     — Newman test suites
simulator- prefix                    — simulator lifecycle (see simulator commands)
mqtt- prefix                         — mosquitto_sub subscriptions for debugging
docker- prefix                       — raw Docker utilities (ps, stats, prune, volumes)
mosquitto-passwd                     — regenerate passwd file from .env
```

---

## Postman test collections

```
tests/postman/
├── climate-control-integration.collection.json   # full suite, ordered, requires make down-hard first
├── climate-control-smoke.collection.json         # repeatable CI/CD suite, uses $timestamp for isolation
├── climate-control-manual.collection.json        # dev tool, auth at collection level, no scripts
├── integration.environment.json
├── smoke.environment.json
└── manual.environment.json
```

- `make test-api-integration` — runs integration suite via Newman (requires fresh DB)
- `make test-api-smoke` — runs smoke suite via Newman (safe against live DB)
- Manual collection: auth set at collection level (`Bearer {{access_token}}`),
  Login test script auto-populates `access_token` and `refresh_token`

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
- `mode = AUTO` + both targets NULL → reject with 422
- `mode = OFF` → always valid

### Device capability conflicts
- DELETE or unassign blocked if device is sole provider of a capability required by
  room's active desired_state or active schedule periods
- Inactive schedules ignored — conflict checked at activation time instead
- Error response includes `hint` field with user-facing guidance

### Schedule period rules
- `days_of_week` values must be 1–7 (ISO 8601)
- `end_time` must be strictly greater than `start_time` — no midnight crossing
- Overlap rejected: `new_start < existing_end AND new_end > existing_start` per shared day
- Capability validation only at activation — not at period create/update
- Schedule name unique per room — `UNIQUE(room_id, name)`

### Schedule activation
- Atomic transaction — deactivates existing active schedule, activates new one
- Partial unique index `one_active_schedule_per_room` enforces at DB level
- Capability check runs before activation — rejects if room lacks required sensors/actuators

---

## Dependency directions (non-negotiable)

```
auth     → user       (login needs user lookup)
device   → room       (ownership check, capability queries)
schedule → room       (ownership check, capability validation on activation)
```

Never reversed. `room` never imports `device` or `schedule`.
`device` never imports `schedule`. `schedule` never imports `device`.

---

## HTTP status code conventions (locked in)

- `400` — malformed request: bad JSON, missing required fields, wrong types,
  `ShouldBindJSON` failure, bad path UUID
- `404` — not found or unauthorized (ownership gate — no information leakage)
- `409` — state conflict: duplicate name, already active/inactive, period overlap,
  capability conflict on device removal
- `422` — semantically invalid request: AUTO mode with no targets, invalid time range
  (`end <= start`), days out of 1–7, unrecognized sensor/actuator type,
  `resolveManualOverride` parse failure
- `500` — internal server error (never leak details)

---

## GORM style (locked in)

- Chain `.Error` directly: `r.db.WithContext(ctx).Where(...).First(&rm).Error`
- No `result` variable — `result.Error` pattern not used
- Pointer returns: repo methods return `*Model, error` — nil pointer on not found
- `Save` for updates — full replacement, always fetch before mutating
- Handler owns field filtering via request structs — keeps `Save` safe
- Empty slices initialized explicitly — never return nil slices in responses

**Responses:**
- Never serialize full model structs — use `gin.H` response helper functions
- No `user_id` in resource responses — implicit from JWT
- Sensors/actuators serialized as `[]string` of type names, not model objects
- Timestamps formatted as RFC3339 UTC in responses
- `start_time`/`end_time` formatted as `"HH:MM"` in responses

---

## Development phases

| Phase | Scope | Status |
|---|---|---|
| 1 | Repo scaffold, Docker Compose, DB schema | ✅ Done |
| 2 | `api-service` — all REST endpoints | ✅ Done |
| 3a | `simulator-service` scaffold | ✅ Done |
| 3b | `device-service` scaffold — cache warm | 🔄 Next |
| 3c | `device-service` ingestion — MQTT + TimescaleDB writes | Pending |
| 3d | `device-service` control loop — bang-bang, commands | Pending |
| 3e | `device-service` LISTEN/NOTIFY — cache invalidation | Pending |
| 4 | Scenario simulator — drift and physics room models | Pending |
| 5 | CI, architecture diagrams, README, NGINX, tests | Pending |
| Later | Cloud deployment | Pending |

---

## Future features (noted, not yet designed)

- Sensor calibration offset — nullable `offset NUMERIC(5,2) DEFAULT 0` on `sensors`,
  applied at query time not write time
- `activatable` bool field on schedule list responses — computed from capability check,
  for client UI to show greyed-out/warning schedules
- Admin API — `DELETE /admin/devices/:hw_id`, `POST /admin/devices/:hw_id/blacklist`,
  `GET /admin/devices` — separate auth, future branch
- Web client and Android app (currently Postman)
- NGINX load balancing for api-service (Phase 5)
- Multiple device-service instances with shared MQTT subscriptions (Phase 5)
- Prometheus metrics + Grafana dashboard (Phase 5)