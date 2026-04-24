# Climate Control — CLAUDE.md

## Project overview

A distributed IoT climate control system built in Go. ESP32 relay devices publish sensor telemetry (temperature, humidity) via MQTT. A backend device-service ingests that telemetry, evaluates a bang-bang control loop per room, and publishes relay commands back to the devices. A REST API (api-service) lets users configure rooms, devices, schedules, and desired climate state. A React web client provides a dashboard for monitoring and control. A simulator service stands in for physical hardware, making the system fully demonstrable without any ESP32s.

The architecture is designed to scale: api-service scales horizontally behind NGINX with no code changes (Redis-backed state, stateless request handling). device-service scales via Kafka partition ownership in Phase 7 — each room hashes to one partition, owned by exactly one instance. The Redis stream cache invalidation layer ensures all device-service instances stay in sync with api-service writes.

The project serves as a portfolio piece demonstrating distributed systems design, event-driven architecture, IoT protocols, and full-stack development across Go backend, React frontend, and infrastructure layers.

---

## Architecture summary

**api-service** — Gin REST API. Owns all user-facing configuration: rooms, devices, schedules, desired state. Writes to PostgreSQL (appdb) via GORM. Publishes cache invalidation events to a Redis Stream on any state change that device-service needs to know about. JWT auth with Redis-backed refresh token rotation. Read-only TimescaleDB access via pgx for climate endpoints.

**device-service** — Standalone Go binary. Minimal HTTP server on `:8081` for health checks only. Subscribes to MQTT telemetry (Phases 3–6) or Kafka (Phase 7), maintains an in-memory cache of room and device state, runs a bang-bang control loop per room, publishes actuator commands via MQTT, writes sensor readings and control logs to TimescaleDB. Consumes the Redis Stream from api-service to keep its cache in sync.

**simulator-service** — Provisions simulated users, rooms, devices, desired states, and schedules via the api-service REST API. Publishes realistic sensor telemetry via MQTT. Reacts to actuator commands — room temperature and humidity evolve based on active actuator contributions and passive return-to-ambient. Allows the full system to be demonstrated without physical hardware. Started via `docker compose --profile simulation` or `make simulator-start`.

**mqtt-bridge** — New service added in Phase 7. Subscribes to Mosquitto telemetry, resolves hw_id to room_id, and produces to Kafka. The only service that talks to Mosquitto for telemetry in Phase 7 — device-service no longer subscribes to Mosquitto directly.

**web-client** — React SPA (Phase 6). Served as static files by NGINX. Consumes the api-service REST API directly via NGINX proxy. Dashboard showing room climate state, control panel, history charts, schedule management, device management.

**NGINX** — Single entry point (Phase 5). Serves React static files. Proxies `/api` to api-service via Docker DNS round-robin load balancing. TLS termination point.

**Infrastructure:** PostgreSQL (appdb), TimescaleDB (metricsdb), Redis, Mosquitto MQTT broker, Apache Kafka (Phase 7), all containerised via Docker Compose.

---

## Current state

**Completed:**
- Phase 1 ✅ — repo scaffold, Docker Compose infrastructure, full DB schema
- Phase 2 ✅ — api-service: all REST endpoints, JWT auth, Redis refresh token rotation
- Phase 3a ✅ — simulator-service scaffold: config, provisioning, MQTT publish loop
- Phase 3b ✅ — device-service scaffold: config, cache warm, appdb repository
- Phase 3c ✅ — device-service ingestion: MQTT source, TimescaleDB writes
- Phase 3d ✅ — device-service control loop: bang-bang, command publishing, scheduler
- Phase 3e ✅ — device-service stream: Redis stream consumer, cache invalidation
- Phase 4a ✅ — CI pipeline, device-service health server, compose health checks, simulator profile
- Phase 4b ✅ — reactive room model, environment model, time scaling, physical units
- Phase 4c ✅ — desired state + schedule provisioning, teardown, demo.yaml 4-user topology
- Phase 5a ✅ — `GET /rooms/:id/climate`, `GET /rooms/:id/climate/history`, Postman verified

**Active branch:** `feat/nginx` — Phase 5b

---

## Development phases

