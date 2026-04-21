# Future Features — Design Notes

Tracked items that are not yet scheduled into a phase. Captured here so design
intent is not lost when they eventually get picked up.

---

## Device connection status

**What:** Real-time online/offline status for assigned devices.

**Design:** Redis hash keyed by `hw_id`. Fields: `status` (`online`/`offline`),
`last_seen` (timestamp). device-service writes on every telemetry message arrival
(sets `online`, updates `last_seen`) and on watchdog timeout (sets `offline`).

**API:** `GET /devices/:id` folds `status` and `last_seen` into the response.
Alternatively a dedicated `GET /devices/:id/status` endpoint.

**Scope restriction:** Only devices currently assigned to a room are tracked.
Unassigned devices have no device-service instance owner and produce no telemetry.

**Foundation:** Unconditional command heartbeat every tick means TTL-based
watchdog detection is straightforward — if device stops publishing telemetry,
the control loop stops sending commands, device goes silent.

---

## Sensor/actuator enabled toggle

**What:** Allow individual sensors and actuators to be disabled without deleting them.

**Schema change:**
```sql
ALTER TABLE sensors   ADD COLUMN enabled BOOLEAN NOT NULL DEFAULT true;
ALTER TABLE actuators ADD COLUMN enabled BOOLEAN NOT NULL DEFAULT true;
```

**API:** `PATCH /sensors/:id` and `PATCH /actuators/:id` to toggle `enabled`.

**Behaviour changes:**
- Ingestion skips disabled sensors — readings not written to TimescaleDB
- Control loop skips disabled sensors when averaging `LatestReadings`
- Capability checks filter to enabled sensors/actuators only
- Cache warm loads `enabled` field into `SensorEntry`/`ActuatorEntry`
- Toggle fires Redis stream event for cache invalidation

---

## Sensor calibration offset

**What:** Per-sensor offset applied at ingestion time to correct systematic
measurement error.

**Schema change:**
```sql
ALTER TABLE sensors ADD COLUMN offset NUMERIC(5,2) DEFAULT 0;
```

**Behaviour:** At ingestion, `value = raw_value + offset`. Both stored in
`sensor_readings`. Current implementation stores identical `value` and `raw_value`
as a placeholder (the `raw_value` column exists for this future use).

**Cache:** `SensorEntry` gains `Offset float64`. Applied in `ingestion.Process`
before writing to TimescaleDB.

**TODO markers:** `// TODO: apply sensor offset when calibration is implemented`
comments exist in ingestion code as callsite markers.

---

## Command acknowledgement

**What:** Device confirms it acted on a command before device-service updates
`ActuatorStates`.

**Current behaviour:** Fire-and-forget. device-service updates `ActuatorStates`
optimistically on command publish, not on device confirmation.

**Design:** Device publishes to `devices/{hw_id}/cmd/ack` after acting on command.
device-service subscribes, updates `ActuatorStates` only on confirmed ack.

**Tradeoff:** Adds complexity and a round-trip delay before state reflects reality.
Current optimistic approach is acceptable for the project scope. Document as a
production consideration in README rather than implementing.

---

## Schedule activatable field

**What:** `activatable bool` on schedule list responses — precomputed indicator
of whether a schedule can currently be activated given the room's device capabilities.

**Current behaviour:** Client discovers non-activatable schedules only on attempted
activation (422 response). Better UX would grey out the Activate button.

**Design:** Computed in `schedule.Service.List()` by running the same capability
checks used at activation time. Not stored — computed on read.

---

## Admin API

**What:** Administrative endpoints for device management.

**Planned endpoints:**
- `DELETE /admin/devices/:hw_id` — force-delete a device by hardware ID
- `POST /admin/devices/:hw_id/blacklist` — prevent a hw_id from registering

**Scope:** Out of scope for the current project phases. Noted for completeness.

---

## Connection pool size configuration

**What:** `DB_POOL_SIZE` env var per service to tune connection pool sizes without
code changes.

**Current:** Pool sizes are hardcoded in `connect/` functions.

---

## Freeform simulator client

**What:** A lightweight UI for runtime simulator management. Allows creating
simulated rooms and devices with specific model parameters at runtime, without
editing YAML files or restarting the simulator.

**Context:** The React app client handles API-level operations (room creation,
device registration, assignment, schedules). The simulator client handles the
simulator-side concern: which simulated room model should back a given device,
what parameters it should use.

**Sync mechanism:** Simulator polls `GET /devices` periodically to detect
assignment changes made via the React client. When a device is reassigned to a
different room, the simulator moves it to the corresponding `RoomState`.

**Key distinction:** React client sets API state. Simulator client sets simulator
physics state. They share only the `hw_id` as the linking key.

**Prerequisite:** Physics room model (Phase 9) — the main motivation for runtime
parameter selection is choosing between model types and tuning physics parameters
without a YAML edit + restart cycle.

---

## Physics room model (Phase 9)

**What:** Full thermal simulation replacing the reactive model's simple drift.

**Parameters (per room template):**
```yaml
model:
  type: physics
  base_temp: 20.0
  base_humidity: 50.0
  physics:
    thermal_mass: 50.0            # kJ/°C — resistance to temperature change
    thermal_conductance: 0.5      # W/°C — heat loss rate to external environment
    external_temp_profile:
      type: sinusoidal
      mean: 10.0                  # °C mean external temperature
      amplitude: 8.0              # °C variation amplitude
      period_hours: 24            # full cycle duration
```

**Tick calculation:**
```
externalTemp = mean + amplitude * sin(2π * elapsedHours / periodHours)
heatLoss = conductance * (Current["temperature"] - externalTemp) * tickSeconds
heatGain = HeatInput["temperature"] * tickSeconds
tempDelta = (heatGain - heatLoss) / thermalMass
Current["temperature"] += tempDelta
```

`RoomState` and the `RoomModelCalculator` interface are unchanged — `PhysicsCalculator`
implements `Tick()` using its own internal parameters.

---

## Android app

**What:** Native Android client (Kotlin + Jetpack Compose) as an alternative to
the React web client.

**Feature parity target:** Same feature set as the React client — dashboard,
room detail with control panel, history charts, schedule management, device management.

**Backend:** Consumes the same api-service REST API as the React client.
No backend changes required.
