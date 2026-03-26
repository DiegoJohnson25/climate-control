# Climate Control Project — Context Handoff

## Project overview

A distributed IoT climate control system backend in Go. Two independent services
communicate via PostgreSQL and MQTT to manage room climate via ESP32 relay devices.
Built as a portfolio project — fully demonstrable without hardware via a simulator
service.

---

## Current state

**Completed:**
- Phase 1 ✅ — repo scaffold, Docker Compose infrastructure, full DB schema migrated
- `feat/api-service-scaffold` ✅ — merged to main

**Active branch:** `feat/api-service-auth`

**Last thing actually done:** Health check endpoint working in the new domain-first
repo structure. User repository written. About to start the auth service layer.

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
│   │   │   ├── handler.go
│   │   │   ├── service.go
│   │   │   ├── repository.go
│   │   │   └── errors.go
│   │   ├── router/
│   │   │   └── router.go         # route registration, middleware chain
│   │   ├── health/
│   │   │   └── health.go         # plain function, no struct
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
│   ├── architecture.md
│   ├── api.md
│   └── diagrams/
├── tests/
│   └── postman/
│       ├── climate-control.postman_collection.json
│       └── climate-control.postman_environment.json
├── docker-compose.yml
├── go.work
├── Makefile
├── .env
├── .env.example
├── .gitignore
├── README.md
├── CLAUDE.md
└── .github/
    ├── workflows/ci.yml
    └── pull_request_template.md
```

---

## Package naming rules (locked in)

- Package name is the domain: `user`, `auth`, `room`
- File name encodes the layer: `handler.go`, `service.go`, `repository.go`
- Types drop the domain prefix: `auth.Service` not `auth.AuthService`
- Constructors keep the layer name: `auth.NewService`, `auth.NewHandler`, `auth.NewRedisRepository`
- Health check is a plain function — `r.GET("/health", health.Check)`
- File names are snake_case, no suffix (e.g. `handler.go` not `user_handler.go`)
- Package/folder names are singular — `service` not `services`, `handler` not `handlers`

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
MQTT_DEVICE_SERVICE_PASSWORD=localdev
MQTT_DEVICE_PASSWORD=localdev

# JWT
JWT_SECRET=localdev-replace-with-32-plus-chars-in-prod
JWT_ACCESS_TTL_MINUTES=15
JWT_REFRESH_TTL_DAYS=7

# Services
API_PORT=8080

# Simulator
SIMULATOR_EMAIL=simulator@local.dev
SIMULATOR_PASSWORD=localdev
```

**Important:** Internal Docker port for both postgres and timescaledb is `5432`
regardless of host-mapped ports. Connection strings inside Docker always use
`port=5432`, `host=postgres`, `host=timescaledb`, `host=redis`.

---

## Config struct (api-service/internal/config/config.go)

```go
type Config struct {
    PostgresUser     string
    PostgresPassword string
    PostgresDB       string
    PostgresPort     int       // host port only — not used in DSN inside Docker

    TimescaleUser     string
    TimescalePassword string
    TimescaleDB       string
    TimescalePort     int

    RedisPassword string
    RedisPort     int

    JWTSecret           string
    JWTAccessTTLMinutes int
    JWTRefreshTTLDays   int

    APIPort int

    SimulatorEmail    string
    SimulatorPassword string
}
```

- Loaded once in `main.go` via `config.Load()`
- `mustInt(name, value)` panics with variable name (not value) on bad input
- JWT TTL constants could be hardcoded — kept as env vars for flexibility
- Config struct passed down field-by-field to components, never the whole struct

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
    value     NUMERIC NOT NULL
);

SELECT create_hypertable('sensor_readings', 'time', chunk_time_interval => INTERVAL '7 days');
CREATE INDEX idx_sensor_readings_sensor_time ON sensor_readings(sensor_id, time DESC);
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
- No constraint tags (`uniqueIndex`, `check`, `not null`) — migrations handle all constraints
- `desired_states` table name set via `TableName()` method (was originally `desired_state`, renamed to plural for consistency)
- GORM used for appdb only — TimescaleDB uses raw pgx

---

## Auth architecture

