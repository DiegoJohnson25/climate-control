// Package stream implements a Redis stream consumer for cache invalidation
// events published by api-service. Each event triggers a targeted cache
// reload so the control loop always operates on fresh config without waiting
// for the periodic refresh ticker.
package stream

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"github.com/DiegoJohnson25/climate-control/device-service/internal/appdb"
	"github.com/DiegoJohnson25/climate-control/device-service/internal/cache"
	"github.com/DiegoJohnson25/climate-control/device-service/internal/scheduler"
)

// streamKey is the Redis stream key published to by api-service.
const streamKey = "stream:cache_invalidation"

// blockTimeout is the XREADGROUP block duration. The consumer checks for
// context cancellation between calls, so this bounds the shutdown response
// time when the stream is idle.
const blockTimeout = 5 * time.Second

// ---------------------------------------------------------------------------
// Consumer
// ---------------------------------------------------------------------------

// Consumer reads cache invalidation events from the Redis stream and dispatches
// them to the appropriate cache reload or scheduler lifecycle methods.
type Consumer struct {
	rdb          *redis.Client
	store        *cache.Store
	appRepo      *appdb.Repository
	sched        *scheduler.Scheduler
	groupName    string
	consumerName string
	wg           sync.WaitGroup
}

// NewConsumer creates a Consumer. groupName and consumerName are both derived
// from HOSTNAME so each instance maintains an independent position in the
// stream — every instance sees every message and decides locally whether the
// affected room is in its cache.
func NewConsumer(
	rdb *redis.Client,
	store *cache.Store,
	appRepo *appdb.Repository,
	sched *scheduler.Scheduler,
) *Consumer {
	hostname, _ := os.Hostname()
	if hostname == "" {
		hostname = "unknown"
	}
	return &Consumer{
		rdb:          rdb,
		store:        store,
		appRepo:      appRepo,
		sched:        sched,
		groupName:    "device-service-" + hostname,
		consumerName: hostname,
	}
}

// Run creates the consumer group if needed, drains any pending messages from a
// previous run, then enters the live read loop. Returns immediately — the
// consumer runs in a background goroutine. Call Wait for clean shutdown.
func (c *Consumer) Run(ctx context.Context) {
	c.wg.Add(1)
	go func() {
		defer c.wg.Done()

		if err := c.ensureGroup(ctx); err != nil {
			log.Fatalf("stream consumer: create group: %v", err)
		}

		c.drainPending(ctx)

		log.Printf("stream consumer: live — group=%s consumer=%s", c.groupName, c.consumerName)
		c.readLoop(ctx)
	}()
}

// Wait blocks until the consumer goroutine has exited. Called from main after
// the root context is cancelled, after sched.Wait().
func (c *Consumer) Wait() {
	c.wg.Wait()
}

// ---------------------------------------------------------------------------
// Startup helpers
// ---------------------------------------------------------------------------

// ensureGroup creates the consumer group at the stream tip with MKSTREAM.
// BUSYGROUP means the group already exists from a previous run — expected,
// not an error.
func (c *Consumer) ensureGroup(ctx context.Context) error {
	err := c.rdb.XGroupCreateMkStream(ctx, streamKey, c.groupName, "$").Err()
	if err != nil && !strings.Contains(err.Error(), "BUSYGROUP") {
		return err
	}
	return nil
}

// drainPending processes any messages that were delivered to this consumer
// during a previous run but never acknowledged. Uses ID "0-0" to read from
// the beginning of the pending-entries list.
func (c *Consumer) drainPending(ctx context.Context) {
	log.Printf("stream consumer: draining pending messages")
	for {
		msgs, err := c.readGroup(ctx, "0-0")
		if err != nil || len(msgs) == 0 {
			if err != nil {
				log.Printf("stream consumer: drain read: %v", err)
			}
			return
		}
		for _, msg := range msgs {
			c.dispatch(ctx, msg)
		}
	}
}

// ---------------------------------------------------------------------------
// Live read loop
// ---------------------------------------------------------------------------

// readLoop blocks on XREADGROUP ">" delivering new messages to this consumer.
// Exits when ctx is cancelled.
func (c *Consumer) readLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		msgs, err := c.readGroup(ctx, ">")
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			log.Printf("stream consumer: read: %v", err)
			continue
		}

		for _, msg := range msgs {
			c.dispatch(ctx, msg)
		}
	}
}

// readGroup issues a single XREADGROUP call for up to 10 messages.
func (c *Consumer) readGroup(ctx context.Context, id string) ([]redis.XMessage, error) {
	streams, err := c.rdb.XReadGroup(ctx, &redis.XReadGroupArgs{
		Group:    c.groupName,
		Consumer: c.consumerName,
		Streams:  []string{streamKey, id},
		Count:    10,
		Block:    blockTimeout,
	}).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, err
	}
	if len(streams) == 0 {
		return nil, nil
	}
	return streams[0].Messages, nil
}

