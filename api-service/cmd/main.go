package main

import (
	"context"
	"fmt"
	"log"

	"github.com/DiegoJohnson25/climate-control/api-service/internal/auth"
	"github.com/DiegoJohnson25/climate-control/api-service/internal/config"
	"github.com/DiegoJohnson25/climate-control/api-service/internal/initializers"
	"github.com/DiegoJohnson25/climate-control/api-service/internal/room"
	"github.com/DiegoJohnson25/climate-control/api-service/internal/router"
	"github.com/DiegoJohnson25/climate-control/api-service/internal/user"
	"github.com/DiegoJohnson25/climate-control/shared/database"
)

func main() {
	cfg := config.Load()

	db, err := database.ConnectPostgres(cfg.PostgresUser, cfg.PostgresPassword, cfg.PostgresDB)
	if err != nil {
		log.Fatalf("failed to connect to postgres: %v", err)
	}

	metricsDB, err := database.ConnectTimescale(cfg.TimescaleUser, cfg.TimescalePassword, cfg.TimescaleDB)
	_ = metricsDB // TODO: used when sensor history endpoints are implemented
	if err != nil {
		log.Fatalf("failed to connect to timescaledb: %v", err)
	}

	rdb := initializers.ConnectRedis(cfg.RedisPassword)
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		log.Fatalf("failed to connect to redis: %v", err)
	}

	userRepo := user.NewRepository(db)
	userSvc := user.NewService(userRepo)
	userHandler := user.NewHandler(userSvc)

	authRepo := auth.NewRepository(rdb, cfg.JWTRefreshTTLDays)
	authSvc := auth.NewService(userRepo, authRepo, cfg.JWTSecret, cfg.JWTAccessTTLMinutes, cfg.JWTRefreshTTLDays)
	authHandler := auth.NewHandler(authSvc)

	roomRepo := room.NewRepository(db)
	roomSvc := room.NewService(roomRepo)
	roomHandler := room.NewHandler(roomSvc)

	r := router.Setup(authHandler, authSvc, userHandler, roomHandler)
	r.Run(fmt.Sprintf(":%d", cfg.APIPort))
}
