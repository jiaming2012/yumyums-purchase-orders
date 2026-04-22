# Stack Research

**Domain:** Go + Postgres backend for Yumyums HQ PWA
**Researched:** 2026-04-15 (v2.0) | Updated: 2026-04-22 (v3.0 additions)
**Scope (v2.0):** NEW backend stack only. Existing frontend stack (vanilla JS, CSS variables, SortableJS, Chart.js, Playwright) is validated and not re-researched.
**Scope (v3.0):** NEW capabilities only — Zoho Cliq webhook alerts, email sending, cron scheduling, Notion CSV import. Everything else below is unchanged.
**Confidence:** HIGH

---

## Recommended Stack

### Core Backend

| Technology | Version | Purpose | Why |
|------------|---------|---------|-----|
| Go | 1.22+ | Backend language | Team choice; strong stdlib, easy deployment, fast compilation |
| chi | v5.2.5 | HTTP router | Lightweight, stdlib-compatible, middleware ecosystem, no framework lock-in |
| pgx/v5 | latest | Postgres driver | Native Postgres protocol, connection pooling, prepared statements, COPY support |
| sqlc | latest | Query codegen | Generates type-safe Go from SQL — schema stays in SQL, no ORM |
| goose | v3.27+ | DB migrations | SQL-file migrations, up/down support, embeddable in Go binary |
| crypto/rand + SHA-256 | stdlib | Auth tokens | Opaque bearer tokens, SHA-256 hash stored in sessions table |

### Infrastructure

| Technology | Purpose | Why |
|------------|---------|-----|
| Postgres 16 | Database | Already chosen in project constraints; LTS, resource-efficient |
| Caddy | Production reverse proxy | Automatic HTTPS via Let's Encrypt, simple config |
| Tailscale Serve | Dev deployment | `tailscale serve https:443 localhost:8080` — automatic HTTPS to tailnet, phone testing |
| embed.FS | Static file serving | Go serves both static PWA files and API from one binary — same-origin, no CORS |

### PWA Offline

| Technology | Purpose | Why |
|------------|---------|-----|
| IndexedDB | Client-side offline queue | Store pending submissions when offline |
| online event listener | Sync trigger | `window.addEventListener('online', replayQueue)` — works on iOS (Background Sync API does NOT) |
| Idempotency keys (UUID) | Dedup on sync | Each field answer gets a client UUID; server uses `ON CONFLICT (client_id) DO NOTHING` |

### What NOT to Use (v2.0)

| Technology | Reason |
|------------|--------|
| GORM / other ORMs | sqlc is simpler — SQL stays in SQL, no magic |
| JWT for sessions | Overkill for 1-5 users; opaque tokens are simpler and revocable |
| Background Sync API | Not supported on iOS Safari — must use online event fallback |
| Separate API origin | Breaks SW scope, requires CORS config, complicates offline queuing |
| Echo / Gin | chi is closer to stdlib, lighter dependency |

---

## Key Architectural Decisions (v2.0)

1. **Same-origin serving** — Go binary serves both `/api/v1/*` and static files via `embed.FS`. No CORS needed.
2. **httpOnly cookies for auth** — Not localStorage (XSS vulnerable), not IndexedDB (not sent automatically). `SameSite=Strict` works because same-origin.
3. **Presigned URLs for photos** — Photos uploaded direct to DigitalOcean Spaces; Go generates presigned PUT URLs. Server stays stateless on file data.
4. **SW fetch handler partitioned** — API calls (`/api/*`) use network-first strategy. Static files keep cache-first. Must be updated before first API call.

---

## v3.0 Stack Additions — Purchase Orders & Shopping Lists

### Scope

Four new capabilities are needed for v3.0. The existing foundation is untouched.

1. **Zoho Cliq webhook** — outbound HTTP POST to Cliq channel for alerts
2. **Email sending** — SMTP via Go library for alert fallback/alternative
3. **Cron scheduling** — cutoff enforcement, pre-cutoff reminders, out-of-stock alerts
4. **Notion CSV import** — one-time seed of ~100 inventory items from existing Notion DB

### New Libraries

| Library | Version | Purpose | Why Recommended |
|---------|---------|---------|-----------------|
| `net/http` (stdlib) | Go stdlib | Zoho Cliq webhook outbound POST | No dependency needed. Cliq webhook is a single outbound POST with JSON body. stdlib `http.Client` is production-ready for this. Adding a third-party HTTP client for one webhook call is over-engineering. |
| `wneessen/go-mail` | v0.7.2 | SMTP email sending | Actively maintained (v0.7.2 released Sep 29, 2025 — security patch). Idiomatic Go API. Tiny dependency footprint (mostly stdlib + golang.org/x). Handles STARTTLS, multiple auth methods (PLAIN, LOGIN, CRAM-MD5, SCRAM-SHA-256). The standard `net/smtp` is frozen and intentionally limited. |
| `go-co-op/gocron` | v2.21.1 | Cron-based scheduling | v2.21.1 released Apr 20, 2026 — actively maintained. Clean API with timezone support (critical for cutoff times). Job tagging enables dynamic rescheduling when admin changes cutoff time. Has test mocks. `robfig/cron` is unmaintained since 2020 with open panic bugs. |
| `encoding/csv` (stdlib) | Go stdlib | Notion CSV import | Notion exports standard UTF-8 CSV. stdlib csv.Reader handles it with zero dependencies. 100-row one-time seed doesn't need struct-tag mapping. |

