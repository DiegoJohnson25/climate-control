# Simulator Service — Architecture Reference

Provisions simulated users, rooms, devices, desired states, and schedules via
the api-service REST API. Publishes sensor telemetry via MQTT. Subscribes to
actuator command topics and feeds commands back into room state models, so
published readings react to device-service control decisions. Fully replaces
physical ESP32 hardware for development and demonstration.

---

## Usage

### Starting a simulation

The simulator-service has `profiles: ["simulation"]` in docker-compose and is
excluded from the normal `docker compose up`. Use dedicated make targets:

```bash
# Start the full stack and run the default simulation
make demo

# Start a named simulation (stack must already be up)
make simulator-start SIM=default
make simulator-start SIM=demo
make simulator-start SIM=cache-test
```

`docker compose run` is used internally by make targets — it allows a
per-invocation `--simulation` flag and is the correct tool for one-shot runs.
`docker compose up` is not used for the simulator.

### Teardown

```bash
make simulator-teardown SIM=default
make simulator-teardown SIM=demo
```

Teardown deletes all resources provisioned for the simulation by deleting each
simulated user via `DELETE /users/me`. All rooms, devices, and schedules cascade
via database foreign key constraints. No ordering required — a single API call
per user is all that is needed.

Device-service caches for deleted rooms become stale after teardown. They are
corrected automatically on the next provisioning run via Redis stream
invalidation when the rooms are recreated.

### Re-running provisioning

Provisioning is fully idempotent. Running `make simulator-start` against an
already-provisioned simulation is safe — all create calls are guarded by 409
handling with name/hw_id lookups. Desired state is always PUT (full replacement).
Schedule periods on 409 are skipped (overlap = already provisioned).

### Credentials file

Interactive user groups write credentials to
`/app/config/credentials/{sim-name}.txt` at the end of provisioning. This file
contains email/password for each simulated user and a summary of their rooms and
devices. Mount or `docker cp` to retrieve it for manual login via the client.

---

## Config system

Two-level separation: template files define reusable types; simulation files
compose them into a concrete topology.

### Template files

Located in `config/templates/`. Four files:

**`rooms.yaml`** — room environment templates. Each defines ambient base values,
noise characteristics, and optional scale overrides for thermal mass and
conductance. Physical base constants are defined once in
`internal/config/config.go` — templates carry only scale multipliers.

**`devices.yaml`** — device templates. Each defines sensors and actuators.
Actuator entries carry `rate_scale` — a multiplier against the base rate constant
for that measurement type.

**`desired_states.yaml`** — desired state templates. Each defines `target_temp`
and/or `target_hum`. Mode is always AUTO and `manual_override_until` is always
`"indefinite"` — neither is configurable. At least one target required.

**`schedules.yaml`** — schedule templates. Each defines a name and a list of
periods. Period name is omitted (optional in API, no functional value here).
`days_of_week` uses 1=Monday, 7=Sunday. AUTO periods require at least one target.

Template references are dissolved into flat concrete structs at load time. The
runtime simulator and provisioning code never reference the template system —
they work entirely with resolved structs.

### Simulation files

Located in `config/simulations/`. Selected via `--simulation=name` CLI flag.
No default — must be explicit on every invocation.

Each simulation file composes templates into a topology via user groups and room
entries. Room entries carry behavioural flags (`type`, `noisy`), optional
physical scale overrides, an optional `desired_state` template reference, and an
optional `schedule` template reference.

**Inline desired state overrides:** `target_temp` and `target_hum` on a room
entry override the template values per field when present. No template ref is
required — inline values alone are sufficient.

**Mutual exclusion:** a room entry with both `desired_state` and `schedule` set
is a load-time error. These are mutually exclusive control mechanisms.

**Schedule content is not overridable inline** — create a new template for
different period values.

**Simulation file template overrides:** room and device template definitions
in a simulation file's `template_overrides` block replace the shared template
for that simulation. Desired state and schedule templates cannot be overridden
inline — create a new template file entry.

### Available simulations

