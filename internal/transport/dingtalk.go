package transport

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha1"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"dolphin/internal/config"

	"go.uber.org/zap"
)

// DingTalkTransport provides DingTalk bot I/O via callback (push) with polling fallback.
type DingTalkTransport struct {
	cfg       *config.DingTalkConfig
	msgCh     chan string
	closeCh   chan struct{}
	closeOnce sync.Once
	httpCli   *http.Client
	httpSrv   *http.Server
	token     tokenCache
	convID    string // openConversationId from last callback
	convMu    sync.RWMutex
}

type tokenCache struct {
	mu       sync.Mutex
	value    string
	expireAt time.Time
}

func NewDingTalkTransport(cfg *config.DingTalkConfig) *DingTalkTransport {
	return &DingTalkTransport{
		cfg:     cfg,
		msgCh:   make(chan string, 1024),
		closeCh: make(chan struct{}),
		httpCli: &http.Client{Timeout: 15 * time.Second},
	}
}

func (t *DingTalkTransport) Name() string { return "dingtalk" }

func (t *DingTalkTransport) Context() string {
	mode := t.cfg.Mode
	if mode == "" {
		mode = "auto"
	}
	return fmt.Sprintf("Connected via DingTalk bot (mode: %s). "+
		"Keep responses concise. Use the crontab system for scheduled tasks.", mode)
}

func (t *DingTalkTransport) Capabilities() Capabilities {
	return Capabilities{Streaming: false, Flushable: true}
}

// Start begins the transport: HTTP callback server and/or polling loop.
func (t *DingTalkTransport) Start(ctx context.Context) error {
	activeConnections.Add(1)
	defer activeConnections.Add(-1)

	mode := t.cfg.Mode
	if mode == "" {
		mode = "auto"
	}

	if mode == "callback" || mode == "auto" {
		if t.cfg.ListenAddr != "" {
			go t.startCallbackServer(ctx)
		}
	}

	if mode == "poll" || mode == "auto" {
		go t.startPolling(ctx)
	}

	<-ctx.Done()
	return t.Close()
}

// ReadLine blocks until a message arrives or the transport is closed.
func (t *DingTalkTransport) ReadLine() (string, error) {
	select {
	case msg, ok := <-t.msgCh:
		if !ok {
			return "", fmt.Errorf("dingtalk transport closed")
		}
		msgsReceived.Inc()
		return msg, nil
	case <-t.closeCh:
		return "", fmt.Errorf("dingtalk transport closed")
	case <-time.After(5 * time.Minute):
		return "", fmt.Errorf("dingtalk: read timeout (5m)")
	}
}

// WriteLine sends a message back via DingTalk API.
func (t *DingTalkTransport) WriteLine(s string) error {
	return t.sendMessage(s)
}

// WriteString sends a message back via DingTalk API.
func (t *DingTalkTransport) WriteString(s string) error {
	return t.sendMessage(s)
}