`auth` imports `user` — one directional. `user` knows nothing about `auth`.

```go
// auth/service.go
type service struct {
    users user.Repository  // looks up user by email for login
    tokens Repository      // Redis refresh token storage
}
```

Wiring in `main.go`:
```go
userRepo    := user.NewPostgresRepository(db)
userSvc     := user.NewService(userRepo)
authRepo    := auth.NewRedisRepository(rdb)
authSvc     := auth.NewService(userRepo, authRepo)  // takes userRepo directly

userHandler := user.NewHandler(userSvc)
authHandler := auth.NewHandler(authSvc)
```

### JWT design
- Access tokens: short-lived JWT (15 min), signed with `JWT_SECRET`
- Refresh tokens: opaque token stored in Redis with TTL (7 days)
- Rotation: every `/auth/refresh` invalidates old token, issues new pair
- Claims: standard `exp` + `user_id`
- Logout: deletes refresh token from Redis

### Redis data model for refresh tokens
- Key: the refresh token string itself
- Value: user ID
- TTL: `JWT_REFRESH_TTL_DAYS`

---

## Layering conventions

| Layer | Responsibility |
|---|---|
| `handler.go` | HTTP only — parse request, validate input, call service, write response |
| `service.go` | Business logic — orchestrate repository calls, enforce rules |
| `repository.go` | Data access only — all DB/Redis queries live here |
| `errors.go` | Sentinel errors for the domain |
| `middleware.go` | JWT validation, injects user ID into Gin context |

**Chain:** `handler → service → repository`. Handlers never call repositories directly.

**Sentinel errors** defined per domain, translated to HTTP status codes in handlers:
```go
var (
    ErrEmailTaken         = errors.New("email already taken")
    ErrInvalidCredentials = errors.New("invalid credentials")
)
```

**Interfaces:** Not used yet — concrete types used directly. Will be added when
tests are written in Phase 5. The refactor is mechanical at that point.

**Constructors return concrete types for now:**
```go
func NewService(users *user.Repository, tokens *RedisRepository) *Service
func NewHandler(svc *Service) *Handler
```

---

## REST API endpoints