### Installation

```bash
# Email sending
go get github.com/wneessen/go-mail@v0.7.2

# Scheduler
go get github.com/go-co-op/gocron/v2@v2.21.1

# Test mocks for scheduler (dev/test only)
go get github.com/go-co-op/gocron/mocks/v2@v2.21.1
```

No installation needed for `net/http`, `encoding/csv`, `encoding/json` — all stdlib.

### Integration Patterns

#### 1. Zoho Cliq Webhook (outbound HTTP POST)

**Mechanism:** Zoho Cliq exposes a channel-specific webhook URL authenticated via `zapikey`
query param (no OAuth flow). Owner generates token in Cliq: Settings → Bots & Tools →
Webhook Tokens (max 5 per user). Channel webhook URL obtained from Channel Info → Connectors.

**Endpoint format:**
```
POST https://cliq.zoho.com/api/v2/channelsbyname/{channel-unique-name}/message?zapikey={token}
Content-Type: application/json
Body: {"text": "your alert text"}
```

**Go implementation (no third-party library):**
```go
func postToCliq(ctx context.Context, webhookURL, message string) error {
    body, _ := json.Marshal(map[string]string{"text": message})
    req, err := http.NewRequestWithContext(ctx, http.MethodPost, webhookURL, bytes.NewReader(body))
    if err != nil {
        return err
    }
    req.Header.Set("Content-Type", "application/json")
    resp, err := http.DefaultClient.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()
    if resp.StatusCode >= 300 {
        return fmt.Errorf("cliq webhook returned %d", resp.StatusCode)
    }
    return nil
}
```

Store `webhookURL` (full URL including zapikey) in environment variable or Postgres user
preferences — NOT in source code.

#### 2. Email Sending (go-mail)

```go
import "github.com/wneessen/go-mail"

func sendEmail(ctx context.Context, to, subject, bodyText string) error {
    m := mail.NewMsg()
    m.From("alerts@yumyums.com")
    m.To(to)
    m.Subject(subject)
    m.SetBodyString(mail.TypeTextPlain, bodyText)

    c, err := mail.NewClient("smtp.mailgun.org",
        mail.WithPort(587),
        mail.WithSMTPAuth(mail.SMTPAuthPlain),
        mail.WithUsername(os.Getenv("SMTP_USER")),
        mail.WithPassword(os.Getenv("SMTP_PASS")),
        mail.WithTLSPolicy(mail.TLSMandatory),
    )
    if err != nil {
        return err
    }
    return c.DialAndSendWithContext(ctx, m)
}
```

Gmail variant: host `smtp.gmail.com:587`, auth with App Password (not account password).

#### 3. Scheduled Task Execution (gocron v2)

**Use cases:**
- Pre-cutoff reminder (e.g., "30 minutes until PO cutoff — submit your orders")
- Cutoff enforcement trigger (lock PO, send alert)
- Out-of-stock alert check (e.g., every morning at 8am)

```go
import "github.com/go-co-op/gocron/v2"

func startScheduler(alertSvc AlertService) error {
    loc, _ := time.LoadLocation("America/Los_Angeles") // owner's timezone from config
    s, err := gocron.NewScheduler(gocron.WithLocation(loc))
    if err != nil {
        return err
    }

    // Weekly cutoff reminder — e.g., every Sunday at 11:30am
    _, err = s.NewJob(
        gocron.CronJob("30 11 * * 0", false), // min hour day month weekday
        gocron.NewTask(func() {
            alertSvc.SendCutoffReminder(context.Background())
        }),
        gocron.WithTags("cutoff-reminder"),
    )
    if err != nil {
        return err
    }

    s.Start()
    return nil
}
```

**Dynamic rescheduling** (admin changes cutoff time):
```go
s.RemoveByTags("cutoff-reminder")
// re-add with new cron expression
```

#### 4. Notion CSV Import (encoding/csv)

**Notion export format:** Zip containing `DatabaseName.csv`, UTF-8, header row = property names.
Relation/formula columns export as plain text strings.

