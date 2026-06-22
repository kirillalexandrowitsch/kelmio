package mailer

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"mime"
	"net"
	"net/mail"
	"net/smtp"
	"strconv"
	"strings"

	"kelmio/backend/internal/config"
)

var ErrDeliveryFailed = errors.New("smtp delivery failed")

type Message struct {
	FromEmail string
	FromName  string
	To        []string
	Subject   string
	TextBody  string
	HTMLBody  string
}

type Client interface {
	Send(ctx context.Context, msg Message) error
}

type NoopClient struct{}

func (NoopClient) Send(context.Context, Message) error {
	return nil
}

type SMTPClient struct {
	host      string
	port      int
	username  string
	password  string
	tlsMode   string
	fromEmail string
	fromName  string
	dial      func(ctx context.Context, address string) (net.Conn, error)
}

func NewClient(cfg config.Config) Client {
	if !cfg.EmailDeliveryEnabled {
		return NoopClient{}
	}
	return NewSMTPClient(cfg)
}

func NewSMTPClient(cfg config.Config) *SMTPClient {
	return &SMTPClient{
		host:      cfg.SMTPHost,
		port:      cfg.SMTPPort,
		username:  cfg.SMTPUsername,
		password:  cfg.SMTPPassword,
		tlsMode:   cfg.SMTPTLSMode,
		fromEmail: cfg.SMTPFromEmail,
		fromName:  cfg.SMTPFromName,
	}
}

func (c *SMTPClient) Send(ctx context.Context, msg Message) error {
	normalized, err := c.normalizeMessage(msg)
	if err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	address := net.JoinHostPort(c.host, strconv.Itoa(c.port))
	conn, err := c.dialSMTP(ctx, address)
	if err != nil {
		return deliveryError("connect to smtp server", err)
	}
	defer conn.Close()
	if deadline, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(deadline)
	}

	client, err := smtp.NewClient(conn, c.host)
	if err != nil {
		return deliveryError("create smtp client", err)
	}
	defer client.Close()

	if c.tlsMode == config.SMTPTLSModeStartTLS {
		if ok, _ := client.Extension("STARTTLS"); !ok {
			return deliveryError("start tls", errors.New("smtp server does not advertise STARTTLS"))
		}
		if err := client.StartTLS(c.tlsConfig()); err != nil {
			return deliveryError("start tls", err)
		}
	}
	if c.username != "" {
		auth := smtp.PlainAuth("", c.username, c.password, c.host)
		if err := client.Auth(auth); err != nil {
			return deliveryError("authenticate smtp client", err)
		}
	}
	if err := client.Mail(normalized.FromEmail); err != nil {
		return deliveryError("set smtp sender", err)
	}
	for _, recipient := range normalized.To {
		if err := client.Rcpt(recipient); err != nil {
			return deliveryError("set smtp recipient", err)
		}
	}
	writer, err := client.Data()
	if err != nil {
		return deliveryError("open smtp data writer", err)
	}
	if _, err := writer.Write(renderMessage(normalized)); err != nil {
		_ = writer.Close()
		return deliveryError("write smtp message", err)
	}
	if err := writer.Close(); err != nil {
		return deliveryError("close smtp message", err)
	}
	if err := client.Quit(); err != nil {
		return deliveryError("quit smtp session", err)
	}
	return nil
}

func (c *SMTPClient) normalizeMessage(msg Message) (Message, error) {
	msg.FromEmail = firstNonEmpty(msg.FromEmail, c.fromEmail)
	msg.FromName = firstNonEmpty(msg.FromName, c.fromName)
	msg.Subject = strings.TrimSpace(msg.Subject)
	msg.TextBody = strings.TrimSpace(msg.TextBody)
	msg.HTMLBody = strings.TrimSpace(msg.HTMLBody)

	if _, err := parseMailbox(msg.FromEmail); err != nil {
		return Message{}, fmt.Errorf("from email is invalid: %w", err)
	}
	if len(msg.To) == 0 {
		return Message{}, errors.New("at least one recipient is required")
	}
	normalizedTo := make([]string, 0, len(msg.To))
	for _, recipient := range msg.To {
		parsed, err := parseMailbox(recipient)
		if err != nil {
			return Message{}, fmt.Errorf("recipient email is invalid: %w", err)
		}
		normalizedTo = append(normalizedTo, parsed)
	}
	msg.To = normalizedTo
	if msg.Subject == "" {
		return Message{}, errors.New("subject is required")
	}
	if strings.ContainsAny(msg.Subject, "\r\n") {
		return Message{}, errors.New("subject must not contain newlines")
	}
	if msg.TextBody == "" && msg.HTMLBody == "" {
		return Message{}, errors.New("text or html body is required")
	}
	return msg, nil
}