| Phase | Branch | Scope |
|---|---|---|
| 1–4c | — | Repo scaffold, full API, device-service, simulator, CI, reactive model, schedules | ✅ Done |
| 5a | `feat/api-service-climate` | `GET /rooms/:id/climate`, `GET /rooms/:id/climate/history`, Postman verified | ✅ Done |
| 5b | `feat/nginx` | Single entry point, static file serving, API proxy via Docker DNS, horizontal scaling demonstrated, Postman environments updated |
| 6a | `feat/client-scaffold` | Vite project, routing, auth flow, JWT handling, persistent nav, SWR setup |
| 6b | `feat/client-rooms` | Dashboard room cards, room detail shell, overview tab, current state + control panel |
| 6c | `feat/client-history` | Climate history chart consuming 5a endpoints |
| 6d | `feat/client-schedules` | Schedule tab, period management modal |
| 6e | `feat/client-devices` | Devices page, inline assignment, room detail devices tab (read-only) |
| 7a | `feat/kafka-bridge` | MQTT bridge service, Kafka cluster in docker-compose |
| 7b | `feat/kafka-device-service` | Replace `mqtt.Source` with `kafka.Source`, partition ownership callbacks, `OwnsRoom()` real implementation |
| 8 | `feat/ci-full` + `feat/docs` | Newman integration + smoke tests in CI, frontend build verification, architecture diagrams, README polish, one-command startup |

**Beyond phase 8 (independent, no strict ordering):**
- Grafana + Prometheus observability
- Physics room model + freeform simulator client
- Cloud deployment (AWS)
- Kubernetes + load testing

---

## Repo structure

```
climate-control/
├── api-service/
│   ├── cmd/main.go
│   └── internal/
│       ├── user/         # handler.go, service.go, repository.go, errors.go
│       ├── auth/         # handler.go, middleware.go, service.go, repository.go, errors.go
│       ├── room/         # handler.go, service.go, repository.go, pg_errors.go, errors.go
│       ├── device/       # handler.go, service.go, repository.go, pg_errors.go, errors.go
│       ├── schedule/     # handler.go, service.go, repository.go, pg_errors.go, errors.go
│       ├── metricsdb/    # repository.go — LatestClimate, ClimateHistory (read-only TimescaleDB)
│       ├── router/       # router.go — route registration, middleware chain
│       ├── health/       # health.go — plain function, no struct
│       ├── ctxkeys/      # keys.go — context key constants, prevents circular imports
│       ├── config/       # config.go — typed config struct, Load()
│       └── initializers/ # redis.go
├── device-service/
│   ├── cmd/main.go       # connections, cache warm, ingestion wiring, signal handling
│   └── internal/
│       ├── config/       # config.go
│       ├── connect/      # postgres.go, timescale.go, redis.go
│       ├── cache/        # cache.go — Store, RoomCache, DeviceCache
│       ├── appdb/        # repository.go — WarmCache, ReloadRoom, ReloadDevice
│       ├── metricsdb/    # repository.go — WriteSensorReadings, WriteControlLogEntry
│       ├── logging/      # logging.go — LogSummary, LogStore, LogFullStore etc.
│       ├── ingestion/    # source.go, ingestion.go, message.go
│       ├── mqtt/         # client.go, source.go
│       ├── health/       # health.go — readiness HTTP server on :8081
│       ├── control/      # bang-bang logic, command publishing
│       ├── scheduler/    # per-room ticker goroutines, lifecycle management
│       └── stream/       # Redis stream consumer, cache invalidation
├── simulator-service/
│   ├── cmd/main.go       # flag parsing, run/teardown modes, signal handling
│   ├── config/
│   │   ├── templates/    # rooms.yaml, devices.yaml, desired_states.yaml, schedules.yaml
│   │   ├── simulations/  # default.yaml, cache-test.yaml, demo.yaml
│   │   └── credentials/  # gitignored, written at runtime for interactive groups
│   └── internal/
│       ├── api/          # client.go — HTTP client for api-service
│       ├── config/       # config.go — env + YAML loading, template resolution, timing derivation
│       ├── mqtt/         # client.go — Paho wrapper
│       ├── provisioning/ # provisioning.go — bootstrap, teardown, identity generation
│       └── simulator/    # simulator.go, room_state.go, model.go
├── web-client/           # Phase 6 — Vite + React SPA
├── mqtt-bridge/          # Phase 7 — MQTT → Kafka bridge service
├── deployments/
│   ├── docker-compose.services.yml
│   ├── docker-compose.prod.yml
│   ├── mosquitto/        # mosquitto.conf, passwd, acl
│   └── nginx/nginx.conf
├── migrations/
│   ├── appdb/
│   └── metricsdb/
├── docs/
│   └── architecture/     # kafka.md, stream.md, auth.md, simulator.md, device-service.md,
│                         # future-features.md, client.md
├── tests/postman/
├── docker-compose.yml
├── go.work
├── Makefile
├── .env.ci               # committed placeholder env for CI
└── .claude/
    └── COMMENTING_STYLE.md
```

---

## Tech stack

### Languages & runtime

