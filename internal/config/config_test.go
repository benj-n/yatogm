package config
package config

import (
	"os"
	"testing"
)

func TestLoadValidConfig(t *testing.T) {
	content := `
gmail:
  email: test@gmail.com
  app_password: gmail-secret
yahoo:
  - email: user1@yahoo.com
    app_password: yahoo-secret-1
  - email: user2@yahoo.com
    app_password: yahoo-secret-2
`
	tmpFile, err := os.CreateTemp(t.TempDir(), "config-*.yml")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tmpFile.WriteString(content); err != nil {
		t.Fatal(err)
	}
	tmpFile.Close()

	cfg, err := Load(tmpFile.Name())
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if cfg.Gmail.Email != "test@gmail.com" {
		t.Errorf("expected gmail email test@gmail.com, got %s", cfg.Gmail.Email)
	}
	if cfg.Gmail.SMTPHost != "smtp.gmail.com" {
		t.Errorf("expected default smtp host, got %s", cfg.Gmail.SMTPHost)
	}
	if cfg.Gmail.SMTPPort != 587 {
		t.Errorf("expected default smtp port 587, got %d", cfg.Gmail.SMTPPort)
	}
	if len(cfg.Yahoo) != 2 {
		t.Fatalf("expected 2 yahoo mailboxes, got %d", len(cfg.Yahoo))
	}
	if cfg.Yahoo[0].POP3Host != "pop.mail.yahoo.com" {
		t.Errorf("expected default pop3 host, got %s", cfg.Yahoo[0].POP3Host)
	}
	if cfg.Yahoo[0].POP3Port != 995 {
		t.Errorf("expected default pop3 port 995, got %d", cfg.Yahoo[0].POP3Port)
	}
}

func TestLoadMissingGmailEmail(t *testing.T) {
	content := `
gmail:
  app_password: secret
yahoo:
  - email: user@yahoo.com
    app_password: secret
`
	tmpFile, err := os.CreateTemp(t.TempDir(), "config-*.yml")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tmpFile.WriteString(content); err != nil {
		t.Fatal(err)
	}
	tmpFile.Close()

	_, err = Load(tmpFile.Name())
	if err == nil {
		t.Fatal("expected validation error for missing gmail.email")
	}
}

func TestLoadNoYahooMailboxes(t *testing.T) {
	content := `
gmail:
  email: test@gmail.com
  app_password: secret
yahoo: []
`
	tmpFile, err := os.CreateTemp(t.TempDir(), "config-*.yml")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tmpFile.WriteString(content); err != nil {
		t.Fatal(err)
	}
	tmpFile.Close()

	_, err = Load(tmpFile.Name())
	if err == nil {
		t.Fatal("expected validation error for empty yahoo list")
	}
}

func TestEnvOverrides(t *testing.T) {
	content := `
gmail:
  email: old@gmail.com
  app_password: old-secret
yahoo:
  - email: old@yahoo.com
    app_password: old-yahoo-secret
`
	tmpFile, err := os.CreateTemp(t.TempDir(), "config-*.yml")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tmpFile.WriteString(content); err != nil {
		t.Fatal(err)
	}
	tmpFile.Close()

	t.Setenv("YATOGM_GMAIL_EMAIL", "new@gmail.com")
	t.Setenv("YATOGM_GMAIL_APP_PASSWORD", "new-secret")
	t.Setenv("YATOGM_YAHOO_0_APP_PASSWORD", "new-yahoo-secret")

	cfg, err := Load(tmpFile.Name())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Gmail.Email != "new@gmail.com" {
		t.Errorf("expected env override for gmail email, got %s", cfg.Gmail.Email)
	}
	if cfg.Gmail.AppPassword != "new-secret" {
		t.Errorf("expected env override for gmail app password")
	}
	if cfg.Yahoo[0].AppPassword != "new-yahoo-secret" {
		t.Errorf("expected env override for yahoo app password")
	}
}

func TestLoadFileNotFound(t *testing.T) {
	_, err := Load("/nonexistent/config.yml")
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
}
