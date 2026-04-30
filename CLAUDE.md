# Climate Control — CLAUDE.md

## Project overview

A distributed IoT climate control system built in Go. ESP32 relay devices publish sensor telemetry (temperature, humidity) via MQTT. A backend Control Service ingests that telemetry, evaluates a bang-bang control loop per room, and publishes relay commands back to the devices. A REST API Server lets users configure rooms, devices, schedules, and desired climate state. A React web client provides a dashboard for monitoring and control. A Simulator stands in for physical hardware, making the system fully demonstrable without any ESP32s.

The architecture is designed to scale: the API Server scales horizontally behind NGINX with no code changes (Redis-backed state, stateless request handling). The Control Service scales via Kafka partition ownership in Phase 7 — each room hashes to one partition, owned by exactly one instance. The Kafka Bridge ingests both MQTT telemetry and Redis stream cache invalidation events into Kafka, routing each by room_id so every event reaches the correct Control Service instance.

The project serves as a portfolio piece demonstrating distributed systems design, event-driven architecture, IoT protocols, and full-stack development across Go backend, React frontend, and infrastructure layers.

---

## Architecture summary

**API Server** (`api-service`) — Gin REST API. Owns all user-facing configuration: rooms, devices, schedules, desired state. Writes to PostgreSQL (appdb) via GORM. Publishes cache invalidation events to a Redis Stream on any state change that the Control Service needs to know about. JWT auth with Redis-backed refresh token rotation. Refresh token stored in httpOnly cookie — not in response body. Read-only TimescaleDB access via pgx for climate endpoints.

**Control Service** (`device-service`) — Standalone Go binary. Minimal HTTP server on `:8081` for health checks only. Subscribes to MQTT telemetry (Phases 3–6) or Kafka (Phase 7), maintains an in-memory cache of room and device state, runs a bang-bang control loop per room, publishes actuator commands via MQTT, writes sensor readings and control logs to TimescaleDB. In Phases 3–6, consumes the Redis Stream from the API Server to keep its cache in sync. In Phase 7, receives both telemetry and cache invalidation events via Kafka — no direct Redis dependency.

**Simulator** (`simulator-service`) — Provisions simulated users, rooms, devices, desired states, and schedules via the API Server REST API. Publishes realistic sensor telemetry via MQTT. Reacts to actuator commands — room temperature and humidity evolve based on active actuator contributions and passive return-to-ambient. Allows the full system to be demonstrated without physical hardware. Started via `make simulator-start SIM=<n>` or `make demo`.

**Kafka Bridge** (`kafka-bridge`) — New service added in Phase 7. The single entry point for all data entering the Kafka pipeline. Subscribes to Mosquitto telemetry, resolves `hw_id` to `room_id`, and produces to Kafka topic `telemetry`. Also consumes the Redis Stream and forwards all cache invalidation events to Kafka topic `cache-invalidation`, keyed by `room_id`. Both topics use the same murmur2 partition key — a room's telemetry and invalidation events always route to the same Control Service instance. The Control Service no longer subscribes to Mosquitto or Redis directly in Phase 7.

**Web Client** (`web-client`) — React SPA (Phase 6). Served as static files by NGINX. Consumes the API Server REST API directly via NGINX proxy. Dashboard showing room climate state, control panel, history charts, schedule management, device management.

**NGINX** — Single entry point (Phase 5b). API Gateway — reverse proxy and load balancer. Serves React static files from `web-client/dist`. Proxies `/api/` to the API Server via Docker DNS round-robin load balancing. API Server port 8080 is not exposed on the host — all traffic enters via NGINX on port 80.

**Infrastructure:** PostgreSQL (appdb), TimescaleDB (metricsdb), Redis, Mosquitto MQTT broker, Apache Kafka (Phase 7), all containerised via Docker Compose.

---

## Current state

