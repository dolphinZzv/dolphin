package transport

import (
	"fmt"
	"net"
	"strings"
	"testing"
	"time"

	"dolphin/internal/config"
)

// newEmailTransportWithSender creates an EmailTransport and sets a default reply-to sender.
func newEmailTransportWithSender(cfg *config.EmailConfig, sender string) *EmailTransport {
	tp := NewEmailTransport(cfg)
	tp.lastSender = sender
	return tp
}

// startTestSMTPServer starts a minimal SMTP server that captures the message body.
func startTestSMTPServer(t *testing.T) (addr string, gotMsg chan string) {
	gotMsg = make(chan string, 1)
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() { ln.Close() })

	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		conn.SetDeadline(time.Now().Add(5 * time.Second))

		// Server sends its banner
		conn.Write([]byte("220 localhost ESMTP test\r\n"))

		// Helper to read a line
		readLine := func() (string, error) {
			var buf strings.Builder
			tmp := make([]byte, 1)
			for {
				n, err := conn.Read(tmp)
				if err != nil {
					return "", err
				}
				if n == 0 {
					continue
				}
				if tmp[0] == '\n' {
					return strings.TrimRight(buf.String(), "\r"), nil
				}
				buf.WriteByte(tmp[0])
			}
		}

		reply := func(code string) {
			conn.Write([]byte(code + "\r\n"))
		}

		for {
			line, err := readLine()
			if err != nil {
				return
			}
			if strings.HasPrefix(line, "EHLO") || strings.HasPrefix(line, "HELO") {
				reply("250-localhost")
				reply("250 AUTH PLAIN")
			} else if strings.HasPrefix(line, "AUTH") {
				reply("235 2.7.0 Authentication successful")
			} else if strings.HasPrefix(line, "MAIL FROM") {
				reply("250 2.1.0 Ok")
			} else if strings.HasPrefix(line, "RCPT TO") {
				reply("250 2.1.5 Ok")
			} else if strings.HasPrefix(line, "DATA") {
				reply("354 End data with <CR><LF>.<CR><LF>")
				// Read body until \r\n.\r\n or \n.\n
				var body strings.Builder
				for {
					b, err := readLine()
					if err != nil {
						return
					}
					if b == "." {
						break
					}
					body.WriteString(b + "\n")
				}
				gotMsg <- body.String()
				reply("250 2.0.0 Ok: queued")
			} else if strings.HasPrefix(line, "QUIT") {
				reply("221 2.0.0 Bye")
				return
			}
		}
	}()

	return ln.Addr().String(), gotMsg
}

func TestEmailTransportName(t *testing.T) {
	tp := &EmailTransport{}
	if n := tp.Name(); n != "email" {
		t.Errorf("Name() = %q", n)
	}
}

func TestEmailTransportCapabilities(t *testing.T) {
	tp := &EmailTransport{}
	caps := tp.Capabilities()
	if caps.Streaming {
		t.Errorf("expected Streaming=false for email")
	}
	if !caps.Flushable {
		t.Errorf("expected Flushable=true for email")
	}
	if caps.ConfirmExit {
		t.Errorf("expected ConfirmExit=false for email")
	}
}

