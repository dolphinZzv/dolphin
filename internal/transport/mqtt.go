package transport

import (
	"context"
	"fmt"
	"log/slog"
	"sync/atomic"
	"time"

	"dolphinzZ/internal/config"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// MQTTTransport provides MQTT pub/sub transport implementing UserIO.
// Messages received on the command topic are delivered via ReadLine.
// Responses are published to the response topic.
type MQTTTransport struct {
	cfg       *config.MQTTConfig
	client    mqtt.Client
	msgCh     chan string
	connected atomic.Bool
}

func NewMQTTTransport(cfg *config.Config) *MQTTTransport {
	return &MQTTTransport{
		cfg:   &cfg.Transport.MQTT,
		msgCh: make(chan string, 64),
	}
}

func (t *MQTTTransport) Name() string { return "mqtt" }

func (t *MQTTTransport) Start(ctx context.Context) error {
	opts := mqtt.NewClientOptions()
	opts.AddBroker(t.cfg.Broker)
	opts.SetClientID(t.cfg.ClientID)
	opts.SetKeepAlive(60 * time.Second)
	opts.SetPingTimeout(10 * time.Second)
	opts.SetCleanSession(true)
	opts.SetConnectionLostHandler(func(c mqtt.Client, err error) {
		slog.Error("mqtt connection lost", "error", err)
	})

	t.client = mqtt.NewClient(opts)
	token := t.client.Connect()
	if token.Wait() && token.Error() != nil {
		return fmt.Errorf("mqtt connect: %w", token.Error())
	}
	t.connected.Store(true)
	slog.Info("mqtt connected", "broker", t.cfg.Broker, "command_topic", t.cfg.Topic, "response_topic", t.cfg.ResponseTopic)

	// Subscribe to command topic — push payloads to msgCh
	token = t.client.Subscribe(t.cfg.Topic, 0, func(c mqtt.Client, msg mqtt.Message) {
		payload := string(msg.Payload())
		slog.Debug("mqtt command received", "payload", truncate(payload, 200))
		select {
		case t.msgCh <- payload:
		default:
			slog.Warn("mqtt message dropped, channel full")
		}
	})
	if token.Wait() && token.Error() != nil {
		return fmt.Errorf("mqtt subscribe: %w", token.Error())
	}

	// Block until context is cancelled
	<-ctx.Done()
	return t.Close()
}

// ReadLine blocks until an MQTT command message arrives.
func (t *MQTTTransport) ReadLine() (string, error) {
	msg, ok := <-t.msgCh
	if !ok {
		return "", fmt.Errorf("mqtt transport closed")
	}
	return msg, nil
}

// WriteLine publishes a line to the response topic with trailing newline.
func (t *MQTTTransport) WriteLine(s string) error {
	return t.publish(s + "\n")
}

// WriteString publishes text to the response topic.
func (t *MQTTTransport) WriteString(s string) error {
	return t.publish(s)
}

func (t *MQTTTransport) publish(payload string) error {
	if !t.connected.Load() {
		return fmt.Errorf("mqtt not connected")
	}
	token := t.client.Publish(t.cfg.ResponseTopic, 0, false, payload)
	token.Wait()
	return token.Error()
}

func (t *MQTTTransport) Close() error {
	if t.client != nil && t.connected.Load() {
		t.connected.Store(false)
		t.client.Unsubscribe(t.cfg.Topic)
		t.client.Disconnect(250)
	}
	return nil
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
