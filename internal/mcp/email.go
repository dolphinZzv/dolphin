package mcp

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/mail"
	"net/smtp"
	"strings"
	"time"

	"dolphin/internal/config"

	goimap "github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
)

// EmailTool provides SMTP send and IMAP search/fetch as a built-in MCP tool.
type EmailTool struct {
	cfg    *config.Config
	schema json.RawMessage
}

func NewEmailTool(cfg *config.Config) *EmailTool {
	schema, _ := json.Marshal(map[string]any{
		"type": "object",
		"properties": map[string]any{
			"action": map[string]any{
				"type":        "string",
				"enum":        []string{"send", "search", "fetch"},
				"description": "send: send an email; search: search inbox; fetch: read a specific email body",
			},
			"to": map[string]any{
				"type":        "string",
				"description": "recipient email address (required for send action)",
			},
			"subject": map[string]any{
				"type":        "string",
				"description": "email subject (required for send action)",
			},
			"body": map[string]any{
				"type":        "string",
				"description": "email body text (required for send action)",
			},
			"query": map[string]any{
				"type":        "string",
				"description": "search text to match in subject or sender (search action)",
			},
			"max_results": map[string]any{
				"type":        "integer",
				"description": "max results to return (search only, default 10, max 50)",
			},
			"seq": map[string]any{
				"type":        "integer",
				"description": "IMAP sequence number (required for fetch action)",
			},
			"unread_only": map[string]any{
				"type":        "boolean",
				"description": "only search unread messages (search only, default false)",
			},
		},
		"required": []string{"action"},
	})
	return &EmailTool{cfg: cfg, schema: schema}
}

func (e *EmailTool) Definition() ToolDefinition {
	return ToolDefinition{
		Name:        "email",
		Description: "Send and receive emails via SMTP/IMAP. Actions: send (send an email), search (search inbox messages), fetch (read a specific email body by sequence number). Requires transport.email to be configured.",
		InputSchema: e.schema,
		Priority:    e.cfg.MCP.Email.Priority,
		Source:      "built-in",
	}
}

func (e *EmailTool) Execute(ctx context.Context, input json.RawMessage) (*ToolResult, error) {
	var params struct {
		Action     string `json:"action"`
		To         string `json:"to,omitempty"`
		Subject    string `json:"subject,omitempty"`
		Body       string `json:"body,omitempty"`
		Query      string `json:"query,omitempty"`
		MaxResults int    `json:"max_results,omitempty"`
		Seq        uint32 `json:"seq,omitempty"`
		UnreadOnly bool   `json:"unread_only,omitempty"`
	}
	if err := json.Unmarshal(input, &params); err != nil {
		return &ToolResult{Content: "Invalid input: " + err.Error(), IsError: true}, nil
	}

	ecfg := &e.cfg.Transport.Email
	if ecfg.Username == "" || ecfg.Password == "" {
		return &ToolResult{Content: "Email not configured. Set transport.email.username and transport.email.password in config.", IsError: true}, nil
	}

	switch params.Action {
	case "send":
		return e.send(ecfg, params.To, params.Subject, params.Body)
	case "search":
		if params.MaxResults <= 0 || params.MaxResults > 50 {
			params.MaxResults = 10
		}
		return e.search(ecfg, params.Query, params.UnreadOnly, params.MaxResults)
	case "fetch":
		if params.Seq == 0 {
			return &ToolResult{Content: "seq (IMAP sequence number) is required for fetch action.", IsError: true}, nil
		}
		return e.fetch(ecfg, params.Seq)
	default:
		return &ToolResult{Content: fmt.Sprintf("Unknown action: %q. Available: send, search, fetch.", params.Action), IsError: true}, nil
	}
}

// ── Send ─────────────────────────────────────────────────────────────

