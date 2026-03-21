# CLAUDE.md — Climate Control Project

This file is the authoritative reference for Claude Code working in this repository.
Read it fully before writing any code. All architectural decisions documented here
are final unless explicitly revisited.

---

## Project overview

A distributed IoT climate control system backend in Go. Two independent services
communicate via PostgreSQL and MQTT to manage room climate via ESP32 relay devices.
Built as a portfolio project — fully demonstrable without hardware via a simulator
service.

---

## Repo structure

```
climate-control/
├── api-service/
│   ├── cmd/main.go
│   ├── internal/
│   │   ├── auth/
│   │   ├── handlers/
│   │   ├── middleware/
│   │   ├── models/
│   │   └── repository/
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
│   ├── models/       # GORM structs — schema contract for both services
│   └── config/       # Typed config struct loaded from env
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
│   ├── architecture.md
│   ├── api.md
│   └── diagrams/
├── tests/
│   └── postman/
│       ├── climate-control.postman_collection.json
│       └── climate-control.postman_environment.json
├── docker-compose.yml       # Infrastructure only — postgres, timescaledb, redis, mosquitto
├── go.work                  # Workspace linking all three service modules + shared
├── .env                     # Never committed — see .env.example
├── .env.example
├── .gitignore
├── README.md
├── CLAUDE.md
└── .github/
    ├── workflows/ci.yml
    └── pull_request_template.md
```

---

## Go module structure

- Each service (`api-service`, `device-service`, `simulator-service`) has its own
  `go.mod` — they are independent modules with separate dependency graphs
- `shared/` is a fourth module imported by both services
- `go.work` at repo root links all four for local development
- Module paths follow `github.com/<owner>/climate-control/<service>`
- Docker builds do NOT use `go.work` — each Dockerfile must `COPY shared/` and
  the relevant service directory into the build context explicitly

---

## Tech stack

| Layer | Technology | Notes |
|---|---|---|
| Language | Go | |
| HTTP framework | Gin | `api-service` only |
| ORM | GORM | App DB only (`api-service`) |
| Time-series queries | pgx raw SQL | TimescaleDB — `time_bucket`, `LATERAL` etc. |
| Auth | JWT | Short-lived access tokens + Redis-backed refresh token rotation |
| Rate limiting | Redis | `api-service` only |
| MQTT client | Eclipse Paho Go | `device-service` and `simulator-service` only |
| Broker | Mosquitto | |
| App DB | PostgreSQL 17 | port 5433 locally (5432 conflicts with native WSL postgres) |
| Time-series DB | TimescaleDB 2.25.2-pg17 | port 5434 locally, separate container |
| Control loop cache | In-process memory | NOT Redis — zero latency on hot path |
| Migrations | golang-migrate | Auto-run on `docker compose up` |
| Containers | Docker Compose | |
| Reverse proxy | NGINX | Later phase — not yet implemented |
| CI | GitHub Actions | Planned |

---

## Environment variables

```bash
# App DB (postgres)
POSTGRES_USER=cc
POSTGRES_PASSWORD=localdev
POSTGRES_DB=appdb
POSTGRES_PORT=5433

# Sensor DB (timescaledb)
TIMESCALE_USER=cc
TIMESCALE_PASSWORD=localdev
TIMESCALE_DB=metricsdb
TIMESCALE_PORT=5434

# Redis
REDIS_PASSWORD=localdev
REDIS_PORT=6379

# Mosquitto
MQTT_PORT=1883
MQTT_DEVICE_SERVICE_PASSWORD=changeme
MQTT_DEVICE_PASSWORD=changeme

# JWT
JWT_SECRET=changeme-replace-with-32-plus-chars-in-prod
JWT_ACCESS_TTL_MINUTES=15
JWT_REFRESH_TTL_DAYS=7

# Services
API_PORT=8080

# Simulator
SIMULATOR_EMAIL=simulator@local.dev
SIMULATOR_PASSWORD=localdev
```

Config is loaded once at startup into a typed struct in `shared/config/`. Never
call `os.Getenv` scattered throughout service code — always use the config struct.

---

## Database schema

### appdb (PostgreSQL 17)

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
    deadband_humidity NUMERIC(5,2) NOT NULL DEFAULT 5.0 CHECK (deadband_humidity > 0),
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

