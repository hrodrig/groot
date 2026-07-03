package notifier

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
	"strconv"
	"strings"
)

type emailSender struct {
	host       string
	port       int
	username   string
	password   string
	from       string
	recipients []string
	useTLS     bool
	skipVerify bool
}

func appendEmailSenders(out []Sender, e emailCfgView) []Sender {
	if !e.enabled {
		return out
	}
	recipients := splitList(e.to)
	if len(recipients) == 0 {
		return out
	}
	out = append(out, &emailSender{
		host:       strings.TrimSpace(e.host),
		port:       e.port,
		username:   strings.TrimSpace(e.username),
		password:   e.password,
		from:       strings.TrimSpace(e.from),
		recipients: recipients,
		useTLS:     e.useTLS,
		skipVerify: e.skipVerify,
	})
	return out
}

type emailCfgView struct {
	enabled    bool
	host       string
	port       int
	username   string
	password   string
	from       string
	to         string
	useTLS     bool
	skipVerify bool
}

func (e *emailSender) Send(_ context.Context, text string) error {
	if e.host == "" || e.from == "" {
		return fmt.Errorf("email: host and from are required")
	}
	port := e.port
	if port == 0 {
		port = 587
	}
	msg := buildEmailMessage(e.from, e.recipients, text)
	auth := emailAuth(e.host, e.username, e.password)
	addr := net.JoinHostPort(e.host, strconv.Itoa(port))

	switch {
	case e.useTLS:
		return e.sendImplicitTLS(addr, auth, msg)
	case port == 587:
		return e.sendSTARTTLS(addr, auth, msg)
	default:
		return smtp.SendMail(addr, auth, e.from, e.recipients, msg)
	}
}

func buildEmailMessage(from string, recipients []string, text string) []byte {
	body := strings.ReplaceAll(text, "\r\n", "\n")
	msg := strings.Join([]string{
		"From: " + from,
		"To: " + strings.Join(recipients, ", "),
		"Subject: GROOT collect notification",
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=UTF-8",
		"",
		body,
	}, "\r\n")
	return []byte(msg)
}

func emailAuth(host, username, password string) smtp.Auth {
	if strings.TrimSpace(username) == "" {
		return nil
	}
	return smtp.PlainAuth("", username, password, host)
}

func (e *emailSender) tlsConfig() *tls.Config {
	cfg := &tls.Config{ServerName: e.host, MinVersion: tls.VersionTLS12}
	if e.skipVerify {
		cfg.InsecureSkipVerify = true
	}
	return cfg
}

func (e *emailSender) sendImplicitTLS(addr string, auth smtp.Auth, msg []byte) error {
	conn, err := tls.Dial("tcp", addr, e.tlsConfig())
	if err != nil {
		return fmt.Errorf("email TLS dial: %w", err)
	}
	client, err := smtp.NewClient(conn, e.host)
	if err != nil {
		_ = conn.Close()
		return fmt.Errorf("email client: %w", err)
	}
	defer client.Close()
	if auth != nil {
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("email auth: %w", err)
		}
	}
	return deliverMessage(client, e.from, e.recipients, msg)
}

func (e *emailSender) sendSTARTTLS(addr string, auth smtp.Auth, msg []byte) error {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return fmt.Errorf("email dial: %w", err)
	}
	client, err := smtp.NewClient(conn, e.host)
	if err != nil {
		_ = conn.Close()
		return fmt.Errorf("email client: %w", err)
	}
	defer client.Close()
	if ok, _ := client.Extension("STARTTLS"); ok {
		if err := client.StartTLS(e.tlsConfig()); err != nil {
			return fmt.Errorf("email STARTTLS: %w", err)
		}
	}
	if auth != nil {
		if ok, _ := client.Extension("AUTH"); ok {
			if err := client.Auth(auth); err != nil {
				return fmt.Errorf("email auth: %w", err)
			}
		}
	}
	return deliverMessage(client, e.from, e.recipients, msg)
}

func deliverMessage(client *smtp.Client, from string, recipients []string, msg []byte) error {
	if err := client.Mail(from); err != nil {
		return fmt.Errorf("email MAIL FROM: %w", err)
	}
	for _, rcpt := range recipients {
		if err := client.Rcpt(rcpt); err != nil {
			return fmt.Errorf("email RCPT TO %s: %w", rcpt, err)
		}
	}
	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("email DATA: %w", err)
	}
	if _, err := w.Write(msg); err != nil {
		_ = w.Close()
		return fmt.Errorf("email write: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("email close data: %w", err)
	}
	return client.Quit()
}

func splitList(raw string) []string {
	parts := strings.Split(raw, ";")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		u := strings.TrimSpace(p)
		if u != "" {
			out = append(out, u)
		}
	}
	return out
}