| File | Description |
|---|---|
| `default.yaml` | Single user, standard room, full climate device set, comfort desired state. `time_scale: 60`, `min_publish_interval_ms: 2000`. Primary simulation for development. |
| `cache-test.yaml` | 5 rooms covering all capability combinations. `time_scale: 1.0`. No desired state or schedules — telemetry-only, used for cache warm verification. |
| `demo.yaml` | 4 users across a 2x2 matrix (device config × control mechanism), 3 rooms each. `time_scale: 10`. |

### demo.yaml user matrix

| User | Device config | Control mechanism |
|---|---|---|
| user-000 | single sensor per type | desired state (indefinite override) |
| user-001 | multiple sensors per type | desired state (indefinite override) |
| user-002 | single sensor per type | schedules |
| user-003 | multiple sensors per type | schedules |

user-000 and user-002 share the same room/device topology. user-001 and user-003
share the same topology (exercises sensor aggregation). The pairing makes the
2x2 explicit: same hardware, different control mechanism.

### Available device templates

`climate-sensor`, `temp-sensor`, `humidity-sensor`, `air-quality-sensor`,
`heater`, `humidifier`, `temp-heater`, `humidity-humidifier`, `climate-controller`

---

## Identity generation

Deterministic — same simulation name always produces the same identities.
This makes provisioning idempotent across hard resets and teardown/re-run cycles.

```
email:  sim-{simulation_name}-user-{000}@{domain}
hw_id:  sim-{simulation_name}-{user_idx}-{room_idx}-{device_idx}
```

Device names include room name prefix (`{room_name}-{device_prefix}-{index}`)
for uniqueness across a user's device namespace.

---

## Provisioning sequence

Per user in the simulation:

1. Attempt login. If 401, register then login. Store JWT for all subsequent calls.
2. Fetch existing rooms (`GET /rooms`) and devices (`GET /devices`). Build lookup
   maps keyed by name (rooms) and hw_id (devices) for idempotent upsert logic.
3. For each room: `POST /rooms` — 409 → lookup by name and use existing ID.
4. For each device: `POST /devices` — 409 → lookup by hw_id and use existing ID.
   `PUT /devices/:id` to assign to room (always called, idempotent).
5. For each room with `desired_state`: `PUT /rooms/:id/desired-state`.
   Always PUT — full replacement, no conflict handling needed.
6. For each room with `schedule`:
   - `POST /rooms/:id/schedules` — 409 → `GET /rooms/:id/schedules`, find by name
   - `POST /schedules/:id/periods` for each period — 409 → skip (overlap = already provisioned)
   - `PATCH /schedules/:id/activate` — 409 (already active) → skip;
     409 (capability conflict) → **fatal error**

Steps 5 and 6 run after all devices are assigned — activation capability check
requires devices to be present.

All provisioning uses the api-service REST API — no direct DB access.

### Capability conflict on activation

A capability conflict means the room's assigned devices cannot satisfy the
schedule's period requirements (e.g. a period with `target_hum` but no humidity
sensor/humidifier assigned). This is a fatal provisioning error — it indicates a
misconfigured simulation file. The schedule template must match the room's device
capabilities.

**Common case:** rooms with only a temp sensor and heater (e.g. the `warm-room`
template with only `temp-sensor` and `heater` devices) cannot use schedule
templates that include `target_hum`. Use a temp-only schedule template for these
rooms.

---

## Room model architecture

### Physical base constants

All physical parameters are derived from base constants defined in
`internal/config/config.go`. Templates and simulation files carry only scale
multipliers — the base values are defined once and never in YAML.

| Constant | Value | Unit |
|---|---|---|
| `baseThermalMassTemperature` | 10,000,000 | J/°C |
| `baseThermalMassHumidity` | 325 | abstract moisture capacity |
| `baseConductanceTemperature` | 100 | W/°C |
| `baseConductanceHumidity` | 0.001 | %RH/s per %RH |
| `baseRateTemperature` | 1000 | W |
| `baseRateHumidity` | 0.009 | %RH/s |

