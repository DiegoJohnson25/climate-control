# Kafka Architecture — Phase 7 Reference

Phase 7 takes the Control Service from single-instance to genuinely horizontally
scalable. The core problem: with multiple Control Service instances all subscribing
to the same Mosquitto telemetry topics, two instances receive the same message
for the same room. They produce incoherent caches and incorrect, conflicting
control decisions. Kafka solves this via partition-based room ownership.

---

## Design

### Single topic, deterministic partitioning

- Topic: `telemetry`, 24 partitions, KRaft mode (no ZooKeeper)
- Partition key: `room_id` bytes via murmur2 hash
- 24 partitions chosen for headroom — supports up to 24 Control Service instances
  without repartitioning
- Go client: franz-go (`github.com/twmb/franz-go`)

Each room hashes deterministically to one partition. One partition is owned by
exactly one Control Service instance. Therefore each room's telemetry is processed
by exactly one instance — no duplicate control decisions, no cache incoherence.

---

## Kafka Bridge (new service, Phase 7a)

The Kafka Bridge is the only service that subscribes to Mosquitto for telemetry in
Phase 7. The Control Service no longer subscribes to Mosquitto for telemetry directly.

### Responsibilities

- Subscribe to `devices/+/telemetry` on Mosquitto
- Maintain local `map[string]{RoomID, DeviceID}` cache keyed on `hw_id`
- Warm cache from appdb on startup
- Consume `stream:cache_invalidation` as consumer group `kafka-bridge` — independent
  from all Control Service consumers, same stream, separate offset
- Resolve `hw_id → room_id` and stamp onto the Kafka message payload
- Produce to Kafka topic `telemetry` with key = `room_id` bytes
- Forward cache invalidation events from Redis stream to Kafka topic
  `cache-invalidation` with key = `room_id` bytes

The Kafka Bridge handles all ingestion into Kafka — both telemetry from MQTT and
cache invalidation events from Redis. It is the single entry point for all data
entering the Kafka pipeline. It performs no business logic — only protocol
translation, device-to-room resolution, and Kafka production.

### Why a separate service

Separating the Kafka Bridge from the Control Service maintains the clean
`ingestion.Source` interface in the Control Service. Swapping `mqtt.Source` for
`kafka.Source` is a one-line change in `main.go`. The `hw_id → room_id` resolution
logic doesn't belong in the Control Service either — it's a routing concern.

Centralising both telemetry and invalidation event ingestion in the Kafka Bridge
means the Control Service has no Redis dependency in Phase 7. All data arrives
via Kafka, partitioned by room, with ownership guaranteed by the consumer group
rebalance protocol.

### Cache invalidation forwarding

The Kafka Bridge consumes the Redis stream as consumer group `kafka-bridge` and
forwards each invalidation event to the Kafka topic `cache-invalidation` with
key = `room_id` bytes — the same murmur2 hash used for telemetry. This means
invalidation events are routed to the same Control Service instance that owns
the affected room's telemetry partition. No `OwnsRoom()` self-filtering is needed
in the Control Service — Kafka's partition assignment guarantees the right instance
receives the right room's events.

The `cache-invalidation` topic is separate from `telemetry` — the Control Service
handles them through separate consumption paths. Invalidation events are low volume
compared to telemetry; a separate topic keeps the consumption logic clean and
independently scalable.

---

## Control Service changes (Phase 7b)

### Source swap

`mqtt.Source` in `main.go` replaced by `kafka.Source`. `ingestion.Process` is
unchanged — `TelemetryMessage` struct is constructed identically by both adapters.
The `ingestion.Source` interface absorbs the transport difference entirely.

```go
// main.go change:
// src := mqtt.NewSource(mqttClient, cfg)
src := kafka.NewSource(kafkaClient, cfg)
ingestor := ingestion.NewIngestor(store, metricsRepo, cfg.StaleThreshold)
ingestor.Run(ctx, src)
```

### kafka.Source

Implements `ingestion.Source`. Runs a franz-go `PollFetches` loop internally.
Constructs `TelemetryMessage` from Kafka record payload (which already has
`room_id` and `device_id` stamped by the Kafka Bridge).

### Cache invalidation via Kafka

The Control Service no longer connects to Redis in Phase 7. Cache invalidation
events arrive via the `cache-invalidation` Kafka topic, consumed alongside
telemetry. Because invalidation events are keyed by `room_id` with the same
murmur2 hash, they are delivered to the same instance that owns the affected
room's partition. The per-instance Redis consumer group design from Phases 3–6
is replaced entirely by Kafka partition ownership.

### Partition ownership callbacks

`OnPartitionsAssigned(partitions map[string][]int32)`:
1. Update `Store.assignedPartitions` under write lock
2. Query appdb for all rooms
3. Filter to rooms where `murmur2(room_id) % numPartitions` is in assigned set
4. Warm only those rooms — existing entries for retained partitions are untouched
5. Start control loop goroutines for newly assigned rooms

`OnPartitionsRevoked(partitions map[string][]int32)`:
1. Stop control loop goroutines for rooms in revoked partitions
2. Evict those rooms from `Store`
3. Update `Store.assignedPartitions` under write lock

