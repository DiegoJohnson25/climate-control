// Package scheduler manages the per-room control loop and cache refresh
// goroutine lifecycles for device-service. It is the orchestration layer
// between the pure control evaluation in the control package and the I/O
// in the metricsdb and mqtt packages.
package scheduler

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/DiegoJohnson25/climate-control/device-service/internal/appdb"
	"github.com/DiegoJohnson25/climate-control/device-service/internal/cache"
	"github.com/DiegoJohnson25/climate-control/device-service/internal/config"
	"github.com/DiegoJohnson25/climate-control/device-service/internal/control"
	"github.com/DiegoJohnson25/climate-control/device-service/internal/metricsdb"
	"github.com/DiegoJohnson25/climate-control/device-service/internal/mqtt"
)

// ---------------------------------------------------------------------------
// Scheduler
// ---------------------------------------------------------------------------

// Scheduler manages the per-room goroutine lifecycle. Each room has two
// goroutines: a control loop ticker and a cache refresh ticker. Control loop
// goroutines are staggered across one tick interval at boot; cache refresh
// goroutines are staggered independently across one refresh interval.
type Scheduler struct {
	mu          sync.Mutex
	activeRooms map[uuid.UUID]context.CancelFunc
	wg          sync.WaitGroup

	store       *cache.Store
	appRepo     *appdb.Repository
	metricsRepo *metricsdb.Repository
	mqttClient  *mqtt.Client
	cfg         config.Config
}

func NewScheduler(
	store *cache.Store,
	appRepo *appdb.Repository,
	metricsRepo *metricsdb.Repository,
	mqttClient *mqtt.Client,
	cfg config.Config,
) *Scheduler {
	return &Scheduler{
		activeRooms: make(map[uuid.UUID]context.CancelFunc),
		store:       store,
		appRepo:     appRepo,
		metricsRepo: metricsRepo,
		mqttClient:  mqttClient,
		cfg:         cfg,
	}
}

// Start launches a control loop and cache refresh goroutine for each room
// currently in the store. Control loop goroutines are staggered evenly across
// one tick interval; cache refresh goroutines are staggered evenly across one
// refresh interval. The two staggers are independent so load is spread within
// each interval separately. Called once from main after cache warm.
func (s *Scheduler) Start(ctx context.Context) {
	roomIDs := s.store.RoomIDs()
	n := len(roomIDs)
	for i, roomID := range roomIDs {
		var controlStagger, refreshStagger time.Duration
		if n > 1 {
			controlStagger = time.Duration(i) * s.cfg.TickInterval / time.Duration(n)
			refreshStagger = time.Duration(i) * s.cfg.CacheRefreshInterval / time.Duration(n)
		}
		s.startRoom(ctx, roomID, controlStagger, refreshStagger)
	}
	log.Printf("scheduler: started goroutines for %d room(s)", n)
}

// AddRoom launches goroutines for a newly created room. Called by the stream
// consumer on room_created events. No stagger is applied.
func (s *Scheduler) AddRoom(ctx context.Context, roomID uuid.UUID) {
	s.startRoom(ctx, roomID, 0, 0)
	log.Printf("scheduler: added room %s", roomID)
}

// RemoveRoom cancels the goroutines for a deleted room. Called by the stream
// consumer on room_deleted events.
func (s *Scheduler) RemoveRoom(roomID uuid.UUID) {
	s.mu.Lock()
	cancel, ok := s.activeRooms[roomID]
	if ok {
		delete(s.activeRooms, roomID)
	}
	s.mu.Unlock()

	if ok {
		cancel()
		log.Printf("scheduler: removed room %s", roomID)
	}
}

// Wait blocks until all room goroutines have exited. Called from main after
// the root context is cancelled to ensure clean shutdown.
func (s *Scheduler) Wait() {
	s.wg.Wait()
}