### Two-axis room behaviour

Room behaviour is determined by two independent fields on the simulation file
room entry, not the template:

**`type`** (default: `static`) — complexity axis:
- `static` — no actuator contributions. `Current` tracks effective ambient only.
  Any device templates with actuators are ignored at the model level.
- `reactive` — actuator contributions drive `Current` via the thermal equation.
- `physics` (Phase 9) — full thermal model with external temperature profile.

**`noisy`** (default: `false`) — stochastic axis:
- `false` — all room-level noise fields zeroed at config load time. Model and
  room advance are deterministic. Sensor noise is unaffected — it is a hardware
  characteristic independent of this flag.
- `true` — room-level noise applied as a perturbation to effective ambient each tick.

Both axes are independent. A static noisy room wanders around ambient without
actuator input. A reactive non-noisy room responds deterministically to commands.

### EnvironmentModel

Single implementation for all non-physics room types. The same thermal equation
runs uniformly across all measurement types — temperature and humidity are handled
identically with different parameter values.

```
effectiveAmbient = Ambient[type] + N(0, roomNoise[type])
energyInput      = heatInput[type] * simulatedTickSeconds
passiveLoss      = conductance[type] * (Current[type] - effectiveAmbient) * simulatedTickSeconds
delta[type]      = (energyInput - passiveLoss) / thermalMass[type]
```

For static rooms, `heatInput` is always zero — `Current` tracks effective ambient
via the passive loss term only. For noisy rooms, `roomNoise` is nonzero and
`effectiveAmbient` wanders each tick. For non-noisy rooms, `roomNoise` was zeroed
at config load — the noise term evaluates to zero.

`PhysicsModel` (Phase 9) implements the same `RoomModel` interface with its own
internal parameters.

### RoomState

Shared runtime state for one room. Protected by `Mu sync.RWMutex`.

```go
type RoomState struct {
    Mu            sync.RWMutex
    Current       map[string]float64             // evolves per tick
    Ambient       map[string]float64             // never changes after init
    contributions map[string]map[string]float64  // hwID/type → measurementType → watts
    heatInput     map[string]float64             // derived sum of contributions
}
```

`contributions` is keyed by `hwID/measurementType` — a device with both
temperature and humidity actuators has two independent contribution entries.
`heatInput` is always derived from `contributions` via `recomputeHeatInput` —
never mutated directly. This eliminates floating-point drift and makes idempotent
commands (same command repeated) a no-op by design.

`HeatInput()` returns a snapshot safe to read without holding `Mu`. The publish
loop snapshots `HeatInput()` before acquiring `Mu.Lock` for the advance call —
this avoids deadlock since `HeatInput()` acquires `Mu.RLock` internally.

### Room template config

```yaml
- id: standard-room
  measurements:
    temperature:
      base: 20.0
      noise: 0.5
    humidity:
      base: 50.0
      noise: 0.5

- id: standard-room-high
  measurements:
    temperature:
      base: 20.0
      noise: 0.2
      conductance_scale: 2.67   # drafty room loses heat faster
    humidity:
      base: 50.0
      noise: 0.8
      conductance_scale: 2.5
```

`thermal_mass_scale` and `conductance_scale` default to 1.0 if absent. Simulation
file room entries can override template scales — simulation value wins if present,
otherwise template value is used.

---

## Actuator command subscriptions

Each actuator device runs a goroutine that subscribes to its command topic at
startup. Multiple actuator goroutines for the same device share one topic — each
filters for its own measurement type.

Command topic: `devices/{hw_id}/cmd` — QoS 2.

Command payload:
```json
{"actuator_type": "heater", "state": true}
```

`actuator_type` uses API-facing names. Translated to measurement types via
`config.ActuatorNameToMeasurement` at command receipt. Each goroutine ignores
commands not matching its measurement type.

On `state: true`: `SetContribution(hwID/type, {type: rate})`.
On `state: false`: `ClearContribution(hwID/type)`.

Device-service re-sends commands every tick as a heartbeat. The idempotent
contribution model means repeated identical commands have no effect.

