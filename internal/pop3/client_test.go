package pop3

import (
	"bufio"
	"fmt"
	"net"
	"strings"
	"testing"
	"time"
)

// mockServer creates a simple POP3 server for testing.
func mockServer(t *testing.T, handler func(conn net.Conn)) net.Listener {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}

	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		handler(conn)
	}()

	return ln
}

// newTestClient creates a Client connected to a plain TCP server (no TLS) for testing.
func newTestClient(t *testing.T, conn net.Conn) *Client {
	t.Helper()
	c := &Client{
		conn:   conn,
		reader: bufio.NewReader(conn),
	}
	// Read greeting
	_, err := c.readResponse()
	if err != nil {
		t.Fatalf("failed to read greeting: %v", err)
	}
	return c
}

func TestClientLogin(t *testing.T) {
	ln := mockServer(t, func(conn net.Conn) {
		fmt.Fprintf(conn, "+OK POP3 server ready\r\n")
		scanner := bufio.NewScanner(conn)
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, "USER ") {
				fmt.Fprintf(conn, "+OK\r\n")
			} else if strings.HasPrefix(line, "PASS ") {
				fmt.Fprintf(conn, "+OK logged in\r\n")
			} else if line == "QUIT" {
				fmt.Fprintf(conn, "+OK bye\r\n")
				return
			}
		}
	})
	defer ln.Close()

	conn, err := net.DialTimeout("tcp", ln.Addr().String(), 2*time.Second)
	if err != nil {
		t.Fatal(err)
	}

	client := newTestClient(t, conn)
	defer client.Close()

	if err := client.Login("user@yahoo.com", "secret"); err != nil {
		t.Fatalf("Login failed: %v", err)
	}
}

func TestClientLoginFail(t *testing.T) {
	ln := mockServer(t, func(conn net.Conn) {
		fmt.Fprintf(conn, "+OK POP3 server ready\r\n")
		scanner := bufio.NewScanner(conn)
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, "USER ") {
				fmt.Fprintf(conn, "+OK\r\n")
			} else if strings.HasPrefix(line, "PASS ") {
				fmt.Fprintf(conn, "-ERR authentication failed\r\n")
				return
			}
		}
	})
	defer ln.Close()

	conn, err := net.DialTimeout("tcp", ln.Addr().String(), 2*time.Second)
	if err != nil {
		t.Fatal(err)
	}

	client := newTestClient(t, conn)
	defer client.Close()

	err = client.Login("user@yahoo.com", "wrong")
	if err == nil {
		t.Fatal("expected login to fail")
	}
}

func TestClientUIDList(t *testing.T) {
	ln := mockServer(t, func(conn net.Conn) {
		fmt.Fprintf(conn, "+OK POP3 server ready\r\n")
		scanner := bufio.NewScanner(conn)
		for scanner.Scan() {
			line := scanner.Text()
			if line == "UIDL" {
				fmt.Fprintf(conn, "+OK\r\n")
				fmt.Fprintf(conn, "1 abc123\r\n")
				fmt.Fprintf(conn, "2 def456\r\n")
				fmt.Fprintf(conn, "3 ghi789\r\n")
				fmt.Fprintf(conn, ".\r\n")
			} else if line == "QUIT" {
				fmt.Fprintf(conn, "+OK bye\r\n")
				return
			}
		}
	})
	defer ln.Close()

	conn, err := net.DialTimeout("tcp", ln.Addr().String(), 2*time.Second)
	if err != nil {
		t.Fatal(err)
	}

	client := newTestClient(t, conn)
	defer client.Close()

	uids, err := client.UIDList()
	if err != nil {
		t.Fatalf("UIDList failed: %v", err)
	}

	if len(uids) != 3 {
		t.Fatalf("expected 3 UIDs, got %d", len(uids))
	}
	if uids[1] != "abc123" {
		t.Errorf("expected uid abc123 for msg 1, got %s", uids[1])
	}
	if uids[2] != "def456" {
		t.Errorf("expected uid def456 for msg 2, got %s", uids[2])
	}
}

func TestClientRetrieve(t *testing.T) {
	ln := mockServer(t, func(conn net.Conn) {
		fmt.Fprintf(conn, "+OK POP3 server ready\r\n")
		scanner := bufio.NewScanner(conn)
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, "RETR ") {
				fmt.Fprintf(conn, "+OK\r\n")
				fmt.Fprintf(conn, "From: sender@example.com\r\n")
				fmt.Fprintf(conn, "To: receiver@example.com\r\n")
				fmt.Fprintf(conn, "Subject: Test\r\n")
				fmt.Fprintf(conn, "\r\n")
				fmt.Fprintf(conn, "Hello World\r\n")
				fmt.Fprintf(conn, ".\r\n")
			} else if line == "QUIT" {
				fmt.Fprintf(conn, "+OK bye\r\n")
				return
			}
		}
	})
	defer ln.Close()

	conn, err := net.DialTimeout("tcp", ln.Addr().String(), 2*time.Second)
	if err != nil {
		t.Fatal(err)
	}

	client := newTestClient(t, conn)
	defer client.Close()

	raw, err := client.Retrieve(1)
	if err != nil {
		t.Fatalf("Retrieve failed: %v", err)
	}

	body := string(raw)
	if !strings.Contains(body, "Subject: Test") {
		t.Errorf("expected Subject header in body, got: %s", body)
	}
	if !strings.Contains(body, "Hello World") {
		t.Errorf("expected body text, got: %s", body)
	}
}

func TestClientDotStuffing(t *testing.T) {
	ln := mockServer(t, func(conn net.Conn) {
		fmt.Fprintf(conn, "+OK POP3 server ready\r\n")
		scanner := bufio.NewScanner(conn)
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, "RETR ") {
				fmt.Fprintf(conn, "+OK\r\n")
				fmt.Fprintf(conn, "Subject: Dots\r\n")
				fmt.Fprintf(conn, "\r\n")
				// Dot-stuffed line (should become single dot)
				fmt.Fprintf(conn, "..leading dot\r\n")
				fmt.Fprintf(conn, "normal line\r\n")
				fmt.Fprintf(conn, ".\r\n")
			} else if line == "QUIT" {
				fmt.Fprintf(conn, "+OK bye\r\n")
				return
			}
		}
	})
	defer ln.Close()

	conn, err := net.DialTimeout("tcp", ln.Addr().String(), 2*time.Second)
	if err != nil {
		t.Fatal(err)
	}

	client := newTestClient(t, conn)
	defer client.Close()

	raw, err := client.Retrieve(1)
	if err != nil {
		t.Fatalf("Retrieve failed: %v", err)
	}

	body := string(raw)
	if !strings.Contains(body, ".leading dot") {
		t.Errorf("expected dot-unstuffed line, got: %s", body)
	}
	if strings.Contains(body, "..leading dot") {
		t.Errorf("dot-stuffing was not removed: %s", body)
	}
}
