# Control Service — Architecture

Standalone Go binary. No public HTTP server — only a minimal health endpoint on
`:8081` for Docker health checks. Fully decoupled from the API Server — they share
PostgreSQL, TimescaleDB, and Redis as infrastructure but have no direct
service-to-service calls.

For low-level detail on data structures, startup sequences, timing constants, and
MQTT topics see [`control-service-reference.md`](control-service-reference.md).

---

## Responsibilities

The Control Service owns everything on the device-facing side of the system:

- **Telemetry ingestion** — receives sensor readings from devices via MQTT
  (Phases 3–6) or Kafka (Phase 7), validates them, and writes them to TimescaleDB
- **Control loop** — evaluates each room's effective state every tick and publishes
  actuator commands back to devices
- **Cache management** — maintains an in-memory snapshot of all room and device
  state so the control loop never touches the database on the hot path

---

## In-memory cache

The most architecturally significant part of the Control Service. The control loop
runs every tick — if it had to query PostgreSQL on each evaluation the database
would be on the hot path for every room, every 10 seconds, indefinitely. The cache
eliminates this entirely.

At startup the cache is warmed from PostgreSQL in a single bulk fetch per data type.
After that, PostgreSQL is only read when a cache invalidation event arrives or the
periodic safety-net refresh fires. The control loop reads only from the in-memory
cache under a read lock — zero database I/O on the hot path.

The cache has two levels:

**`Store`** — top-level container. Holds a map of `RoomCache` entries keyed by
`room_id` and a map of `DeviceCache` entries keyed by `hw_id`. A single
`sync.RWMutex` protects the maps themselves — inserting and deleting entries.
Per-struct mutexes protect individual cache entries.

**`RoomCache`** — full runtime state for one room. Contains everything the control
loop needs: desired state, active schedule periods, latest sensor readings, last
commanded actuator states, resolved timezone, actuator hw_id lookup, and pre-parsed
schedule period time windows. The exported `Mu sync.RWMutex` allows the control
loop to hold a read lock across the entire tick evaluation without re-acquiring it.

**`DeviceCache`** — device metadata. Sensor and actuator maps keyed by measurement
type for O(1) lookup during ingestion and cache warm.

### Pre-computation at warm time

Several expensive operations are done once at warm/reload time and cached — never
recomputed at tick time:

- Timezone string → `*time.Location` resolution
- Schedule period `"HH:MM"` strings → integer minutes
- `days_of_week` array → `[8]bool` bitmask indexed by ISO day
- Actuator hw_id lookup maps per measurement type

This keeps the tick evaluation path as cheap as possible — a read lock, a few map
lookups, and arithmetic.

### Runtime fields survive reloads

When a room cache is reloaded (via invalidation event or periodic refresh),
runtime-only fields are explicitly preserved from the existing entry:

- `LatestReadings` — accumulated sensor readings in the sliding window
- `ActuatorStates` — last commanded state per actuator
- `LastActivePeriod` — used for grace period evaluation

These fields represent live state that cannot be recovered from PostgreSQL. Clobbering
them on reload would break the control loop — it would lose its sensor reading
history and actuator state, producing incorrect commands on the next tick.

---

## Cache invalidation

The cache needs to stay in sync with API Server writes. When a user changes a
desired state, activates a schedule, or reassigns a device, the Control Service
must learn about it promptly — otherwise it continues evaluating against stale data.

The mechanism is a Redis Stream (`stream:cache_invalidation`). The API Server
publishes an event on any write that affects the Control Service's view of the
world. The Control Service consumes that stream and calls the appropriate reload
function.

**Why a Redis Stream and not polling?** Polling PostgreSQL on an interval would
put the database on a secondary hot path and introduce latency proportional to the
poll interval. The stream is push-based — the Control Service learns about changes
as soon as the API Server commits them.

**Why not direct database triggers?** Keeping the two services decoupled through
shared infrastructure (Redis) rather than database-level coupling means neither
service depends on the other's internal implementation.

