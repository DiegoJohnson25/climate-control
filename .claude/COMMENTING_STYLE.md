# Climate Control — Go Commenting Style Guide

This document defines the commenting conventions for the entire climate-control
codebase. Apply when writing new code and when refactoring existing comments.
Based on the official Go doc comment spec (go.dev/doc/comment) and Go Code
Review Comments (go.dev/wiki/CodeReviewComments).

---

## 1. Package comments

Every package gets a package comment. One sentence minimum, starting with
"Package <name>". No blank line between the comment and the package declaration.

For packages whose purpose is self-evident from their name, one line is enough.
For packages with non-obvious constraints or usage rules, add more.

For multi-file packages, the package comment lives in the file containing the
primary type or entry point for the package — the one a reader would open first
when navigating from godoc. Examples in this repo:

| Package | Primary file |
|---|---|
| `device-service/internal/cache` | `cache.go` (defines `Store`, `RoomCache`, `DeviceCache`) |
| `device-service/internal/ingestion` | `ingestion.go` (defines `Ingestor`) |
| `device-service/internal/appdb` | `repository.go` |
| `api-service/internal/room` | `service.go` |
| `simulator-service/internal/simulator` | `simulator.go` (defines `Run`) |

No `doc.go` files — those are for standard library scale packages with
substantial introductory documentation.

```go
// Package ctxkeys defines context key constants used across api-service
// to prevent circular imports between the auth and user packages.
package ctxkeys

// Package cache provides the in-memory store for device-service.
// All RoomCache fields are protected by Mu — callers must acquire the lock
// before reading or writing fields. DeviceCache.RoomID is protected by its
// own unexported mutex; use GetRoomID and SetRoomID.
package cache

// Package metricsdb provides write-only access to the TimescaleDB instance.
// Read access is handled exclusively by api-service.
package metricsdb

// Package room provides HTTP handlers, service logic, and repository access
// for the rooms domain.
package room
```

---

## 2. Exported type and function doc comments (godoc style)

Exported types, functions, and methods get a doc comment **unless they fall
under one of the carve-outs below**. Per the Go spec:

- Start with the name of the symbol being documented.
- Write in complete sentences.
- Capitalise the first word and end with a period.
- No blank line between the comment and the declaration.

```go
// RoomCache holds the complete runtime state for a single room.
// Mu protects all fields — the control loop holds Mu.RLock for the duration
// of its tick evaluation; ingestion and stream consumer hold Mu.Lock for updates.
type RoomCache struct {

// NewIngestor constructs an Ingestor with the given store, metrics repository,
// and stale reading threshold.
func NewIngestor(store *cache.Store, metrics *metricsdb.Repository, stale time.Duration) *Ingestor {

// WriteSensorReadings inserts all readings from a single telemetry message in
// one pgx batch — one round trip regardless of reading count.
func (r *Repository) WriteSensorReadings(ctx context.Context, readings []SensorReading) error {
```

For functions reporting a boolean result, use "reports whether":
```go
// OwnsRoom reports whether this instance is responsible for the given room.
func (s *Store) OwnsRoom(roomID uuid.UUID) bool {
```

### Carve-outs (no doc comment required)

These exist to keep godoc signal high and prevent comments that just restate
the symbol name.

**Trivial constructors.** A `New*` function that takes its dependencies and
returns `&T{...}` does not need a doc comment if the type itself has one.
Reaching for `// NewService returns a new Service.` is noise, not documentation.

```go
// No comment needed:
func NewRepository(db *gorm.DB) *Repository {
    return &Repository{db: db}
}

func NewHandler(svc *Service) *Handler {
    return &Handler{svc: svc}
}
```

If the constructor does meaningful work (validates inputs, opens connections,
spawns goroutines), document it — `mqtt.NewClient` and `cache.NewStore` qualify.

**Schema/DTO packages.** Packages whose only purpose is to declare data shapes
(`shared/models`) need a package comment, but individual structs whose field
names are self-documenting do not need per-type doc comments. The same applies
to handler-local request/response types.

```go
// No comment needed — shape is the documentation:
type User struct {
    ID           uuid.UUID
    Email        string
    PasswordHash string
    Timezone     string
    CreatedAt    time.Time
    UpdatedAt    time.Time
}
```

