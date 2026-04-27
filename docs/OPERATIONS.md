# Operations Guide

Operational reference for running, scaling, and debugging the Climate Control System locally.

---

## Prerequisites

- Docker Desktop with Docker Compose v2
- Go 1.25 (for local builds and `go vet`)
- Newman (`npm install -g newman`) — for running API tests

---

## Quick start

```bash
# One-time setup
cp .env.example .env
make mosquitto-passwd

# Start the full stack
make up

# Start the demo simulation
make demo
```

The web client is available at `http://localhost` once the stack is healthy.

---

## Make targets

### Lifecycle

| Target | Description |
|---|---|
| `make up` | Start the full stack — single API Server instance. Idempotent. |
| `make up-scaled [API=N]` | Start with N API Server instances (default 2). Always resyncs NGINX. |
| `make down` | Stop and remove containers. Volumes preserved. |
| `make down-hard` | Stop and remove containers and volumes. Destroys all data. |
| `make rebuild` | Rebuild all images and restart the stack. |
| `make infra` | Start infrastructure only — PostgreSQL, TimescaleDB, Redis, Mosquitto. |
| `make infra-down` | Stop infrastructure containers. Volumes preserved. |
| `make infra-down-hard` | Stop infrastructure containers and remove volumes. |
| `make mockup` | Serve the UI mockup on port 8090. |
| `make mockup-down` | Stop the mockup server. |

### Service rebuild

| Target | Description |
|---|---|
| `make rebuild-api` | Rebuild and restart API Server and NGINX together. |
| `make rebuild-device` | Rebuild and restart Control Service. |
| `make rebuild-simulator` | Rebuild and restart Simulator. |
| `make restart-device` | Restart Control Service without rebuilding. |

### Scaling

| Target | Description |
|---|---|
| `make scale-api [API=N]` | Rescale API Server to N instances (default 2). Force-recreates API Server and NGINX together so DNS is fresh. Infrastructure untouched. |

### Simulator

| Target | Description |
|---|---|
| `make simulator-start SIM=<name>` | Start simulator with the named simulation file. |
| `make simulator-stop` | Stop the simulator. Data preserved in DB. |
| `make simulator-resume` | Restart a stopped simulator container without rebuilding. |
| `make simulator-restart` | Soft restart — fresh container, re-runs provisioning. |
| `make simulator-restart-hard` | Hard restart — teardown first, then fresh start. Use when device capabilities or room topology have changed. |
| `make simulator-switch SIM=<name>` | Switch to a different simulation. Old data stays in DB. |
| `make simulator-switch-hard SIM=<name>` | Switch to a different simulation with teardown first. |
| `make simulator-teardown [SIM=<name>]` | Delete all provisioned data for the simulation via API cascade. |
| `make simulator-status` | Show current simulation name and container status. |
| `make demo` | Start the demo simulation. Stack must be running. |

### Logs

| Target | Description |
|---|---|
| `make logs` | Tail all service logs. |
| `make logs-api` | Tail API Server logs. |
| `make logs-device` | Tail Control Service logs. |
| `make logs-nginx` | Tail NGINX logs. |
| `make logs-simulator` | Tail Simulator logs. |
| `make logs-postgres` | Tail PostgreSQL logs. |
| `make logs-redis` | Tail Redis logs. |
| `make logs-mqtt` | Tail Mosquitto logs. |

### Shell access

| Target | Description |
|---|---|
| `make shell-api` | Shell into API Server container. |
| `make shell-device` | Shell into Control Service container. |
| `make shell-postgres` | psql shell into PostgreSQL (appdb). |
| `make shell-timescale` | psql shell into TimescaleDB (metricsdb). |
| `make shell-redis` | redis-cli shell into Redis. |

### MQTT inspection

| Target | Description |
|---|---|
| `make mqtt-telemetry` | Subscribe to all device telemetry topics. |
| `make mqtt-commands` | Subscribe to all device command topics. |
| `make mqtt-all` | Subscribe to all device topics. |
| `make mqtt-device HW_ID=<id>` | Subscribe to all topics for a specific device. |

### Testing

| Target | Description |
|---|---|
| `make test-api-integration` | Full integration suite. Requires a fresh database — run `make down-hard && make up` first. |
| `make test-api-smoke` | Smoke suite. Safe to run against a live database. |

### Go

| Target | Description |
|---|---|
| `make go-vet` | Run `go vet` across all services. |
| `make go-build` | Build all services locally. |

---

## Environment variables

All variables are defined in `.env`. Copy `.env.example` to `.env` on first setup — the example file contains working defaults for local development.

### Databases

| Variable | Default | Description |
|---|---|---|
| `POSTGRES_USER` | `cc` | PostgreSQL username |
| `POSTGRES_PASSWORD` | `changeme` | PostgreSQL password |
| `POSTGRES_DB` | `appdb` | PostgreSQL database name |
| `POSTGRES_PORT` | `5433` | Host port for PostgreSQL |
| `TIMESCALE_USER` | `cc` | TimescaleDB username |
| `TIMESCALE_PASSWORD` | `changeme` | TimescaleDB password |
| `TIMESCALE_DB` | `metricsdb` | TimescaleDB database name |
| `TIMESCALE_PORT` | `5434` | Host port for TimescaleDB |

