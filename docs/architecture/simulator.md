# Simulator — Architecture

Standalone Go service that fully replaces physical ESP32 hardware for development
and demonstration. Connects to Mosquitto with the same credentials as a real
device, publishes to the same telemetry topics, and subscribes to the same command
topics. From the Control Service's perspective, simulated devices are
indistinguishable from physical ones.

For low-level detail on config YAML structure, data structures, timing derivation,
and provisioning sequence see [`simulator-reference.md`](simulator-reference.md).

---

## Responsibilities

**Provisioning** — on startup, the Simulator calls the API Server REST API to
create users, rooms, devices, desired states, and schedules. Provisioning is fully
idempotent — re-running against an already-provisioned simulation is safe.

**Telemetry publication** — each simulated device runs a sensor goroutine that
ticks at the configured rate and publishes readings to
`devices/{hw_id}/telemetry` QoS 1.

**Command subscription** — each simulated actuator runs a goroutine that
subscribes to `devices/{hw_id}/cmd` QoS 2. Received commands update the room's
physics state, so the next telemetry publish reflects what the actuator actually
did. The control loop closes in real time.

**Teardown** — on `--mode=teardown`, deletes all provisioned users via the API.
Foreign key cascades handle rooms, devices, and schedules automatically.

---

## Physics model

All rooms use a single thermal equation applied uniformly across measurement types.
Temperature and humidity are handled identically with different parameter values:

```
effectiveAmbient = Ambient[type] + N(0, roomNoise[type])
energyInput      = heatInput[type] * simulatedTickSeconds
passiveLoss      = conductance[type] * (Current[type] - effectiveAmbient) * simulatedTickSeconds
delta[type]      = (energyInput - passiveLoss) / thermalMass[type]
```

When an actuator is commanded ON, its contribution to `heatInput` is set. When
commanded OFF or when the watchdog fires, the contribution is cleared. The next
tick advance reflects the changed energy input immediately.

**Two axes of room behaviour:**

`type` controls the complexity axis:
- `static` — no actuator contributions. `Current` passively tracks ambient. Used
  for sensor-only rooms and controlled testing.
- `reactive` — actuator contributions drive `Current` via the thermal equation.
  Used for realistic simulation.
- `physics` (Phase 9) — full thermal model with external temperature profile.

`noisy` controls the stochastic axis:
- `false` — room model runs deterministically. Sensor noise is unaffected — it is
  applied at publish time and represents hardware measurement error, independent
  of model noise.
- `true` — Gaussian noise perturbs `effectiveAmbient` each tick.

Both axes are independent — a static noisy room wanders around ambient without
actuator input; a reactive non-noisy room responds deterministically to commands.

### Why this model

The model is intentionally simple. The goal is realistic enough behaviour to drive
convincing charts and exercise the control loop correctly — not physical accuracy.
The single equation with scale multipliers gives enough tunable variation between
room types (a well-insulated room vs a drafty one) without requiring per-type
physics implementations. Phase 9 adds a proper thermal model for scenarios where
realistic external temperature dynamics matter.

---

## Time scaling

Simulations can run faster than real time via `time_scale`. At `time_scale: 60`,
simulated time moves 60× faster — a 24-hour schedule cycle completes in 24
minutes. This allows the full system behaviour (schedule transitions, temperature
equilibration, grace periods) to be observed without waiting.

A floor on the publish interval (`min_publish_interval_ms`, default 500ms)
prevents overwhelming Mosquitto and TimescaleDB at high time scales. Above the
floor, physics remain correct because `simulatedTickSeconds` grows proportionally
— the model advances by a larger simulated step per publish.

The base tick interval (`CONTROL_TICK_INTERVAL_SECONDS`) is shared with the
Control Service as the system heartbeat. The Simulator derives all its timing from
this single shared constant.

---

## Config system

Two-level separation: **templates** define reusable room and device types;
**simulation files** compose templates into a concrete topology.

Templates carry only what makes a type distinct — room templates define ambient
base values and optional scale multipliers; device templates define sensor and
actuator types with rate scales. Physical base constants live in code, not in YAML.

Simulation files define the user/room topology — how many users, which room
templates each user gets, which device templates are assigned, whether the room
uses a desired state or a schedule for control, and behavioural flags (`type`,
`noisy`).

This separation means adding a new simulation scenario only requires a new
simulation file — no new templates unless a new room or device type is required.

Template references are dissolved into flat concrete structs at config load time.
Runtime code works entirely with resolved structs and has no knowledge of the
template system.

---

## Identity generation

Device and user identities are deterministic:

```
email:  sim-{simulation_name}-user-{000}@{domain}
hw_id:  sim-{simulation_name}-{user_idx}-{room_idx}-{device_idx}
```

The same simulation name always produces the same identities. This makes
provisioning idempotent across hard resets and teardown/re-run cycles — the
Simulator can always find its own previously provisioned resources by identity
without querying by name.

---

## Goroutine structure

Goroutines are split by responsibility — one goroutine per concern, not one per
device:

**Sensor goroutine** (one per device with sensors) — ticks at the effective
publish interval, advances room state via the physics model, publishes telemetry.

**Actuator goroutine** (one per actuator per device) — subscribes to the device
command topic, filters for its own measurement type, updates room contributions
when a command arrives, runs a watchdog that clears contributions if commands stop.

A device with both sensors and actuators (e.g. a `climate-controller`) has one
sensor goroutine and two actuator goroutines (temperature and humidity), each
independently managing its own contribution and watchdog.

### Why the watchdog matters

The Control Service sends commands every tick as a heartbeat. If the Control
Service stops, commands stop arriving. The watchdog fires after `3 ×
baseTickSeconds` of silence and clears all actuator contributions — the room
passively returns to ambient. This mirrors what a real device would do if it lost
its controller connection.

---

## Relationship to the Control Service

The Simulator makes the control loop observable. Without it, verifying that the
Control Service is making correct decisions requires physical hardware and patience.
With it, a full control cycle — sensor reading, bang-bang evaluation, command
dispatch, state update, next reading — completes in seconds at elevated time scale.

The Simulator is a development and demonstration tool, not a production component.
It is excluded from normal `docker compose up` via Docker Compose profiles and
started independently via `make simulator-start` or `make demo`.