func (c *SMTPClient) tlsConfig() *tls.Config {
	return &tls.Config{
		ServerName: c.host,
		MinVersion: tls.VersionTLS12,
	}
}

func (c *SMTPClient) dialSMTP(ctx context.Context, address string) (net.Conn, error) {
	var (
		conn net.Conn
		err  error
	)
	if c.dial != nil {
		conn, err = c.dial(ctx, address)
	} else {
		dialer := &net.Dialer{}
		conn, err = dialer.DialContext(ctx, "tcp", address)
	}
	if err != nil {
		return nil, err
	}
	if c.tlsMode != config.SMTPTLSModeTLS {
		return conn, nil
	}
	tlsConn := tls.Client(conn, c.tlsConfig())
	if err := tlsConn.HandshakeContext(ctx); err != nil {
		_ = conn.Close()
		return nil, err
	}
	return tlsConn, nil
}

func renderMessage(msg Message) []byte {
	var body bytes.Buffer
	from := (&mail.Address{Name: msg.FromName, Address: msg.FromEmail}).String()
	writeHeader(&body, "From", from)
	writeHeader(&body, "To", strings.Join(msg.To, ", "))
	writeHeader(&body, "Subject", mime.QEncoding.Encode("utf-8", msg.Subject))
	writeHeader(&body, "MIME-Version", "1.0")

	if msg.TextBody != "" && msg.HTMLBody != "" {
		boundary := "kelmio-message-boundary"
		writeHeader(&body, "Content-Type", `multipart/alternative; boundary="`+boundary+`"`)
		body.WriteString("\r\n")
		writePart(&body, boundary, "text/plain; charset=utf-8", msg.TextBody)
		writePart(&body, boundary, "text/html; charset=utf-8", msg.HTMLBody)
		body.WriteString("--" + boundary + "--\r\n")
		return body.Bytes()
	}

	contentType := "text/plain; charset=utf-8"
	content := msg.TextBody
	if msg.HTMLBody != "" {
		contentType = "text/html; charset=utf-8"
		content = msg.HTMLBody
	}
	writeHeader(&body, "Content-Type", contentType)
	writeHeader(&body, "Content-Transfer-Encoding", "8bit")
	body.WriteString("\r\n")
	body.WriteString(normalizeNewlines(content))
	body.WriteString("\r\n")
	return body.Bytes()
}

func writePart(w io.StringWriter, boundary string, contentType string, content string) {
	w.WriteString("--" + boundary + "\r\n")
	writeHeader(w, "Content-Type", contentType)
	writeHeader(w, "Content-Transfer-Encoding", "8bit")
	w.WriteString("\r\n")
	w.WriteString(normalizeNewlines(content))
	w.WriteString("\r\n")
}

func writeHeader(w io.StringWriter, key string, value string) {
	w.WriteString(key + ": " + value + "\r\n")
}

func normalizeNewlines(value string) string {
	value = strings.ReplaceAll(value, "\r\n", "\n")
	value = strings.ReplaceAll(value, "\r", "\n")
	return strings.ReplaceAll(value, "\n", "\r\n")
}

func parseMailbox(value string) (string, error) {
	parsed, err := mail.ParseAddress(strings.TrimSpace(value))
	if err != nil {
		return "", err
	}
	return parsed.Address, nil
}

func firstNonEmpty(value string, fallback string) string {
	if strings.TrimSpace(value) != "" {
		return strings.TrimSpace(value)
	}
	return strings.TrimSpace(fallback)
}

func deliveryError(operation string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", operation, ErrDeliveryFailed)
}
