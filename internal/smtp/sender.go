// Package smtp implements an SMTP sender for forwarding emails to Gmail.
package smtp

import (
	"bytes"
	"fmt"
	"net"
	"net/mail"
	netsmtp "net/smtp"
	"strconv"
	"strings"
)

// Sender handles forwarding emails via SMTP to Gmail.
type Sender struct {
	host     string
	port     int
	username string
	password string
	to       string
}

// NewSender creates a new SMTP Sender configured for Gmail.
func NewSender(host string, port int, username, password, to string) *Sender {
	return &Sender{
		host:     host,
		port:     port,
		username: username,
		password: password,
		to:       to,
	}
}

// Send forwards a raw email message to the configured Gmail account.
// It parses the original email to extract the From address and rewrites
// headers so that Gmail's filtering system processes the email correctly.
func (s *Sender) Send(rawEmail []byte, originalFrom string) error {
	// Parse the original message to extract headers.
	msg, err := mail.ReadMessage(bytes.NewReader(rawEmail))
	if err != nil {
		// If we can't parse, send as-is with a wrapper.
		return s.sendRaw(rawEmail, originalFrom)
	}

	// Build the forwarded message with proper headers for Gmail filtering.
	var buf bytes.Buffer

	// Preserve important original headers.
	origFrom := msg.Header.Get("From")
	origDate := msg.Header.Get("Date")
	origSubject := msg.Header.Get("Subject")
	origMessageID := msg.Header.Get("Message-Id")
	origTo := msg.Header.Get("To")
	origCc := msg.Header.Get("Cc")
	origReplyTo := msg.Header.Get("Reply-To")
	contentType := msg.Header.Get("Content-Type")
	contentTransferEncoding := msg.Header.Get("Content-Transfer-Encoding")
	mimeVersion := msg.Header.Get("MIME-Version")

	// Write headers that Gmail will use for filtering.
	// The "From" must be the authenticated sender (Gmail requirement),
	// but we put the original sender in Reply-To and X-Original-From.
	fmt.Fprintf(&buf, "From: %s\r\n", s.to)
	fmt.Fprintf(&buf, "To: %s\r\n", s.to)
	if origSubject != "" {
		fmt.Fprintf(&buf, "Subject: %s\r\n", origSubject)
	}
	if origDate != "" {
		fmt.Fprintf(&buf, "Date: %s\r\n", origDate)
	}

	// Preserve original sender info.
	if origFrom != "" {
		fmt.Fprintf(&buf, "X-Original-From: %s\r\n", origFrom)
		fmt.Fprintf(&buf, "Resent-From: %s\r\n", origFrom)
	}
	if origTo != "" {
		fmt.Fprintf(&buf, "X-Original-To: %s\r\n", origTo)
	}
	if origCc != "" {
		fmt.Fprintf(&buf, "X-Original-Cc: %s\r\n", origCc)
	}
	if origReplyTo != "" {
		fmt.Fprintf(&buf, "Reply-To: %s\r\n", origReplyTo)
	} else if origFrom != "" {
		fmt.Fprintf(&buf, "Reply-To: %s\r\n", origFrom)
	}
	if origMessageID != "" {
		fmt.Fprintf(&buf, "X-Original-Message-Id: %s\r\n", origMessageID)
	}

	// Source identification.
	fmt.Fprintf(&buf, "X-YaToGm-Source: %s\r\n", originalFrom)
	fmt.Fprintf(&buf, "X-Mailer: YaToGm/1.0\r\n")

	// MIME headers.
	if mimeVersion != "" {
		fmt.Fprintf(&buf, "MIME-Version: %s\r\n", mimeVersion)
	}
	if contentType != "" {
		fmt.Fprintf(&buf, "Content-Type: %s\r\n", contentType)
	}
	if contentTransferEncoding != "" {
		fmt.Fprintf(&buf, "Content-Transfer-Encoding: %s\r\n", contentTransferEncoding)
	}

	// Copy any remaining headers that we haven't already handled.
	handled := map[string]bool{
		"From": true, "To": true, "Subject": true, "Date": true,
		"Message-Id": true, "Cc": true, "Reply-To": true,
		"Content-Type": true, "Content-Transfer-Encoding": true,
		"Mime-Version": true,
	}
	for key, values := range msg.Header {
		if handled[key] {
			continue
		}
		for _, v := range values {
			fmt.Fprintf(&buf, "%s: %s\r\n", key, v)
		}
	}

	// End of headers.
	fmt.Fprintf(&buf, "\r\n")

	// Copy the body.
	body, err := readBody(msg.Body)
	if err != nil {
		return fmt.Errorf("reading message body: %w", err)
	}
	buf.Write(body)

	return s.sendBytes(buf.Bytes())
}

// sendRaw sends the raw email bytes directly when parsing fails.
func (s *Sender) sendRaw(rawEmail []byte, originalFrom string) error {
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "X-YaToGm-Source: %s\r\n", originalFrom)
	fmt.Fprintf(&buf, "X-YaToGm-Note: original message could not be parsed\r\n")
	buf.Write(rawEmail)
	return s.sendBytes(buf.Bytes())
}

// sendBytes sends the given email bytes via SMTP.
func (s *Sender) sendBytes(data []byte) error {
	addr := net.JoinHostPort(s.host, strconv.Itoa(s.port))

	auth := netsmtp.PlainAuth("", s.username, s.password, s.host)

	err := netsmtp.SendMail(addr, auth, s.to, []string{s.to}, data)
	if err != nil {
		return fmt.Errorf("smtp send: %w", err)
	}

	return nil
}

// readBody reads the entire body from a mail.Message.
func readBody(r interface{ Read([]byte) (int, error) }) ([]byte, error) {
	var buf bytes.Buffer
	_, err := buf.ReadFrom(r)
	return buf.Bytes(), err
}

// ExtractEmailAddress extracts the bare email address from a From header value
// like "John Doe <john@example.com>" or just "john@example.com".
func ExtractEmailAddress(from string) string {
	addr, err := mail.ParseAddress(from)
	if err != nil {
		// Fallback: try to extract from angle brackets.
		if i := strings.Index(from, "<"); i >= 0 {
			if j := strings.Index(from[i:], ">"); j >= 0 {
				return from[i+1 : i+j]
			}
		}
		return strings.TrimSpace(from)
	}
	return addr.Address
}