### OwnsRoom in Phase 7

```go
func (s *Store) OwnsRoom(roomID uuid.UUID) bool {
    s.mu.RLock()
    defer s.mu.RUnlock()
    p := murmur2([]byte(roomID.String())) % uint32(s.numPartitions)
    _, owned := s.assignedPartitions[int32(p)]
    return owned
}
```

Uses franz-go's exported murmur2 — identical function to what the Kafka Bridge
producer uses. The coupling between the Kafka Bridge and the Control Service on
hash function is real but contained: both import the same franz-go package, same
function, same result.

`OwnsRoom()` is no longer used for self-filtering invalidation events — Kafka
partition ownership guarantees delivery to the correct instance. It is retained
for cache warm filtering during `OnPartitionsAssigned`.

### Startup sequence change

Cache warm cannot happen before joining the consumer group — partition assignment
is asynchronous. The new startup sequence:

1. Load config, connect to all infrastructure
2. Register `OnPartitionsAssigned` and `OnPartitionsRevoked` callbacks with franz-go
3. Join consumer group → callbacks fire asynchronously
4. `OnPartitionsAssigned` warms cache for assigned rooms
5. franz-go begins delivering records for assigned partitions
6. Begin processing (ingestion and control loops are started inside the callback)

The old `appdb.WarmCache(store)` call in `main.go` is removed. Warm is now
entirely owned by `OnPartitionsAssigned`.

---

## Commands still bypass Kafka

Actuator commands flow Control Service → Mosquitto → ESP32 directly. Permanently.

The Kafka Bridge handles ingestion only. Routing commands through Kafka would add
latency and complexity for no benefit — commands are point-to-point, low-volume,
and don't benefit from Kafka's fan-out or ordering guarantees.

Multiple Control Service instances publish to Mosquitto for commands using the
same credentials but distinct client IDs (`control-service-{HOSTNAME}`). Mosquitto
allows this. The ACL grants publish rights by username, not client ID.

---

## Redis stream in Phase 7

The `stream:cache_invalidation` stream continues to operate in Phase 7. The
Kafka Bridge consumes it as consumer group `kafka-bridge` and forwards all events
into Kafka topic `cache-invalidation`.

The Control Service no longer consumes the Redis stream directly in Phase 7. The
per-instance consumer group design from Phases 3–6 is superseded by Kafka partition
ownership. Invalidation events are delivered to the correct Control Service instance
by Kafka, not by self-filtering in each instance.

---

## Kafka topics

| Topic | Producer | Consumer | Key | Purpose |
|---|---|---|---|---|
| `telemetry` | Kafka Bridge | Control Service | `room_id` | Sensor readings from devices |
| `cache-invalidation` | Kafka Bridge | Control Service | `room_id` | State change events from API Server |

Both topics use the same `room_id` murmur2 partition key. A given room's telemetry
and invalidation events always route to the same Control Service instance.

---

## Docker Compose additions (Phase 7a)

- Kafka broker in KRaft mode
- Kafka Bridge service (`kafka-bridge`)
- Kafka topic creation as a one-shot init container

Scaling the Control Service in Phase 7:
```bash
docker compose up --scale control-service=3
```

Kafka's consumer group rebalance protocol assigns partitions across the 3 instances
automatically. No config changes to NGINX, API Server, or any other service.

---

## Open design questions (resolve before Phase 7a)

**Device reassignment invalidation in the Kafka Bridge**
The Bridge maintains a `hw_id → room_id` cache warmed from appdb and
updated via `stream:cache_invalidation`. When a device is reassigned,
the Bridge must handle `device_assigned` and `device_unassigned` events
to keep its routing map current. This parallels what the Control Service
does via `ReloadDevice` — the Bridge needs the same event handling or
it will silently misroute telemetry after a reassignment.

**murmur2 hash input must match between Bridge and Control Service**
`OwnsRoom()` in the Control Service hashes `roomID.String()` (the
36-character UUID string form). The Kafka Bridge producer must use the
identical representation as the partition key — not raw UUID bytes
(`room_id[:]`, 16 bytes). The hash value differs between string and
byte representations. Both services must import and call the same
franz-go murmur2 function with the same input format. Add an explicit
comment at both callsites referencing this constraint.

**`ingestion.Source` handles one message type — Phase 7 needs two**
The current `ingestion.Source` interface yields `TelemetryMessage`
only. Phase 7b requires the Control Service to consume both `telemetry`
and `cache-invalidation` Kafka topics. Options:
- Second `Source` interface for invalidation events, consumed by the
  stream consumer goroutine (cleanest separation)
- Union message type with a Kind discriminator field
- Separate Kafka consumer outside the `ingestion` package entirely
Decide before starting Phase 7b — the choice shapes the Phase 7b
component structure.

**Health server ready timing in Phase 7**
In Phase 7, `OnPartitionsAssigned` fires asynchronously inside a
franz-go callback goroutine. `SetReady()` must not be called in
`main.go` after `Join()` returns — it must be called inside
`OnPartitionsAssigned` after cache warm and control loop startup
complete for the assigned partition set. The current Phase 3–6 startup
sequence (linear, synchronous) does not apply in Phase 7.