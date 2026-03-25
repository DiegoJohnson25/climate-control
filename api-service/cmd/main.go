package main

import (
	"context"
	"fmt"
	"log"

	"github.com/DiegoJohnson25/climate-control/api-service/internal/config"
	"github.com/DiegoJohnson25/climate-control/api-service/internal/health"
	"github.com/DiegoJohnson25/climate-control/api-service/internal/initializers"
	"github.com/DiegoJohnson25/climate-control/shared/database"
	"github.com/gin-gonic/gin"
)

func main() {
	cfg := config.Load()

	db, err := database.ConnectPostgres(cfg.PostgresUser, cfg.PostgresPassword, cfg.PostgresDB)
	_ = db // TODO: Remove this once db is used
	if err != nil {
		log.Fatalf("failed to connect to postgres: %v", err)
	}

	metricsDB, err := database.ConnectTimescale(cfg.TimescaleUser, cfg.TimescalePassword, cfg.TimescaleDB)
	_ = metricsDB // TODO: Remove this once metricsDB is used
	if err != nil {
		log.Fatalf("failed to connect to timescaledb: %v", err)
	}

	rdb := initializers.ConnectRedis(cfg.RedisPassword)
	ctx := context.Background()
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Fatalf("failed to connect to redis: %v", err)
	}

	r := gin.Default()
	r.GET("/health", health.Check)
	r.Run(fmt.Sprintf(":%d", cfg.APIPort))

}
