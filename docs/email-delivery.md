# Email Delivery And Account Recovery

Kelmio delivers system email through a durable PostgreSQL outbox and a separate
worker. HTTP requests enqueue messages transactionally; they do not wait for an
SMTP provider.

## Local Email Stack

The default development stack includes Mailpit and the email worker:

```sh
docker compose up -d --build
make setup-db
```

- Mailpit inbox: `http://localhost:8025`
- Mailpit SMTP: `localhost:1025`
- Email worker metrics: `http://localhost:9091/metrics`

Mailpit stores development messages only. Do not use it as a durable archive.

## Delivery Flow

1. An application transaction writes an `email_outbox` row with template data.
2. The email worker claims pending rows with PostgreSQL row locking.
3. The worker renders the template and sends it through the configured SMTP
   client.
4. Successful rows become `sent`; temporary failures use the configured retry
   policy; invalid templates and exhausted retries become `failed`.

Outbox records store template data rather than raw rendered messages. Logs and
diagnostics omit passwords, reset/invite tokens, message bodies and provider
responses that may contain secrets.

## Supported Product Flows

### Password reset

The sign-in screen links to the password reset request flow. Requests return the
same response for known, unknown and inactive email addresses. A valid reset
token can be used once, expires after `PASSWORD_RESET_TTL`, and revokes all
existing sessions after the password changes.

The raw reset token exists only inside the queued and delivered reset URL. The
password-reset domain table stores only its SHA-256 hash.

### Team invites

Creating an invite queues an email and still returns a one-time copy-link
fallback. Workspace admins can resend pending invites after the cooldown. A
resend reuses the existing invite URL; it does not rotate the token.

Invite list responses expose only delivery status and timestamps. They never
return a reusable raw token.

## Configuration

Development defaults are defined in `.env.example`:

| Variable | Default | Purpose |
|---|---|---|
| `EMAIL_DELIVERY_ENABLED` | `true` | Enables SMTP delivery |
| `SMTP_HOST` | `mailpit` in Compose | SMTP server |
| `SMTP_PORT` | `1025` | SMTP port |
| `SMTP_USERNAME` | empty | Optional SMTP username |
| `SMTP_PASSWORD` | empty | Optional SMTP password |
| `SMTP_FROM_EMAIL` | `no-reply@kelmio.local` | Sender address |
| `SMTP_FROM_NAME` | `Kelmio` | Sender name |
| `SMTP_TLS_MODE` | `none` | `none`, `starttls` or `tls` |
| `EMAIL_WORKER_POLL_INTERVAL` | `10s` | Idle polling interval |
| `EMAIL_MAX_ATTEMPTS` | `5` | Terminal retry limit |
| `PASSWORD_RESET_TTL` | `30m` | Reset-token lifetime |

Production-like configuration may disable delivery. If it enables delivery,
SMTP host, sender, valid port and TLS mode are required. Store credentials only
in a private env file; never commit them.

## Diagnostics

Workspace admins can inspect read-only delivery health in the Team screen. The
panel shows outbox counts, oldest pending/processing timestamps and the ten most
recent terminal failures with masked recipients and sanitized errors.

The same summary is available from the terminal:

```sh
make email-diagnostics
```

The script uses the authenticated diagnostics endpoint when the backend is
available. Its database fallback returns only status counts, masked recipients
and sanitized errors. There is no generic retry endpoint; invite resend remains
the only user-triggered redelivery action.

## Troubleshooting

### A message remains pending

1. Confirm `email-worker` and Mailpit are running with `docker compose ps`.
2. Open `http://localhost:8025` and check whether the message was delivered.
3. Run `make email-diagnostics`.
4. Inspect worker logs with `docker compose logs email-worker` using the outbox
   ID, type, status and attempt count.

### Delivery retries after an SMTP outage

Run the isolated recovery smoke:

```sh
make smoke-email-delivery
```

It stops Mailpit, confirms an outbox retry, restarts Mailpit and waits for the
same message to be delivered. Cleanup always restores Mailpit.

### Metrics are unavailable

Confirm `METRICS_ENABLED=true`. Development allows metrics without a token.
When `METRICS_AUTH_TOKEN` is configured, send it as a bearer token and keep it
out of shell history and logs.

## Security Invariants

- Password reset requests do not disclose whether an account exists.
- Reset and invite tokens are never stored raw in their domain tables.
- SMTP credentials, cookies, CSRF values and email bodies are not logged.
- Metrics contain no email address, username, token or message content labels.
- Diagnostics are workspace-admin-only and expose sanitized data.
