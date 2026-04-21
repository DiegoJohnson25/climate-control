// Package events publishes cache-invalidation notifications to the Redis stream
// consumed by device-service. All functions are fire-and-forget — a failed
// XADD is logged but does not fail the calling operation. The periodic cache
// refresh in device-service is the safety net for missed events.
package events

import (
	"context"
	"log"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// Stream is the Redis stream key shared by all publishers and consumers.
const Stream = "stream:cache_invalidation"

// ---------------------------------------------------------------------------
// Room events
// ---------------------------------------------------------------------------

// NotifyRoomCreated publishes a room_created event so device-service warms the
// new room into its cache and starts its control loop goroutine.
func NotifyRoomCreated(ctx context.Context, rdb *redis.Client, roomID uuid.UUID) {
	xadd(ctx, rdb, map[string]any{
		"event":   "room_created",
		"room_id": roomID.String(),
	})
}

// NotifyRoomDeleted publishes a room_deleted event so device-service evicts the
// room from its cache and stops its control loop goroutine.
func NotifyRoomDeleted(ctx context.Context, rdb *redis.Client, roomID uuid.UUID) {
	xadd(ctx, rdb, map[string]any{
		"event":   "room_deleted",
		"room_id": roomID.String(),
	})
}

// NotifyRoomConfigChanged publishes a room_config_changed event so device-service
// reloads deadband values for the room.
func NotifyRoomConfigChanged(ctx context.Context, rdb *redis.Client, roomID uuid.UUID) {
	xadd(ctx, rdb, map[string]any{
		"event":   "room_config_changed",
		"room_id": roomID.String(),
	})
}

// NotifyDesiredStateChanged publishes a desired_state_changed event so
// device-service reloads the room's mode, targets, and override expiry.
func NotifyDesiredStateChanged(ctx context.Context, rdb *redis.Client, roomID uuid.UUID) {
	xadd(ctx, rdb, map[string]any{
		"event":   "desired_state_changed",
		"room_id": roomID.String(),
	})
}

// ---------------------------------------------------------------------------
// Device events
// ---------------------------------------------------------------------------

// NotifyDeviceAssigned publishes a device_assigned event so device-service
// reloads both the device cache entry and the room it joined.
func NotifyDeviceAssigned(ctx context.Context, rdb *redis.Client, roomID uuid.UUID, hwID string) {
	xadd(ctx, rdb, map[string]any{
		"event":   "device_assigned",
		"room_id": roomID.String(),
		"hw_id":   hwID,
	})
}

// NotifyDeviceUnassigned publishes a device_unassigned event so device-service
// reloads both the device cache entry and the room it left.
func NotifyDeviceUnassigned(ctx context.Context, rdb *redis.Client, prevRoomID uuid.UUID, hwID string) {
	xadd(ctx, rdb, map[string]any{
		"event":   "device_unassigned",
		"room_id": prevRoomID.String(),
		"hw_id":   hwID,
	})
}

// ---------------------------------------------------------------------------
// Schedule events
// ---------------------------------------------------------------------------

// NotifyScheduleChanged publishes a schedule_changed event so device-service
// reloads the active period set for the room.
func NotifyScheduleChanged(ctx context.Context, rdb *redis.Client, roomID uuid.UUID) {
	xadd(ctx, rdb, map[string]any{
		"event":   "schedule_changed",
		"room_id": roomID.String(),
	})
}

// ---------------------------------------------------------------------------
// Internal
// ---------------------------------------------------------------------------

func xadd(ctx context.Context, rdb *redis.Client, values map[string]any) {
	if err := rdb.XAdd(ctx, &redis.XAddArgs{
		Stream: Stream,
		MaxLen: 10000,
		Approx: true,
		Values: values,
	}).Err(); err != nil {
		log.Printf("warn: events.xadd stream=%s event=%v: %v", Stream, values["event"], err)
	}
}
