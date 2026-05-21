package mqtt

import (
	"testing"
	"time"

	"dolphin/internal/config"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

func testBrokerCfg() config.MQTTBrokerConfig {
	return config.MQTTBrokerConfig{
		Enabled: true,
		Addr:    ":19999",
		Accounts: []config.MQTTAccount{
			{Username: "test", Password: "secret"},
		},
	}
}

func TestNewBroker(t *testing.T) {
	b := New(testBrokerCfg())
	if b == nil {
		t.Fatal("New returned nil")
	}
	if b.server != nil {
		t.Error("server should be nil before Start")
	}
}

func TestBrokerStartClose(t *testing.T) {
	cfg := testBrokerCfg()
	cfg.Addr = ":19998"
	b := New(cfg)
	if err := b.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	if b.server == nil {
		t.Fatal("server should be non-nil after Start")
	}
	if err := b.Close(); err != nil {
		t.Errorf("Close: %v", err)
	}
}

func TestBrokerClientAddr(t *testing.T) {
	tests := []struct {
		addr string
		want string
	}{
		{":1883", "localhost:1883"},
		{"0.0.0.0:1883", "localhost:1883"},
		{"127.0.0.1:1883", "127.0.0.1:1883"},
		{"192.168.1.1:8888", "192.168.1.1:8888"},
		{"", "localhost:1883"},
		{"invalid", "localhost:1883"},
	}
	for _, tt := range tests {
		cfg := testBrokerCfg()
		cfg.Addr = tt.addr
		b := New(cfg)
		got := b.ClientAddr()
		if got != tt.want {
			t.Errorf("ClientAddr(%q) = %q, want %q", tt.addr, got, tt.want)
		}
	}
}

func TestBrokerClientConnect(t *testing.T) {
	cfg := testBrokerCfg()
	cfg.Addr = ":19997"
	b := New(cfg)
	if err := b.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer b.Close()

	opts := mqtt.NewClientOptions()
	opts.AddBroker("tcp://" + b.ClientAddr())
	opts.SetClientID("test-client")
	opts.SetUsername("test")
	opts.SetPassword("secret")
	opts.SetConnectTimeout(5 * time.Second)
	opts.SetAutoReconnect(false)

	client := mqtt.NewClient(opts)
	token := client.Connect()
	if token.WaitTimeout(10*time.Second) && token.Error() != nil {
		t.Fatalf("client connect: %v", token.Error())
	}
	defer client.Disconnect(250)

	if !client.IsConnected() {
		t.Fatal("client should be connected")
	}
}

func TestBrokerClientConnectBadAuth(t *testing.T) {
	cfg := testBrokerCfg()
	cfg.Addr = ":19996"
	b := New(cfg)
	if err := b.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer b.Close()

	opts := mqtt.NewClientOptions()
	opts.AddBroker("tcp://" + b.ClientAddr())
	opts.SetClientID("test-client-bad")
	opts.SetUsername("test")
	opts.SetPassword("wrong")
	opts.SetConnectTimeout(5 * time.Second)
	opts.SetAutoReconnect(false)

	client := mqtt.NewClient(opts)
	token := client.Connect()
	if token.WaitTimeout(10*time.Second) && token.Error() == nil {
		t.Fatal("client should fail to connect with wrong password")
	}
}

func TestBrokerPubSub(t *testing.T) {
	cfg := testBrokerCfg()
	cfg.Addr = ":19995"
	b := New(cfg)
	if err := b.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer b.Close()

	addr := "tcp://" + b.ClientAddr()

	clientOpts := func(id string) *mqtt.ClientOptions {
		o := mqtt.NewClientOptions()
		o.AddBroker(addr)
		o.SetClientID(id)
		o.SetUsername("test")
		o.SetPassword("secret")
		o.SetConnectTimeout(5 * time.Second)
		o.SetAutoReconnect(false)
		return o
	}

	time.Sleep(100 * time.Millisecond)

	sub := mqtt.NewClient(clientOpts("sub-client"))
	if token := sub.Connect(); token.WaitTimeout(10*time.Second) && token.Error() != nil {
		t.Fatalf("sub connect: %v", token.Error())
	}
	defer sub.Disconnect(250)

	received := make(chan string, 1)
	sub.Subscribe("test/topic", 0, func(_ mqtt.Client, msg mqtt.Message) {
		received <- string(msg.Payload())
	})

	pub := mqtt.NewClient(clientOpts("pub-client"))
	if token := pub.Connect(); token.WaitTimeout(10*time.Second) && token.Error() != nil {
		t.Fatalf("pub connect: %v", token.Error())
	}
	defer pub.Disconnect(250)

	pub.Publish("test/topic", 0, false, "hello from test")
	select {
	case msg := <-received:
		if msg != "hello from test" {
			t.Errorf("got %q, want %q", msg, "hello from test")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for message")
	}
}
