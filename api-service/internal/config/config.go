package config

import (
	"os"
	"strconv"
)

type Config struct {
	// Appdb
	PostgresUser     string
	PostgresPassword string
	PostgresDB       string
	PostgresPort     int

	// Metricsdb
	TimescaleUser     string
	TimescalePassword string
	TimescaleDB       string
	TimescalePort     int

	// Redis
	RedisPassword string
	RedisPort     int

	// JWT
	JWTSecret           string
	JWTAccessTTLMinutes int
	JWTRefreshTTLDays   int

	// API
	APIPort int
}

func Load() Config {
	return Config{
		PostgresUser:        os.Getenv("POSTGRES_USER"),
		PostgresPassword:    os.Getenv("POSTGRES_PASSWORD"),
		PostgresDB:          os.Getenv("POSTGRES_DB"),
		TimescaleUser:       os.Getenv("TIMESCALE_USER"),
		TimescalePassword:   os.Getenv("TIMESCALE_PASSWORD"),
		TimescaleDB:         os.Getenv("TIMESCALE_DB"),
		RedisPassword:       os.Getenv("REDIS_PASSWORD"),
		JWTSecret:           os.Getenv("JWT_SECRET"),
		JWTAccessTTLMinutes: mustInt("JWT_ACCESS_TTL_MINUTES"),
		JWTRefreshTTLDays:   mustInt("JWT_REFRESH_TTL_DAYS"),
		APIPort:             mustInt("API_PORT"),
	}
}

func mustInt(envName string) int {
	v, err := strconv.Atoi(os.Getenv(envName))
	if err != nil {
		panic("invalid integer config value for: " + envName)
	}
	return v
}
