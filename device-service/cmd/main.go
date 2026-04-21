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
	"github.com/DiegoJohnson25/climate-control/device-service/internal/scheduler"
	"github.com/DiegoJohnson25/climate-control/device-service/internal/stream"
)

func main() {
	cfg := config.Load()

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

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	store := cache.NewStore()
	appRepo := appdb.NewRepository(db)
	if err := appRepo.WarmCache(ctx, store); err != nil {
		log.Fatalf("cache warm: %v", err)
	}
	logging.LogSummary(store)

	metricsRepo := metricsdb.NewRepository(metricsPool)
	source := mqtt.NewSource(mqttClient)
	ingestor := ingestion.NewIngestor(source, store, metricsRepo, cfg.StaleThreshold)

	if err := ingestor.Run(ctx); err != nil {
		log.Fatalf("telemetry ingestion start: %v", err)
	}
	log.Println("telemetry ingestion: started")

	sched := scheduler.NewScheduler(store, appRepo, metricsRepo, mqttClient, cfg)
	sched.Start(ctx)

	consumer := stream.NewConsumer(rdb, store, appRepo, sched)
	consumer.Run(ctx)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit
	log.Printf("device-service: shutting down (signal: %s)", sig)

	cancel()
	ingestor.Stop()
	sched.Wait()
	consumer.Wait()
	log.Println("device-service: shutdown complete")
}
