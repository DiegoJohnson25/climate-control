package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/DiegoJohnson25/climate-control/device-service/internal/appdb"
	"github.com/DiegoJohnson25/climate-control/device-service/internal/cache"
	"github.com/DiegoJohnson25/climate-control/device-service/internal/config"
	"github.com/DiegoJohnson25/climate-control/device-service/internal/connect"
	"github.com/DiegoJohnson25/climate-control/device-service/internal/ingestion"
	"github.com/DiegoJohnson25/climate-control/device-service/internal/logging"
	"github.com/DiegoJohnson25/climate-control/device-service/internal/metricsdb"
	"github.com/DiegoJohnson25/climate-control/device-service/internal/mqtt"
)

func main() {
	// ---------------------------------------------------------------------------
	// Config
	// ---------------------------------------------------------------------------

	cfg := config.Load()

	// ---------------------------------------------------------------------------
	// Connections
	// ---------------------------------------------------------------------------

	db, err := connect.Postgres(cfg.PostgresUser, cfg.PostgresPassword, cfg.PostgresDB)
	if err != nil {
		log.Fatalf("appdb connect: %v", err)
	}
	log.Println("appdb: connected")

	metricsPool, err := connect.Timescale(cfg.TimescaleUser, cfg.TimescalePassword, cfg.TimescaleDB)
	if err != nil {
		log.Fatalf("metricsdb connect: %v", err)
	}
	defer metricsPool.Close()
	log.Println("metricsdb: connected")

	rdb := connect.Redis(cfg.RedisPassword)
	log.Println("redis: connected")

	mqttClient, err := mqtt.NewClient(&cfg)
	if err != nil {
		log.Fatalf("mqtt connect: %v", err)
	}
	log.Println("mqtt: connected")

	// ---------------------------------------------------------------------------
	// Cache warm
	// ---------------------------------------------------------------------------

	store := cache.NewStore()
	appRepo := appdb.NewRepository(db)
	if err := appRepo.WarmCache(store); err != nil {
		log.Fatalf("cache warm: %v", err)
	}
	logging.LogSummary(store)
	// logging.LogFullStore(store)
	// logging.LogDevices(store)

	// ---------------------------------------------------------------------------
	// Telemetry Ingestion
	// ---------------------------------------------------------------------------

	metricsRepo := metricsdb.NewRepository(metricsPool)
	source := mqtt.NewSource(mqttClient)
	ingestor := ingestion.NewIngestor(source, store, metricsRepo, cfg.StaleThreshold)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := ingestor.Run(ctx); err != nil {
		log.Fatalf("telemetry ingestion start: %v", err)
	}
	log.Println("telemetry ingestion: started")

	// TODO Phase 3d: start control loop / scheduler
	// TODO Phase 3e: start Redis stream consumer

	// suppress unused variable warning — rdb unused until Phase 3e
	_ = rdb

	// ---------------------------------------------------------------------------
	// Signal handling
	// ---------------------------------------------------------------------------

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit
	log.Printf("device-service: shutting down (signal: %s)", sig)

	cancel()
	ingestor.Stop()

	// TODO Phase 3d: wg.Wait() once scheduler goroutines are added

	log.Println("device-service: shutdown complete")
}
