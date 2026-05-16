package transport

import (
	"context"
	"fmt"
	"sync"
	"time"

	"dolphin/internal/config"

	"github.com/open-dingtalk/dingtalk-stream-sdk-go/chatbot"
	"github.com/open-dingtalk/dingtalk-stream-sdk-go/client"
	"go.uber.org/zap"
)

// DingTalkTransport provides DingTalk bot I/O via Stream mode using the official SDK.
type DingTalkTransport struct {
	cfg       *config.DingTalkConfig
	msgCh     chan string
	closeCh   chan struct{}
	closeOnce sync.Once

	sdkCli    *client.StreamClient
	webhook   string
	webhookMu sync.RWMutex
}

func NewDingTalkTransport(cfg *config.DingTalkConfig) *DingTalkTransport {
	return &DingTalkTransport{
		cfg:     cfg,
		msgCh:   make(chan string, 1024),
		closeCh: make(chan struct{}),
	}
}

func (t *DingTalkTransport) Name() string { return "dingtalk" }

func (t *DingTalkTransport) Context() string {
	return "Connected via DingTalk bot (Stream mode). " +
		"The user is on a mobile device. Keep responses concise."
}

func (t *DingTalkTransport) Capabilities() Capabilities {
	return Capabilities{Streaming: false, Flushable: true}
}

// Start creates the SDK stream client and connects to DingTalk.
// The SDK handles auto-reconnect internally.
func (t *DingTalkTransport) Start(ctx context.Context) error {
	activeConnections.Add(1)
	defer activeConnections.Add(-1)

	cred := client.NewAppCredentialConfig(t.cfg.ClientID, t.cfg.ClientSecret)
	t.sdkCli = client.NewStreamClient(
		client.WithAppCredential(cred),
	)
	t.sdkCli.RegisterChatBotCallbackRouter(t.onMessage)

	if err := t.sdkCli.Start(ctx); err != nil {
		return fmt.Errorf("dingtalk stream start: %w", err)
	}
	zap.S().Infow("dingtalk stream connected")

	<-ctx.Done()
	return t.Close()
}

// onMessage is the chatbot callback registered with the SDK.
func (t *DingTalkTransport) onMessage(ctx context.Context, data *chatbot.BotCallbackDataModel) ([]byte, error) {
	if data.SessionWebhook != "" {
		t.webhookMu.Lock()
		t.webhook = data.SessionWebhook
		t.webhookMu.Unlock()
	}

	msgText := data.Text.Content
	if msgText == "" {
		return nil, nil
	}

	zap.S().Infow("dingtalk message received", "sender", data.SenderNick, "len", len(msgText))

	select {
	case t.msgCh <- msgText:
		msgsReceived.Inc()
	default:
		zap.S().Warnw("dingtalk message dropped, channel full")
	}

	return nil, nil
}

// ReadLine blocks until a message arrives or the transport is closed.
func (t *DingTalkTransport) ReadLine() (string, error) {
	select {
	case msg, ok := <-t.msgCh:
		if !ok {
			return "", fmt.Errorf("dingtalk transport closed")
		}
		return msg, nil
	case <-t.closeCh:
		return "", fmt.Errorf("dingtalk transport closed")
	case <-time.After(5 * time.Minute):
		return "", fmt.Errorf("dingtalk: read timeout (5m)")
	}
}

// WriteLine sends a message back via the session webhook.
func (t *DingTalkTransport) WriteLine(s string) error {
	return t.sendMessage(s)
}

// WriteString sends a message back via the session webhook.
func (t *DingTalkTransport) WriteString(s string) error {
	return t.sendMessage(s)
}

// sendMessage sends a reply using the SDK's chatbot replier via session webhook.
func (t *DingTalkTransport) sendMessage(body string) error {
	msgsSent.Inc()

	t.webhookMu.RLock()
	webhook := t.webhook
	t.webhookMu.RUnlock()

	if webhook == "" {
		return fmt.Errorf("dingtalk: no session webhook — wait for a user to @ the bot first")
	}

	replier := chatbot.NewChatbotReplier()
	if err := replier.SimpleReplyText(context.Background(), webhook, []byte(body)); err != nil {
		return fmt.Errorf("dingtalk reply: %w", err)
	}

	zap.S().Debugw("dingtalk message sent", "len", len(body))
	return nil
}

// Close shuts down the transport.
func (t *DingTalkTransport) Close() error {
	t.closeOnce.Do(func() {
		close(t.closeCh)
		if t.sdkCli != nil {
			t.sdkCli.Close()
		}
	})
	return nil
}
