// Package config handles loading and validating the application configuration.
package config

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config is the top-level application configuration.
type Config struct {
	// Gmail holds the destination Gmail SMTP settings.
	Gmail GmailConfig `yaml:"gmail"`
	// Yahoo holds a list of Yahoo mailboxes to fetch from.
	Yahoo []YahooMailbox `yaml:"yahoo"`
	// StatePath is the file path for persisting fetched email UIDs.
	StatePath string `yaml:"state_path"`
	// LogLevel controls verbosity: "debug", "info", "warn", "error".
	LogLevel string `yaml:"log_level"`
}

// GmailConfig holds Gmail SMTP credentials and settings.
type GmailConfig struct {
	// Email is the Gmail address to deliver emails to.
	Email string `yaml:"email"`
	// AppPassword is the Gmail App Password for SMTP authentication.
	// Can be overridden by the YATOGM_GMAIL_APP_PASSWORD environment variable.
	AppPassword string `yaml:"app_password"`
	// SMTPHost is the Gmail SMTP server (default: smtp.gmail.com).
	SMTPHost string `yaml:"smtp_host"`
	// SMTPPort is the Gmail SMTP port (default: 587).
	SMTPPort int `yaml:"smtp_port"`
}

// YahooMailbox holds credentials for a single Yahoo mailbox.
type YahooMailbox struct {
	// Email is the Yahoo email address.
	Email string `yaml:"email"`
	// AppPassword is the Yahoo App Password for POP3 authentication.
	// Can be overridden by YATOGM_YAHOO_<INDEX>_APP_PASSWORD environment variable.
	AppPassword string `yaml:"app_password"`
	// POP3Host is the POP3 server (default: pop.mail.yahoo.com).
	POP3Host string `yaml:"pop3_host"`
	// POP3Port is the POP3S port (default: 995).
	POP3Port int `yaml:"pop3_port"`
}

// Load reads the configuration from the given YAML file path and applies
// environment variable overrides.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config file %s: %w", path, err)
	}

	cfg := &Config{
		StatePath: "/data/state.json",
		LogLevel:  "info",
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parsing config file %s: %w", path, err)
	}

	applyEnvOverrides(cfg)
	applyDefaults(cfg)

	if err := validate(cfg); err != nil {
		return nil, fmt.Errorf("config validation: %w", err)
	}

	return cfg, nil
}

// applyEnvOverrides replaces config values with environment variables when set.
func applyEnvOverrides(cfg *Config) {
	if v := os.Getenv("YATOGM_GMAIL_EMAIL"); v != "" {
		cfg.Gmail.Email = v
	}
	if v := os.Getenv("YATOGM_GMAIL_APP_PASSWORD"); v != "" {
		cfg.Gmail.AppPassword = v
	}
	if v := os.Getenv("YATOGM_STATE_PATH"); v != "" {
		cfg.StatePath = v
	}
	if v := os.Getenv("YATOGM_LOG_LEVEL"); v != "" {
		cfg.LogLevel = v
	}

	// Override individual Yahoo mailbox passwords: YATOGM_YAHOO_0_APP_PASSWORD, etc.
	for i := range cfg.Yahoo {
		key := fmt.Sprintf("YATOGM_YAHOO_%d_APP_PASSWORD", i)
		if v := os.Getenv(key); v != "" {
			cfg.Yahoo[i].AppPassword = v
		}
		emailKey := fmt.Sprintf("YATOGM_YAHOO_%d_EMAIL", i)
		if v := os.Getenv(emailKey); v != "" {
			cfg.Yahoo[i].Email = v
		}
	}
}

// applyDefaults sets default values for optional fields.
func applyDefaults(cfg *Config) {
	if cfg.Gmail.SMTPHost == "" {
		cfg.Gmail.SMTPHost = "smtp.gmail.com"
	}
	if cfg.Gmail.SMTPPort == 0 {
		cfg.Gmail.SMTPPort = 587
	}
	for i := range cfg.Yahoo {
		if cfg.Yahoo[i].POP3Host == "" {
			cfg.Yahoo[i].POP3Host = "pop.mail.yahoo.com"
		}
		if cfg.Yahoo[i].POP3Port == 0 {
			cfg.Yahoo[i].POP3Port = 995
		}
	}
}

// validate checks that all required configuration fields are present.
func validate(cfg *Config) error {
	var errs []string

	if cfg.Gmail.Email == "" {
		errs = append(errs, "gmail.email is required")
	}
	if cfg.Gmail.AppPassword == "" {
		errs = append(errs, "gmail.app_password is required (set via config or YATOGM_GMAIL_APP_PASSWORD)")
	}
	if len(cfg.Yahoo) == 0 {
		errs = append(errs, "at least one yahoo mailbox must be configured")
	}
	for i, y := range cfg.Yahoo {
		if y.Email == "" {
			errs = append(errs, fmt.Sprintf("yahoo[%d].email is required", i))
		}
		if y.AppPassword == "" {
			errs = append(errs, fmt.Sprintf("yahoo[%d].app_password is required (set via config or YATOGM_YAHOO_%d_APP_PASSWORD)", i, i))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("missing required fields:\n  - %s", strings.Join(errs, "\n  - "))
	}
	return nil
}
