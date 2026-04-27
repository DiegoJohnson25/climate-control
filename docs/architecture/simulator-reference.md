# Simulator — Reference

Low-level reference for the Simulator. Covers config YAML structure, data
structures, provisioning sequence, timing constants, command payload format,
and goroutine detail.

For architecture overview and design decisions see [`simulator.md`](simulator.md).

---

## Usage

### Starting a simulation

The Simulator has `profiles: ["simulation"]` in docker-compose and is excluded
from normal `docker compose up`. Use dedicated make targets:

```bash
# Start the full stack and run the demo simulation
make demo

# Start a named simulation (stack must already be running via make up)
make simulator-start SIM=default
make simulator-start SIM=demo
make simulator-start SIM=cache-test
```

`docker compose run` is used internally by make targets — it allows a
per-invocation `--simulation` flag and is the correct tool for one-shot runs.
`docker compose up` is not used for the Simulator.

### Teardown

```bash
make simulator-teardown SIM=default
make simulator-teardown SIM=demo
```

Calls `DELETE /users/me` for each simulated user. All rooms, devices, and
schedules cascade via database foreign key constraints. No ordering required.

Control Service caches for deleted rooms become stale after teardown. Corrected
automatically on the next provisioning run via cache invalidation events when
the rooms are recreated.

### Re-running provisioning

Provisioning is fully idempotent. All create calls are guarded by 409 handling
with name/hw_id lookups. Desired state is always PUT (full replacement). Schedule
periods on 409 are skipped.

### Credentials file

Interactive user groups write credentials to
`/app/config/credentials/{sim-name}.txt` at the end of provisioning. Contains
email/password per simulated user and a summary of their rooms and devices.
Retrieve with:

```bash
docker cp climate-control-simulator-service-1:/app/config/credentials/demo.txt ./demo-credentials.txt
```

---

## Config system

### Template files

Located in `simulator-service/config/templates/`. Four files:

**`rooms.yaml`** — room environment templates.

```yaml
- id: standard-room
  measurements:
    temperature:
      base: 20.0          # ambient base value °C
      noise: 0.5          # room model noise σ per tick
    humidity:
      base: 50.0
      noise: 0.5

- id: standard-room-high
  measurements:
    temperature:
      base: 20.0
      noise: 0.2
      conductance_scale: 2.67   # room loses heat faster — drafty
    humidity:
      base: 50.0
      noise: 0.8
      conductance_scale: 2.5
```

`thermal_mass_scale` and `conductance_scale` default to 1.0 if absent. Simulation
file room entries can override template scales — simulation value wins if present.

**`devices.yaml`** — device templates.

```yaml
- id: climate-controller
  sensors:
    - measurement_type: temperature
      noise: 0.1          # sensor noise σ — applied at publish, not model
      offset: 0.0         # sensor calibration offset
    - measurement_type: humidity
      noise: 0.5
      offset: 0.0
  actuators:
    - actuator_type: heater
      rate_scale: 1.0     # multiplier against base rate constant
    - actuator_type: humidifier
      rate_scale: 1.0
```

**`desired_states.yaml`** — desired state templates.

```yaml
- id: comfort
  target_temp: 22.0
  target_hum: 50.0
```

Mode is always AUTO and `manual_override_until` is always `"indefinite"`.
At least one target required.

**`schedules.yaml`** — schedule templates.

```yaml
- id: weekday-comfort
  name: Weekday Comfort
  periods:
    - days_of_week: [1, 2, 3, 4, 5]  # Mon–Fri, ISO 8601
      start_time: "08:00"
      end_time: "22:00"
      target_temp: 22.0
      target_hum: 50.0
```

Period names are omitted — optional in the API, no functional value here.
AUTO periods require at least one target.

### Simulation files

Located in `simulator-service/config/simulations/`. Selected via
`--simulation=name` CLI flag. No default — must be explicit.

```yaml
time_scale: 10
min_publish_interval_ms: 500  # optional, default 500

user_groups:
  - count: 1
    rooms:
      - template: standard-room
        devices: [climate-controller]
        type: reactive        # static | reactive
        noisy: false
        desired_state:
          template: comfort
          target_temp: 23.0   # inline override — takes precedence over template
```

**Mutual exclusion:** `desired_state` and `schedule` on the same room entry is a
load-time error.

