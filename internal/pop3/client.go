// Package pop3 implements a POP3S client for fetching emails from Yahoo Mail.
package pop3

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"time"
)

// Message represents a fetched email message.
type Message struct {
	// UID is the unique identifier for this message on the POP3 server.
	UID string
	// Raw is the complete raw email (headers + body) as bytes.
	Raw []byte
}

// Client is a POP3S client that connects over TLS.
type Client struct {
	conn   net.Conn
	reader *bufio.Reader
}

// Dial connects to a POP3S server and returns a Client.
func Dial(host string, port int, timeout time.Duration) (*Client, error) {
	addr := net.JoinHostPort(host, strconv.Itoa(port))

	dialer := &net.Dialer{Timeout: timeout}
	tlsConn, err := tls.DialWithDialer(dialer, "tcp", addr, &tls.Config{
		MinVersion: tls.VersionTLS12,
	})
	if err != nil {
		return nil, fmt.Errorf("pop3 dial %s: %w", addr, err)
	}

	c := &Client{
		conn:   tlsConn,
		reader: bufio.NewReader(tlsConn),
	}

	// Read the server greeting.
	if _, err := c.readResponse(); err != nil {
		tlsConn.Close()
		return nil, fmt.Errorf("pop3 greeting: %w", err)
	}

	return c, nil
}

// Login authenticates with the POP3 server using USER/PASS.
func (c *Client) Login(user, pass string) error {
	if _, err := c.command("USER " + user); err != nil {
		return fmt.Errorf("pop3 USER: %w", err)
	}
	if _, err := c.command("PASS " + pass); err != nil {
		return fmt.Errorf("pop3 PASS: %w", err)
	}
	return nil
}

// UIDList returns a map of message number to UID for all messages.
func (c *Client) UIDList() (map[int]string, error) {
	if _, err := c.command("UIDL"); err != nil {
		return nil, fmt.Errorf("pop3 UIDL: %w", err)
	}

	result := make(map[int]string)
	for {
		line, err := c.reader.ReadString('\n')
		if err != nil {
			return nil, fmt.Errorf("pop3 UIDL read: %w", err)
		}
		line = strings.TrimRight(line, "\r\n")
		if line == "." {
			break
		}
		parts := strings.SplitN(line, " ", 2)
		if len(parts) != 2 {
			continue
		}
		num, err := strconv.Atoi(parts[0])
		if err != nil {
			continue
		}
		result[num] = parts[1]
	}

	return result, nil
}

// Retrieve fetches the full message content for the given message number.
func (c *Client) Retrieve(msgNum int) ([]byte, error) {
	if _, err := c.command(fmt.Sprintf("RETR %d", msgNum)); err != nil {
		return nil, fmt.Errorf("pop3 RETR %d: %w", msgNum, err)
	}

	var buf strings.Builder
	for {
		line, err := c.reader.ReadString('\n')
		if err != nil {
			return nil, fmt.Errorf("pop3 RETR %d read: %w", msgNum, err)
		}
		// Dot-stuffing: a line with just "." signals end of message.
		trimmed := strings.TrimRight(line, "\r\n")
		if trimmed == "." {
			break
		}
		// Remove dot-stuffing (leading dot on lines starting with "..")
		if strings.HasPrefix(trimmed, "..") {
			line = line[1:]
		}
		buf.WriteString(line)
	}

	return []byte(buf.String()), nil
}

// Delete marks the given message for deletion on the server.
func (c *Client) Delete(msgNum int) error {
	if _, err := c.command(fmt.Sprintf("DELE %d", msgNum)); err != nil {
		return fmt.Errorf("pop3 DELE %d: %w", msgNum, err)
	}
	return nil
}

// Quit sends the QUIT command and closes the connection.
// Deleted messages are only removed after a successful QUIT.
func (c *Client) Quit() error {
	_, _ = c.command("QUIT")
	return c.conn.Close()
}

// Close closes the connection without sending QUIT.
func (c *Client) Close() error {
	return c.conn.Close()
}

// command sends a POP3 command and reads the single-line response.
func (c *Client) command(cmd string) (string, error) {
	if err := c.conn.SetDeadline(time.Now().Add(30 * time.Second)); err != nil {
		return "", err
	}

	if _, err := fmt.Fprintf(c.conn, "%s\r\n", cmd); err != nil {
		return "", fmt.Errorf("sending command: %w", err)
	}

	return c.readResponse()
}

// readResponse reads a single-line POP3 response and checks for +OK or -ERR.
func (c *Client) readResponse() (string, error) {
	line, err := c.reader.ReadString('\n')
	if err != nil {
		if err == io.EOF {
			return "", fmt.Errorf("server closed connection")
		}
		return "", err
	}

	line = strings.TrimRight(line, "\r\n")

	if strings.HasPrefix(line, "+OK") {
		return line, nil
	}
	if strings.HasPrefix(line, "-ERR") {
		return "", fmt.Errorf("server error: %s", line)
	}

	return line, nil
}