**Phase 7 change:** in Phase 7 the Kafka Bridge consumes the Redis stream and
forwards invalidation events into a dedicated Kafka topic keyed by `room_id`. The
Control Service receives them via the same Kafka consumer group as telemetry —
routed to the correct instance by partition ownership. The Control Service has no
direct Redis dependency in Phase 7. See [`kafka.md`](kafka.md) for detail.

---

## Control loop

One goroutine per room, staggered at startup to spread load evenly across the tick
interval. Each tick:

1. Acquires the room cache read lock for the entire evaluation
2. Calls `resolveEffectiveState` to determine what the room should be doing
3. Evaluates the bang-bang decision for each measurement type with a target
4. Publishes commands to all actuators in the room
5. Releases the read lock
6. Updates `ActuatorStates`
7. Writes a `room_control_logs` row to TimescaleDB

### Effective state resolution

`resolveEffectiveState` synthesises the room's effective state from durable
sources in priority order:

| Priority | Condition | Source |
|---|---|---|
| 1 | Manual override active and not expired | `desired_states` — manual hold |
| 2 | Active schedule period matches current day/time | `schedule_periods` |
| 3 | Within 60s grace period after period end | Last active period |
| 4 | None of the above | Mode OFF |

The `desired_states` table is the source of truth for **manual override intent**
— it is one input to `resolveEffectiveState`, not the complete source of truth for
what the system does moment-to-moment. Effective state is computed, ephemeral, and
never persisted. This means correctness is guaranteed across restarts and schedule
transitions with no state synchronisation required — a restarted instance warms its
cache and immediately computes the correct effective state for every room.

### Bang-bang control

Simple two-position control: a measurement either crosses a threshold or it
doesn't. No PID, no integral term, no prediction. Appropriate for climate control
where thermal inertia already provides natural smoothing.

The deadband prevents rapid toggling near the setpoint. When the measurement is
within target ± deadband, the last commanded state is re-sent unchanged. When it
crosses a threshold, the command flips.

**Safe failure to OFF** — if no fresh readings are available (device offline,
stale beyond the threshold), all actuators in the room are commanded OFF. The
system never holds the last commanded state when it cannot verify current
conditions. This prevents runaway heaters or humidifiers if a sensor goes offline.

**Unconditional commands as heartbeat** — commands are published every tick
regardless of whether the state changed. This doubles as a liveness signal to
devices — devices run watchdog timers that revert to a safe state if they stop
receiving commands. A device that stops receiving commands treats the Control
Service as absent.

---

## Transport-agnostic ingestion

The `ingestion.Source` interface abstracts the telemetry transport entirely:

```go
type Source interface {
    Start(ctx context.Context, handler func(context.Context, TelemetryMessage)) error
    Stop()
}
```

`mqtt.Source` (Phases 3–6) and `kafka.Source` (Phase 7) both implement this
interface. `ingestion.Process` — the function that validates a message, updates
the cache, and writes to TimescaleDB — is identical regardless of transport.
Swapping from MQTT to Kafka in Phase 7 is a one-line change in `main.go`.

This abstraction also means the control loop and ingestion path are completely
decoupled from each other. Ingestion updates the cache; the control loop reads
from it. They share the cache under lock but never call each other.

---

## Horizontal scaling (Phase 7)

In Phases 3–6 a single Control Service instance owns all rooms. Scaling to
multiple instances naively would cause every instance to receive the same
telemetry message for the same room — producing duplicate, potentially conflicting
control decisions and incoherent caches.

Phase 7 solves this via Kafka partition ownership. Each room's `room_id` hashes
deterministically to one of 24 partitions. One Control Service instance owns each
partition subset. A room's telemetry and invalidation events always route to the
same instance — no duplicate processing, no conflicts.

See [`kafka.md`](kafka.md) for the full Phase 7 design.