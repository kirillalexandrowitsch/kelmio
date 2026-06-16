package mailer

import (
	"bufio"
	"context"
	"errors"
	"net"
	"strings"
	"testing"
	"time"

	"team-task-tracker/backend/internal/config"
)

func TestNoopClientSendReturnsSuccess(t *testing.T) {
	if err := (NoopClient{}).Send(context.Background(), Message{}); err != nil {
		t.Fatalf("Send() error = %v, want nil", err)
	}
}

func TestSMTPClientRejectsInvalidMessages(t *testing.T) {
	client := NewSMTPClient(validSMTPConfig("127.0.0.1", 1025))

	tests := []struct {
		name string
		msg  Message
		want string
	}{
		{name: "missing recipient", msg: Message{Subject: "Subject", TextBody: "Body"}, want: "at least one recipient is required"},
		{name: "invalid recipient", msg: Message{To: []string{"bad"}, Subject: "Subject", TextBody: "Body"}, want: "recipient email is invalid"},
		{name: "missing subject", msg: Message{To: []string{"member@example.com"}, TextBody: "Body"}, want: "subject is required"},
		{name: "subject newline", msg: Message{To: []string{"member@example.com"}, Subject: "Bad\nSubject", TextBody: "Body"}, want: "subject must not contain newlines"},
		{name: "missing body", msg: Message{To: []string{"member@example.com"}, Subject: "Subject"}, want: "text or html body is required"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := client.Send(context.Background(), tt.msg)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("Send() error = %v, want %q", err, tt.want)
			}
		})
	}
}

func TestSMTPClientDeliversToFakeServer(t *testing.T) {
	dial, received := fakeSMTPDialer(t)
	client := NewSMTPClient(validSMTPConfig("smtp.test", 1025))
	client.dial = dial

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err := client.Send(ctx, Message{
		To:       []string{"member@example.com"},
		Subject:  "SMTP smoke",
		TextBody: "Hello from Team Task Tracker",
	})
	if err != nil {
		t.Fatalf("Send() error = %v", err)
	}

	select {
	case body := <-received:
		if !strings.Contains(body, "Hello from Team Task Tracker") {
			t.Fatalf("message body = %q, want text body", body)
		}
		if !strings.Contains(body, "To: member@example.com") {
			t.Fatalf("message body = %q, want recipient header", body)
		}
	case <-ctx.Done():
		t.Fatal("timed out waiting for SMTP message")
	}
}

func TestSMTPClientSanitizesDeliveryErrors(t *testing.T) {
	cfg := validSMTPConfig("smtp.test", 1025)
	cfg.SMTPPassword = "super-secret-password"
	client := NewSMTPClient(cfg)
	client.dial = func(context.Context, string) (net.Conn, error) {
		return nil, errors.New("provider rejected super-secret-password")
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	err := client.Send(ctx, Message{
		To:       []string{"member@example.com"},
		Subject:  "Subject",
		TextBody: "Body",
	})
	if !errors.Is(err, ErrDeliveryFailed) {
		t.Fatalf("Send() error = %v, want ErrDeliveryFailed", err)
	}
	if strings.Contains(err.Error(), cfg.SMTPPassword) {
		t.Fatalf("Send() error leaked SMTP password: %v", err)
	}
}

func validSMTPConfig(host string, port int) config.Config {
	return config.Config{
		SMTPHost:      host,
		SMTPPort:      port,
		SMTPFromEmail: "no-reply@example.com",
		SMTPFromName:  "Team Task Tracker",
		SMTPTLSMode:   config.SMTPTLSModeNone,
	}
}

func fakeSMTPDialer(t *testing.T) (func(context.Context, string) (net.Conn, error), <-chan string) {
	t.Helper()
	received := make(chan string, 1)

	dial := func(context.Context, string) (net.Conn, error) {
		clientConn, serverConn := net.Pipe()
		go serveFakeSMTP(serverConn, received)
		return clientConn, nil
	}
	return dial, received
}

func serveFakeSMTP(conn net.Conn, received chan<- string) {
	defer conn.Close()
	reader := bufio.NewReader(conn)
	writer := bufio.NewWriter(conn)
	writeSMTPLine(writer, "220 localhost ESMTP")

	var data strings.Builder
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return
		}
		command := strings.ToUpper(strings.TrimSpace(line))
		switch {
		case strings.HasPrefix(command, "EHLO"), strings.HasPrefix(command, "HELO"):
			writeSMTPLine(writer, "250-localhost")
			writeSMTPLine(writer, "250 OK")
		case strings.HasPrefix(command, "MAIL FROM:"):
			writeSMTPLine(writer, "250 OK")
		case strings.HasPrefix(command, "RCPT TO:"):
			writeSMTPLine(writer, "250 OK")
		case command == "DATA":
			writeSMTPLine(writer, "354 End data with <CR><LF>.<CR><LF>")
			for {
				dataLine, err := reader.ReadString('\n')
				if err != nil {
					return
				}
				if strings.TrimSpace(dataLine) == "." {
					break
				}
				data.WriteString(dataLine)
			}
			writeSMTPLine(writer, "250 OK")
		case command == "QUIT":
			writeSMTPLine(writer, "221 Bye")
			received <- data.String()
			return
		default:
			writeSMTPLine(writer, "250 OK")
		}
	}
}

func writeSMTPLine(writer *bufio.Writer, line string) {
	_, _ = writer.WriteString(line + "\r\n")
	_ = writer.Flush()
}

func TestRenderMessageUsesConfiguredSender(t *testing.T) {
	client := NewSMTPClient(validSMTPConfig("127.0.0.1", 1025))
	msg, err := client.normalizeMessage(Message{
		To:       []string{"member@example.com"},
		Subject:  "Subject",
		TextBody: "Body",
	})
	if err != nil {
		t.Fatalf("normalizeMessage() error = %v", err)
	}
	rendered := string(renderMessage(msg))
	if !strings.Contains(rendered, `From: "Team Task Tracker" <no-reply@example.com>`) {
		t.Fatalf("rendered message = %q, want configured sender", rendered)
	}
}