### Redis

| Variable | Default | Description |
|---|---|---|
| `REDIS_PASSWORD` | `changeme` | Redis password |
| `REDIS_PORT` | `6379` | Host port for Redis |

### Mosquitto

| Variable | Default | Description |
|---|---|---|
| `MQTT_PORT` | `1883` | Host port for Mosquitto |
| `MQTT_DEVICE_SERVICE_USERNAME` | `device-service` | MQTT username for Control Service and Kafka Bridge |
| `MQTT_DEVICE_SERVICE_PASSWORD` | `changeme` | MQTT password for Control Service and Kafka Bridge |
| `MQTT_DEVICE_USERNAME` | `device` | MQTT username for devices and Simulator |
| `MQTT_DEVICE_PASSWORD` | `changeme` | MQTT password for devices and Simulator |

After changing any MQTT passwords, regenerate the password file:
```bash
make mosquitto-passwd
```

### JWT

| Variable | Default | Description |
|---|---|---|
| `JWT_SECRET` | `changeme` | JWT signing secret — use 32+ random characters in production |
| `JWT_ACCESS_TTL_MINUTES` | `15` | Access token lifetime in minutes |
| `JWT_REFRESH_TTL_DAYS` | `7` | Refresh token lifetime in days |

### Control Service tuning

| Variable | Default | Description |
|---|---|---|
| `CONTROL_STALE_THRESHOLD_SECONDS` | `90` | Readings older than this are considered stale — device treated as unavailable |
| `CONTROL_TICK_INTERVAL_SECONDS` | `10` | Control loop evaluation interval. Shared with Simulator as the system heartbeat. |
| `CONTROL_CACHE_REFRESH_MINUTES` | `5` | Periodic full cache reload interval — safety net for missed invalidation events |
| `DEVICE_DEBUG` | *(empty)* | Set to `info` for key event logging, `verbose` for full cache and tick detail |
| `DEVICE_TRACE_INGESTION` | `false` | Log every telemetry message received |
| `DEVICE_TRACE_TICK` | `false` | Log every control loop tick evaluation |

### Services

| Variable | Default | Description |
|---|---|---|
| `API_PORT` | `8080` | Internal API Server port (not published on host — all traffic enters via NGINX on port 80) |

### Simulator

| Variable | Default | Description |
|---|---|---|
| `SIMULATOR_EMAIL` | `simulator@local.dev` | Email for the simulator's own API account |
| `SIMULATOR_PASSWORD` | `changeme` | Password for the simulator's own API account |
| `SIMULATOR_SIMULATION` | `default` | Default simulation file name |

---

## Debug flags

The Control Service has three runtime debug flags configurable via `.env` without rebuilding.

**`DEVICE_DEBUG`**

| Value | Output |
|---|---|
| *(empty)* | Errors and fatal only |
| `info` | Cache warm summary, stream events received, control loop mode changes |
| `verbose` | Full cache contents on warm, every tick evaluation with readings and targets |

**`DEVICE_TRACE_INGESTION=true`**

Logs every telemetry message received — `hw_id`, `room_id`, measurement types, and values. Useful for verifying the Simulator is publishing and the Control Service is receiving correctly.

**`DEVICE_TRACE_TICK=true`**

Logs every control loop tick — effective state resolution, fresh readings, comparison against targets, and commands issued. Useful for debugging why a room is not responding as expected.

Use the dedicated targets — `debug-device-info`, `debug-device-verbose`, and `debug-device-off` update `.env` and restart the Control Service automatically:

```bash
make debug-device-info
make debug-device-verbose
make debug-device-off
```

The trace flag targets only update `.env` so you can set multiple flags before restarting. Once all flags are set, apply with a single restart:

```bash
make debug-device-trace-ingestion-on   # or -off
make debug-device-trace-tick-on        # or -off
make restart-device
```

---

## Horizontal scaling

### API Server

The API Server is stateless — scale freely:

```bash
# Start with 3 instances
make up-scaled API=3

# Rescale a running stack
make scale-api API=3
```

NGINX round-robins across all instances via Docker DNS. Always use `make scale-api` rather than `docker compose up --scale` directly — the make target force-recreates NGINX so DNS resolution is fresh.

### Control Service

The Control Service is stateful — scaling requires Kafka (Phase 7). In Phases 3–6, run a single instance only.

---

## Metrics database size

Check TimescaleDB hypertable sizes:

```bash
make db-metrics-size
```

High time-scale simulations generate data quickly. As a reference:

| Simulation | Time scale | Approximate write rate |
|---|---|---|
| `default.yaml` | 60× | ~50–100 MB/day |
| `demo.yaml` | 10× | ~200–400 MB/day |

To reset all metrics data:
```bash
make simulator-teardown
make down-hard
make up
```

A TimescaleDB retention policy and continuous aggregates are planned for Phase 8.