func TestEmailTransportSendMailWithPort(t *testing.T) {
	addr, gotMsg := startTestSMTPServer(t)

	cfg := &config.EmailConfig{
		SMTPHost: "localhost",
		SMTPPort: portFromAddr(addr),
		Username: "test@example.com",
		Password: "pass",
		From:     "test@example.com",
		UseTLS:   false,
	}
	tp := newEmailTransportWithSender(cfg, "recipient@example.com")
	err := tp.WriteLine("test with explicit port")
	if err != nil {
		t.Fatalf("WriteLine() error: %v", err)
	}

	select {
	case msg := <-gotMsg:
		if !strings.Contains(msg, "test with explicit port") {
			t.Errorf("expected body to contain message, got: %q", msg)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timeout waiting for SMTP server")
	}
}

func TestEmailTransportFromFallback(t *testing.T) {
	addr, gotMsg := startTestSMTPServer(t)

	cfg := &config.EmailConfig{
		SMTPHost: "localhost",
		SMTPPort: portFromAddr(addr),
		Username: "user@example.com",
		Password: "pass",
		From:     "",
		UseTLS:   false,
	}
	tp := newEmailTransportWithSender(cfg, "recipient@example.com")
	err := tp.WriteLine("test from fallback")
	if err != nil {
		t.Fatalf("WriteLine() error: %v", err)
	}

	select {
	case <-gotMsg:
	case <-time.After(3 * time.Second):
		t.Fatal("timeout waiting for SMTP server")
	}
}

func TestEmailTransportReadLine(t *testing.T) {
	tp := NewEmailTransport(&config.EmailConfig{})
	tp.msgCh <- "hello command"
	got, err := tp.ReadLine()
	if err != nil {
		t.Fatalf("ReadLine() error: %v", err)
	}
	if got != "hello command" {
		t.Errorf("ReadLine() = %q, want %q", got, "hello command")
	}
}

func TestEmailTransportReadLineClosed(t *testing.T) {
	tp := NewEmailTransport(&config.EmailConfig{})
	tp.Close()
	_, err := tp.ReadLine()
	if err == nil {
		t.Error("expected error after close")
	}
}

func TestEmailTransportWriteLine(t *testing.T) {
	addr, gotMsg := startTestSMTPServer(t)

	cfg := &config.EmailConfig{
		SMTPHost: "localhost",
		SMTPPort: portFromAddr(addr),
		Username: "test@example.com",
		Password: "pass",
		From:     "test@example.com",
		UseTLS:   false,
	}
	tp := newEmailTransportWithSender(cfg, "recipient@example.com")
	err := tp.WriteLine("hello response")
	if err != nil {
		t.Fatalf("WriteLine() error: %v", err)
	}

	select {
	case msg := <-gotMsg:
		if !strings.Contains(msg, "hello response") {
			t.Errorf("expected body to contain 'hello response', got: %q", msg)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timeout waiting for SMTP server to receive message")
	}
}

func TestEmailTransportWriteString(t *testing.T) {
	addr, gotMsg := startTestSMTPServer(t)

	cfg := &config.EmailConfig{
		SMTPHost: "localhost",
		SMTPPort: portFromAddr(addr),
		Username: "test@example.com",
		Password: "pass",
		From:     "test@example.com",
		UseTLS:   false,
	}
	tp := newEmailTransportWithSender(cfg, "recipient@example.com")
	err := tp.WriteString("hello world")
	if err != nil {
		t.Fatalf("WriteString() error: %v", err)
	}

	select {
	case msg := <-gotMsg:
		if !strings.Contains(msg, "hello world") {
			t.Errorf("expected body to contain 'hello world', got: %q", msg)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timeout waiting for SMTP server to receive message")
	}
}

func TestEmailTransportSendMailPlain(t *testing.T) {
	addr, gotMsg := startTestSMTPServer(t)

	cfg := &config.EmailConfig{
		SMTPHost: "localhost",
		SMTPPort: portFromAddr(addr),
		Username: "test@example.com",
		Password: "pass",
		From:     "test@example.com",
		UseTLS:   false,
	}
	tp := newEmailTransportWithSender(cfg, "recipient@example.com")
	err := tp.sendMail("test body")
	if err != nil {
		t.Fatalf("sendMail() error: %v", err)
	}

	select {
	case msg := <-gotMsg:
		if !strings.Contains(msg, "test body") {
			t.Errorf("expected 'test body' in message, got: %q", msg)
		}
		if !strings.Contains(msg, "Subject: Re: dolphin Agent") {
			t.Errorf("expected Subject header")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timeout waiting for SMTP server")
	}
}

// portFromAddr extracts the port from a host:port string.
func portFromAddr(addr string) int {
	_, port, _ := net.SplitHostPort(addr)
	var p int
	fmt.Sscanf(port, "%d", &p)
	return p
}

func TestEmailTransportSendMailSMTPPortDefault(t *testing.T) {
	// When SMTPPort is 0, it defaults to 587
	// We don't have a server on 587, so this should fail gracefully
	cfg := &config.EmailConfig{
		SMTPHost: "localhost",
		SMTPPort: 0,
		Username: "test@example.com",
		Password: "pass",
		From:     "test@example.com",
		UseTLS:   false,
	}
	tp := newEmailTransportWithSender(cfg, "recipient@example.com")
	err := tp.sendMail("test")
	if err == nil {
		t.Error("expected error when connecting to port 587")
	}
}

// TestEmailSendIntegration sends a real email to test@siciv.space using the configured SMTP.
func TestEmailSendIntegration(t *testing.T) {
	cfg, err := config.Load("../../.dolphin/config.yaml")
	if err != nil {
		t.Skipf("config load failed: %v", err)
	}
	ec := cfg.Transport.Email
	t.Logf("email enabled=%v user=%q from=%q smtp=%s:%d imap=%s:%d useTLS=%v",
		ec.Enabled, ec.Username, ec.From, ec.SMTPHost, ec.SMTPPort, ec.IMAPHost, ec.IMAPPort, ec.UseTLS)
	if !ec.Enabled {
		t.Skip("email transport disabled")
	}

	tp := newEmailTransportWithSender(&ec, "test@siciv.space")
	err = tp.sendMail("[dolphin integration test] SMTP connectivity check at " + time.Now().Format(time.RFC3339))
	if err != nil {
		t.Fatalf("sendMail failed: %v", err)
	}
	t.Log("email sent successfully")
}

// TestEmailIMAPIntegration checks IMAP connectivity and lists unseen messages.
func TestEmailIMAPIntegration(t *testing.T) {
	cfg, err := config.Load("../../.dolphin/config.yaml")
	if err != nil {
		t.Skipf("config load failed: %v", err)
	}
	ec := cfg.Transport.Email
	t.Logf("email enabled=%v user=%q from=%q smtp=%s:%d imap=%s:%d useTLS=%v",
		ec.Enabled, ec.Username, ec.From, ec.SMTPHost, ec.SMTPPort, ec.IMAPHost, ec.IMAPPort, ec.UseTLS)
	if !ec.Enabled {
		t.Skip("email transport disabled")
	}
	if ec.Username == "" || ec.Password == "" {
		t.Skip("email credentials not configured")
	}
	t.Logf("IMAP: %s:%d, user=%s, TLS=%v",
		ec.IMAPHost, ec.IMAPPort, ec.Username, ec.UseTLS)

	tp := NewEmailTransport(&ec)
	tp.poll()
	t.Log("IMAP poll complete — check logs for results")
}

func TestEmailTransportSendMailTLSNotUsed(t *testing.T) {
	// With UseTLS=false and SMTPPort=587, sendMailPlain is used
	// This test verifies the non-TLS path works with a mock server
	addr, gotMsg := startTestSMTPServer(t)
	cfg := &config.EmailConfig{
		SMTPHost: "localhost",
		SMTPPort: portFromAddr(addr),
		Username: "test@example.com",
		Password: "pass",
		From:     "test@example.com",
		UseTLS:   false,
	}
	tp := newEmailTransportWithSender(cfg, "recipient@example.com")
	err := tp.WriteString("non-tls test")
	if err != nil {
		t.Fatalf("WriteString error: %v", err)
	}
	select {
	case msg := <-gotMsg:
		if !strings.Contains(msg, "non-tls test") {
			t.Errorf("expected body, got: %q", msg)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timeout")
	}
}