### Watchdog

Each actuator goroutine runs a watchdog ticker. If no matching command arrives
within `watchdogTimeout`, the contribution is cleared — device-service is treated
as absent. This prevents actuators remaining on indefinitely if device-service
stops.

```
watchdogTimeout = baseTickSeconds * watchdogMultiplier
```

`baseTickSeconds` = `CONTROL_TICK_INTERVAL_SECONDS` — device-service's real
wall-clock tick rate. `watchdogMultiplier` = 3 (hardcoded constant). The timeout
is real wall-clock time, independent of `time_scale`.

---

## Time scaling

The simulator supports accelerated simulation via `time_scale` in the simulation
YAML. All timing is derived from this single user-facing parameter.

```
naturalInterval          = baseTickSeconds / timeScale
effectivePublishInterval = max(naturalInterval, minPublishIntervalMS)
simulatedTickSeconds     = timeScale * effectivePublishInterval.Seconds()
```

`baseTickSeconds` = `CONTROL_TICK_INTERVAL_SECONDS` (shared with device-service).
`minPublishIntervalMS` defaults to 500ms — overridable in simulation YAML via
`min_publish_interval_ms`. `maxTimeScale` = 400 — hard cap enforced at load time.

The floor crossover occurs at `timeScale = baseTickSeconds / minPublishInterval`.
With default values (`baseTickSeconds=10`, `minPublishIntervalMS=500ms`): crossover
at `timeScale=20`. Below the crossover, `simulatedTickSeconds` always equals
`baseTickSeconds`. Above the crossover, ticks are capped at the floor rate and
`simulatedTickSeconds` grows to compensate.

All timing values are computed once in `config.Load()` and stored on `Config` —
`simulator.go` reads `cfg.EffectivePublishInterval` and `cfg.SimulatedTickSeconds`
directly.

### Write volume considerations

High `time_scale` combined with a low `min_publish_interval_ms` generates a large
number of TimescaleDB rows quickly. As a reference: `demo.yaml` at `time_scale: 10`
with default 500ms floor generates roughly 1–1.5 GB/day of raw sensor readings.
`default.yaml` at `time_scale: 60` and `min_publish_interval_ms: 2000` is
appropriate for sustained local development — approximately 50–100 MB/day.

A TimescaleDB retention policy and continuous aggregates are planned for Phase 8.
Until then, use `make simulator-teardown` and `docker compose down -v` to reset
data between extended demo runs.

---

## Goroutine structure

One goroutine per concern — split by responsibility:

**Sensor goroutine** (one per device with sensors):
- Waits for stagger offset
- Ticks at `effectivePublishInterval`
- Calls `advanceRoom` each tick (snapshots `HeatInput`, acquires `Mu.Lock`,
  calls `model.Advance`, applies deltas, clamps bounds)
- Calls `publishTelemetry` (acquires `Mu.RLock`, reads `Current`, applies
  sensor noise and offset, publishes)

**Actuator goroutine** (one per actuator per device):
- Subscribes to `devices/{hw_id}/cmd` at startup
- Filters commands by measurement type
- Updates contributions via `SetContribution` / `ClearContribution`
- Runs watchdog ticker, clears contribution on timeout

Devices with no sensors (actuator-only) get only actuator goroutines.
Devices with no actuators (sensor-only) get only a sensor goroutine.

---

## Publish loop

Sensor goroutines staggered by:
```
staggerOffset = effectivePublishInterval * deviceIndex / totalDevices
```

Published value per sensor:
```
value = Current[type] + N(0, sensor.Noise) + sensor.Offset
```

Sensor noise is applied at publish time, independent of room model. It represents
measurement error of the physical sensor, not environmental variation.

---

## Mosquitto credentials

Three users: `device`, `device-service`, `healthcheck`.
`healthcheck` credentials are hardcoded (`healthcheck`/`healthcheck`) — not in `.env`.
Password file generated via `make mosquitto-passwd`. ACL restricts topics per username.

Simulator connects as the `device` user. All simulated devices share this credential.