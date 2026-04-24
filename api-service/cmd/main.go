package main

import (
	"context"
	"fmt"
	"log"

	"github.com/DiegoJohnson25/climate-control/api-service/internal/auth"
	"github.com/DiegoJohnson25/climate-control/api-service/internal/config"
	"github.com/DiegoJohnson25/climate-control/api-service/internal/connect"
	"github.com/DiegoJohnson25/climate-control/api-service/internal/device"
	"github.com/DiegoJohnson25/climate-control/api-service/internal/metricsdb"
	"github.com/DiegoJohnson25/climate-control/api-service/internal/room"
	"github.com/DiegoJohnson25/climate-control/api-service/internal/router"
	"github.com/DiegoJohnson25/climate-control/api-service/internal/schedule"
	"github.com/DiegoJohnson25/climate-control/api-service/internal/user"
)

func main() {
	cfg := config.Load()

	db, err := connect.Postgres(cfg.PostgresUser, cfg.PostgresPassword, cfg.PostgresDB)
	if err != nil {
		log.Fatalf("failed to connect to postgres: %v", err)
	}

	metricsDB, err := connect.Timescale(cfg.TimescaleUser, cfg.TimescalePassword, cfg.TimescaleDB)
	if err != nil {
		log.Fatalf("failed to connect to timescaledb: %v", err)
	}

	rdb := connect.Redis(cfg.RedisPassword)
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		log.Fatalf("failed to connect to redis: %v", err)
	}

	userRepo := user.NewRepository(db)
	userSvc := user.NewService(userRepo)
	userHandler := user.NewHandler(userSvc)

	authRepo := auth.NewRepository(rdb, cfg.JWTRefreshTTLDays)
	authSvc := auth.NewService(userRepo, authRepo, cfg.JWTSecret, cfg.JWTAccessTTLMinutes, cfg.JWTRefreshTTLDays)
	authHandler := auth.NewHandler(authSvc)

	metricsRepo := metricsdb.NewRepository(metricsDB)

	roomRepo := room.NewRepository(db)
	roomSvc := room.NewService(roomRepo, metricsRepo, rdb)
	roomHandler := room.NewHandler(roomSvc)

	deviceRepo := device.NewRepository(db)
	deviceSvc := device.NewService(deviceRepo, roomRepo, rdb)
	deviceHandler := device.NewHandler(deviceSvc)

	scheduleRepo := schedule.NewRepository(db)
	scheduleSvc := schedule.NewService(scheduleRepo, roomRepo, rdb)
	scheduleHandler := schedule.NewHandler(scheduleSvc)

	r := router.Setup(authHandler, authSvc, userHandler, roomHandler, deviceHandler, scheduleHandler)
	r.Run(fmt.Sprintf(":%d", cfg.APIPort))
}
