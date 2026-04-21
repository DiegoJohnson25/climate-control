// Package config loads device-service configuration from environment variables.
// Internal Docker hostnames and ports are hardcoded in the connect package;
// only credentials and tunables live here.
package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	// Appdb
	PostgresUser     string
	PostgresPassword string
	PostgresDB       string

	// Metricsdb
	TimescaleUser     string
	TimescalePassword string
	TimescaleDB       string

	// Redis
	RedisPassword string

	// MQTT
	MQTTClientID              string
	MQTTDeviceServiceUsername string
	MQTTDeviceServicePassword string

	// Control
	StaleThreshold       time.Duration
	TickInterval         time.Duration
	CacheRefreshInterval time.Duration

	// Debug
	DebugLevel       string // "info", "verbose", or "" (off)
	TraceIngestion   bool   // log every processed telemetry message
	TraceTick        bool   // log every control tick evaluation
}

// Load reads environment variables into a Config. Panics if any variable
// marked with mustGetEnv is unset.
func Load() Config {
	staleSeconds := getEnvInt("CONTROL_STALE_THRESHOLD_SECONDS", 90)
	tickSeconds := getEnvInt("CONTROL_TICK_INTERVAL_SECONDS", 30)
	cacheRefreshMinutes := getEnvInt("CONTROL_CACHE_REFRESH_MINUTES", 5)

	return Config{
		PostgresUser:     os.Getenv("POSTGRES_USER"),
		PostgresPassword: os.Getenv("POSTGRES_PASSWORD"),
		PostgresDB:       os.Getenv("POSTGRES_DB"),

		TimescaleUser:     os.Getenv("TIMESCALE_USER"),
		TimescalePassword: os.Getenv("TIMESCALE_PASSWORD"),
		TimescaleDB:       os.Getenv("TIMESCALE_DB"),

		RedisPassword: os.Getenv("REDIS_PASSWORD"),

		MQTTClientID:              "device-service-" + mustGetEnv("HOSTNAME"),
		MQTTDeviceServiceUsername: os.Getenv("MQTT_DEVICE_SERVICE_USERNAME"),
		MQTTDeviceServicePassword: os.Getenv("MQTT_DEVICE_SERVICE_PASSWORD"),

		StaleThreshold:       time.Duration(staleSeconds) * time.Second,
		TickInterval:         time.Duration(tickSeconds) * time.Second,
		CacheRefreshInterval: time.Duration(cacheRefreshMinutes) * time.Minute,

		DebugLevel:     os.Getenv("DEVICE_DEBUG"),
		TraceIngestion: os.Getenv("DEVICE_TRACE_INGESTION") == "true",
		TraceTick:      os.Getenv("DEVICE_TRACE_TICK") == "true",
	}
}

func mustGetEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		panic("missing required env var: " + key)
	}
	return v
}

func getEnvInt(key string, defaultVal int) int {
	s := os.Getenv(key)
	if s == "" {
		return defaultVal
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return defaultVal
	}
	return v
}
