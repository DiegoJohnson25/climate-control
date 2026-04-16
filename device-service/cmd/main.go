package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/DiegoJohnson25/climate-control/device-service/internal/appdb"
	"github.com/DiegoJohnson25/climate-control/device-service/internal/cache"
	"github.com/DiegoJohnson25/climate-control/device-service/internal/config"
	"github.com/DiegoJohnson25/climate-control/device-service/internal/connect"
	"github.com/DiegoJohnson25/climate-control/device-service/internal/logging"
)

func main() {
	// ---- config ----
	cfg := config.Load()

	// ---- connections ----
	db, err := connect.Postgres(cfg.PostgresUser, cfg.PostgresPassword, cfg.PostgresDB)
	if err != nil {
		log.Fatalf("appdb connect: %v", err)
	}
	log.Println("appdb: connected")

	metricsDB, err := connect.Timescale(cfg.TimescaleUser, cfg.TimescalePassword, cfg.TimescaleDB)
	if err != nil {
		log.Fatalf("metricsdb connect: %v", err)
	}
	defer metricsDB.Close()
	log.Println("metricsdb: connected")

	rdb := connect.Redis(cfg.RedisPassword)
	log.Println("redis: connected")

	// ---- cache warm ----
	store := cache.NewStore()
	repo := appdb.NewRepository(db)

	if err := repo.WarmCache(store); err != nil {
		log.Fatalf("cache warm: %v", err)
	}

	logging.LogSummary(store)

	logging.LogFullStore(store)
	logging.LogDevices(store)

	// ---- placeholders for future phases ----
	// TODO Phase 3c: start MQTT subscriber (ingestion)
	// TODO Phase 3d: start control loop / scheduler
	// TODO Phase 3e: start Redis stream consumer

	// suppress unused variable warnings for connections not yet used
	_ = rdb

	// ---- signal handling ----
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit
	log.Printf("device-service: shutting down (signal: %s)", sig)

	// TODO: wg.Wait() once goroutines are added in Phase 3c/3d
	log.Println("device-service: shutdown complete")
}
