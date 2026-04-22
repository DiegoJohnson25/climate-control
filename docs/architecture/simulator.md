# Simulator Service — Architecture Reference

Provisions simulated users, rooms, devices, and schedules via the api-service
REST API. Publishes sensor telemetry via MQTT. Subscribes to actuator command
topics and feeds commands back into room state models, so the published readings
react to device-service control decisions. Fully replaces physical ESP32 hardware
for development and demonstration.

---

## Config system

Two-file separation: templates define reusable room and device types; simulation
files compose them into a concrete topology.

### Template files

`config/templates/rooms.yaml` — room environment templates. Each defines ambient
base values, noise characteristics, and optional scale overrides for thermal mass
and conductance. Physical base constants are defined once in
`internal/config/config.go` — templates only carry scale multipliers.

`config/templates/devices.yaml` — device templates. Each defines sensors,
actuators, and their characteristics. Actuator entries carry `rate_scale` — a
multiplier against the base rate constant for that measurement type.

Template references are dissolved into flat concrete structs at load time. The
runtime simulator never references the template system — it works entirely with
resolved structs.

### Simulation files

Located in `config/simulations/`. Selected via `--simulation=name` CLI flag.
No default — must be explicit on every invocation.

Each simulation file references room and device templates by ID and defines the
topology: how many users, rooms per user, devices per room. Room entries carry
`type` and `noisy` behavioural flags, and optional scale overrides that take
precedence over template scales.

### Available simulations

| File | Description |
|---|---|
| `default.yaml` | Single user, standard room, separate sensor + actuator devices, `time_scale: 120` |
| `cache-test.yaml` | 5 rooms covering all capability combinations — used for cache warm verification, `time_scale: 1.0` |
| `demo.yaml` | 2 users (single-sensor and multi-sensor variants), 3 rooms each, `time_scale: 120` |

### Available device templates

`climate-sensor`, `temp-sensor`, `humidity-sensor`, `air-quality-sensor`,
`heater`, `humidifier`, `temp-heater`, `humidity-humidifier`, `climate-controller`

---

## Identity generation

Deterministic — same simulation name always produces the same identities.
This makes provisioning idempotent across hard resets.

```
email:  sim-{simulation_name}-user-{000}@{domain}
hw_id:  sim-{simulation_name}-{user_idx}-{room_idx}-{device_idx}
```

Device names include room name prefix (`{room_name}-{device_prefix}-{index}`)
for uniqueness across a user's device namespace.

---

## Provisioning sequence

1. For each user in the simulation:
   - Attempt login. If 401, register then login.
   - Store JWT for subsequent API calls.
2. Fetch existing rooms and devices via `GET /rooms` and `GET /devices`.
   Build lookup maps keyed by name for idempotent upsert logic.
3. For each room in topology: create if not exists (409 → lookup and use existing).
4. For each device in topology: create if not exists, assign to room.
5. (Phase 4c) For each room: create schedule with periods from simulation YAML,
   activate it.
6. Write credentials file to `/app/config/credentials/{sim-name}.txt` for
   simulations with interactive groups.

All provisioning uses the api-service REST API — no direct DB access.

Actuator types in YAML use measurement type names (`temperature`, `humidity`).
These are translated to API-facing names (`heater`, `humidifier`) at provisioning
time via `config.MeasurementToActuatorName`. The simulator's internal model uses
measurement types exclusively.

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
- `false` — all room-level and actuator noise fields zeroed at config load time.
  Model and room advance are deterministic. Sensor noise is unaffected — it is a
  hardware characteristic independent of this flag.
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
at config load — the equation is unchanged, the noise term just evaluates to zero.

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

`contributions` is keyed by `hwID/measurementType` — a device with both temperature
and humidity actuators has two independent contribution entries. `heatInput` is
always derived from `contributions` via `recomputeHeatInput` — never mutated
directly. This eliminates floating-point drift and makes idempotent commands
(same command repeated) a no-op by design.

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
      conductance_scale: 2.67   # 400/150 — drafty room loses heat faster
    humidity:
      base: 50.0
      noise: 0.8
      conductance_scale: 2.5    # 0.005/0.002
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
`baseTickSeconds` — acceleration comes from more frequent ticks. Above the
crossover, ticks are capped at the floor rate and `simulatedTickSeconds` grows
to compensate.

All timing values are computed once in `config.Load()` and stored on `Config` —
`simulator.go` reads `cfg.EffectivePublishInterval` and `cfg.SimulatedTickSeconds`
directly.

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

## Schedule provisioning (Phase 4c)

Schedule definitions will live in the simulation YAML alongside room definitions.
At bootstrap, after rooms and devices are provisioned, schedules are created and
activated per room. Two additional users will be added to `demo.yaml` — one
schedule-driven, one on indefinite manual override.

---

## Teardown (Phase 4c)

`--mode=teardown` deletes all resources provisioned for a simulation in strict
dependency order:

1. Deactivate active schedules
2. Delete schedule periods
3. Delete schedules
4. Unassign devices from rooms
5. Delete devices
6. Delete rooms
7. Delete user (`DELETE /users/me`)

Teardown is idempotent — 404 responses are treated as success.

---

## Invocation

```bash
# Run a simulation
docker compose run -d simulator-service --simulation=default

# Teardown a simulation
docker compose run simulator-service --simulation=default --mode=teardown
```

`docker compose run` (not `up`) allows per-invocation `--simulation` flag.

---

## Mosquitto credentials

Three users: `device`, `device-service`, `healthcheck`.
`healthcheck` credentials are hardcoded (`healthcheck`/`healthcheck`) — not in `.env`.
Password file generated via `make mosquitto-passwd`. ACL restricts topics per username.

Simulator connects as the `device` user. All simulated devices share this credential.