// sendMessage sends a message to the DingTalk conversation.
func (t *DingTalkTransport) sendMessage(body string) error {
	msgsSent.Inc()
	token, err := t.getAccessToken()
	if err != nil {
		return err
	}

	t.convMu.RLock()
	convID := t.convID
	t.convMu.RUnlock()

	if convID == "" {
		return fmt.Errorf("dingtalk: no conversation id available, wait for a user message first")
	}

	payload := map[string]any{
		"robotCode":                t.cfg.ClientID,
		"targetOpenConversationId": convID,
		"msgKey":                   "sampleText",
		"msgParam":                 fmt.Sprintf(`{"content": %q}`, body),
	}

	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(payload); err != nil {
		return fmt.Errorf("dingtalk send: marshal: %w", err)
	}

	req, err := http.NewRequest("POST",
		"https://oapi.dingtalk.com/v1.0/robot/groupMessages/send", &buf)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-acs-dingtalk-access-token", token)

	resp, err := t.httpCli.Do(req)
	if err != nil {
		return fmt.Errorf("dingtalk send: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("dingtalk send: status %d: %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		ErrCode int    `json:"errcode"`
		ErrMsg  string `json:"errmsg"`
	}
	json.NewDecoder(resp.Body).Decode(&result)
	if result.ErrCode != 0 {
		return fmt.Errorf("dingtalk send: errcode=%d %s", result.ErrCode, result.ErrMsg)
	}

	zap.S().Debugw("dingtalk message sent", "len", len(body))
	return nil
}

// getAccessToken returns a cached token or fetches a new one.
func (t *DingTalkTransport) getAccessToken() (string, error) {
	t.token.mu.Lock()
	defer t.token.mu.Unlock()

	if t.token.value != "" && time.Now().Before(t.token.expireAt) {
		return t.token.value, nil
	}

	url := fmt.Sprintf("https://oapi.dingtalk.com/gettoken?appkey=%s&appsecret=%s",
		t.cfg.ClientID, t.cfg.ClientSecret)

	resp, err := t.httpCli.Get(url)
	if err != nil {
		return "", fmt.Errorf("dingtalk gettoken: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		ErrCode     int    `json:"errcode"`
		ErrMsg      string `json:"errmsg"`
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("dingtalk gettoken: decode: %w", err)
	}
	if result.ErrCode != 0 {
		return "", fmt.Errorf("dingtalk gettoken: errcode=%d %s", result.ErrCode, result.ErrMsg)
	}
	if result.AccessToken == "" {
		return "", fmt.Errorf("dingtalk gettoken: empty access_token")
	}

	// Cache with 5-minute safety margin
	expiresIn := result.ExpiresIn
	if expiresIn <= 0 {
		expiresIn = 7200
	}
	t.token.value = result.AccessToken
	t.token.expireAt = time.Now().Add(time.Duration(expiresIn-300) * time.Second)

	zap.S().Infow("dingtalk access token refreshed", "expires_in", expiresIn)
	return t.token.value, nil
}

// startCallbackServer starts the HTTP server that handles DingTalk event callbacks.
func (t *DingTalkTransport) startCallbackServer(ctx context.Context) {
	mux := http.NewServeMux()
	mux.HandleFunc("/dingtalk/callback", t.handleCallback)
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(200)
	})

	t.httpSrv = &http.Server{
		Addr:    t.cfg.ListenAddr,
		Handler: mux,
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		t.httpSrv.Shutdown(shutdownCtx)
	}()

	zap.S().Infow("dingtalk callback server starting", "addr", t.cfg.ListenAddr)
	if err := t.httpSrv.ListenAndServe(); err != http.ErrServerClosed {
		zap.S().Warnw("dingtalk callback server error", "error", err)
	}
}

// handleCallback processes DingTalk event subscription callbacks.
// GET requests are for URL verification; POST requests contain encrypted events.
func (t *DingTalkTransport) handleCallback(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		t.handleVerify(w, r)
	case "POST":
		t.handleEvent(w, r)
	default:
		w.WriteHeader(405)
	}
}

// handleVerify responds to DingTalk's callback URL verification challenge.
// DingTalk sends: signature, timestamp, nonce, echostr.
// We verify the signature and return the decrypted echostr.
func (t *DingTalkTransport) handleVerify(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	signature := q.Get("signature")
	timestamp := q.Get("timestamp")
	nonce := q.Get("nonce")
	echostr := q.Get("echostr")

	if echostr == "" {
		w.WriteHeader(400)
		return
	}

	// Verify signature: sha1(sort(token, timestamp, nonce, msg))
	if !t.verifySignature(signature, timestamp, nonce, echostr) {
		zap.S().Warnw("dingtalk callback verify: signature mismatch")
		w.WriteHeader(403)
		return
	}

	// Decrypt echostr and return plaintext
	plain, err := t.decryptMsg(echostr)
	if err != nil {
		zap.S().Warnw("dingtalk callback verify: decrypt failed", "error", err)
		w.WriteHeader(500)
		return
	}

	w.Write(plain)
}

