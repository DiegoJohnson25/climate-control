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
	StaleThreshold time.Duration
}

func Load() Config {
	staleSeconds := getEnvInt("CONTROL_STALE_THRESHOLD_SECONDS", 90)

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

		StaleThreshold: time.Duration(staleSeconds) * time.Second,
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