| Technology | Purpose |
|---|---|
| Go 1.25 | All backend services |
| JavaScript (ES2022) | Web client |

### API layer

| Technology | Purpose |
|---|---|
| Gin | HTTP framework — api-service only |
| GORM | ORM — appdb queries in api-service |
| golang-jwt | JWT access token signing and validation |
| go-redis/v9 | Refresh token storage, rate limiting, Redis Streams |

### Device service

| Technology | Purpose |
|---|---|
| pgx/v5 | Raw SQL — all TimescaleDB queries, device-service appdb queries use Raw+Scan |
| Eclipse Paho Go | MQTT client — aliased `pahomqtt` to avoid package name collision |
| franz-go | Kafka client — Phase 7, replaces Paho as telemetry source in device-service |

### Simulator service

| Technology | Purpose |
|---|---|
| Eclipse Paho Go | MQTT client — telemetry publish, command subscribe |
| goccy/go-yaml | Simulation config and template loading |

### Web client (Phase 6)

| Technology | Purpose |
|---|---|
| React 19 | UI component framework |
| Vite | Build tooling and dev server |
| SWR | Server state — data fetching, polling, cache |
| shadcn/ui + Tailwind | Component library and utility CSS |
| Recharts | Time series charts for climate history |
| React Router | Client-side routing |

### Infrastructure & data

| Technology | Purpose |
|---|---|
| PostgreSQL 17 | Application data (appdb) — host port 5433 |
| TimescaleDB 2.25-pg17 | Sensor readings + control logs hypertables (metricsdb) — host port 5434 |
| Redis | Refresh tokens, rate limiting, cache invalidation stream — host port 6379 |
| golang-migrate | Schema migrations — auto-run on `docker compose up` |
| Docker Compose | Container orchestration |

### Messaging

| Technology | Purpose |
|---|---|
| Mosquitto 2.x | MQTT broker — auth enabled, ACL per username — host port 1883 |
| Apache Kafka (KRaft) | Telemetry pipeline — Phase 7, 24 partitions, no ZooKeeper |

### Tooling & testing

| Technology | Purpose |
|---|---|
| Newman + Postman | API integration and smoke tests |
| NGINX | Reverse proxy, static file serving, TLS termination — Phase 5 |
| GitHub Actions | CI — build/vet all Go services, Docker Compose stack startup verification |

---

## Environment variables

```bash
# App DB (postgres)
POSTGRES_USER=cc
POSTGRES_PASSWORD=changeme
POSTGRES_DB=appdb
POSTGRES_PORT=5433

# Metrics DB (timescaledb)
TIMESCALE_USER=cc
TIMESCALE_PASSWORD=changeme
TIMESCALE_DB=metricsdb
TIMESCALE_PORT=5434

# Redis
REDIS_PASSWORD=changeme
REDIS_PORT=6379

# Mosquitto
MQTT_PORT=1883
MQTT_DEVICE_SERVICE_USERNAME=device-service
MQTT_DEVICE_SERVICE_PASSWORD=changeme
MQTT_DEVICE_USERNAME=device
MQTT_DEVICE_PASSWORD=changeme
# healthcheck user hardcoded: username=healthcheck password=healthcheck — not in .env
# After changing any MQTT passwords, regenerate the password file: make mosquitto-passwd

# Device service tuning
CONTROL_STALE_THRESHOLD_SECONDS=90
CONTROL_TICK_INTERVAL_SECONDS=10
CONTROL_CACHE_REFRESH_MINUTES=5
DEVICE_DEBUG=           # "info" for key event logging, "verbose" for full cache/tick detail
DEVICE_TRACE_INGESTION=false
DEVICE_TRACE_TICK=false

# JWT
JWT_SECRET=changeme-replace-with-32-plus-chars-in-prod
JWT_ACCESS_TTL_MINUTES=15
JWT_REFRESH_TTL_DAYS=7

# Services
API_PORT=8080

# Simulator
SIMULATOR_EMAIL=simulator@local.dev
SIMULATOR_PASSWORD=changeme
SIMULATOR_SIMULATION=default
```

Internal Docker ports are always hardcoded in connection strings — env var ports are
host-machine mappings only. `host=postgres port=5432`, `host=timescaledb port=5432`,
`host=redis port=6379`, `host=mosquitto port=1883`, `http://api-service:8080`.

---

## Locked-in conventions

### Package naming

