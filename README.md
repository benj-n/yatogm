# YaToGm — Yahoo To Gmail

[![CI](https://github.com/benj-n/yatogm/actions/workflows/ci.yml/badge.svg)](https://github.com/benj-n/yatogm/actions/workflows/ci.yml)

A self-hosted Docker application that fetches emails from Yahoo mailboxes via POP3S and forwards them to a Gmail account via SMTP. Emails pass through Gmail's full filtering system (spam detection, labels, forwarding rules).

## Features

- **POP3S fetch** — Securely retrieves emails from multiple Yahoo mailboxes
- **SMTP forwarding** — Delivers to Gmail via standard SMTP (no API keys or Google Cloud needed)
- **Gmail filtering** — Forwarded emails go through Gmail's complete filtering pipeline
- **Deduplication** — Tracks fetched email UIDs to avoid processing the same email twice
- **Scheduled** — Runs on a configurable cron schedule (default: every 15 minutes)
- **Lightweight** — ~15MB Docker image based on Alpine Linux
- **Secure** — Non-root container, read-only filesystem, no-new-privileges, TLS enforced
- **Free** — No paid services, APIs, or subscriptions required

## Prerequisites

1. **Yahoo App Password** — For each Yahoo mailbox:
   - Enable [2-Step Verification](https://login.yahoo.com/myc/security) on your Yahoo account
   - Generate an [App Password](https://login.yahoo.com/myc/security/app-password) (select "Other App")
   - Enable POP3 in Yahoo Mail settings: Settings → More Settings → POP3/IMAP → Enable POP

2. **Gmail App Password** — For the destination Gmail account:
   - Enable [2-Step Verification](https://myaccount.google.com/signinoptions/two-step-verification) on your Google account
   - Generate an [App Password](https://myaccount.google.com/apppasswords) (select "Mail" and your device)

## Quick Start

### 1. Clone and configure

```bash
git clone https://github.com/benj-n/yatogm.git
cd yatogm

# Create configuration from template
cp config.example.yml config.yml
cp .env.example .env
```

### 2. Edit configuration

Edit `config.yml` with your email addresses:

```yaml
gmail:
  email: "your-gmail@gmail.com"

yahoo:
  - email: "your-yahoo@yahoo.com"
```

Edit `.env` with your app passwords:

```bash
YATOGM_GMAIL_APP_PASSWORD=your-gmail-app-password
YATOGM_YAHOO_0_APP_PASSWORD=your-yahoo-app-password
```

### 3. Run

```bash
docker compose up -d
```

Check logs:

```bash
docker compose logs -f
```

## Configuration

### Config File (`config.yml`)

| Field | Description | Default |
|-------|-------------|---------|
| `gmail.email` | Gmail address to forward to | (required) |
| `gmail.app_password` | Gmail App Password | (required, prefer env var) |
| `gmail.smtp_host` | Gmail SMTP server | `smtp.gmail.com` |
| `gmail.smtp_port` | Gmail SMTP port | `587` |
| `yahoo[].email` | Yahoo email address | (required) |
| `yahoo[].app_password` | Yahoo App Password | (required, prefer env var) |
| `yahoo[].pop3_host` | Yahoo POP3 server | `pop.mail.yahoo.com` |
| `yahoo[].pop3_port` | Yahoo POP3 port | `995` |
| `state_path` | Path to state file | `/data/state.json` |
| `log_level` | Log verbosity: debug, info, warn, error | `info` |

### Environment Variables

Environment variables override config file values:

| Variable | Description |
|----------|-------------|
| `YATOGM_GMAIL_EMAIL` | Gmail address |
| `YATOGM_GMAIL_APP_PASSWORD` | Gmail App Password |
| `YATOGM_YAHOO_0_APP_PASSWORD` | App password for first Yahoo mailbox |
| `YATOGM_YAHOO_1_APP_PASSWORD` | App password for second Yahoo mailbox |
| `YATOGM_YAHOO_N_APP_PASSWORD` | App password for Nth Yahoo mailbox |
| `YATOGM_STATE_PATH` | State file path |
| `YATOGM_LOG_LEVEL` | Log level |
| `TZ` | Timezone (e.g., `America/New_York`) |

### Cron Schedule

The default schedule runs every 15 minutes. To customize, create your own `crontab` file and mount it:

```bash
# crontab - run every 5 minutes
*/5 * * * * /usr/local/bin/yatogm -config /etc/yatogm/config.yml
```

Uncomment the crontab volume mount in `docker-compose.yml`:

```yaml
volumes:
  - ./crontab:/etc/yatogm/crontab:ro
```

## How It Works

1. **Fetch**: Connects to each Yahoo mailbox via POP3S (TLS on port 995)
2. **Deduplicate**: Checks each email's UID against previously processed UIDs
3. **Forward**: Sends new emails to Gmail via SMTP with STARTTLS (port 587)
4. **Track**: Saves the UID to the state file to prevent re-processing
5. **Preserve**: Original sender info is preserved in `X-Original-From`, `Resent-From`, and `Reply-To` headers

### Why SMTP Instead of Gmail API?

| | SMTP | Gmail API |
|---|---|---|
| Cost | Free | Free (with quotas) |
| Setup | App Password only | OAuth2 + Cloud project |
| Gmail Filters | Full support | Bypasses filters |
| Dependencies | None | Google Cloud SDK |
| Token management | None | Refresh tokens needed |

SMTP delivery goes through Gmail's standard intake pipeline, meaning spam filters, label rules, and forwarding rules all apply — exactly as if the email arrived naturally.

## Development

### Build locally

```bash
go build -o yatogm ./cmd/yatogm/
```

### Run tests

```bash
go test -v -race ./...
```

### Run with coverage

```bash
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html
```

### Build Docker image

```bash
docker build -t yatogm .
```

## Architecture

```
cmd/yatogm/main.go          Entry point, CLI flags, logging setup
internal/config/config.go    YAML + env var configuration loading
internal/pop3/client.go      POP3S client (TLS, UIDL, RETR)
internal/smtp/sender.go      SMTP forwarder with header rewriting
internal/state/tracker.go    JSON-based UID deduplication tracker
internal/worker/worker.go    Orchestration: fetch → forward → track
```

## Security

See [SECURITY.md](SECURITY.md) for security practices and responsible disclosure information.

**Key security measures:**
- All connections use TLS (POP3S + SMTP STARTTLS)
- Container runs as non-root user (UID 1000)
- Read-only root filesystem
- `no-new-privileges` security option
- Secrets via environment variables (never in image)
- State file written with 0600 permissions
- Atomic state file writes (temp + rename)
- CI includes `govulncheck` security scanning

## License

MIT
