// Package mqtt wraps the Paho MQTT client for the simulator-service, exposing
// connect, publish, subscribe, and disconnect with 10 s timeouts and
// auto-reconnect enabled.
package mqtt

import (
	"fmt"
	"time"

	pahomqtt "github.com/eclipse/paho.mqtt.golang"

	"github.com/DiegoJohnson25/climate-control/simulator-service/internal/config"
)

type Client struct {
	paho pahomqtt.Client
}

func NewClient(cfg *config.Config) (*Client, error) {
	opts := pahomqtt.NewClientOptions()
	opts.AddBroker(fmt.Sprintf("tcp://%s:%d", cfg.MQTTHost, cfg.MQTTPort))
	opts.SetClientID(cfg.MQTTClientID)
	opts.SetUsername(cfg.MQTTUsername)
	opts.SetPassword(cfg.MQTTPassword)
	opts.SetAutoReconnect(true)
	opts.SetMaxReconnectInterval(30 * time.Second)
	opts.SetConnectionLostHandler(func(_ pahomqtt.Client, err error) {
		_ = err
	})

	c := pahomqtt.NewClient(opts)

	token := c.Connect()
	if !token.WaitTimeout(10 * time.Second) {
		return nil, fmt.Errorf("mqtt connection timed out after 10s")
	}
	if err := token.Error(); err != nil {
		return nil, fmt.Errorf("mqtt connect: %w", err)
	}

	return &Client{paho: c}, nil
}

func (c *Client) Publish(topic string, qos byte, payload []byte) error {
	token := c.paho.Publish(topic, qos, false, payload)
	if !token.WaitTimeout(10 * time.Second) {
		return fmt.Errorf("mqtt publish timed out: topic %s", topic)
	}
	return token.Error()
}

func (c *Client) Subscribe(topic string, qos byte, handler func(topic string, payload []byte)) error {
	token := c.paho.Subscribe(topic, qos, func(_ pahomqtt.Client, msg pahomqtt.Message) {
		handler(msg.Topic(), msg.Payload())
	})
	if !token.WaitTimeout(10 * time.Second) {
		return fmt.Errorf("mqtt subscribe timed out: topic %s", topic)
	}
	return token.Error()
}

func (c *Client) Disconnect() {
	c.paho.Disconnect(250)
}