- Package name is the domain: `user`, `auth`, `room`, `device`, `schedule`
- File name encodes the layer: `handler.go`, `service.go`, `repository.go`
- Types drop the domain prefix: `auth.Service` not `auth.AuthService`
- Constructors: `NewService`, `NewHandler`, `NewRepository` — always `NewRepository` regardless of backing store
- File names are snake_case, no suffix (`handler.go` not `user_handler.go`)
- Package/folder names are singular
- Postgres unique violation detection in a private `pg_errors.go` per package —
  unexported helper, intentionally duplicated, never extracted to shared utility

### Naming conventions

- `models.User` → `usr`, `models.Room` → `rm`, `models.Device` → `dev`
- `models.Schedule` → `sched`, `models.SchedulePeriod` → `period`
- Service struct fields named by concept: `users`, `tokens`, `rooms`, `devices`, `schedules`
- Single-repo services use plain `repo`
- Method receivers use single letter: `(s *Service)`, `(h *Handler)`, `(r *Repository)`
- `List` prefix for slice-returning methods
- No `ID` suffix on method names: `ListByRoom` not `ListByRoomID`
- Service layer parameters use `input` naming (`req` reserved for handler request structs)

### GORM style (api-service)

- Chain `.Error` directly — no `result` variable, no `RowsAffected`
- Repo methods return pointer types
- All take `context.Context` first with `.WithContext(ctx)` on all GORM calls
- `Save` for updates — full replacement, always fetch-before-mutate
- Handler owns field filtering via request structs
- Empty slices initialized explicitly
- Responses never serialize full model structs — use `gin.H`
- No `user_id` in resource responses
- Sensors/actuators serialized as `[]string` of type names
- Timestamps: RFC3339 UTC. `start_time`/`end_time`: `"HH:MM"`

### GORM Raw+Scan (device-service)

device-service never uses GORM model-based queries. All appdb queries use
`.Raw().Scan()` into unexported local scan structs. GORM tag required for array
types: `gorm:"type:integer[]"` on `pq.Int64Array` fields.

### GORM model conventions (shared/models)

- `ID uuid.UUID \`gorm:"type:uuid;primaryKey;default:gen_random_uuid()"\``
- Nullable foreign keys: `RoomID *uuid.UUID \`gorm:"type:uuid"\``
- Nullable values: pointer types (`*float64`, `*string`, `*time.Time`)
- `HwID` needs explicit tag: `\`gorm:"column:hw_id"\``
- `days_of_week`: `pq.Int64Array \`gorm:"type:integer[]"\``
- No constraint tags — migrations handle all constraints
- No GORM association fields on shared models — enriched types live in domain packages

### HTTP status codes

- `400` — malformed request
- `404` — not found or unauthorized (ownership gate — no information leakage)
- `409` — state conflict
- `422` — semantically invalid request
- `500` — internal server error (never leak details)

### Handler conventions

- `c.JSON` + `return` in handlers — not `AbortWithStatusJSON` (middleware only)
- `ShouldBindJSON` over `BindJSON`
- Sentinel errors defined per domain, translated to HTTP status in handlers not services
- `ErrNotFound` maps to `ErrInvalidCredentials` in login — prevents user enumeration
- Logout swallows `ErrTokenNotFound`
- PUT = full replacement of all mutable fields
- PATCH reserved for targeted state transitions (e.g. schedule activate/deactivate)

### Commenting style

All Go code follows `.claude/COMMENTING_STYLE.md`. Key rules:
- Package comments in primary file, starting with "Package <n>."
- Godoc on exported symbols that need them — start with symbol name, end with period.
- Trivial constructors, self-documenting DTOs, descriptive error sentinels need no comment.
- File-level section breaks: three-line 75-dash bars with title case label.
- No structural dividers inside functions.
- `// TODO Phase N:` for phase-tagged items, always with explanation.
- Inline comments only for struct field annotations.

### Connection conventions

All services use `internal/connect/` with separate files per connection type.
Package name `connect` — reads as `connect.Postgres(...)`, `connect.Redis(...)`.

### Dependency directions (non-negotiable)

```
auth     → user
device   → room
schedule → room
ingestion → (nothing — transport-agnostic)
mqtt     → ingestion
```

`room` never imports `device` or `schedule`. `device` never imports `schedule`.
`ingestion` never imports `mqtt` or `kafka`.

---

## REST API endpoints

### Implemented

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

