package transport

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"mime"
	"net"
	"net/smtp"
	"strings"
	"sync"
	"time"

	"dolphin/internal/config"

	goimap "github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
	"go.uber.org/zap"
)

// EmailTransport provides email-based I/O via SMTP (send) and IMAP (receive).
type EmailTransport struct {
	cfg         *config.EmailConfig
	msgCh       chan string
	closeCh     chan struct{}
	closeOnce   sync.Once
	pollTicker  *time.Ticker
	closeMu     sync.Mutex
	startTime   time.Time
	lastSender  string
	lastMsgID   string
	lastSubject string
	senderMu    sync.RWMutex
}

func NewEmailTransport(cfg *config.EmailConfig) *EmailTransport {
	return &EmailTransport{
		cfg:       cfg,
		msgCh:     make(chan string, 1024),
		closeCh:   make(chan struct{}),
		startTime: time.Now(),
	}
}

func (t *EmailTransport) Name() string { return "email" }

func (t *EmailTransport) Context() string {
	return fmt.Sprintf("Connected via email (IMAP: %s:%d, SMTP: %s:%d). User replies may be delayed by minutes. Be patient, self-contained, and thorough — each response is a separate email.",
		t.cfg.IMAPHost, t.cfg.IMAPPort, t.cfg.SMTPHost, t.cfg.SMTPPort)
}

func (t *EmailTransport) Capabilities() Capabilities {
	return Capabilities{Streaming: false, Flushable: true}
}

// Start begins IMAP polling and blocks until context is cancelled.
func (t *EmailTransport) Start(ctx context.Context) error {
	activeConnections.Add(1)
	interval, _ := time.ParseDuration(t.cfg.PollInterval)
	if interval <= 0 {
		interval = 10 * time.Second
	}
	t.pollTicker = time.NewTicker(interval)
	t.poll()
	for {
		select {
		case <-ctx.Done():
			return t.Close()
		case <-t.pollTicker.C:
			t.poll()
		}
	}
}

// ReadLine blocks until a new email command arrives or the transport is closed.
func (t *EmailTransport) ReadLine() (string, error) {
	select {
	case msg, ok := <-t.msgCh:
		if !ok {
			return "", fmt.Errorf("email transport closed")
		}
		msgsReceived.Inc()
		return msg, nil
	case <-t.closeCh:
		return "", fmt.Errorf("email transport closed")
	case <-time.After(5 * time.Minute):
		return "", fmt.Errorf("email transport: read timeout (5m)")
	}
}

// WriteLine sends an email response via SMTP.
func (t *EmailTransport) WriteLine(s string) error {
	return t.sendMail(s + "\n")
}

// WriteString sends an email response via SMTP.
func (t *EmailTransport) WriteString(s string) error {
	return t.sendMail(s)
}

func (t *EmailTransport) sendMail(body string) error {
	msgsSent.Inc()
	host := t.cfg.SMTPHost
	port := t.cfg.SMTPPort
	if port <= 0 {
		port = 587
	}
	addr := fmt.Sprintf("%s:%d", host, port)
	from := t.cfg.From
	if from == "" {
		from = t.cfg.Username
	}

	t.senderMu.RLock()
	to := t.lastSender
	msgID := t.lastMsgID
	subject := t.lastSubject
	t.senderMu.RUnlock()
	if to == "" {
		return fmt.Errorf("email: no sender yet — wait for an incoming message")
	}

	// Decode RFC 2047 encoded subject if needed
	if decoded, err := (&mime.WordDecoder{}).DecodeHeader(subject); err == nil {
		subject = decoded
	}
	if subject == "" {
		subject = "dolphin Agent"
	}
	if !strings.HasPrefix(strings.ToLower(subject), "re:") {
		subject = "Re: " + subject
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("From: %s\r\n", from))
	sb.WriteString(fmt.Sprintf("To: %s\r\n", to))
	sb.WriteString(fmt.Sprintf("Subject: %s\r\n", subject))
	sb.WriteString(fmt.Sprintf("Date: %s\r\n", time.Now().Format(time.RFC1123Z)))
	if msgID != "" {
		sb.WriteString(fmt.Sprintf("In-Reply-To: <%s>\r\n", msgID))
		sb.WriteString(fmt.Sprintf("References: <%s>\r\n", msgID))
	}
	sb.WriteString("MIME-Version: 1.0\r\n")
	sb.WriteString("Content-Type: text/plain; charset=\"utf-8\"\r\n")
	sb.WriteString("\r\n")
	sb.WriteString(body)

	if t.cfg.UseTLS && t.cfg.SMTPPort == 465 {
		return t.sendMailTLS(addr, host, sb.String(), to)
	}
	return t.sendMailPlain(addr, sb.String(), to)
}

func (t *EmailTransport) sendMailTLS(addr, host, msg, to string) error {
	tconn, err := tls.Dial("tcp", addr, &tls.Config{ServerName: host})
	if err != nil {
		return fmt.Errorf("tls connect: %w", err)
	}
	defer tconn.Close()

	sc, err := smtp.NewClient(tconn, host)
	if err != nil {
		return fmt.Errorf("smtp client: %w", err)
	}
	defer sc.Close()

	auth := smtp.PlainAuth("", t.cfg.Username, t.cfg.Password, host)
	if err := sc.Auth(auth); err != nil {
		return fmt.Errorf("smtp auth: %w", err)
	}
	sc.Mail(t.cfg.From)
	sc.Rcpt(to)
	w, err := sc.Data()
	if err != nil {
		return err
	}
	w.Write([]byte(msg))
	return w.Close()
}