**Completed:**
- Phase 1 ✅ — repo scaffold, Docker Compose infrastructure, full DB schema
- Phase 2 ✅ — API Server: all REST endpoints, JWT auth, Redis refresh token rotation
- Phase 3a ✅ — Simulator scaffold: config, provisioning, MQTT publish loop
- Phase 3b ✅ — Control Service scaffold: config, cache warm, appdb repository
- Phase 3c ✅ — Control Service ingestion: MQTT source, TimescaleDB writes
- Phase 3d ✅ — Control Service control loop: bang-bang, command publishing, scheduler
- Phase 3e ✅ — Control Service stream: Redis stream consumer, cache invalidation
- Phase 4a ✅ — CI pipeline, Control Service health server, compose health checks, simulator profile
- Phase 4b ✅ — reactive room model, environment model, time scaling, physical units
- Phase 4c ✅ — desired state + schedule provisioning, teardown, demo.yaml 4-user topology
- Phase 5a ✅ — `GET /rooms/:id/climate`, `GET /rooms/:id/climate/history`, Postman verified
- Phase 5b ✅ — NGINX reverse proxy, static file serving, API proxy, horizontal scaling verified, auth cookie rewrite
- Phase docs ✅ — UI/UX mockup, README, full architecture docs overhaul, diagram specifications
- Phase 6a ✅ — Vite scaffold, design tokens, auth flow, routing, Nav, login/register pages, SWR fetcher, dark mode
- Phase 6b ✅ — Dashboard, room cards, room detail shell, overview tab, useUser/useRoom/useRooms/useClimate/useSchedules hooks, rename/delete modals, PUT /users/me, room capabilities API
- Phase 6c ✅ — manual_active schema migration, useDesiredState hook, fully wired control panel, tolerances modal, blur validation, capability toggles, CSS tooltips
**Active branch:** `feat/client-history` — Phase 6d


---

## Development phases

| Phase | Branch | Scope |
|---|---|---|
| 1–4c | — | Repo scaffold, full API, Control Service, Simulator, CI, reactive model, schedules | ✅ Done |
| 5a | `feat/api-service-climate` | `GET /rooms/:id/climate`, `GET /rooms/:id/climate/history`, Postman verified | ✅ Done |
| 5b | `feat/nginx` | NGINX reverse proxy, static file serving, API proxy, horizontal scaling, auth cookie rewrite | ✅ Done |
| docs | `feat/docs-and-mockup` | UI/UX mockup, README, architecture docs split (explanation + reference pairs), diagram specifications, naming convention updates | ✅ Done |
| 6a | `feat/client-scaffold` | Vite project, routing, auth flow, JWT handling, persistent nav, SWR setup | ✅ Done |
| 6b | `feat/client-rooms` | Dashboard room cards, room detail shell, overview tab (read-only), useUser hook, Nav email | ✅ Done |
| 6c | `feat/client-control` | manual_active schema migration, useDesiredState hook, fully wired control panel, tolerances modal, blur validation, capability toggles, CSS tooltips | ✅ Done |
| 6d | `feat/client-history` | History tab — two stacked Recharts charts, window selector, duty cycle overlays |
| 6e | `feat/client-schedules` | Schedules tab — schedule list, period accordion, period modal (clock + timeline modes) |
| 6f | `feat/client-devices` | Room-scoped devices tab + global devices page, inline room assignment |
| 6g | `feat/client-polish` | Account settings modal, timezone picker, empty states, error states, loading skeletons, client-reference.md |
| 7a | `feat/kafka-bridge` | Kafka Bridge service, Kafka cluster in docker-compose |
| 7b | `feat/kafka-control-service` | Replace `mqtt.Source` with `kafka.Source`, partition ownership callbacks, `OwnsRoom()` real implementation, Kafka-routed cache invalidation |
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
│       └── stream/       # Redis stream consumer, cache invalidation (Phases 3–6)
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
│   ├── src/
│   │   ├── api/          # auth.jsx, fetcher.js, users.js
│   │   ├── components/   # Nav.jsx, ProtectedRoute.jsx, TimezonePrompt.jsx, RoomCard.jsx
│   │   ├── hooks/        # useUser.js, useRooms.js, useRoom.js, useClimate.js, useSchedules.js
│   │   ├── lib/          # helpers.js, utils.js
│   │   ├── pages/        # LoginPage.jsx, RegisterPage.jsx, DashboardPage.jsx,
│   │   │                 # RoomDetailPage.jsx, DevicesPage.jsx
│   │   │   └── tabs/     # OverviewTab.jsx (+ history/schedules/devices in later phases)
│   │   └── styles/       # tokens.css
│   ├── mockup/           # Static HTML/CSS mockup — served on port 8090 via make mockup
│   └── dist/             # Build output served by NGINX
├── kafka-bridge/         # Phase 7 — Kafka Bridge service (MQTT + Redis → Kafka)
├── deployments/
│   ├── docker-compose.services.yml
│   ├── docker-compose.prod.yml
│   ├── mosquitto/        # mosquitto.conf, passwd, acl
│   ├── mockup/nginx.conf # NGINX config for mockup profile — mirrors deployments/nginx/nginx.conf
│   └── nginx/nginx.conf  # NGINX config — upstream, rate limiting, static serving, SPA fallback
├── migrations/
│   ├── appdb/
│   └── metricsdb/
├── docs/
│   ├── architecture/
│   │   ├── assets/           # SVG diagrams — system-topology.svg (+ others as built)
│   │   ├── control-service.md          # architecture explanation
│   │   ├── control-service-reference.md # data structures, startup, timing, topics
│   │   ├── simulator.md                # architecture explanation
│   │   ├── simulator-reference.md      # YAML schema, data structures, timing math
│   │   ├── kafka.md                    # Kafka Bridge + Phase 7 scaling design
│   │   ├── client.md                   # web client architecture explanation
│   │   ├── client-reference.md         # file structure, hooks, routes, design tokens
│   │   ├── schema.md                   # Mermaid ERD — appdb + metricsdb
│   │   └── future-features.md
│   ├── OPERATIONS.md         # make targets, env vars, scaling, debug flags
│   └── DEMO.md               # demo walkthrough, simulation configs
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
| Gin | HTTP framework — API Server only |
| GORM | ORM — appdb queries in API Server |
| golang-jwt | JWT access token signing and validation |
| go-redis/v9 | Refresh token storage, rate limiting, Redis Streams |

