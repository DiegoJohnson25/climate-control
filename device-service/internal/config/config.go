package config

import (
	"os"
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
}

func Load() Config {
	return Config{
		PostgresUser:     os.Getenv("POSTGRES_USER"),
		PostgresPassword: os.Getenv("POSTGRES_PASSWORD"),
		PostgresDB:       os.Getenv("POSTGRES_DB"),

		TimescaleUser:     os.Getenv("TIMESCALE_USER"),
		TimescalePassword: os.Getenv("TIMESCALE_PASSWORD"),
		TimescaleDB:       os.Getenv("TIMESCALE_DB"),

		RedisPassword: os.Getenv("REDIS_PASSWORD"),

		MQTTClientID:              "device-service-" + os.Getenv("HOSTNAME"),
		MQTTDeviceServiceUsername: os.Getenv("MQTT_DEVICE_SERVICE_USERNAME"),
		MQTTDeviceServicePassword: os.Getenv("MQTT_DEVICE_SERVICE_PASSWORD"),
	}
}
