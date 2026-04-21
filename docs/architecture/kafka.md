# Kafka Architecture — Phase 7 Reference

Phase 7 takes device-service from single-instance to genuinely horizontally
scalable. The core problem: with multiple device-service instances all subscribing
to the same Mosquitto telemetry topics, two instances receive the same message
for the same room. They produce incoherent caches and incorrect, conflicting
control decisions. Kafka solves this via partition-based room ownership.

---

## Design

### Single topic, deterministic partitioning

- Topic: `telemetry`, 24 partitions, KRaft mode (no ZooKeeper)
- Partition key: `room_id` bytes via murmur2 hash
- 24 partitions chosen for headroom — supports up to 24 device-service instances
  without repartitioning
- Go client: franz-go (`github.com/twmb/franz-go`)

Each room hashes deterministically to one partition. One partition is owned by
exactly one device-service instance. Therefore each room's telemetry is processed
by exactly one instance — no duplicate control decisions, no cache incoherence.

---

## MQTT bridge (new service, Phase 7a)

The bridge is the only service that subscribes to Mosquitto for telemetry in
Phase 7. device-service no longer subscribes to Mosquitto for telemetry directly.

### Responsibilities

- Subscribe to `devices/+/telemetry` on Mosquitto
- Maintain local `map[string]{RoomID, DeviceID}` cache keyed on `hw_id`
- Warm cache from appdb on startup
- Consume `stream:cache_invalidation` as consumer group `bridge` — independent
  from the `device-service` consumer group, same stream, separate offset
- Resolve `hw_id → room_id` and stamp onto the Kafka message payload
- Produce to Kafka topic `telemetry` with key = `room_id` bytes

The bridge is pure stateless routing — no control loop, no DB writes, no business
logic. It translates MQTT messages into Kafka records with room context attached.

### Why a separate service

Separating the bridge from device-service maintains the clean `ingestion.Source`
interface in device-service. Swapping `mqtt.Source` for `kafka.Source` in
device-service is a one-line change in `main.go`. The bridge's `hw_id → room_id`
resolution logic doesn't belong in device-service either — it's a routing concern.

---

## device-service changes (Phase 7b)

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
`room_id` and `device_id` stamped by the bridge).

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

### OwnsRoom becomes real

```go
func (s *Store) OwnsRoom(roomID uuid.UUID) bool {
    s.mu.RLock()
    defer s.mu.RUnlock()
    p := murmur2([]byte(roomID.String())) % uint32(s.numPartitions)
    _, owned := s.assignedPartitions[int32(p)]
    return owned
}
```

Uses franz-go's exported murmur2 — identical function to what the bridge producer
uses. The coupling between bridge and device-service on hash function is real but
contained: both import the same franz-go package, same function, same result.

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

Actuator commands flow device-service → Mosquitto → ESP32 directly. Permanently.

The Kafka bridge handles telemetry ingestion only. Routing commands through Kafka
would add latency and complexity for no benefit — commands are point-to-point,
low-volume, and don't benefit from Kafka's fan-out or ordering guarantees.

Multiple device-service instances publish to Mosquitto for commands using the
same credentials but distinct client IDs (`device-service-{HOSTNAME}`). Mosquitto
allows this. The ACL grants publish rights by username, not client ID.

---

## Redis stream in Phase 7

The `stream:cache_invalidation` stream continues to operate in Phase 7. The
consumer group name changes to `device-service-{hostname}` (per-instance) —
this was already the design from Phase 3e. No changes to the stream consumer logic.

The bridge adds a second independent consumer group `bridge` on the same stream.
This is a native Redis Streams feature — multiple consumer groups on one stream,
each maintaining its own offset, each receiving all events independently.

---

## Docker Compose additions (Phase 7a)

- Kafka broker in KRaft mode
- MQTT bridge service
- Kafka topic creation as a one-shot init container

Scaling device-service in Phase 7:
```bash
docker compose up --scale device-service=3
```

Kafka's consumer group rebalance protocol assigns partitions across the 3 instances
automatically. No config changes to NGINX, api-service, or any other service.