### Control Service

| Technology | Purpose |
|---|---|
| pgx/v5 | Raw SQL — all TimescaleDB queries, Control Service appdb queries use Raw+Scan |
| Eclipse Paho Go | MQTT client — aliased `pahomqtt` to avoid package name collision |
| franz-go | Kafka client — Phase 7, replaces Paho as telemetry source in Control Service |

### Simulator

| Technology | Purpose |
|---|---|
| Eclipse Paho Go | MQTT client — telemetry publish, command subscribe |
| goccy/go-yaml | Simulation config and template loading |

### Web Client (Phase 6)

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
| NGINX | Reverse proxy, load balancer, static file serving — port 80 |

### Messaging

| Technology | Purpose |
|---|---|
| Mosquitto 2.x | MQTT broker — auth enabled, ACL per username — host port 1883 |
| Apache Kafka (KRaft) | Telemetry + cache invalidation pipeline — Phase 7, 24 partitions, no ZooKeeper |

### Tooling & testing

| Technology | Purpose |
|---|---|
| Newman + Postman | API integration and smoke tests |
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

# Control Service tuning
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

API Server port 8080 is internal only — not published on the host. All external
traffic enters via NGINX on port 80.

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

### GORM style (API Server)

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

### GORM Raw+Scan (Control Service)

Control Service never uses GORM model-based queries. All appdb queries use
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
- Refresh token transported as httpOnly cookie — never in request/response body

### Auth cookie conventions

- Cookie name: `refresh_token`
- `httpOnly: true` — inaccessible to JavaScript
- `secure: false` — plain HTTP in development
  - TODO: set Secure to true if TLS is configured
- `path: /` — sent on all requests to the same origin
- `maxAge` matches `JWT_REFRESH_TTL_DAYS` — computed in `NewHandler` as `days * 24 * 60 * 60`
- Login: sets cookie, returns only `access_token` in body
- Refresh: reads cookie, sets new cookie, returns only `access_token` in body
- Logout: reads cookie, clears cookie (maxAge -1), returns confirmation message

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
PUT    /api/v1/users/me
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

`PUT /api/v1/users/me` — accepts `{ "timezone": string }` (IANA timezone string).
Validates via `time.LoadLocation`. Returns 204 on success. Pointer field — nil means
not provided, skip. Additional profile fields added here in future phases.