**Template overrides:** room and device templates can be overridden per simulation:

```yaml
template_overrides:
  rooms:
    - id: standard-room
      measurements:
        temperature:
          conductance_scale: 0.5  # well-insulated for this simulation only
```

Desired state and schedule templates cannot be overridden inline — create a new
template file entry.

### Available simulations

| File | Time scale | Description |
|---|---|---|
| `default.yaml` | 60× | Single user, standard room, full climate device, comfort desired state. `min_publish_interval_ms: 2000`. |
| `cache-test.yaml` | 1× | 5 rooms covering all capability combinations. No desired state or schedules. |
| `demo.yaml` | 10× | 4 users × 3 rooms. 2×2 matrix of device config and control mechanism. |

### demo.yaml user matrix

| User | Device config | Control mechanism |
|---|---|---|
| user-000 | single sensor per type | desired state (indefinite override) |
| user-001 | multiple sensors per type | desired state (indefinite override) |
| user-002 | single sensor per type | schedules |
| user-003 | multiple sensors per type | schedules |

### Available device templates

`climate-sensor`, `temp-sensor`, `humidity-sensor`, `air-quality-sensor`,
`heater`, `humidifier`, `temp-heater`, `humidity-humidifier`, `climate-controller`

---

## Identity generation

```
email:  sim-{simulation_name}-user-{000}@{domain}
hw_id:  sim-{simulation_name}-{user_idx}-{room_idx}-{device_idx}
```

Device names: `{room_name}-{device_prefix}-{index}` — unique across a user's
device namespace.

Same simulation name always produces the same identities — provisioning is
idempotent across hard resets.

---

## Provisioning sequence

Per user in the simulation:

1. Attempt login. If 401, register then login. Store JWT.
2. `GET /rooms` + `GET /devices` → build lookup maps (name → ID for rooms,
   hw_id → ID for devices).
3. For each room: `POST /rooms` — 409 → lookup by name.
4. For each device: `POST /devices` — 409 → lookup by hw_id.
   `PUT /devices/:id` to assign to room (always called, idempotent).
5. For each room with `desired_state`: `PUT /rooms/:id/desired-state`.
6. For each room with `schedule`:
   - `POST /rooms/:id/schedules` — 409 → find by name
   - `POST /schedules/:id/periods` per period — 409 → skip
   - `PATCH /schedules/:id/activate` — already active → skip; capability conflict → **fatal error**

Steps 5 and 6 run after all devices are assigned — activation capability check
requires devices to be present.

### Capability conflict on activation

Fatal error — indicates a misconfigured simulation file. The schedule template
must match the room's device capabilities.

Common case: a room with only `temp-sensor` and `heater` cannot activate a
schedule with `target_hum`. Use a temp-only schedule template for that room.

---

## Physical base constants

Defined once in `simulator-service/internal/config/config.go`. Templates carry
only scale multipliers against these values.

| Constant | Value | Unit |
|---|---|---|
| `baseThermalMassTemperature` | 10,000,000 | J/°C |
| `baseThermalMassHumidity` | 325 | abstract moisture capacity |
| `baseConductanceTemperature` | 100 | W/°C |
| `baseConductanceHumidity` | 0.001 | %RH/s per %RH |
| `baseRateTemperature` | 1,000 | W |
| `baseRateHumidity` | 0.009 | %RH/s |

**Equilibrium reference** (standard room, single device, ambient 20°C / 50% RH):
- Temperature: `1000 / 100 = 10°C above ambient → 30°C`
- Humidity: `0.009 / 0.001 = 9% RH above ambient → 59% RH`

---

## RoomState

```go
type RoomState struct {
    Mu            sync.RWMutex
    Current       map[string]float64             // evolves per tick
    Ambient       map[string]float64             // never changes after init
    contributions map[string]map[string]float64  // hwID/type → measurementType → rate
    heatInput     map[string]float64             // derived sum of contributions
}
```

`contributions` keyed by `hwID/measurementType` — a device with both temperature
and humidity actuators has two independent entries.

`heatInput` always derived from `contributions` via `recomputeHeatInput` — never
mutated directly. Eliminates floating-point drift. Repeated identical commands are
a no-op by design.

`HeatInput()` returns a snapshot safe to read without holding `Mu`. The sensor
goroutine snapshots `HeatInput()` before acquiring `Mu.Lock` for `advanceRoom` —
avoids deadlock since `HeatInput()` acquires `Mu.RLock` internally.