If a field has a non-obvious invariant (a sentinel value, a units convention,
a nullable-meaning), document the field, not the struct.

**Error sentinels.** Sentinel errors declared in a `var (...)` block whose
message is itself the documentation do not need individual doc comments. The
error message *is* the doc comment.

```go
// No per-sentinel comments needed:
var (
    ErrNotFound        = errors.New("room not found")
    ErrNameTaken       = errors.New("room name already taken")
    ErrNoCapability    = errors.New("room lacks required sensors or actuators for requested mode")
    ErrInvalidState    = errors.New("AUTO mode requires at least one target (temp or humidity)")
)
```

If a sentinel's name is opaque (`ErrConflict` rather than `ErrNameTaken`), or
its meaning depends on context (when it's returned and why), document it.

### Unexported symbols

Unexported types and functions only get comments when the logic or purpose is
non-obvious. Obvious helpers, simple constructors, and thin wrappers need no comment.

```go
// trimStale scans from the front — readings are always appended in
// chronological order so all stale entries are at the head of the slice.
func trimStale(readings []cache.TimestampedReading, now time.Time, threshold time.Duration) []cache.TimestampedReading {

// No comment needed — name is self-documenting:
func parseTimeToMinutes(s string) (int, error) {
```

---

## 3. File-level section breaks

Use a three-line long-bar divider to separate major groups of top-level
declarations within a file. Only use when a file has three or more distinct groups.

```go
// ---------------------------------------------------------------------------
// Scan types
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// Repository
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// Query helpers
// ---------------------------------------------------------------------------
```

Rules:
- One blank line before the opening bar, one blank line after the closing bar.
- Label is title case.
- Bar width is 75 dashes.
- Do not use in short files where the structure is obvious without them.

---

## 4. Within-function comments

No routine structural labels or dividers inside functions. Blank lines between
logical stages are sufficient for most functions.

Use a plain label comment only when a block of code is genuinely non-obvious
or benefits from being named in a long function:

```go
// resolve effective state from override and schedule
effectiveMode, targets := resolveEffectiveState(rc, now)

// preserve runtime-only fields from existing cache entry
existing := store.Room(roomID)
if existing != nil {
    existing.Mu.RLock()
    rc.ActuatorStates = existing.ActuatorStates
    rc.LatestReadings = existing.LatestReadings
    rc.LastActivePeriod = existing.LastActivePeriod
    existing.Mu.RUnlock()
}
```

Rules:
- Plain text, no dashes, no punctuation suffix.
- Lower case (these are headings, not sentences).
- Only when the block would be unclear without it.
- Never as a routine structural divider — if every stage gets a label,
  none of them are adding value.
- No numbered steps (`Step 1 —`, `// 1.`). If a function is long enough that
  numbering helps, the function probably wants splitting.

---

## 5. TODO markers

Phase-tagged for items tied to a specific phase. Plain for open-ended items.
Explanatory lines follow immediately in the same comment block.

```go
// TODO Phase 6: replace with Kafka partition ownership check.
// murmur2(room_id) % numPartitions must be in assignedPartitions.
// assignedPartitions is populated by OnPartitionsAssigned /
// OnPartitionsRevoked callbacks from the franz-go Kafka consumer.

// TODO Phase 8: add Prometheus instrumentation callsites.
// telemetry_messages_total{outcome}, sensor_readings_written_total{type}

// TODO: apply sensor offset when calibration is implemented.
```

Rules:
- `// TODO Phase N:` when a phase is known.
- `// TODO:` when genuinely open-ended with no phase context.
- No other formats — no FIXME, HACK, XXX.
- Never leave a TODO without at least a brief explanation of what needs doing.

---

## 6. Inline end-of-line comments

Permitted only for struct field annotations. Never for logic explanation —
that always goes on its own line above the statement.

```go
// Permitted — field annotation:
ActuatorHwIDs  map[string][]string             // actuator_type → []hw_id
LatestReadings map[string][]TimestampedReading // sensor_type   → readings
RoomID         *uuid.UUID                      // nil if device is unassigned

// Not permitted — logic explanation inline:
return readings[i:]  // skip stale entries  ← wrong, put this above the line
```

---

## 7. Multi-line explanatory blocks

Preferred for non-obvious behaviour, drop conditions, concurrency notes, and
design decisions embedded in code. Explain why, not what. Use Go list syntax
for enumerated conditions (three spaces, then a dash):

```go
// Process handles a single telemetry message. It updates LatestReadings in the
// store and writes sensor readings to TimescaleDB in one batch.
//
// Drop conditions (silent):
//   - hw_id not in device cache — unknown device
//   - device has no room assignment — unassigned devices have no instance owner
//
// Drop conditions (warning logged):
//   - room not owned by this instance — cache inconsistency, should not occur
//   - room not found in store — cache inconsistency, should not occur
```

### Concurrency annotations

Apply only to **self-locking** methods — methods whose body acquires a lock
internally and releases it before returning. The reader needs to know they can
call this method from any goroutine without external synchronisation.

```go
// SetRoomID updates the device's room assignment.
// This method is safe for concurrent use.
func (dc *DeviceCache) SetRoomID(roomID *uuid.UUID) {
    dc.mu.Lock()
    defer dc.mu.Unlock()
    dc.RoomID = roomID
}
```

Do **not** add this annotation to:
- Types that expose their lock for caller-managed locking (e.g. `RoomCache.Mu`)
  — those are documented at the type level, where the locking contract lives.
- Methods that read fields without locking because the caller is required to
  hold the lock.

---

## 8. What not to do

```go
// Bad — restates what the code obviously does:
i++ // increment counter

// Bad — three-line bar inside a function:
// ---------------------------------------------------------------------------
// Device lookup
// ---------------------------------------------------------------------------

// Bad — dash-style section labels inside a function:
// --- device lookup ---

// Bad — numbered steps:
// Step 1 — fetch room IDs

// Bad — phase-less TODO when phase is known:
// TODO: wire up Kafka consumer

// Bad — no explanation on a TODO:
// TODO Phase 6:

// Bad — commented-out code left in permanently:
// dc := n.store.Device(msg.HwID)

// Bad — doc comment that just restates the symbol name:
// NewService returns a new Service.
func NewService(repo *Repository) *Service { ... }

// Bad — doc comment on a struct whose fields are self-documenting:
// User represents a user.
type User struct {
    ID    uuid.UUID
    Email string
    ...
}
```

---

## Quick reference — when does a symbol need a doc comment?

| Symbol | Comment? |
|---|---|
| Exported type with non-obvious fields, invariants, or concurrency rules | **Yes** |
| Exported type whose fields are self-documenting (DTOs, models, request/response structs) | **No** |
| Exported function with logic, side effects, or non-obvious behaviour | **Yes** |
| Trivial `New*` constructor returning `&T{...}` | **No** (the type's comment covers it) |
| `New*` constructor that validates, connects, or spawns work | **Yes** |
| Exported error sentinel whose message is self-explanatory | **No** |
| Exported error sentinel whose meaning depends on context | **Yes** |
| Unexported symbol with non-obvious behaviour | **Yes** |
| Unexported symbol with self-documenting name | **No** |
| Method that is safe for concurrent use (self-locking) | **Yes** — annotate it |
| Method on a type with caller-managed locking | **No** — type-level doc covers it |

---

## Refactor checklist (apply at branch boundary via Claude Code)

When refactoring existing files to this style:

- [ ] Add missing package comments — one line minimum, in primary file only
- [ ] Add missing godoc comments to exported symbols, **respecting carve-outs in §2**
- [ ] Ensure all doc comments start with the symbol name and end with a period
- [ ] Replace single-line bar dividers with three-line long-bar style
- [ ] Remove `// --- label ---` dash-style section labels from function bodies
- [ ] Remove `// Step N —` numbered step markers
- [ ] Replace plain `// TODO:` with `// TODO Phase N:` where phase is known
- [ ] Add explanatory lines to any bare TODO markers
- [ ] Remove comments that restate what the code obviously does
- [ ] Move inline logic comments to their own line above the statement
- [ ] Add concurrency safety notes to self-locking methods only (see §7)