GET    /api/v1/rooms/:id/climate
GET    /api/v1/rooms/:id/climate/history?window=1h|6h|24h|7d[&density=N]
```

`/climate` — current snapshot sourced from the most recent `room_control_logs` row.
Returns `null` body (200) if the room has no data yet. Fields: `time`, `avg_temp`,
`avg_hum`, `mode`, `target_temp`, `target_hum`, `control_source`, `heater_cmd`
(bool or null), `humidifier_cmd` (bool or null), `deadband_temp`, `deadband_hum`.

`/climate/history` — array of time-bucketed objects from `room_control_logs`.
Response shape:
```json
{
  "window": "24h",
  "bucket_seconds": 600,
  "points": [{ "time", "avg_temp", "avg_hum", "heater_duty", "humidifier_duty",
               "target_temp", "target_hum", "deadband_temp", "deadband_hum" }]
}
```
`window` defaults to `"24h"` if omitted. Optional `?density=N` overrides the
target point count (must be a positive integer). Bucket size targets
`DefaultDensity = 120` data points per window, rounded to the nearest value on a
fixed ladder `[30, 60, 120, 300, 600, 900, 1800, 3600, 7200, 10800, 21600]`.
Duty fields are `AVG(heater_cmd)` / `AVG(humidifier_cmd)` — 0.0–1.0 fraction of
ticks the actuator was on within the bucket. `deadband_temp` / `deadband_hum` are
null when no target is set for that type (deadband is only meaningful alongside a
target). Null buckets are returned as-is — gaps in the chart indicate offline
devices or stale readings.

---

## Business logic rules

### desired_states
- Every room always has exactly one row — created in same transaction as room
- `api-service` writes when user makes direct control request
- `device-service` writes on schedule period transition — does NOT set `manual_override_until`
- `manual_override_until = NULL` means scheduler controls the room
- `manual_override_until = 9999-12-31T23:59:59Z` means indefinite override
- API contract: client sends `"indefinite"`, timestamp string, or `null`
- `desired_states.id` is vestigial — `room_id` is the natural PK but migration not worth it
- Default deadbands set in `room.Service.Create` as named constants — not in DB migration:
  `defaultTempDeadband = 0.5` and `defaultHumDeadband = 2.0`

### Capability checks
- `mode = AUTO` + `target_temp NOT NULL` → room must have temperature sensor + heater
- `mode = AUTO` + `target_hum NOT NULL` → room must have humidity sensor + humidifier
- `mode = AUTO` + both targets NULL → reject 422
- `mode = OFF` → always valid
- Sensor and actuator may be on separate devices — EXISTS + EXISTS pattern (not JOIN)
- Capability queries live in `room.Repository` — moving to `device.Repository` would
  create a circular import
- Capability checks enforced at activation time only; inactive schedules represent
  stored future intent

### Device capability conflicts
- DELETE or unassign blocked if device is sole provider of a capability required by
  active desired_state or active schedule periods
- Inactive schedules ignored at assignment time
- Error response includes `hint` field

### Schedule period rules
- `days_of_week` values: 1–7 (ISO 8601, Monday=1 Sunday=7)
- `end_time` must be strictly greater than `start_time` — no midnight crossing
- Overlap rejected per shared day
- Period overlap detection uses PostgreSQL `&&` array overlap operator

---

## Device-service design

See `docs/architecture/device-service.md` for full detail. Key points:

**Startup sequence:** load config → connect → start health server (returns 503) →
warm cache → start ingestion → create/drain Redis consumer group → start stream
consumer → start per-room control loop goroutines (staggered) → start periodic
cache refresh → mark health server ready (returns 200) → block on signal.

**Health server:** minimal HTTP server on `:8081`, internal to container only —
not published on host. Returns 503 until `SetReady()` called after full startup,
then 200. Port hardcoded in `internal/health` package.

**Timing constants:**

| Parameter | Env var | Default | Rationale |
|---|---|---|---|
| Simulator tick interval | — | 10s | Matches CONTROL_TICK_INTERVAL_SECONDS |
| Stale threshold | `CONTROL_STALE_THRESHOLD_SECONDS` | 90s | 3-reading window — silent >90s means unavailable |
| Control loop tick | `CONTROL_TICK_INTERVAL_SECONDS` | 10s | Shared with simulator as system heartbeat |
| Cache refresh interval | `CONTROL_CACHE_REFRESH_MINUTES` | 5min | Safety net for missed stream events |
| Stream block timeout | — | 5s | Bounds shutdown latency |

**Control loop truth table:**

| Readings | Mode | Value vs target | Command |
|---|---|---|---|
| Stale/missing | any | — | OFF (safe failure) |
| Fresh | OFF | — | OFF |
| Fresh | AUTO | Below target − deadband | ON |
| Fresh | AUTO | Above target + deadband | ON→OFF |
| Fresh | AUTO | Within deadband | Re-send last state |

Unconditional command every tick — doubles as device heartbeat. Deadband re-sends
last state to service device watchdog timers. Safe failure to OFF on stale readings.

**Redis stream:** `stream:cache_invalidation`. Per-instance consumer group
`device-service-{hostname}` — every instance needs every event. Created at stream
tip (`$`) on first start. On restart: drain pending with `0-0` before live reads.
Unknown events acked and skipped. Failed dispatches not acked — redelivered on restart.

**MQTT topics:**
- Telemetry: `devices/{hw_id}/telemetry` — QoS 1, device → device-service
- Commands: `devices/{hw_id}/cmd` — QoS 2, device-service → device

Commands always flow device-service → Mosquitto → device directly, even in Phase 7.
The Kafka bridge handles telemetry only.

---

## TimescaleDB schema (metricsdb)

```sql
CREATE TABLE sensor_readings (
    time      TIMESTAMPTZ NOT NULL,
    sensor_id UUID NOT NULL,
    room_id   UUID,
    value     NUMERIC NOT NULL,
    raw_value NUMERIC NOT NULL
);

