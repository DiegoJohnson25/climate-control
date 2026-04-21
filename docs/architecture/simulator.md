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

`config/templates/rooms.yaml` — room model templates. Each template defines a
model type and its parameters.

`config/templates/devices.yaml` — device templates. Each defines sensors,
actuators, noise characteristics, and actuator rates.

Template references are dissolved into flat concrete structs at load time. The
runtime simulator never references the template system — it works entirely with
resolved structs.

### Simulation files

Located in `config/simulations/`. Selected via `--simulation=name` CLI flag.
No default — must be explicit on every invocation.

Each simulation file references room and device templates by ID and defines the
topology: how many users, rooms per user, devices per room.

### Available simulations

| File | Description |
|---|---|
| `default.yaml` | Single user, one room with full capability |
| `multi-room.yaml` | Single user, multiple rooms |
| `multi-user.yaml` | Multiple independent users |
| `sensor-only.yaml` | Rooms with sensors but no actuators |
| `multi-sensor.yaml` | Multiple sensor devices per room |
| `cache-test.yaml` | 5 rooms covering all capability combinations — used for cache warm verification |

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

---

## Room model architecture

### RoomState

Shared runtime state for one room. Protected by `sync.Mutex`. Updated by
actuator command goroutines, read by the publish loop tick.

```go
type RoomState struct {
    mu        sync.Mutex
    Current   map[string]float64  // current environmental values, evolves per tick
    Ambient   map[string]float64  // equilibrium — set from base_temp/base_humidity, never changes
    HeatInput map[string]float64  // sum of active actuator rate contributions by measurement type
}
```

`Current` starts equal to `Ambient` at startup for all model types.

### RoomModelCalculator interface

Assigned once at startup based on room template `model.type`. Called every tick
by the publish loop to compute deltas.

```go
type RoomModelCalculator interface {
    Tick(state *RoomState) map[string]float64  // returns deltas to apply to Current
}
```

The interface isolates model-specific parameters from shared runtime state.
`ReactiveCalculator` holds `passiveRate` as its own field — not on `RoomState`.
`PhysicsCalculator` (Phase 9) holds `thermalMass`, `conductance` etc. as its own
fields. `RoomState` has no knowledge of model type or parameters.

### Model types

**`noise`** — `Current` never moves. `Tick` returns zero deltas. Gaussian noise
applied per device at publish time, independently of room state. Used when
actuator feedback is not needed.

**`reactive`** (Phase 4b) — `Current` drifts based on active actuator contributions
minus a passive return-to-ambient rate. Physically: heater on → temperature rises,
heater off → temperature falls back toward ambient. Symmetric for humidity.

Tick calculation:
```
for each measurement type:
    netDelta = (HeatInput[type] * tickSeconds) - (passiveRate[type] * tickSeconds)
    Current[type] += netDelta
    // clamp to reasonable bounds (5–40°C for temperature, 0–100% for humidity)
```

`passiveRate` is a field on `ReactiveCalculator`, not on `RoomState`. It
represents degrees/percent per second return toward ambient when there is no
heat input.

**`physics`** (Phase 9) — thermal equation consuming `HeatInput`, external
temperature profile (sinusoidal), thermal mass and conductance parameters.
`RoomState.HeatInput` is the same interface — `PhysicsCalculator` consumes it
differently. No changes to `RoomState` or the actuator command goroutines.

### Room template config

```yaml
room_templates:
  - id: reactive-room
    model:
      type: reactive
      base_temp: 20.0
      base_humidity: 50.0
      passive_rate:
        temperature: 0.1    # degrees per second back toward ambient
        humidity: 0.3       # percent per second back toward ambient
```

---

## Actuator command subscriptions

Each actuator device runs a goroutine that subscribes to its command topic
on startup. The goroutine updates `RoomState.HeatInput` when commands arrive.

Command topic: `devices/{hw_id}/cmd` — QoS 2.

On `on` command (and previously `off`): add device's `rates` to `HeatInput`.
On `off` command (and previously `on`): subtract device's `rates` from `HeatInput`.
Same command repeated: no-op — idempotent. This matters because device-service
re-sends commands every tick as a heartbeat.

Rate accumulation is additive across multiple devices of the same actuator type
in the same room. Two heaters each with `temperature rate: 0.5` produce
`HeatInput["temperature"] = 1.0` when both are on. Physically correct —
two heaters heat a room faster than one.

### Device template config

```yaml
device_templates:
  - id: climate-controller
    sensors: [temperature, humidity]
    actuators:
      - type: heater
        rates:
          temperature: 0.5    # degrees per second when commanded on
      - type: humidifier
        rates:
          humidity: 1.0       # percent per second when commanded on
    noise:
      temperature: 0.1        # Gaussian noise std dev applied at publish time
      humidity: 0.5
    offset:
      temperature: 0.0        # sensor measurement offset
      humidity: 0.0
```

Rates are per-second. The tick calculation multiplies by `tick_interval_seconds`
so rate values are independent of tick frequency.

---

## Publish loop

One goroutine per device. Staggered start offset: `tickInterval * deviceIndex / totalDevices`.

On each tick:
1. Acquire `RoomState.mu` read lock
2. Read `Current` values for each sensor type the device reports
3. Release lock
4. Apply per-device Gaussian noise and offset to each value
5. Publish `TelemetryMessage` to `devices/{hw_id}/telemetry` — QoS 1

Published value = `RoomState.Current[type] + noise + offset`.

---

## Schedule provisioning (Phase 4c)

Schedule definitions live in the simulation YAML alongside room definitions.
At bootstrap, after rooms and devices are provisioned:

1. For each room, create a schedule with the defined periods.
2. Activate the schedule via `PATCH /schedules/:id/activate`.

Example simulation YAML extension:
```yaml
rooms:
  - template: reactive-room
    name_prefix: living-room
    schedule:
      name: "Weekday Comfort"
      periods:
        - days: [1,2,3,4,5]
          start_time: "07:00"
          end_time: "22:00"
          mode: AUTO
          target_temp: 21.0
          target_hum: 50.0
```

---

## Teardown (Phase 4c)

`--mode=teardown` deletes all resources provisioned for a simulation in strict
dependency order:

1. Deactivate active schedules (PATCH deactivate)
2. Delete schedule periods
3. Delete schedules
4. Unassign devices from rooms (PUT with null room_id)
5. Delete devices
6. Delete rooms
7. Delete user (`DELETE /users/me`)

Teardown is idempotent — 404 responses are treated as success (already deleted).
Uses the same deterministic identity generation as provisioning to know which
resources to target.

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