```go
import "encoding/csv"

func importNotionCSV(r io.Reader, db *sql.DB) error {
    reader := csv.NewReader(r)
    headers, err := reader.Read() // first row
    if err != nil {
        return err
    }
    idx := make(map[string]int)
    for i, h := range headers {
        idx[strings.ToLower(strings.TrimSpace(h))] = i
    }
    for {
        record, err := reader.Read()
        if err == io.EOF {
            break
        }
        if err != nil {
            return err
        }
        name := normalizeItemName(record[idx["name"]])
        category := record[idx["category"]]
        // upsert into items table via existing catalog upsert logic
    }
    return nil
}
```

**Delivery:** Admin-only endpoint `POST /api/v1/inventory/import-csv` with
`multipart/form-data`. One-time operation; return created/skipped counts. No scheduled job.

### Alternatives Considered (v3.0)

| Recommended | Alternative | When to Use Alternative |
|-------------|-------------|-------------------------|
| `net/http` stdlib for Cliq | `go-resty/resty` | Only if project already uses resty elsewhere. For one webhook, stdlib is simpler. |
| `wneessen/go-mail` | `net/smtp` (stdlib) | Never — `net/smtp` is frozen, no STARTTLS retry, Go team recommends third-party for production mail. |
| `wneessen/go-mail` | `gomail.v2` | Never — gomail.v2 is unmaintained since 2016, no security patches. |
| `go-co-op/gocron/v2` | `robfig/cron/v3` | robfig/cron unmaintained since 2020, open panic bugs on DST transitions, 50+ ignored PRs. |
| `go-co-op/gocron/v2` | `time.AfterFunc` / ticker | Only for single one-shot timers. Recurring cron expressions need a scheduler library. |
| `encoding/csv` stdlib | `gocarina/gocsv` | gocsv adds struct tag mapping — useful for complex CSVs. Overkill for 100-row one-time import. |
| Zoho Cliq webhook token | Zoho Cliq OAuth API | OAuth needed only for user-level actions. Channel webhook token is sufficient for system alerts — far simpler. |

### What NOT to Use (v3.0)

| Avoid | Why | Use Instead |
|-------|-----|-------------|
| `gomail.v2` | Unmaintained since 2016; no SCRAM-SHA auth; no security patches | `wneessen/go-mail` v0.7.2 |
| `robfig/cron/v3` | Unmaintained since 2020; panic bugs on DST; 50+ open PRs unaddressed | `go-co-op/gocron/v2` |
| Zoho Cliq OAuth 2.0 for alerts | Requires OAuth server, token refresh, Zoho app registration — massive complexity for a text message | Webhook token via `zapikey` query param |
| External HTTP client library for Cliq | No benefit over stdlib for a single outbound webhook | `net/http` stdlib |
| Third-party CSV library | 100-row one-time import doesn't need struct mapping | `encoding/csv` stdlib |
| Storing `zapikey` in source code | Credential leak risk | Environment variable or DB-stored user preference |

### Version Compatibility (v3.0)

| Package | Compatible With | Notes |
|---------|-----------------|-------|
| `go-co-op/gocron/v2@v2.21.1` | Go 1.21+ | v2 uses generics internally; requires Go 1.21 minimum |
| `wneessen/go-mail@v0.7.2` | Go 1.16+ | Works on Go 1.21+ used by existing backend |
| `encoding/csv` | All Go versions | Stdlib; no compatibility concerns |
| `net/http` | All Go versions | Stdlib; no compatibility concerns |

Verify existing Go version before adding gocron:
```bash
head -5 go.mod  # expect go 1.21 or later
```

---

## Sources (v3.0 research)

- [Zoho Cliq Webhook Tokens docs](https://www.zoho.com/cliq/help/platform/webhook-tokens.html) — endpoint format, zapikey auth, 5-token limit (MEDIUM — URL format confirmed, payload via community examples)
- [Zoho Cliq REST API v2](https://www.zoho.com/cliq/help/restapi/v2/) — channel message endpoint, OAuth scopes (MEDIUM — `channelsbyname` endpoint confirmed via multiple community sources)
- [wneessen/go-mail pkg.go.dev](https://pkg.go.dev/github.com/wneessen/go-mail) — v0.7.2 verified, API signatures (HIGH)
- [wneessen/go-mail GitHub releases](https://github.com/wneessen/go-mail/releases) — maintenance status, security release history (HIGH)
- [go-co-op/gocron/v2 pkg.go.dev](https://pkg.go.dev/github.com/go-co-op/gocron/v2) — v2.21.1 released Apr 20 2026, API signatures (HIGH)
- [encoding/csv pkg.go.dev](https://pkg.go.dev/encoding/csv) — stdlib, stable (HIGH)
- [Notion export help](https://www.notion.com/help/export-your-content) — CSV format, UTF-8 requirement, column behavior (HIGH)

---

*Stack research for: Yumyums HQ v2.0 + v3.0 Purchase Orders & Shopping Lists*
*Originally researched: 2026-04-15 | Updated: 2026-04-22*