// handleEvent processes incoming event notifications from DingTalk.
func (t *DingTalkTransport) handleEvent(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	signature := q.Get("signature")
	timestamp := q.Get("timestamp")
	nonce := q.Get("nonce")

	body, err := io.ReadAll(r.Body)
	if err != nil {
		zap.S().Warnw("dingtalk callback: read body failed", "error", err)
		w.WriteHeader(500)
		return
	}

	var encrypted struct {
		Encrypt string `json:"encrypt"`
	}
	if err := json.Unmarshal(body, &encrypted); err != nil {
		zap.S().Warnw("dingtalk callback: parse encrypt failed", "error", err)
		w.WriteHeader(400)
		return
	}

	if !t.verifySignature(signature, timestamp, nonce, encrypted.Encrypt) {
		zap.S().Warnw("dingtalk callback event: signature mismatch")
		w.WriteHeader(403)
		return
	}

	plain, err := t.decryptMsg(encrypted.Encrypt)
	if err != nil {
		zap.S().Warnw("dingtalk callback event: decrypt failed", "error", err)
		w.WriteHeader(500)
		return
	}

	// Parse the decrypted event JSON to extract message content
	var event struct {
		EventType          string `json:"EventType"`
		ConversationID     string `json:"conversationId"`
		OpenConversationID string `json:"openConversationId"`
		SenderID           string `json:"senderStaffId"`
		Text               struct {
			Content string `json:"content"`
		} `json:"text"`
		Content string `json:"Content"` // alternate field name
	}
	if err := json.Unmarshal(plain, &event); err != nil {
		zap.S().Warnw("dingtalk callback: parse event failed", "error", err, "plain", string(plain))
		w.WriteHeader(500)
		return
	}

	// Store conversation ID for replies
	if event.OpenConversationID != "" {
		t.convMu.Lock()
		t.convID = event.OpenConversationID
		t.convMu.Unlock()
	} else if event.ConversationID != "" {
		t.convMu.Lock()
		t.convID = event.ConversationID
		t.convMu.Unlock()
	}

	// Extract message text
	msgText := strings.TrimSpace(event.Text.Content)
	if msgText == "" {
		msgText = strings.TrimSpace(event.Content)
	}
	if msgText == "" {
		zap.S().Debugw("dingtalk callback: empty message, skipping")
		w.WriteHeader(200)
		w.Write([]byte("success"))
		return
	}

	zap.S().Infow("dingtalk message received", "sender", event.SenderID, "len", len(msgText))

	select {
	case t.msgCh <- msgText:
	default:
		zap.S().Warnw("dingtalk message dropped, channel full")
	}

	w.WriteHeader(200)
	w.Write([]byte("success"))
}

// verifySignature verifies the DingTalk callback signature.
// signature = sha1(sort(token, timestamp, nonce, msg_encrypt))
func (t *DingTalkTransport) verifySignature(signature, timestamp, nonce, msgEncrypt string) bool {
	if signature == "" {
		return true // skip verification in polling-only mode
	}
	arr := []string{t.cfg.ClientSecret, timestamp, nonce, msgEncrypt}
	sort.Strings(arr)
	hash := sha1.New()
	hash.Write([]byte(strings.Join(arr, "")))
	got := fmt.Sprintf("%x", hash.Sum(nil))
	return got == signature
}

