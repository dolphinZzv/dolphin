// Package dingtalk provides DingTalk bot stream transport.
package dingtalk

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"dolphin/internal/config"
	transport "dolphin/internal/transport"

	"github.com/open-dingtalk/dingtalk-stream-sdk-go/chatbot"
	"github.com/open-dingtalk/dingtalk-stream-sdk-go/client"
	"go.uber.org/zap"
)

func init() { transport.Register("dingtalk", New) }

// DingTalkTransport provides DingTalk bot I/O via Stream mode using the official SDK.
type DingTalkTransport struct {
	cfg         *config.DingTalkConfig
	msgCh       chan string
	closeCh     chan struct{}
	closeOnce   sync.Once
	reconnectCh chan struct{}

	sdkCli         *client.StreamClient
	webhook        string
	webhookMu      sync.RWMutex
	startupWebhook string
}

func New(cfg *config.Config) (transport.Transport, error) {
	return &DingTalkTransport{
		cfg:            &cfg.Transport.DingTalk,
		msgCh:          make(chan string, 1024),
		closeCh:        make(chan struct{}),
		reconnectCh:    make(chan struct{}, 1),
		startupWebhook: cfg.Transport.DingTalk.StartupWebhook,
	}, nil
}

func (t *DingTalkTransport) OnConfigChange(oldCfg, newCfg *config.Config) {
	oldID, oldSecret := t.cfg.ClientID, t.cfg.ClientSecret
	t.cfg = &newCfg.Transport.DingTalk
	t.startupWebhook = newCfg.Transport.DingTalk.StartupWebhook
	if t.cfg.ClientID != oldID || t.cfg.ClientSecret != oldSecret {
		select {
		case t.reconnectCh <- struct{}{}:
		default:
		}
		zap.S().Infow("dingtalk credentials changed, reconnecting")
	}
}

func (t *DingTalkTransport) Name() string { return "dingtalk" }

func (t *DingTalkTransport) Banner() string {
	return "  DingTalk bot active (Stream mode)\n"
}

func (t *DingTalkTransport) Context() string {
	return "Connected via DingTalk bot (Stream mode). " +
		"The user is on a mobile device. Keep responses concise."
}

func (t *DingTalkTransport) Capabilities() transport.Capabilities {
	return transport.Capabilities{Streaming: false}
}

func (t *DingTalkTransport) Start(ctx context.Context) error {
	transport.ActiveConnections.Add(1)
	defer transport.ActiveConnections.Add(-1)

	for {
		cred := client.NewAppCredentialConfig(t.cfg.ClientID, t.cfg.ClientSecret)
		cli := client.NewStreamClient(
			client.WithAppCredential(cred),
		)
		cli.RegisterChatBotCallbackRouter(t.onMessage)
		t.sdkCli = cli

		zap.S().Infow("dingtalk stream starting", "client_id", t.cfg.ClientID)
		if err := cli.Start(ctx); err != nil {
			if ctx.Err() != nil {
				return t.Close()
			}
			zap.S().Errorw("dingtalk stream start failed, retrying", "error", err)
			time.Sleep(time.Second)
			continue
		}
		zap.S().Infow("dingtalk stream connected")

		// Send startup notification via robot webhook if configured
		t.sendStartupNotification()

		// cli.Start is non-blocking — it spawns an internal processLoop and returns.
		// The SDK handles reconnection automatically (AutoReconnect: true).
		// Wait here for either program shutdown or a credential-change signal.
		select {
		case <-ctx.Done():
			return t.Close()
		case <-t.reconnectCh:
			zap.S().Infow("dingtalk credentials changed, reconnecting")
			cli.Close()
		}
	}
}

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
		transport.MsgsReceived.Inc()
	default:
		zap.S().Warnw("dingtalk message dropped, channel full")
	}

	return nil, nil
}

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

func (t *DingTalkTransport) WriteLine(s string) error {
	return t.sendMessage(s)
}

func (t *DingTalkTransport) WriteString(s string) error {
	return t.sendMessage(s)
}

func (t *DingTalkTransport) Flush() error { return nil }

func (t *DingTalkTransport) sendMessage(body string) error {
	transport.MsgsSent.Inc()

	t.webhookMu.RLock()
	webhook := t.webhook
	t.webhookMu.RUnlock()

	if webhook == "" {
		return fmt.Errorf("dingtalk: no session webhook — wait for a user to @ the bot first")
	}

	replier := chatbot.NewChatbotReplier()

	if isMarkdownContent(body) {
		if err := replier.SimpleReplyMarkdown(context.Background(), webhook, []byte("Dolphin"), []byte(body)); err != nil {
			return fmt.Errorf("dingtalk markdown reply: %w", err)
		}
	} else {
		if err := replier.SimpleReplyText(context.Background(), webhook, []byte(body)); err != nil {
			return fmt.Errorf("dingtalk reply: %w", err)
		}
	}

	zap.S().Debugw("dingtalk message sent", "len", len(body))
	return nil
}

// sendStartupNotification sends a startup message via the DingTalk robot webhook.
func (t *DingTalkTransport) sendStartupNotification() {
	webhook := t.startupWebhook
	if webhook == "" {
		return
	}

	payload := map[string]any{
		"msgtype": "text",
		"text": map[string]string{
			"content": "Dolphin bot started — stream connection established",
		},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		zap.S().Errorw("dingtalk startup webhook marshal failed", "error", err)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, webhook, bytes.NewReader(body))
	if err != nil {
		zap.S().Errorw("dingtalk startup webhook request failed", "error", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		zap.S().Errorw("dingtalk startup webhook call failed", "error", err)
		return
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		zap.S().Errorw("dingtalk startup webhook non-200",
			"status", resp.StatusCode, "body", string(respBody))
		return
	}

	zap.S().Infow("dingtalk startup notification sent", "webhook", truncateToken(webhook))
}

// truncateToken masks the access_token portion of a webhook URL for logging.
func truncateToken(url string) string {
	if idx := strings.Index(url, "access_token="); idx >= 0 {
		token := url[idx+13:]
		if len(token) > 8 {
			return url[:idx+13] + token[:4] + "****"
		}
	}
	return url
}

func isMarkdownContent(s string) bool {
	markdownIndicators := []string{
		"# ", "**", "```", "`", "- ", "* ", "1. ", "> ", "---", "[](",
	}
	for _, indicator := range markdownIndicators {
		if strings.Contains(s, indicator) {
			return true
		}
	}
	return false
}

func (t *DingTalkTransport) Close() error {
	t.closeOnce.Do(func() {
		close(t.closeCh)
		if t.sdkCli != nil {
			t.sdkCli.Close()
		}
	})
	return nil
}