// startRoom creates a child context for the room, registers its cancel func,
// and launches the control loop and cache refresh goroutines with independent
// stagger delays.
func (s *Scheduler) startRoom(ctx context.Context, roomID uuid.UUID, controlStagger, refreshStagger time.Duration) {
	roomCtx, cancel := context.WithCancel(ctx)

	s.mu.Lock()
	// cancel any existing goroutines for this room before replacing
	if existing, ok := s.activeRooms[roomID]; ok {
		existing()
	}
	s.activeRooms[roomID] = cancel
	s.mu.Unlock()

	s.wg.Add(2)
	go s.runControlLoop(roomCtx, roomID, controlStagger)
	go s.runCacheRefresh(roomCtx, roomID, refreshStagger)
}

// ---------------------------------------------------------------------------
// Control loop
// ---------------------------------------------------------------------------

// runControlLoop runs the bang-bang control evaluation for one room on each
// tick. After evaluation it publishes actuator commands, updates ActuatorStates
// and LastActivePeriod under a write lock, then writes the control log entry.
//
// The stagger delay is applied once before the first tick so rooms do not all
// fire simultaneously at startup.
func (s *Scheduler) runControlLoop(ctx context.Context, roomID uuid.UUID, stagger time.Duration) {
	defer s.wg.Done()

	if stagger > 0 {
		select {
		case <-time.After(stagger):
		case <-ctx.Done():
			return
		}
	}

	ticker := time.NewTicker(s.cfg.TickInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case now := <-ticker.C:
			s.tick(ctx, roomID, now)
		}
	}
}

// tick executes one control evaluation for a room. Separated from the ticker
// loop so the logic is readable without the select scaffolding.
func (s *Scheduler) tick(ctx context.Context, roomID uuid.UUID, now time.Time) {
	rc := s.store.Room(roomID)
	if rc == nil {
		log.Printf("scheduler: room %s not found in store during tick — skipping", roomID)
		return
	}

	result := control.Evaluate(rc, now, s.cfg.StaleThreshold)

	// publish commands — update ActuatorStates only for successful publishes
	for _, cmd := range result.Commands {
		payload, err := cmd.MarshalPayload()
		if err != nil {
			log.Printf("scheduler: marshal command for %s/%s: %v", roomID, cmd.HwID, err)
			continue
		}
		topic := fmt.Sprintf("devices/%s/cmd", cmd.HwID)
		if err := s.mqttClient.Publish(topic, 2, payload); err != nil {
			log.Printf("scheduler: publish command to %s: %v", topic, err)
			continue
		}
		rc.Mu.Lock()
		rc.ActuatorStates[cmd.ActuatorType] = cmd.State
		rc.Mu.Unlock()
	}

	// update LastActivePeriod when a schedule period or grace period was the control source
	if result.LastActivePeriod != nil {
		rc.Mu.Lock()
		rc.LastActivePeriod = result.LastActivePeriod
		rc.Mu.Unlock()
	}

	if err := s.metricsRepo.WriteControlLogEntry(ctx, result.LogEntry); err != nil {
		log.Printf("scheduler: write control log for room %s: %v", roomID, err)
	}
}

// ---------------------------------------------------------------------------
// Cache refresh
// ---------------------------------------------------------------------------

// runCacheRefresh reloads the room cache from appdb on each refresh interval.
// This is a safety net for missed stream events — stream events handle
// real-time invalidation; this catches anything that slips through.
//
// The stagger is computed independently from the control loop stagger, spread
// across one refresh interval so DB reload load is distributed at startup.
func (s *Scheduler) runCacheRefresh(ctx context.Context, roomID uuid.UUID, stagger time.Duration) {
	defer s.wg.Done()

	select {
	case <-time.After(s.cfg.CacheRefreshInterval + stagger):
	case <-ctx.Done():
		return
	}

	ticker := time.NewTicker(s.cfg.CacheRefreshInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := s.appRepo.ReloadRoom(ctx, s.store, roomID); err != nil {
				log.Printf("scheduler: cache refresh for room %s: %v", roomID, err)
			}
		}
	}
}