```
POST   /api/v1/auth/register
POST   /api/v1/auth/login
POST   /api/v1/auth/refresh
POST   /api/v1/auth/logout

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

All endpoints except auth require `Authorization: Bearer <access_token>`.
All responses JSON. All timestamps UTC.

---

## Architectural rules (non-negotiable)

1. `api-service` never connects to MQTT
2. `device-service` never exposes HTTP endpoints (except optional `/health`)
3. Shared models are the schema contract — both services import from `shared/models`
4. Handlers never call repositories directly — always through service
5. No business logic in handlers — HTTP concerns only
6. `desired_states` is never nil — every room always has a row, even if `mode = OFF`
7. All timestamps UTC — no timezone conversion on backend
8. Actuator state never persisted — lives in `device-service` in-memory cache only
9. Deadbands live on `rooms` — room configuration, not target setpoints
10. Schedule timezone comes from `users.timezone` — no per-schedule timezone
11. `sensor_readings.sensor_id` has no FK — integrity enforced by `device-service`
12. `OFF` mode publishes nothing — ESP32 failsafe handles physical off state
13. One active schedule per room — partial unique index enforces at DB level
14. Schedule period overlap rejected at API layer
15. Primary keys always named `id`
16. Config loaded once at startup — never call `os.Getenv` in business logic
17. GORM for appdb only — all TimescaleDB queries use raw pgx
18. Room creation and `desired_states` row creation in the same transaction
19. Rate limiting applied at router level, not inside handlers
20. Never log config values — log variable names only on panic

---

## Business logic rules

### desired_states
- Every room always has exactly one row — created in same transaction as room
- `api-service` writes when user makes direct control request — sets `manual_override_until`
- `device-service` writes on schedule period transition — does NOT set `manual_override_until`
- `manual_override_until = NULL` means scheduler controls the room

### Validation for desired_states and schedule_periods
- `mode = AUTO` + `target_temp NOT NULL` → room must have temperature sensor + heater
- `mode = AUTO` + `target_humidity NOT NULL` → room must have humidity sensor + humidifier
- `mode = AUTO` + both targets NULL → reject, use `OFF`
- `mode = OFF` → always valid

### Schedule period overlap
Rejected at API layer. Check: `new_start < existing_end AND new_end > existing_start`
Per-day — only periods sharing at least one day need checking.

### Schedule midnight crossing
API transparently splits into two periods:
- Period A: original days, `start_time` to `23:59`
- Period B: days +1, `00:00` to `end_time`
- Sunday (7) wraps to Monday (1) for Period B

### Schedule activation
Atomic transaction — deactivates existing active schedule, activates new one.

### Device deletion
When deleted, check if room still has sensors/actuators to support current
`desired_states` and active `schedule_periods`. Clear orphaned targets.
**Currently a TODO in the device deletion handler.**

### Rooms are user-scoped
All room queries must filter by `user_id` from JWT claims.

---

## Control loop (device-service — for future reference)

Event-driven — triggered by incoming MQTT telemetry. Bang-bang control with
hysteresis. In-memory cache invalidated via Postgres LISTEN/NOTIFY.

```go
type RoomCache struct {
    DeadbandTemp     float64
    DeadbandHumidity float64
    DesiredState     DesiredState
    ActivePeriods    []SchedulePeriod
    UserTimezone     string
    ActuatorStates   map[string]bool
    LastActivePeriod *SchedulePeriod
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

---

## MQTT conventions

- Telemetry: `devices/{device_id}/telemetry` — QoS 1
- Commands: `devices/{device_id}/cmd` — QoS 2
- Only `device-service`, `simulator-service`, ESP32s connect to Mosquitto

---

## Measurement unit conventions

| Type | Unit |
|---|---|
| `temperature` | Celsius |
| `humidity` | % relative humidity (0-100) |
| `air_quality` | PPM |

## Days of week convention

ISO 8601: Monday = 1, Sunday = 7.
Go's `time.Weekday()` uses Sunday = 0 — must convert explicitly.
Write a test case specifically for Sunday.

---

## Docker Compose

- `docker-compose.yml` — infrastructure only (postgres, timescaledb, redis, mosquitto)
- `deployments/docker-compose.services.yml` — application services overlay
- All services have healthchecks
- `depends_on` uses `condition: service_healthy`
- `restart` policy not yet set on application services (dev phase)
- Dockerfile `context: ..` points to repo root so `shared/` is accessible
- `replace` directive in `api-service/go.mod` for local `shared` module resolution in Docker

## Makefile targets

```
up            — bring all services up (no rebuild)
rebuild       — bring all services up with --build
down          — bring all services down
down-hard     — bring all services down, wipe volumes
infra         — infrastructure only
infra-down    — infrastructure down
infra-down-v  — infrastructure down, wipe volumes
rebuild-api   — rebuild api-service only
logs          — tail all logs
logs-api      — tail api-service logs
logs-device   — tail device-service logs
logs-postgres — tail postgres logs
logs-redis    — tail redis logs
logs-mqtt     — tail mosquitto logs
shell-api     — shell into api-service container
shell-postgres — psql into postgres
shell-timescale — psql into timescaledb
shell-redis   — redis-cli into redis
ps            — show container status
build-api     — build api-service image directly (for debugging build errors)
vet           — go vet all modules
```

---

## Simulator service

On startup:
1. Register user using `SIMULATOR_EMAIL`/`SIMULATOR_PASSWORD` (or login if exists)
2. POST rooms via REST API
3. POST devices into rooms
4. Publish MQTT telemetry using `hw_id` from API response
5. Uses `device_type = 'simulator'`

Future: configurable scenario file for room/device counts, sensor profiles,
heating/cooling curves.

---

## Development phases

| Phase | Scope | Status |
|---|---|---|
| 1 | Repo scaffold, Docker Compose, DB schema | ✅ Done |
| 2 | `api-service` — auth, all REST endpoints | 🔄 In progress |
| 3 | `device-service` — MQTT, control loop, cache | Pending |
| 4 | `simulator-service` — API registration, MQTT telemetry | Pending |
| 5 | CI, architecture diagrams, README, tests | Pending |
| Later | NGINX load balancing, cloud deployment | Pending |