func (e *EmailTool) send(ecfg *config.EmailConfig, to, subject, body string) (*ToolResult, error) {
	if to == "" {
		return &ToolResult{Content: "Missing required field: to.", IsError: true}, nil
	}
	if subject == "" {
		return &ToolResult{Content: "Missing required field: subject.", IsError: true}, nil
	}

	from := ecfg.From
	if from == "" {
		from = ecfg.Username
	}

	host := ecfg.SMTPHost
	port := ecfg.SMTPPort
	if port <= 0 {
		port = 587
	}
	addr := fmt.Sprintf("%s:%d", host, port)

	var msg strings.Builder
	msg.WriteString(fmt.Sprintf("From: %s\r\n", from))
	msg.WriteString(fmt.Sprintf("To: %s\r\n", to))
	msg.WriteString(fmt.Sprintf("Subject: %s\r\n", subject))
	msg.WriteString(fmt.Sprintf("Date: %s\r\n", time.Now().Format(time.RFC1123Z)))
	msg.WriteString("MIME-Version: 1.0\r\n")
	msg.WriteString("Content-Type: text/plain; charset=\"utf-8\"\r\n")
	msg.WriteString("\r\n")
	msg.WriteString(body)

	if ecfg.UseTLS && ecfg.SMTPPort == 465 {
		if err := sendTLS(addr, host, from, []string{to}, msg.String(), ecfg.Username, ecfg.Password); err != nil {
			return &ToolResult{Content: fmt.Sprintf("Send failed (TLS): %s", err.Error()), IsError: true}, nil
		}
	} else {
		if err := sendPlain(addr, host, from, []string{to}, msg.String(), ecfg.Username, ecfg.Password); err != nil {
			return &ToolResult{Content: fmt.Sprintf("Send failed: %s", err.Error()), IsError: true}, nil
		}
	}

	return &ToolResult{Content: fmt.Sprintf("Email sent to %s: %s", to, subject)}, nil
}

func sendTLS(addr, host, from string, to []string, msg, user, pass string) error {
	tconn, err := tls.Dial("tcp", addr, &tls.Config{ServerName: host})
	if err != nil {
		return fmt.Errorf("connect: %w", err)
	}
	defer tconn.Close()

	sc, err := smtp.NewClient(tconn, host)
	if err != nil {
		return fmt.Errorf("smtp client: %w", err)
	}
	defer sc.Close()

	auth := smtp.PlainAuth("", user, pass, host)
	if err := sc.Auth(auth); err != nil {
		return fmt.Errorf("auth: %w", err)
	}
	if err := sc.Mail(from); err != nil {
		return fmt.Errorf("mail from: %w", err)
	}
	for _, rcpt := range to {
		if err := sc.Rcpt(rcpt); err != nil {
			return fmt.Errorf("rcpt %s: %w", rcpt, err)
		}
	}
	w, err := sc.Data()
	if err != nil {
		return fmt.Errorf("data: %w", err)
	}
	if _, err := w.Write([]byte(msg)); err != nil {
		return fmt.Errorf("write: %w", err)
	}
	return w.Close()
}

func sendPlain(addr, host, from string, to []string, msg, user, pass string) error {
	auth := smtp.PlainAuth("", user, pass, host)
	return smtp.SendMail(addr, auth, from, to, []byte(msg))
}

// ── Search ───────────────────────────────────────────────────────────

func (e *EmailTool) search(ecfg *config.EmailConfig, query string, unreadOnly bool, maxResults int) (*ToolResult, error) {
	c, err := dialIMAP(ecfg)
	if err != nil {
		return &ToolResult{Content: fmt.Sprintf("IMAP connection failed: %s", err.Error()), IsError: true}, nil
	}
	defer c.Logout()

	mbox, err := c.Select("INBOX", false)
	if err != nil {
		return &ToolResult{Content: fmt.Sprintf("Failed to select INBOX: %s", err.Error()), IsError: true}, nil
	}
	if mbox.Messages == 0 {
		return &ToolResult{Content: "No messages in inbox."}, nil
	}

	criteria := goimap.NewSearchCriteria()
	if unreadOnly {
		criteria.WithoutFlags = []string{"\\Seen"}
	}
	if query != "" {
		// Support "from:..." and "subject:..." prefixes, otherwise search both
		criteria.Text = []string{query}
		if strings.HasPrefix(strings.ToLower(query), "from:") {
			criteria.Text = nil
			fromVal := strings.TrimSpace(query[5:])
			criteria.Header = map[string][]string{"From": {fromVal}}
		} else if strings.HasPrefix(strings.ToLower(query), "subject:") {
			criteria.Text = nil
			subjVal := strings.TrimSpace(query[8:])
			criteria.Header = map[string][]string{"Subject": {subjVal}}
		}
	}

	seqNums, err := c.Search(criteria)
	if err != nil {
		return &ToolResult{Content: fmt.Sprintf("Search failed: %s", err.Error()), IsError: true}, nil
	}
	if len(seqNums) == 0 {
		return &ToolResult{Content: "No matching emails found."}, nil
	}

	// Take the most recent N messages
	start := 0
	if len(seqNums) > maxResults {
		start = len(seqNums) - maxResults
	}
	latest := seqNums[start:]

	seqset := new(goimap.SeqSet)
	seqset.AddNum(latest...)

	messages := make(chan *goimap.Message, len(latest))
	if err := c.Fetch(seqset, []goimap.FetchItem{goimap.FetchEnvelope}, messages); err != nil {
		return &ToolResult{Content: fmt.Sprintf("Fetch failed: %s", err.Error()), IsError: true}, nil
	}

	var results []string
	for msg := range messages {
		if msg == nil || msg.Envelope == nil {
			continue
		}
		from := ""
		if len(msg.Envelope.From) > 0 {
			from = msg.Envelope.From[0].PersonalName
			if from == "" {
				from = msg.Envelope.From[0].MailboxName + "@" + msg.Envelope.From[0].HostName
			}
		}
		results = append(results, fmt.Sprintf("Seq:%d | %s | %s | %s",
			msg.SeqNum,
			msg.Envelope.Date.Format("2006-01-02 15:04"),
			truncate(from, 40),
			truncate(msg.Envelope.Subject, 60)))
	}

	if len(results) == 0 {
		return &ToolResult{Content: "No matching emails found."}, nil
	}

	summary := fmt.Sprintf("Found %d matching emails (showing %d):\n\n%s",
		len(seqNums), len(results), strings.Join(results, "\n"))
	return &ToolResult{Content: summary}, nil
}