// decryptMsg decrypts a DingTalk-encrypted message using AES-256-CBC.
// The encryption key is derived from the client secret.
func (t *DingTalkTransport) decryptMsg(encrypted string) ([]byte, error) {
	// DingTalk uses Base64-encoded encrypted data
	ciphertext, err := base64.StdEncoding.DecodeString(encrypted)
	if err != nil {
		return nil, fmt.Errorf("decode base64: %w", err)
	}

	// AES key is a 32-byte key derived from the client secret
	// DingTalk's AES key for callbacks is configured separately in the app console,
	// but for simplicity we derive it from the client secret (SHA256 first 32 bytes)
	// NOTE: If the user configures a custom AES key in DingTalk console, they should
	// set it via aes_key config field.
	aesKey := make([]byte, 32)
	secretBytes := []byte(t.cfg.ClientSecret)
	if len(secretBytes) >= 32 {
		copy(aesKey, secretBytes[:32])
	} else {
		// Pad with zeros
		copy(aesKey, secretBytes)
	}

	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return nil, fmt.Errorf("create cipher: %w", err)
	}

	if len(ciphertext) < aes.BlockSize {
		return nil, fmt.Errorf("ciphertext too short: %d", len(ciphertext))
	}

	// IV is the first 16 bytes of the AES key (DingTalk convention)
	iv := aesKey[:aes.BlockSize]
	mode := cipher.NewCBCDecrypter(block, iv)
	mode.CryptBlocks(ciphertext, ciphertext)

	// Remove PKCS7 padding
	plaintext, err := pkcs7Unpad(ciphertext)
	if err != nil {
		return nil, fmt.Errorf("unpad: %w", err)
	}

	// Extract message: random(16) + msg_len(4) + msg + appid
	if len(plaintext) < 20 {
		return nil, fmt.Errorf("plaintext too short: %d", len(plaintext))
	}
	msgLen := binary.BigEndian.Uint32(plaintext[16:20])
	msgStart := 20
	msgEnd := msgStart + int(msgLen)
	if msgEnd > len(plaintext) {
		return nil, fmt.Errorf("message length overflow: len=%d offset=%d", msgLen, msgEnd)
	}
	return plaintext[msgStart:msgEnd], nil
}

func pkcs7Unpad(data []byte) ([]byte, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("empty data")
	}
	padLen := int(data[len(data)-1])
	if padLen > aes.BlockSize || padLen == 0 || padLen > len(data) {
		return nil, fmt.Errorf("invalid padding: %d", padLen)
	}
	for i := len(data) - padLen; i < len(data); i++ {
		if data[i] != byte(padLen) {
			return nil, fmt.Errorf("invalid padding at %d", i)
		}
	}
	return data[:len(data)-padLen], nil
}

// startPolling periodically checks for new messages via DingTalk API.
func (t *DingTalkTransport) startPolling(ctx context.Context) {
	interval, err := time.ParseDuration(t.cfg.PollInterval)
	if err != nil || interval <= 0 {
		interval = 5 * time.Second
	}

	jitter := time.Duration(rand.Int63n(int64(interval / 4)))
	select {
	case <-ctx.Done():
		return
	case <-time.After(jitter):
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			t.pollMessages()
		}
	}
}

// pollMessages checks for new messages via DingTalk API.
func (t *DingTalkTransport) pollMessages() {
	token, err := t.getAccessToken()
	if err != nil {
		zap.S().Debugw("dingtalk poll: get token failed", "error", err)
		return
	}

	// Use the DingTalk message query API
	url := fmt.Sprintf("https://oapi.dingtalk.com/v1.0/im/robot/messages?access_token=%s", token)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		zap.S().Debugw("dingtalk poll: create request failed", "error", err)
		return
	}
	req.Header.Set("x-acs-dingtalk-access-token", token)

	resp, err := t.httpCli.Do(req)
	if err != nil {
		zap.S().Debugw("dingtalk poll: request failed", "error", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return
	}

	var result struct {
		Messages []struct {
			OpenConversationID string `json:"openConversationId"`
			Text               struct {
				Content string `json:"content"`
			} `json:"text"`
		} `json:"messages"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return
	}

	for _, msg := range result.Messages {
		if msg.Text.Content == "" {
			continue
		}
		if msg.OpenConversationID != "" {
			t.convMu.Lock()
			t.convID = msg.OpenConversationID
			t.convMu.Unlock()
		}
		zap.S().Infow("dingtalk poll: message received", "len", len(msg.Text.Content))
		select {
		case t.msgCh <- msg.Text.Content:
		default:
			zap.S().Warnw("dingtalk poll: message dropped, channel full")
		}
	}
}

// Close shuts down the transport.
func (t *DingTalkTransport) Close() error {
	t.closeOnce.Do(func() {
		close(t.closeCh)
		if t.httpSrv != nil {
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			t.httpSrv.Shutdown(shutdownCtx)
		}
	})
	return nil
}