func (t *EmailTransport) sendMailPlain(addr, msg, to string) error {
	auth := smtp.PlainAuth("", t.cfg.Username, t.cfg.Password, t.cfg.SMTPHost)
	return smtp.SendMail(addr, auth, t.cfg.From, []string{to}, []byte(msg))
}

func (t *EmailTransport) poll() {
	host := t.cfg.IMAPHost
	if host == "" {
		host = t.cfg.SMTPHost
	}
	port := t.cfg.IMAPPort
	if port <= 0 {
		port = 993
	}
	addr := fmt.Sprintf("%s:%d", host, port)

	d := &net.Dialer{Timeout: 30 * time.Second}
	tlsConn, err := tls.DialWithDialer(d, "tcp", addr, nil)
	if err != nil {
		zap.S().Warnw("email imap connect failed", "error", err)
		return
	}
	c, err := client.New(tlsConn)
	if err != nil {
		zap.S().Warnw("email imap connect failed", "error", err)
		return
	}
	defer c.Logout()

	if err := c.Login(t.cfg.Username, t.cfg.Password); err != nil {
		zap.S().Warnw("email imap login failed", "error", err)
		return
	}

	mbox, err := c.Select("INBOX", false)
	if err != nil {
		zap.S().Warnw("email imap select inbox failed", "error", err)
		return
	}
	if mbox.Messages == 0 {
		return
	}

	criteria := goimap.NewSearchCriteria()
	criteria.WithoutFlags = []string{"\\Seen"}
	seqNums, err := c.Search(criteria)
	if err != nil {
		zap.S().Debugw("email imap search failed", "error", err)
		return
	}
	if len(seqNums) == 0 {
		return
	}

	// Mark all unseen as read first
	allUnseen := new(goimap.SeqSet)
	allUnseen.AddNum(seqNums...)
	c.Store(allUnseen, goimap.AddFlags, []interface{}{"\\Seen"}, nil)

	// Only process the newest message
	latest := seqNums[len(seqNums)-1]
	seqset := new(goimap.SeqSet)
	seqset.AddNum(latest)

	messages := make(chan *goimap.Message, 1)
	if err := c.Fetch(seqset, []goimap.FetchItem{
		goimap.FetchEnvelope,
		goimap.FetchItem("BODY[TEXT]"),
	}, messages); err != nil {
		zap.S().Debugw("email imap fetch failed", "error", err)
		return
	}

	msg := <-messages
	if msg == nil || msg.Envelope == nil {
		return
	}

	// Skip messages sent before agent started
	if !msg.Envelope.Date.IsZero() && msg.Envelope.Date.Before(t.startTime) {
		return
	}

	// Skip self-sent messages to avoid infinite self-reply loop
	if isOwnAddress(msg.Envelope.From, t.cfg.From, t.cfg.Username) {
		return
	}

	// Only process emails from allowed senders (if configured)
	if len(t.cfg.AllowedSenders) > 0 && !isOwnAddress(msg.Envelope.From, t.cfg.AllowedSenders...) {
		zap.S().Debugw("email from non-allowed sender, skipped",
			"from", formatAddresses(msg.Envelope.From))
		return
	}

	// Decode RFC 2047 encoded subject
	rawSubject := msg.Envelope.Subject
	decSubject := rawSubject
	if d, err := (&mime.WordDecoder{}).DecodeHeader(rawSubject); err == nil {
		decSubject = d
	}
	if decSubject == "" {
		return
	}

	// Read body text
	var bodyText string
	for _, lit := range msg.Body {
		data, _ := io.ReadAll(lit)
		bodyText = strings.TrimSpace(string(data))
		break
	}

	// Build command: prefer body text, fall back to subject
	cmd := bodyText
	if cmd == "" {
		cmd = decSubject
	}

	// Store reply metadata
	if len(msg.Envelope.From) > 0 && msg.Envelope.From[0] != nil {
		t.senderMu.Lock()
		t.lastSender = msg.Envelope.From[0].Address()
		t.lastMsgID = msg.Envelope.MessageId
		t.lastSubject = rawSubject
		t.senderMu.Unlock()
	}

	zap.S().Infow("email received", "from", t.lastSender, "subject", truncate(decSubject, 80))

	select {
	case t.msgCh <- cmd:
	default:
		zap.S().Warnw("email message dropped, channel full")
	}
}

// formatAddresses formats a list of IMAP addresses for logging.
func formatAddresses(from []*goimap.Address) string {
	var parts []string
	for _, addr := range from {
		if addr != nil {
			parts = append(parts, addr.Address())
		}
	}
	return strings.Join(parts, ", ")
}

// isOwnAddress checks whether any sender address matches the reference list.
// Entries starting with "@" match any address ending with that domain suffix.
func isOwnAddress(from []*goimap.Address, refs ...string) bool {
	for _, addr := range from {
		if addr == nil {
			continue
		}
		addrStr := strings.ToLower(addr.Address())
		for _, ref := range refs {
			if ref == "" {
				continue
			}
			ref = strings.ToLower(ref)
			if strings.HasPrefix(ref, "@") {
				if strings.HasSuffix(addrStr, ref) {
					return true
				}
			} else if addrStr == ref {
				return true
			}
		}
	}
	return false
}

func (t *EmailTransport) Close() error {
	t.closeOnce.Do(func() {
		activeConnections.Add(-1)
		t.closeMu.Lock()
		if t.pollTicker != nil {
			t.pollTicker.Stop()
		}
		t.closeMu.Unlock()
		close(t.closeCh)
	})
	return nil
}