---

## EnvironmentModel tick equation

```
effectiveAmbient = Ambient[type] + N(0, roomNoise[type])
energyInput      = heatInput[type] * simulatedTickSeconds
passiveLoss      = conductance[type] * (Current[type] - effectiveAmbient) * simulatedTickSeconds
delta[type]      = (energyInput - passiveLoss) / thermalMass[type]
Current[type]   += delta[type]   (clamped to physical bounds)
```

For static rooms: `heatInput` always zero.
For non-noisy rooms: `roomNoise` zeroed at config load — noise term evaluates to zero.
Sensor noise is separate — applied at publish time, independent of model noise.

`PhysicsModel` (Phase 9) implements the same `RoomModel` interface with its own
internal parameters.

---

## Actuator command subscriptions

**Topic:** `devices/{hw_id}/cmd` — QoS 2

**Payload:**
```json
{"actuator_type": "heater", "state": true}
```

`actuator_type` uses API-facing names. Translated to measurement types via
`config.ActuatorNameToMeasurement` at receipt. Each goroutine ignores commands
not matching its measurement type.

`state: true` → `SetContribution(hwID/type, {type: rate})`
`state: false` → `ClearContribution(hwID/type)`

### Watchdog

```
watchdogTimeout = baseTickSeconds * watchdogMultiplier
```

`watchdogMultiplier` = 3 (hardcoded). Real wall-clock time — independent of
`time_scale`. Fires if no matching command arrives within the timeout.
Clears contribution — room passively returns to ambient.

---

## Time scaling

```
naturalInterval          = baseTickSeconds / timeScale
effectivePublishInterval = max(naturalInterval, minPublishIntervalMS)
simulatedTickSeconds     = timeScale * effectivePublishInterval.Seconds()
```

`baseTickSeconds` = `CONTROL_TICK_INTERVAL_SECONDS` (shared with Control Service).
`minPublishIntervalMS` default 500ms, overridable via `min_publish_interval_ms`.
`maxTimeScale` = 400 — hard cap at load time.

Floor crossover: `timeScale = baseTickSeconds / minPublishInterval`.
With defaults (`baseTickSeconds=10`, `minPublishIntervalMS=500ms`): crossover at `timeScale=20`.

Below crossover: `simulatedTickSeconds = baseTickSeconds`.
Above crossover: publish rate capped, `simulatedTickSeconds` grows proportionally.

All timing computed once in `config.Load()`, stored on `Config`.

### Write volume

| Simulation | Time scale | Approx. write rate |
|---|---|---|
| `default.yaml` | 60× | ~50–100 MB/day |
| `demo.yaml` | 10× | ~1–1.5 GB/day |

---

## Goroutine detail

**Sensor goroutine** (one per device with sensors):
1. Wait stagger offset: `effectivePublishInterval * deviceIndex / totalDevices`
2. Tick loop at `effectivePublishInterval`:
   - Snapshot `HeatInput()` (acquires/releases `Mu.RLock` internally)
   - `advanceRoom`: acquire `Mu.Lock`, call `model.Advance(snapshot)`, apply
     deltas to `Current`, clamp to bounds, release lock
   - `publishTelemetry`: acquire `Mu.RLock`, read `Current[type]`, apply
     `N(0, sensor.Noise) + sensor.Offset`, publish, release lock

**Actuator goroutine** (one per actuator per device):
1. Subscribe to `devices/{hw_id}/cmd` at startup
2. On message: decode payload, check `actuator_type` matches — ignore if not
3. `state: true` → `SetContribution`, `state: false` → `ClearContribution`
4. Watchdog ticker running concurrently — `ClearContribution` on timeout

Devices with no sensors: actuator goroutines only.
Devices with no actuators: sensor goroutine only.

---

## Mosquitto credentials

| User | Purpose | Source |
|---|---|---|
| `device` | Simulator + ESP32 devices | `.env` |
| `device-service` | Control Service + Kafka Bridge | `.env` |
| `healthcheck` | Docker health check | Hardcoded `healthcheck`/`healthcheck` |

Password file generated via `make mosquitto-passwd`. ACL restricts topics per
username. Simulator connects as `device` — all simulated devices share this
credential.