CREATE TABLE desired_state (
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

-- Only one active schedule per room at a time
CREATE UNIQUE INDEX one_active_schedule_per_room
ON schedules(room_id) WHERE is_active = true;

CREATE TABLE schedule_periods (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    schedule_id     UUID NOT NULL REFERENCES schedules(id) ON DELETE CASCADE,
    name            TEXT,
    days_of_week    INTEGER[] NOT NULL,
    start_time      TIME NOT NULL,
    end_time        TIME NOT NULL,
    mode            TEXT NOT NULL CHECK (mode IN ('OFF', 'AUTO')),
    target_temp     NUMERIC(5,2) CHECK (target_temp BETWEEN 5 AND 40),
    target_humidity NUMERIC(5,2) CHECK (target_humidity BETWEEN 0 AND 100),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK (end_time > start_time),
    CHECK (mode = 'OFF' OR target_temp IS NOT NULL OR target_humidity IS NOT NULL),
    CHECK (array_length(days_of_week, 1) > 0)
);
```

### metricsdb (TimescaleDB 2.25.2-pg17)

```sql
CREATE TABLE sensor_readings (
    time      TIMESTAMPTZ NOT NULL,
    sensor_id UUID NOT NULL,
    value     NUMERIC NOT NULL
);

SELECT create_hypertable('sensor_readings', 'time',
    chunk_time_interval => INTERVAL '7 days');

-- Column order matters: sensor_id first for equality filter, time DESC for latest-first scan
CREATE INDEX idx_sensor_readings_sensor_time
ON sensor_readings(sensor_id, time DESC);
```

`sensor_readings.sensor_id` has **no FK constraint**. Cross-server FK constraints
are impossible — `sensors` lives in appdb, `sensor_readings` lives in metricsdb.
Referential integrity is enforced by `device-service` only writing sensor IDs
sourced from its appdb sensor cache.

---

## Architecture

### Two-service split

| Service | Faces | Protocol | Responsibilities |
|---|---|---|---|
| `api-service` | Human clients | HTTP/REST | CRUD — rooms, devices, schedules, users. JWT auth. Writes `desired_state` to DB. **Never touches MQTT.** |
| `device-service` | ESP32 / simulator | MQTT | Subscribes to telemetry, writes sensor readings to TimescaleDB, runs control loop, publishes actuator commands. Reads `desired_state` from DB via cache. |

### MQTT conventions

- Telemetry topic: `devices/{device_id}/telemetry` — QoS 1 (at least once)
- Command topic: `devices/{device_id}/cmd` — QoS 2 (exactly once)
- Payloads: JSON
- Only `device-service`, `simulator-service`, and ESP32 devices connect to Mosquitto
- `api-service` never connects to MQTT under any circumstances

### Layering conventions

| Package | Responsibility |
|---|---|
| `handlers/` | HTTP concerns only — parse request, validate input, call service, write response |
| `service/` | Business logic — orchestrates repository calls, enforces rules |
| `repository/` | Data access only — all DB queries live here, nothing else |
| `models/` | Structs only — no methods with business logic |
| `middleware/` | JWT validation, rate limiting, request logging |

**Chain is always:** `handlers → service → repository`. Handlers never call
repositories directly.

---

## Control loop (device-service)

### Overview

Event-driven — triggered by incoming MQTT telemetry messages, not polling.

### In-memory cache (hot path)

```go
type RoomCache struct {
    DeadbandTemp     float64
    DeadbandHumidity float64
    DesiredState     DesiredState
    ActivePeriods    []SchedulePeriod
    UserTimezone     string
    ActuatorStates   map[string]bool  // actuator_type → on/off, never persisted
    LastActivePeriod *SchedulePeriod
    LastPeriodEndTime time.Time
}
```

Cache is invalidated via Postgres `LISTEN/NOTIFY` when `api-service` writes to
`desired_state`, `schedules`, `schedule_periods`, or `rooms`. Triggers are written
alongside `device-service` implementation — not before.

**Never query the DB on the control loop hot path.** All reads are from cache.

### Bang-bang control with hysteresis

Unidirectional actuators only (heater adds heat, humidifier adds humidity).
No active cooling or dehumidification.

```
current_value = avg(all sensors of this type in room)

if current_value < target - deadband  →  command actuator ON
if current_value > target + deadband  →  command actuator OFF
else (in band)                        →  hold current state (hysteresis)
```

### Command publishing rules

| Condition | Behaviour |
|---|---|
| `mode = AUTO`, target NOT NULL | Publish explicit ON/OFF at boundaries |
| `mode = AUTO`, target NULL | Publish nothing for that actuator type |
| `mode = OFF` | Publish nothing — ESP32 failsafe timeout handles physical off |

### Actuator state

Not persisted to DB. Lives exclusively in `device-service` in-memory cache.
Initialises to OFF on startup. Self-corrects on first telemetry batch.

Known edge case: if `device-service` restarts while an actuator is ON and the room
value is in the deadband, the actuator times out on the ESP32 side and turns off.
Self-corrects when value crosses the lower boundary. Acceptable for a home system.

### Schedule evaluation (runs on each telemetry message)

```
1. Check manual_override_until:
   - Still active  →  use current desired_state, skip schedule evaluation
   - Just expired  →  clear it, write to DB, fall through
   - NULL          →  fall through

2. Find active period from cache (pure in-memory):
   - Get current day (ISO: Mon=1, Sun=7) and time in user's timezone
   - Scan periods for matching day and time window

3. Apply grace period (~60s) if no active period found near a boundary

4. If active period differs from current desired_state  →  write to DB

5. Run bang-bang logic against current desired_state

6. Publish command if actuator state should change
```

### Schedule midnight crossing

Periods cannot cross midnight (`CHECK end_time > start_time` at DB level).
The API **transparently splits** a midnight-crossing period into two on behalf of
the user:

- Period A: original days, `start_time` to `23:59`
- Period B: days shifted +1, `00:00` to `end_time`
- `days_of_week = [7]` (Sunday) wraps to `[1]` (Monday) for Period B

### Schedule activation

Atomically deactivates any existing active schedule for the room in the same
transaction:

```sql
BEGIN;
  UPDATE schedules SET is_active = false WHERE room_id = $1 AND is_active = true;
  UPDATE schedules SET is_active = true  WHERE id = $2;
COMMIT;
```

### Stale device monitoring

Lightweight goroutine in `device-service`. Every 60 seconds, scan live state
cache for devices that haven't reported in over 5 minutes. Log a warning.
Observability only — does not write `desired_state` or publish commands.

---

## REST API endpoints

All endpoints except auth require `Authorization: Bearer <access_token>`.
All responses are JSON. All timestamps UTC.

```
POST   /api/v1/auth/register
POST   /api/v1/auth/login
POST   /api/v1/auth/refresh

GET    /api/v1/rooms
POST   /api/v1/rooms
GET    /api/v1/rooms/:id
PUT    /api/v1/rooms/:id
DELETE /api/v1/rooms/:id

GET    /api/v1/rooms/:id/devices
POST   /api/v1/rooms/:id/devices
GET    /api/v1/devices/:id
PUT    /api/v1/devices/:id
DELETE /api/v1/devices/:id

GET    /api/v1/rooms/:id/desired-state
PUT    /api/v1/rooms/:id/desired-state

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

---

## Auth design

- Access tokens: short-lived JWT (default 15 min), signed with `JWT_SECRET`
- Refresh tokens: opaque token stored as a key in Redis with TTL (default 7 days)
- Refresh token rotation: each `/auth/refresh` call invalidates the old token and
  issues a new one — prevents replay attacks
- Rate limiting via Redis on auth endpoints and globally per user

---

## Business logic rules

### desired_state

- Every room **always** has exactly one `desired_state` row — created in the same
  transaction as the room, never deleted independently
- `mode = OFF` is an explicit valid state, not the absence of one
- `api-service` writes `desired_state` when user makes a direct control request —
  sets `manual_override_until`
- `device-service` writes `desired_state` when a schedule period transitions —
  does NOT set `manual_override_until`
- `manual_override_until = NULL` means the scheduler controls the room
- `manual_override_until` in the future means user override is active

### Validation rules for desired_state and schedule_periods

- `mode = AUTO` + `target_temp NOT NULL` → room must have a `temperature` sensor
  AND a `heater` actuator
- `mode = AUTO` + `target_humidity NOT NULL` → room must have a `humidity` sensor
  AND a `humidifier` actuator
- `mode = AUTO` + both targets NULL → reject — use `OFF` instead
- `mode = OFF` → always valid regardless of room configuration

### Schedule period overlap

Rejected at the API layer. Overlap check:

```
new_start < existing_end AND new_end > existing_start
```

Applied per-day — only periods sharing at least one day in `days_of_week` need
to be checked.

### Device deletion edge case

When a device is deleted, check whether the room still has the sensors and
actuators required to support the current `desired_state` and active
`schedule_periods`. Clear orphaned targets if not. **Document as TODO in the
device deletion handler** — not yet implemented.

### Rooms are user-scoped

Rooms belong to a single user. Cross-user room conflicts are impossible by design.
All room queries must filter by `user_id` from the JWT claims.

---

## Measurement unit conventions

Enforced by convention, not a DB column:

| Measurement type | Unit |
|---|---|
| `temperature` | Degrees Celsius |
| `humidity` | Percent relative humidity (0–100) |
| `air_quality` | PPM (parts per million) |

---

## Days of week convention

ISO 8601: Monday = 1, Sunday = 7.
Go's `time.Weekday()` uses Sunday = 0 — **must convert explicitly**.
Write a test case specifically for Sunday boundary behaviour.

---

## TimescaleDB query patterns

Use `LATERAL` join for "latest reading per sensor" — this is the primary hot-path
query pattern:

```sql
SELECT s.id, s.measurement_type, lr.value, lr.time
FROM sensors s
CROSS JOIN LATERAL (
    SELECT value, time
    FROM sensor_readings
    WHERE sensor_id = s.id
    ORDER BY time DESC
    LIMIT 1
) lr
WHERE s.device_id = ANY($1);
```

Use raw `pgx` for all TimescaleDB queries — not GORM.
Composite index column order `(sensor_id, time DESC)` is intentional — equality
filter on `sensor_id` first, then scan `time DESC` for the latest value.

---

## Simulator service

On startup, the simulator:

1. Registers a user using `SIMULATOR_EMAIL` / `SIMULATOR_PASSWORD` from env (or
   logs in if already exists)
2. POSTs rooms via the REST API
3. POSTs devices into those rooms
4. Begins publishing MQTT telemetry for each registered device using the `hw_id`
   received from the API

The simulator uses `device_type = 'simulator'` when registering devices.

Future scope (not yet implemented): configurable scenario file (JSON/YAML) defining
room count, device count per room, sensor types, and simulated reading profiles
(stable, drifting, oscillating, heating/cooling curves).

---

## Architectural rules (non-negotiable)

1. `api-service` **never** connects to MQTT
2. `device-service` **never** exposes HTTP endpoints (except optional internal `/health`)
3. Shared models are the schema contract — both services import from `shared/models`,
   never duplicate structs
4. Handlers never call repositories directly — chain is `handlers → service → repository`
5. Control loop never queries DB on hot path — reads from in-memory cache only
6. `desired_state` is never nil — every room always has a row, even if `mode = OFF`
7. All timestamps UTC — no timezone conversion on the backend, client's responsibility
8. Actuator state never persisted — lives in `device-service` in-memory cache only
9. Deadbands live on `rooms`, not `desired_state` — they are room configuration,
   not target setpoints
10. Schedule timezone comes from `users.timezone` — no per-schedule timezone column
11. `sensor_readings.sensor_id` has no FK — integrity enforced by `device-service`
12. `OFF` mode publishes nothing — ESP32 failsafe handles physical off state
13. One active schedule per room — partial unique index enforces this at DB level
14. Schedule period overlap rejected at API layer — not left to DB constraints
15. Primary keys are always named `id`, never `table_name_id`
16. Config loaded once at startup into typed struct — never call `os.Getenv` in
    business logic
17. GORM for appdb only — all TimescaleDB queries use raw `pgx`
18. Room creation and `desired_state` row creation happen in the same transaction

---

## Docker Compose notes

- `docker-compose.yml` at repo root is infrastructure only (postgres, timescaledb,
  redis, mosquitto, golang-migrate runners)
- `deployments/docker-compose.services.yml` adds application services (api-service,
  device-service, simulator-service) as an overlay
- `docker-compose.yml` must always be the first `-f` argument so `.env` is resolved
  from the repo root
- Postgres on port 5433, TimescaleDB on port 5434 — offset from 5432 to avoid
  conflict with native WSL PostgreSQL instance
- `go.work` is not used inside Docker — each service Dockerfile copies `shared/`
  explicitly into the build context

---

## Development phases

| Phase | Scope |
|---|---|
| 1 | Repo scaffold, Docker Compose infrastructure, full DB schema migrated |
| 2 | `api-service` — Go modules, shared config/models, Gin server, JWT auth, all REST endpoints |
| 3 | `device-service` — MQTT consumer, control loop, in-memory cache, LISTEN/NOTIFY invalidation |
| 4 | `simulator-service` — API registration on startup, MQTT telemetry publishing |
| 5 | CI (GitHub Actions), architecture diagrams, README polish |
| Later | NGINX load balancing layer, cloud deployment (AWS or GCP) |

---

## What to build next

**Branch:** `feat/api-service-scaffold`

Goals:
1. Initialize `shared/go.mod`, `api-service/go.mod`, `simulator-service/go.mod`,
   update `go.work` to include all four modules
2. Add dependencies — Gin, GORM, pgx, Redis client, JWT library
3. `shared/config/` — typed config struct loaded from env
4. `shared/models/` — GORM models matching schema above
5. `api-service/cmd/main.go` — starts Gin server, connects to postgres and Redis,
   serves `GET /health`
6. `api-service/Dockerfile` — multi-stage build
7. Update `docker-compose.yml` — add `api-service` service entry

This branch produces a service that compiles, starts, connects to its dependencies,
and passes a CI health check. No business logic yet.