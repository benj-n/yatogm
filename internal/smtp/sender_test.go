package smtp

import (
	"testing"
)

func TestExtractEmailAddress(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"john@example.com", "john@example.com"},
		{"John Doe <john@example.com>", "john@example.com"},
		{"<john@example.com>", "john@example.com"},
		{"\"John Doe\" <john@example.com>", "john@example.com"},
	}

	for _, tt := range tests {
		got := ExtractEmailAddress(tt.input)
		if got != tt.want {
			t.Errorf("ExtractEmailAddress(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestFormatOriginalSender(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"john@example.com", `"john@example.com"`},
		{"John Doe <john@example.com>", `"John Doe via john@example.com"`},
		{"<john@example.com>", `"john@example.com"`},
		{`"John Doe" <john@example.com>`, `"John Doe via john@example.com"`},
	}

	for _, tt := range tests {
		got := formatOriginalSender(tt.input)
		if got != tt.want {
			t.Errorf("formatOriginalSender(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestNewSender(t *testing.T) {
	s := NewSender("smtp.gmail.com", 587, "user@gmail.com", "secret", "dest@gmail.com")
	if s.host != "smtp.gmail.com" {
		t.Errorf("expected host smtp.gmail.com, got %s", s.host)
	}
	if s.port != 587 {
		t.Errorf("expected port 587, got %d", s.port)
	}
	if s.username != "user@gmail.com" {
		t.Errorf("expected username user@gmail.com, got %s", s.username)
	}
	if s.to != "dest@gmail.com" {
		t.Errorf("expected to dest@gmail.com, got %s", s.to)
	}
}