// ---------------------------------------------------------------------------
// Dispatch
// ---------------------------------------------------------------------------

// dispatch routes a single stream message to the appropriate handler and
// ACKs on success. On dispatch error the message is not ACKed so it remains
// in the pending-entries list for retry on next restart. Unknown events and
// unparseable messages are ACKed and skipped to prevent indefinite accumulation.
func (c *Consumer) dispatch(ctx context.Context, msg redis.XMessage) {
	event, err := extractString(msg.Values, "event")
	if err != nil {
		log.Printf("stream consumer: message %s missing event field — skipping", msg.ID)
		c.ack(ctx, msg.ID)
		return
	}

	var dispatchErr error

	switch event {
	case "room_created":
		dispatchErr = c.onRoomCreated(ctx, msg.Values)
	case "room_deleted":
		dispatchErr = c.onRoomDeleted(ctx, msg.Values)
	case "room_config_changed", "desired_state_changed", "schedule_changed":
		dispatchErr = c.onRoomReload(ctx, msg.Values)
	case "device_assigned", "device_unassigned":
		dispatchErr = c.onDeviceChanged(ctx, msg.Values)
	default:
		log.Printf("stream consumer: unknown event %q in message %s — skipping", event, msg.ID)
		c.ack(ctx, msg.ID)
		return
	}

	if dispatchErr != nil {
		log.Printf("stream consumer: dispatch %s (msg %s): %v — will retry on restart", event, msg.ID, dispatchErr)
		return
	}

	c.ack(ctx, msg.ID)
	// logging.LogStreamEvent(event, msg.ID, msg.Values)
}

// ---------------------------------------------------------------------------
// Event handlers
// ---------------------------------------------------------------------------

func (c *Consumer) onRoomCreated(ctx context.Context, values map[string]interface{}) error {
	roomID, err := extractRoomID(values)
	if err != nil {
		return err
	}
	if err := c.appRepo.ReloadRoom(ctx, c.store, roomID); err != nil {
		return fmt.Errorf("reload room: %w", err)
	}
	c.sched.AddRoom(ctx, roomID)
	return nil
}

func (c *Consumer) onRoomDeleted(ctx context.Context, values map[string]interface{}) error {
	roomID, err := extractRoomID(values)
	if err != nil {
		return err
	}
	c.sched.RemoveRoom(roomID)
	c.store.DeleteRoom(roomID)
	return nil
}

func (c *Consumer) onRoomReload(ctx context.Context, values map[string]interface{}) error {
	roomID, err := extractRoomID(values)
	if err != nil {
		return err
	}
	if err := c.appRepo.ReloadRoom(ctx, c.store, roomID); err != nil {
		return fmt.Errorf("reload room: %w", err)
	}
	return nil
}

func (c *Consumer) onDeviceChanged(ctx context.Context, values map[string]interface{}) error {
	hwID, err := extractString(values, "hw_id")
	if err != nil {
		return err
	}
	roomID, err := extractRoomID(values)
	if err != nil {
		return err
	}
	if err := c.appRepo.ReloadDevice(ctx, c.store, hwID); err != nil {
		return fmt.Errorf("reload device: %w", err)
	}
	if err := c.appRepo.ReloadRoom(ctx, c.store, roomID); err != nil {
		return fmt.Errorf("reload room: %w", err)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func (c *Consumer) ack(ctx context.Context, msgID string) {
	if err := c.rdb.XAck(ctx, streamKey, c.groupName, msgID).Err(); err != nil {
		log.Printf("stream consumer: ack %s: %v", msgID, err)
	}
}

// extractString retrieves a string value from an XMessage Values map.
// go-redis returns field values as interface{} — they are always strings for
// entries written by XAdd with string/any values.
func extractString(values map[string]interface{}, key string) (string, error) {
	v, ok := values[key]
	if !ok {
		return "", fmt.Errorf("missing field %q", key)
	}
	s, ok := v.(string)
	if !ok {
		return "", fmt.Errorf("field %q is not a string (got %T)", key, v)
	}
	return s, nil
}

// extractRoomID extracts and parses the "room_id" field from an XMessage Values map.
func extractRoomID(values map[string]interface{}) (uuid.UUID, error) {
	s, err := extractString(values, "room_id")
	if err != nil {
		return uuid.Nil, err
	}
	roomID, err := uuid.Parse(s)
	if err != nil {
		return uuid.Nil, fmt.Errorf("parse room_id %q: %w", s, err)
	}
	return roomID, nil
}