CREATE TABLE room_control_logs (
    time               TIMESTAMPTZ NOT NULL,
    room_id            UUID NOT NULL,
    avg_temp           NUMERIC,
    avg_hum            NUMERIC,
    mode               TEXT CHECK (mode IN ('OFF', 'AUTO')),
    target_temp        NUMERIC,
    target_hum         NUMERIC,
    control_source     TEXT CHECK (control_source IN ('manual_override', 'schedule', 'grace_period', 'none')),
    heater_cmd         SMALLINT CHECK (heater_cmd IN (0, 1)),
    humidifier_cmd     SMALLINT CHECK (humidifier_cmd IN (0, 1)),
    deadband_temp      NUMERIC,
    deadband_hum       NUMERIC,
    reading_count_temp SMALLINT,
    reading_count_hum  SMALLINT,
    schedule_period_id UUID
);
```

Both tables are TimescaleDB hypertables partitioned by `time` with 1-day chunks.
`sensor_readings.room_id` is snapshotted at write time — accurate historical metrics
even after device reassignment. `heater_cmd`/`humidifier_cmd` are SMALLINT not
BOOLEAN — `AVG()` produces a duty cycle fraction without casting.
`deadband_temp`/`deadband_hum` are snapshotted from the room cache at write time —
preserves historical context when deadbands change. api-service reads
`room_control_logs` for climate endpoints; `sensor_readings` is currently write-only
from device-service.

---

## Simulator design

See `docs/architecture/simulator.md` for full detail. Key points:

**Invocation:** simulator-service has `profiles: ["simulation"]` in docker-compose —
excluded from normal `docker compose up`. Use `make simulator-start SIM=<n>` or
`make demo` (starts stack + default simulation). `docker compose run` used for teardown.

**Config system:** four template files (`rooms.yaml`, `devices.yaml`,
`desired_states.yaml`, `schedules.yaml`) and simulation files that compose them into
a topology. Templates dissolved into flat concrete structs at load time. Simulation
selected via `--simulation` flag — no default, must be explicit on every invocation.

**Identity generation (deterministic):**
```
email:  sim-{sim_name}-user-{000}@{domain}
hw_id:  sim-{sim_name}-{user_idx}-{room_idx}-{device_idx}
```
Same simulation name always produces the same identities — makes provisioning
idempotent across hard resets and teardown/re-run cycles.

**Provisioning sequence (per user):**
1. Login-first auth — register only on 401
2. Fetch existing rooms and devices — build name/hw_id lookup maps
3. Create rooms (409 → lookup by name)
4. Create and assign devices (409 → lookup by hw_id)
5. PUT desired state if configured for the room (always PUT — idempotent full replacement)
6. Create schedule + periods + activate if configured for the room
   (name 409 → lookup by name; period 409 → skip; capability conflict → fatal error)

**Desired state vs schedule — mutually exclusive per room:**
- `desired_state` — AUTO mode with indefinite override. Used for direct control
  behaviour testing. Mode and override are implicit — not configurable in YAML.
- `schedule` — time-windowed control. Used for realistic simulation. Content not
  overridable inline — create a new template for different values.
- A room entry with both set is a load-time error.

**Teardown:** `--mode=teardown` calls `DELETE /users/me` for each simulated user.
Cascades to all rooms, devices, and schedules via DB foreign key constraints.
Device-service caches for deleted rooms become stale but are corrected on the next
provisioning run via Redis stream invalidation.

**Environment model:**

All rooms use a single `EnvironmentModel` with a thermal equation applied uniformly
per measurement type. The model is agnostic to measurement type — temperature and
humidity use the same equation with different parameters.

```
effectiveAmbient = Ambient[type] + N(0, roomNoise[type])
energyInput      = heatInput[type] * simulatedTickSeconds
passiveLoss      = conductance[type] * (Current[type] - effectiveAmbient) * simulatedTickSeconds
delta[type]      = (energyInput - passiveLoss) / thermalMass[type]
```

Base physical constants defined once in `internal/config/config.go`:

| Constant | Value | Unit |
|---|---|---|
| `baseThermalMassTemperature` | 10,000,000 | J/°C |
| `baseThermalMassHumidity` | 325 | abstract moisture units |
| `baseConductanceTemperature` | 100 | W/°C |
| `baseConductanceHumidity` | 0.001 | %RH/s per %RH |
| `baseRateTemperature` | 1,000 | W |
| `baseRateHumidity` | 0.009 | %RH/s |

Templates carry only scale multipliers — `thermal_mass_scale` and `conductance_scale`
on room templates, `rate_scale` on device actuator entries. Simulation file entries
can override template scales. All default to 1.0 if absent.

**Two-axis room behaviour:**

- `type` field on simulation room entry: `static` (default) or `reactive`. Static rooms have
  no actuators contributing to `HeatInput` — `Current` tracks `EffectiveAmbient` only.
  Reactive rooms accumulate actuator contributions. Both use the same `EnvironmentModel`.
- `noisy` field on simulation room entry: `false` (default) or `true`. When false, all
  noise fields are zeroed at config load time — model and sensors run deterministically.
  Sensor noise is never zeroed — it is a hardware characteristic independent of `noisy`.

**`RoomState`:**
```go
type RoomState struct {
    Mu            sync.RWMutex
    Current       map[string]float64             // evolves per tick
    Ambient       map[string]float64             // never changes after init
    contributions map[string]map[string]float64  // hwID/type → measurementType → watts
    heatInput     map[string]float64             // derived sum, recomputed on change
}
```

Actuator contributions keyed by `hwID/measurementType` — a climate-controller with
both temperature and humidity actuators has two independent contribution entries.
`HeatInput()` returns a snapshot safe to read without holding `Mu`.

**Goroutine structure (split concerns):**
- Sensor goroutine per device with sensors — publish loop, calls `advanceRoom` each tick
- Actuator goroutine per actuator per device — command subscription + watchdog
- `advanceRoom` snapshots `HeatInput()` before acquiring `Mu.Lock`, passes snapshot
  to `model.Advance` to avoid deadlock

**Actuator command format:**
```json
{"actuator_type": "heater", "state": true}
```
`actuator_type` uses API-facing names (`heater`, `humidifier`). Translated to
measurement types via `config.ActuatorNameToMeasurement` map at command receipt.
Each actuator goroutine filters for its own measurement type and ignores others.

**Time scaling:**

```
naturalInterval           = baseTickSeconds / timeScale
effectivePublishInterval  = max(naturalInterval, minPublishInterval)
simulatedTickSeconds      = timeScale * effectivePublishInterval.Seconds()
watchdogTimeout           = baseTickSeconds * watchdogMultiplier (real wall-clock)
```

`baseTickSeconds` = `CONTROL_TICK_INTERVAL_SECONDS` — shared with device-service.
`minPublishIntervalMS` defaults to 500ms, overridable in simulation YAML via
`min_publish_interval_ms`. `maxTimeScale` = 400 — hard cap, validated at load time.
`watchdogMultiplier` = 3 — hardcoded constant.

Below the floor crossover (`timeScale ≤ baseTickSeconds / minPublishInterval`),
`simulatedTickSeconds` always equals `baseTickSeconds`. Above the floor, publish rate
is capped and `simulatedTickSeconds` grows proportionally to maintain correct physics.

**Equilibrium reference** (standard room, single device, ambient 20°C / 50% RH):
- Temperature: `1000 / 100 = 10°C above ambient → 30°C`
- Humidity: `0.009 / 0.001 = 9% above ambient → 59% RH`
Parameters may need tuning once the client graphs are available.

**Simulation files:**
- `default.yaml` — single user, standard room, full climate device set, `time_scale: 60`,
  `min_publish_interval_ms: 2000`, comfort desired state
- `cache-test.yaml` — 5 rooms covering all capability combinations, `time_scale: 1.0`,
  no desired state or schedules (telemetry-only, used for cache warm verification)
- `demo.yaml` — 4 users across a 2x2 matrix (device config × control mechanism),
  3 rooms each, `time_scale: 10`

**demo.yaml user matrix:**

| User | Device config | Control mechanism |
|---|---|---|
| user-000 | single sensor per type | desired state (indefinite override) |
| user-001 | multiple sensors per type | desired state (indefinite override) |
| user-002 | single sensor per type | schedules |
| user-003 | multiple sensors per type | schedules |

**Publish loop:** sensor goroutines staggered by `effectivePublishInterval * deviceIndex / totalDevices`.

**Mosquitto users:** `device`, `device-service`, `healthcheck` (hardcoded credentials,
not in `.env`). ACL restricts topics per username. Password file generated in CI
via `make mosquitto-passwd` equivalent — not committed to repo.

---

## Web client design (Phase 6)

See `docs/architecture/client.md` for full detail. Key points:

**Stack:** React 19 + JavaScript + Vite + SWR + shadcn/ui + Tailwind + Recharts +
React Router. No TypeScript. No Redux.

**Navigation structure:**
```
Login → Dashboard (room cards grid)
         ├─→ Room detail
         │     ├─ Overview tab   (current state + control panel, side-by-side)
         │     ├─ History tab    (climate chart, window selector)
         │     ├─ Schedules tab  (schedule list, inline period expansion, modal for period edit)
         │     └─ Devices tab    (read-only, links to Devices page)
         └─→ Devices page        (all devices, inline assignment/unassignment)