`GET /rooms` and `GET /rooms/:id` — both include a nested `capabilities` object:
```json
{ "capabilities": { "temperature": true, "humidity": false } }
```
Temperature capability = temperature sensor + heater both assigned to the room.
Humidity capability = humidity sensor + humidifier both assigned. EXISTS + EXISTS
pattern in `room.Repository`. Bulk query for list endpoint, per-room for detail.
`RoomCapabilities` and `RoomWithCapabilities` types defined in `room` package.
`HasTemperatureCapability` and `HasHumidityCapability` delegate to `RoomCapabilities`
to keep SQL in one place.

`/climate` — current snapshot sourced from the most recent `room_control_logs` row.
Returns 204 (no body) if the room has no data yet — valid no-data state, not an error.
Fields: `time`, `avg_temp`, `avg_hum`, `mode`, `target_temp`, `target_hum`,
`control_source`, `heater_cmd` (bool or null), `humidifier_cmd` (bool or null),
`deadband_temp`, `deadband_hum`.

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
- API Server writes when user makes direct control request
- Control Service reads at each tick via `resolveEffectiveState` — never writes to desired_states
- `manual_active = false` means scheduler controls the room; `manual_override_until` is always null in this state
- `manual_active = true` + `manual_override_until = 9999-12-31T23:59:59Z` means indefinite override
- `manual_active = true` + `manual_override_until = <timestamp>` means timed override (UI does not expose this yet)
- API contract: client sends `"indefinite"`, RFC3339 timestamp string, or `null` for `manual_override_until`
- `mode` stores the configured manual preference — always `OFF` or `AUTO`, never null
- `target_temp` / `target_hum` persist independently of `manual_active` and `mode` — represent saved user preferences even when manual control is inactive. Only nulled when user explicitly disables that capability toggle in the control panel.
- `desired_states.id` is vestigial — `room_id` is the natural PK but migration not worth it
- Default deadbands set in `room.Service.Create` as named constants — not in DB migration:
  `defaultTempDeadband = 0.5` and `defaultHumDeadband = 2.0`

### Effective state resolution

`resolveEffectiveState` in the Control Service synthesises effective state from
multiple durable inputs. `desired_states` is the source of truth for manual
override intent — it is one input among several, not the complete source of truth
for moment-to-moment system behaviour. Effective state is computed, ephemeral,
and never persisted.

| Priority | Condition | Source |
|---|---|---|
| 1 | `manual_active = true` and `manual_override_until` in future | `desired_states` — manual hold |
| 2 | Active schedule period matches current day/time | `schedule_periods` |
| 3 | Within 60s grace period after period end | Last active period |
| 4 | None of the above | Mode OFF |

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

## NGINX design

Single entry point for the entire stack. All traffic enters via NGINX on port 80.
API Server port 8080 is internal to the Docker network — not published on the host.

```nginx
upstream api {
    server api-service:8080;  # Docker DNS resolves to all instances
}
```

**Routing:**
- `location /api/` — proxied to API Server. No trailing slash on `proxy_pass` —
  preserves the full URI including the `/api/` prefix.
- `location /` — static files from `/usr/share/nginx/html` (bind-mounted from
  `web-client/dist`). `try_files $uri $uri/ /index.html` for React Router SPA fallback.

**Rate limiting:** `limit_req_zone $binary_remote_addr zone=api:10m rate=30r/s`
with `burst=60 nodelay`. Intentionally loose — framework for future tuning, not
a strict guard. Tune before any public deployment.

**Cache headers:**
- `.js` and `.css` assets: `Cache-Control: public, immutable`, `expires 1y` —
  safe because Vite content-hashes asset filenames on every build.
- Everything else (including `index.html`): `Cache-Control: no-cache` — always
  revalidated, never stale.

**Proxy headers forwarded:** `X-Real-IP`, `X-Forwarded-For`, `X-Forwarded-Proto`, `Host`.

**Horizontal scaling:**
- Scale via `make up-scaled API=N` or `make scale-api API=N`.
- Docker DNS round-robins across all API Server instances automatically — no
  nginx.conf changes needed.