// ── Fetch ────────────────────────────────────────────────────────────

func (e *EmailTool) fetch(ecfg *config.EmailConfig, seq uint32) (*ToolResult, error) {
	c, err := dialIMAP(ecfg)
	if err != nil {
		return &ToolResult{Content: fmt.Sprintf("IMAP connection failed: %s", err.Error()), IsError: true}, nil
	}
	defer c.Logout()

	if _, err := c.Select("INBOX", true); err != nil {
		return &ToolResult{Content: fmt.Sprintf("Failed to select INBOX: %s", err.Error()), IsError: true}, nil
	}

	seqset := new(goimap.SeqSet)
	seqset.AddNum(seq)

	messages := make(chan *goimap.Message, 1)
	items := []goimap.FetchItem{goimap.FetchEnvelope, goimap.FetchRFC822}
	if err := c.Fetch(seqset, items, messages); err != nil {
		return &ToolResult{Content: fmt.Sprintf("Fetch failed: %s", err.Error()), IsError: true}, nil
	}

	msg := <-messages
	if msg == nil {
		return &ToolResult{Content: fmt.Sprintf("Message seq %d not found.", seq), IsError: true}, nil
	}

	// Read the RFC822 body
	var bodyText string
	for _, literal := range msg.Body {
		data, err := io.ReadAll(literal)
		if err != nil {
			continue
		}
		bodyText = string(data)
		break
	}

	if bodyText == "" {
		return &ToolResult{Content: "No body content found."}, nil
	}

	// Parse with net/mail for headers + body
	r := strings.NewReader(bodyText)
	parsed, err := mail.ReadMessage(r)
	if err != nil {
		// Return raw body if parsing fails
		bodyPreview := truncate(bodyText, 2000)
		return &ToolResult{Content: fmt.Sprintf("Seq: %d\n(raw content, parse failed: %s)\n\n%s", seq, err.Error(), bodyPreview)}, nil
	}

	header := parsed.Header
	from := header.Get("From")
	to := header.Get("To")
	subject := header.Get("Subject")
	date := header.Get("Date")
	bodyBytes, _ := io.ReadAll(parsed.Body)
	bodyStr := string(bodyBytes)

	result := fmt.Sprintf("From: %s\nTo: %s\nSubject: %s\nDate: %s\nSeq: %d\n\n%s",
		from, to, subject, date, seq, truncate(bodyStr, 10000))

	return &ToolResult{Content: result}, nil
}

// ── Helpers ──────────────────────────────────────────────────────────

func dialIMAP(ecfg *config.EmailConfig) (*client.Client, error) {
	host := ecfg.IMAPHost
	if host == "" {
		host = ecfg.SMTPHost
	}
	port := ecfg.IMAPPort
	if port <= 0 {
		port = 993
	}
	addr := fmt.Sprintf("%s:%d", host, port)

	d := &net.Dialer{Timeout: 30 * time.Second}
	tlsConn, err := tls.DialWithDialer(d, "tcp", addr, nil)
	if err != nil {
		return nil, err
	}
	c, err := client.New(tlsConn)
	if err != nil {
		tlsConn.Close()
		return nil, err
	}
	if err := c.Login(ecfg.Username, ecfg.Password); err != nil {
		c.Logout()
		return nil, err
	}
	return c, nil
}