```

**Overview tab:** side-by-side panels. Left: live readings (temperature, humidity),
actuator state indicators, control source, last updated. Right: control panel —
mode selector (OFF/AUTO), target temp/humidity, manual override toggle with duration.

**History tab:** Recharts line chart. `connectNulls={false}` — gaps mean device
offline. Target band overlay (`target ± deadband`) rendered as dashed lines.
Heater/humidifier duty cycle on secondary axis or separate panel. Window selector:
`1h`, `6h`, `24h`, `7d` buttons, default `24h`. Re-fetches on window change.

**SWR polling:**
- Dashboard + overview tab: poll every 30s
- History tab: poll every 60s + `revalidateOnFocus: true`. Full re-fetch on each
  poll — avoids mixing raw and bucketed data at the chart edge.

**Auth:** JWT access token in memory (not localStorage). Refresh token in httpOnly
cookie. SWR intercepts 401 → trigger refresh → retry. Silent re-auth on page load.

**Capability-aware rendering:** client uses `GET /rooms/:id` sensor/actuator list
to determine which controls and indicators to render. Distinguishes structural nulls
(no humidifier) from transient nulls (humidifier exists, no recent reading).

---

## NGINX design (Phase 5)

Single entry point for the entire stack. React static files served directly.
`/api` proxied to api-service via Docker service DNS name — Docker round-robins
across all running api-service instances automatically.

```nginx
upstream api {
    server api-service:8080;  # Docker DNS resolves to all instances
}
```

No instance enumeration in nginx.conf — scaling via `docker compose up --scale
api-service=N` requires no config change. NGINX does not know or care how many
instances exist.

Postman environments: `nginx` environment targets NGINX port (default 80).
Direct `manual` environment retained for debugging against api-service directly.

---

## Kafka architecture (Phase 7)

See `docs/architecture/kafka.md` for full detail. Key points:

Single topic `telemetry`, 24 partitions, KRaft mode (no ZooKeeper).
Partition key: `room_id` bytes via murmur2 hash.

**MQTT bridge** (new service, Phase 7a): subscribes to Mosquitto telemetry,
stamps `room_id`/`device_id` onto payload, produces to Kafka. Maintains local
`hw_id → room_id` cache. Consumes Redis Stream as consumer group `bridge`
(independent from `device-service` group). Stateless routing — no control logic.

**device-service Phase 7b:** `mqtt.Source` replaced by `kafka.Source` — same
`ingestion.Source` interface, `TelemetryMessage` struct unchanged. Partition
ownership callbacks (`OnPartitionsAssigned`, `OnPartitionsRevoked`) manage which
rooms each instance owns. `OwnsRoom()` becomes real murmur2 check. Cache warm
moves inside `OnPartitionsAssigned` callback.

Commands still flow device-service → Mosquitto → device directly — permanently.

---

## Future features

See `docs/architecture/future-features.md` for design notes. Tracked items:

- Device connection status (Redis hash, online/offline + last-seen)
- Sensor/actuator `enabled` boolean with PATCH endpoints
- Sensor calibration offset (nullable `offset` on sensors, applied at ingestion)
- Command acknowledgement (device publishes ack, device-service confirms state)
- `activatable` bool on schedule list responses
- Admin API (device blacklist)
- Connection pool size via env (`DB_POOL_SIZE`)
- Freeform simulator client (runtime room/device creation, physics model)
- Android app (Kotlin + Jetpack Compose)
- TimescaleDB retention policy + continuous aggregates (Phase 8 cleanup)