- NGINX resolves upstream DNS once at startup. Instances added after NGINX starts
  will not receive traffic until NGINX restarts. Always start with the desired
  instance count using `make up-scaled` — do not scale up after the fact.
- `make scale-api` force-recreates both API Server and NGINX together to ensure
  DNS is always fresh. Infrastructure services are untouched (`--no-deps`).

**Makefile targets:**
- `make up` — single API Server instance, idempotent
- `make up-scaled [API=N]` — N instances (default 2), always resyncs NGINX
- `make scale-api [API=N]` — rescale running stack, resyncs NGINX, no infra restart

---

## API Server design

**Auth handler (`auth/handler.go`):**
- `NewHandler(svc *Service, refreshTTLDays int)` — TTL passed in from `main.go`
  via `cfg.JWTRefreshTTLDays`, stored as `refreshTTLSecs` (converted at construction)
- `setRefreshCookie` / `clearRefreshCookie` — unexported helpers, called by
  Login, Refresh, and Logout handlers
- Postman collections send no refresh token in request body — cookie is managed
  automatically by Postman's cookie jar within a collection run

---

## Control Service design

See `docs/architecture/device-service.md` for full detail. Key points:

**Startup sequence (Phases 3–6):** load config → connect → start health server
(returns 503) → warm cache → start ingestion → create/drain Redis consumer group
→ start stream consumer → start per-room control loop goroutines (staggered) →
start periodic cache refresh → mark health server ready (returns 200) → block on signal.

**Startup sequence (Phase 7):** load config → connect (no Redis) → register
partition callbacks → join Kafka consumer group → `OnPartitionsAssigned` warms
cache and starts control loops → mark health server ready → block on signal.

**Health server:** minimal HTTP server on `:8081`, internal to container only —
not published on host. Returns 503 until `SetReady()` called after full startup,
then 200. Port hardcoded in `internal/health` package.

**Timing constants:**

| Parameter | Env var | Default | Rationale |
|---|---|---|---|
| Simulator tick interval | — | 10s | Matches CONTROL_TICK_INTERVAL_SECONDS |
| Stale threshold | `CONTROL_STALE_THRESHOLD_SECONDS` | 90s | 3-reading window — silent >90s means unavailable |
| Control loop tick | `CONTROL_TICK_INTERVAL_SECONDS` | 10s | Shared with simulator as system heartbeat |
| Cache refresh interval | `CONTROL_CACHE_REFRESH_MINUTES` | 5min | Safety net for missed invalidation events |
| Stream block timeout | — | 5s | Bounds shutdown latency (Phases 3–6) |

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

**Cache invalidation (Phases 3–6):**
Redis stream `stream:cache_invalidation`. Per-instance consumer group
`control-service-{hostname}` — every instance needs every event. Created at stream
tip (`$`) on first start. On restart: drain pending with `0-0` before live reads.
Unknown events acked and skipped. Failed dispatches not acked — redelivered on restart.

**Cache invalidation (Phase 7):**
The Control Service no longer consumes Redis directly. The Kafka Bridge forwards
all invalidation events from `stream:cache_invalidation` to Kafka topic
`cache-invalidation`, keyed by `room_id`. The Control Service consumes this topic
alongside `telemetry` — same partition ownership guarantees delivery to the correct
instance. No `OwnsRoom()` self-filtering required.

**MQTT topics:**
- Telemetry: `devices/{hw_id}/telemetry` — QoS 1, device → Control Service (Phases 3–6) / Kafka Bridge (Phase 7)
- Commands: `devices/{hw_id}/cmd` — QoS 2, Control Service → device (always direct, all phases)

Commands always flow Control Service → Mosquitto → device directly, even in Phase 7.
The Kafka Bridge handles ingestion only — telemetry and invalidation events.

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
preserves historical context when deadbands change. API Server reads
`room_control_logs` for climate endpoints; `sensor_readings` is currently write-only
from the Control Service.

---

## Simulator design

See `docs/architecture/simulator.md` for full detail. Key points:

**Invocation:** Simulator has `profiles: ["simulation"]` in docker-compose —
excluded from normal `docker compose up`. Use `make simulator-start SIM=<n>`.
`make demo` starts the demo simulation only — stack must already be running via
`make up`. `docker compose run` used for teardown.

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
Control Service caches for deleted rooms become stale but are corrected on the next
provisioning run via cache invalidation events.

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

`baseTickSeconds` = `CONTROL_TICK_INTERVAL_SECONDS` — shared with Control Service.
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

## Web Client design (Phase 6)

See `docs/architecture/client.md` for full detail. Key points:

**Stack:** React 19 + JavaScript (ES2022) + Vite 8 + SWR 2 +
React Router v7 + Tailwind v4 + shadcn 4.6 (Radix primitives only) +
Recharts + lucide-react. No TypeScript. No Redux. No TanStack Query.
Node 22 required (`nvm alias default 22`).

**Dev server:** `npm run dev` in `web-client/`. Vite runs on `:5173`.
All `/api` requests proxied to `http://localhost` (NGINX on port 80).
Full stack must be running (`make up`) for API calls to resolve.

**Design system:** `src/styles/tokens.css` — single source of truth for
all `--cc-*` CSS custom properties (colors, typography, spacing, radii,
shadows, motion, layout). Full dark mode via `[data-theme="dark"]` on
`<html>`. Component-level `cc-*` CSS classes defined in `tokens.css` —
used directly in JSX for all named components (buttons, badges, inputs,
cards, modals, tables). Tailwind utilities handle layout only (flex,
grid, gap, padding). shadcn used for complex interactive primitives
(Popover, Dialog, DropdownMenu) where accessibility behavior is needed.

**File naming:** `.jsx` for any file containing JSX. `.js` for pure
utility/hook files with no JSX.

**Auth pattern:**
- Module-level token store in `src/api/auth.jsx`: `getToken()`,
  `setToken()` (private), `clearToken()` (exported)
- `doRefresh()` — `POST /auth/refresh`, deduplicates concurrent calls
  via shared in-flight promise, returns token data
- `AuthContext` + `AuthProvider` — provides `{ isAuthenticated, login,
  logout }` to component tree. `login(token)` sets module variable +
  flips boolean. `logout()` clears both.
- `useAuth()` hook — throws if used outside `AuthProvider`
- SWR global fetcher in `src/api/fetcher.js` — attaches Authorization
  header, intercepts 401, refreshes + retries once, hard redirects to
  `/login` on double 401 via `window.location.href` (outside React tree)

**Routing (React Router v7):**
- Public: `/login`, `/register`
- Protected (wrapped in `ProtectedRoute`): `/dashboard`, `/rooms/:id`,
  `/devices`
- `/` redirects to `/dashboard`
- `ProtectedRoute` — attempts silent refresh on mount if no token in
  memory; renders null while checking; redirects to `/login` on failure

**Dark mode:** `ThemeProvider` (render props pattern) in `App.jsx`.
Persisted to `localStorage` under key `cc-theme`. Toggle in Nav.
All token values swap automatically — no component changes needed.

**Registration flow:** `POST /auth/register` (email + password, min 4
chars) → immediate `POST /auth/login` with same credentials →
`login(access_token)` → redirect to `/dashboard`. No timezone at
registration — UTC default with `TimezonePrompt` banner on dashboard.

**`TimezonePrompt`:** Shown when `user.timezone === 'UTC'`. Auto-detects
browser timezone. `onSave` calls `PUT /users/me` then mutates useUser
SWR cache. Dismissed state persisted to localStorage under
`cc-timezone-prompt-dismissed`. Note: dismiss key is not per-user —
acceptable for single-user self-hosted context.

**Helpers (`src/lib/helpers.js`):** `timeAgo`, `fmtTime12`, `fmtMin12`,
`fmtTick12` — all use `Intl.DateTimeFormat` with user's IANA timezone
(not browser local time).

**Mockup:** `web-client/mockup/` — visual reference only. Do not port
code patterns — the `window.*` global pattern, raw SVG charts, and
module structure do not apply to the production app. Use mockup for
visual structure, CSS class usage, and interaction states only.

