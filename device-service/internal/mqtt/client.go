// Package mqtt provides an MQTT client for device-service, handling telemetry
// subscriptions and actuator command publishing.
package mqtt

import (
	"fmt"
	"time"

	pahomqtt "github.com/eclipse/paho.mqtt.golang"

	"github.com/DiegoJohnson25/climate-control/device-service/internal/config"
)

// Client wraps the Paho MQTT client with the subset of functionality needed
// by device-service: telemetry subscription and command publishing.
type Client struct {
	paho pahomqtt.Client
}

// NewClient constructs a Client and establishes a connection to the broker.
// Fails hard if the connection cannot be established within 10 seconds.
//
// The broker address is hardcoded to the internal Docker hostname. Credentials
// and client ID are loaded from config. Client ID is unique per instance via
// the HOSTNAME environment variable, allowing multiple device-service instances
// to connect with the same credentials but distinct identities.
func NewClient(cfg *config.Config) (*Client, error) {
	opts := pahomqtt.NewClientOptions()
	opts.AddBroker("tcp://mosquitto:1883")
	opts.SetClientID(cfg.MQTTClientID)
	opts.SetUsername(cfg.MQTTDeviceServiceUsername)
	opts.SetPassword(cfg.MQTTDeviceServicePassword)
	opts.SetAutoReconnect(true)
	opts.SetMaxReconnectInterval(30 * time.Second)
	opts.SetConnectionLostHandler(func(_ pahomqtt.Client, err error) {
		// connection lost is handled by auto-reconnect
		_ = err
	})

	c := pahomqtt.NewClient(opts)

	token := c.Connect()
	if !token.WaitTimeout(10 * time.Second) {
		return nil, fmt.Errorf("mqtt connect: timed out after 10s")
	}
	if err := token.Error(); err != nil {
		return nil, fmt.Errorf("mqtt connect: %w", err)
	}

	return &Client{paho: c}, nil
}

// Subscribe registers a handler for the given topic. The handler is called
// by Paho's internal goroutines — it must not block.
func (c *Client) Subscribe(topic string, qos byte, handler func(topic string, payload []byte)) error {
	token := c.paho.Subscribe(topic, qos, func(_ pahomqtt.Client, msg pahomqtt.Message) {
		handler(msg.Topic(), msg.Payload())
	})
	if !token.WaitTimeout(10 * time.Second) {
		return fmt.Errorf("mqtt subscribe timed out: topic %s", topic)
	}
	return token.Error()
}

// Publish sends a payload to the given topic.
func (c *Client) Publish(topic string, qos byte, payload []byte) error {
	token := c.paho.Publish(topic, qos, false, payload)
	if !token.WaitTimeout(10 * time.Second) {
		return fmt.Errorf("mqtt publish timed out: topic %s", topic)
	}
	return token.Error()
}

// Disconnect performs a clean shutdown, allowing in-flight messages up to
// 250ms to complete before closing the connection.
func (c *Client) Disconnect() {
	c.paho.Disconnect(250)
}
