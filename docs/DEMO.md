# Demo Guide

How to run the demo simulation and what to expect.

---

## Starting the demo

```bash
# One-time setup if not already done
cp .env.example .env
make mosquitto-passwd

# Start the full stack
make up

# Start the demo simulation
make demo
```

`make demo` provisions four simulated users with rooms and devices, activates schedules and desired state configurations, and begins publishing sensor telemetry. The Control Service starts evaluating and dispatching commands immediately.

The web client is available at `http://localhost`.

---

## Demo topology

The demo simulation (`demo.yaml`) provisions a 2×2 matrix designed to exercise all major system behaviours simultaneously.

| User | Device configuration | Control mechanism |
|---|---|---|
| user-000 | Single sensor per type | Desired state — indefinite hold |
| user-001 | Multiple sensors per type | Desired state — indefinite hold |
| user-002 | Single sensor per type | Schedules |
| user-003 | Multiple sensors per type | Schedules |

Each user has 3 rooms. This gives 12 rooms total, each with independent climate state evolving in real time.

The pairing is deliberate — user-000 and user-002 share the same hardware topology, letting you directly compare desired state control vs schedule control on identical hardware. user-001 and user-003 exercise sensor averaging — the Control Service averages multiple fresh readings before evaluating the bang-bang decision.

---

## Simulation credentials

After provisioning, the Simulator writes credentials for each user to the container at `/app/config/credentials/demo.txt`. Retrieve them with:

```bash
docker cp climate-control-simulator-service-1:/app/config/credentials/demo.txt ./demo-credentials.txt
cat ./demo-credentials.txt
```

Log in to the web client with any of these credentials to view that user's rooms and climate data.

---

## What to observe

**Control loop responding to sensor readings**
On the Overview tab for any room, the current state card updates every 30 seconds. With `time_scale: 10`, simulated time moves 10× faster — schedule transitions and temperature changes that would take hours in a real room happen in minutes.

**Bang-bang control in action**
Watch the heater or humidifier state toggle as temperature crosses the target ± deadband boundary. The history chart duty cycle fill shows what fraction of recent ticks the actuator was on.

**Schedule transitions**
Rooms controlled by schedules (user-002 and user-003) will transition between schedule periods, grace periods, and idle states visibly on the control source badge and in the history chart.

**Sensor averaging**
Rooms for user-001 and user-003 have multiple sensors per measurement type. The Control Service averages all fresh readings before evaluating — watch the `avg_temp` and `avg_hum` values on the climate endpoint or history chart.

---

## Stopping the demo

```bash
# Stop the simulator — data preserved in DB
make simulator-stop

# Stop the full stack — data preserved
make down

# Stop and destroy all data
make down-hard
```

To reset the simulation data and start fresh:
```bash
make simulator-teardown SIM=demo
make down-hard
make up
make demo
```

---

## Other simulations

| Simulation | Command | Description |
|---|---|---|
| Default | `make simulator-start SIM=default` | Single user, one room, `time_scale: 60`. Good for development. |
| Demo | `make demo` | 4 users, 12 rooms, `time_scale: 10`. |
| Cache test | `make simulator-start SIM=cache-test` | 5 rooms, all capability combinations, `time_scale: 1`. Telemetry only — no desired state or schedules. |

Switch between simulations without destroying data:
```bash
make simulator-switch SIM=default
```

Switch with a clean slate:
```bash
make simulator-switch-hard SIM=default
```