**Navigation structure:**
```
Login / Register
└── Dashboard (room cards grid)
    ├── Room detail
    │     ├── Overview tab   (current state card + control panel)
    │     ├── History tab    (two stacked Recharts charts)
    │     ├── Schedules tab  (schedule list, period modal)
    │     └── Devices tab    (full device management)
    └── Devices page         (all devices, inline room assignment)
```

**Tab routing:** local `useState` only — URL stays `/rooms/:id` regardless
of active tab. No nested routes under `/rooms/:id`.

**SWR polling intervals:**
- Dashboard room list + per-card climate: 30s
- History tab: 60s + `revalidateOnFocus: true`
- On-demand hooks (room detail, schedules, devices): no polling

**SWR hooks inventory:**
- `useUser` — `GET /users/me`, no polling, exposes `mutate`
- `useRooms` — `GET /rooms`, 30s polling, exposes `mutate`
- `useRoom(roomId)` — `GET /rooms/:id`, no polling, exposes `mutate`
- `useClimate(roomId)` — `GET /rooms/:id/climate`, 30s polling, custom
  fetcher handles 204 as null (valid no-data state)
- `useDesiredState(roomId)` — `GET /rooms/:id/desired-state`, no polling,
  `revalidateOnFocus: false` — prevents draft clobber on tab-away. Exposes
  `mutate` — called after Apply to sync hook with newly saved state.
- `useSchedules(roomId)` — `GET /rooms/:id/schedules`, no polling

**Capability-aware rendering:**
- Capabilities come from `room.capabilities.temperature` and
  `room.capabilities.humidity` on the room object — not inferred from
  climate readings. Temperature capability = sensor + heater. Humidity
  capability = sensor + humidifier.
- Dashboard cards: `ClimateReading` null fields used for `—` display.
  No capability logic needed at the card level.
- Overview current state card: actuator rows always render — null
  `heater_cmd`/`humidifier_cmd` shows `—`, not hidden.
- Control panel: capability-aware greying deferred to 6c when real
  desired state data is wired in.

**Always-show rendering philosophy:**
Components render all rows and sections regardless of data availability.
Null values show `—`. Unavailable states grey out via `cc-row--disabled`
or `opacity: 0.5`. No conditional hiding that causes layout shifts.
CSS grid with `minmax()` or proportional `fr` units preferred over
flexbox with fixed gaps — handles responsive reflow without awkward
whitespace.

**Control source label mapping:**
- `manual_override` → "Manual" (not "Hold active" — "Hold" terminology
  dropped entirely from the UI)
- `schedule` → "Schedule"
- `grace_period` → "Grace period"
- `none` → source row shows "None" in muted style, no badge variant

**Control panel design (OverviewTab card 2):**
**Control panel design (OverviewTab card 2):**
Two top-level states driven by "Control type" segmented control:
- "Schedule" — schedule section active, manual settings section greyed
- "Manual" — manual settings active, schedule section shows "Overridden by manual"
Mode (OFF/AUTO) and capability rows are subordinate to Control type.
Each capability row has an independent enable/disable toggle (`cc-togdot`).
The togdot has three visual states:
- `cc-togdot--disabled` — grey, not clickable (hardware unavailable, not in manual+auto)
- base `cc-togdot` — white/neutral, clickable (available but user chose not to regulate)
- `cc-togdot--on` — active color, clickable (available and actively regulating)
Togdot and content opacity are independent — the togdot is never dimmed by its
surrounding row's opacity, preserving its affordance as an interactive control.

`isDirty` gate: Apply and Revert only enabled when draft differs from saved
desired state. Apply button style reflects dirty state (`cc-btn--primary` when
dirty, `cc-btn--ghost` when clean).

Draft state initialised from `useDesiredState` via `useEffect` on
`[desiredState]` dep. `revalidateOnFocus: false` prevents SWR revalidation on
tab-away from clobbering unsaved edits.

Apply payload only nulls targets when capability toggle is explicitly off —
not when switching control type or mode. This preserves saved preferences across
state transitions.

Blur validation on target inputs: 5–40°C temperature, 10–90% humidity.
Blur validation on tolerance inputs: 0.1–10.0°C, 0.5–20.0%.
Red border + error message on invalid blur. Apply blocked if errors present.

CSS tooltips explain disabled states:
- `cc-tooltip` — `::after` pseudo-element anchored above element, centered
- `cc-tooltip--right` — right-edge anchor modifier for right-aligned controls
**Tolerances modal:**
- Accessible from dbpills in control panel (`showHints={true}` — threshold hints
  computed from live draft targets)
- Accessible from room detail kebab "Edit tolerances" (`showHints={false}`)
- Title: "Tolerances". Subtitle: "Wider tolerances save energy but allow more drift."
- `onSave` calls `PUT /rooms/:id` with full room body. Callers handle fetch
  and `mutateRoom()` independently.
- Deadband pills display `room.deadband_temp`/`room.deadband_hum` (not climate
  snapshot) — updates immediately after save via `mutateRoom()`.
Uncontrolled inputs use `useRef` reset counter (`resetCount`) for stable cursor
behaviour. `resetCount.current` incremented on Revert and `useEffect` reinit.

**Known limitation:** when the user disables a capability toggle in Manual+AUTO
mode and hits Apply, that target is nulled in the database. Re-enabling the toggle
later requires re-entering the target. See `future-features.md` for full note.

**Button conventions:**
- Title case throughout: "Add Room", "Delete Room", "Log Out",
  "Save Timezone", "Create Room", etc.
- Action buttons include relevant lucide-react icon where appropriate
  (Plus for add actions, etc.)

**Modals:**
- `cc-modal-bg` overlay + `cc-modal` pattern throughout
- Overlay click closes modal, inner card click does not (stopPropagation)
- Enter key submits single-input modals
- 409 conflicts surface specific error messages, other errors show
  generic "Something went wrong."
- Successful mutations call `mutate()` on relevant SWR hooks immediately
  — no waiting for next poll interval

**Deferred to 6g:**
- Account Settings modal with timezone picker (full curated IANA selector,
  ~40 entries, no external library needed)
- Loading skeletons across all pages
- Empty states across all pages
- TimezonePrompt dismiss key is not per-user — acceptable for single-user
  self-hosted deployment

---

## Kafka architecture (Phase 7)

See `docs/architecture/kafka.md` for full detail. Key points:

Two Kafka topics: `telemetry` and `cache-invalidation`. Both use `room_id` bytes
as the partition key via murmur2 hash — 24 partitions, KRaft mode (no ZooKeeper).

**Kafka Bridge** (new service, Phase 7a, `kafka-bridge`): the single ingestion
point for the Kafka pipeline. Subscribes to Mosquitto telemetry, stamps
`room_id`/`device_id` onto payload, produces to `telemetry` topic. Also consumes
Redis Stream `stream:cache_invalidation` as consumer group `kafka-bridge` and
forwards all events to `cache-invalidation` topic keyed by `room_id`. Maintains
local `hw_id → room_id` cache. Performs protocol translation and device-to-room
resolution — no business logic.

**Control Service Phase 7b:** `mqtt.Source` replaced by `kafka.Source` — same
`ingestion.Source` interface, `TelemetryMessage` struct unchanged. Consumes both
`telemetry` and `cache-invalidation` topics. Partition ownership callbacks
(`OnPartitionsAssigned`, `OnPartitionsRevoked`) manage which rooms each instance
owns. `OwnsRoom()` becomes real murmur2 check used for cache warm filtering only —
not for invalidation self-filtering. Cache warm moves inside `OnPartitionsAssigned`
callback. No Redis dependency in Phase 7.

Commands still flow Control Service → Mosquitto → device directly — permanently.

---

## Future features

See `docs/architecture/future-features.md` for design notes. Tracked items:

- Device connection status (Redis hash, online/offline + last-seen)
- Sensor/actuator `enabled` boolean with PATCH endpoints
- Sensor calibration offset (nullable `offset` on sensors, applied at ingestion)
- Command acknowledgement (device publishes ack, Control Service confirms state)
- `activatable` bool on schedule list responses
- Admin API (device blacklist)
- Connection pool size via env (`DB_POOL_SIZE`)
- Freeform simulator client (runtime room/device creation, physics model)
- Android app (Kotlin + Jetpack Compose)
- TimescaleDB retention policy + continuous aggregates (Phase 8 cleanup)