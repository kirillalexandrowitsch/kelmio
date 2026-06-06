# V3 Local Production QA

This guide validates the V3 production hardening without changing application data. Deployment and update instructions remain separate from this QA flow.

## Baseline Checks

Run the production configuration and Compose checks from the repository root:

```sh
make prod-config-check
make prod-compose-check
```

`make prod-config-check` confirms that unsafe production backend settings are rejected.

`make prod-compose-check` validates `docker-compose.prod.yml` with the safe placeholder values from `deploy/production.env.example` and validates the Caddyfile.

## Localhost Hardening Smoke

Start and prepare the development stack:

```sh
make dev
make setup-db
```

Run the production-sensitive smoke against the direct localhost backend:

```sh
make smoke-production
```

The smoke checks:

- health and runtime version metadata;
- generated, preserved, and replaced `X-Request-ID` values;
- backend security headers;
- trusted and untrusted CORS preflight behavior;
- session cookie flags;
- missing, invalid, and valid CSRF tokens;
- the `1 MiB` request body limit;
- login rate limiting using a unique nonexistent login.

The localhost backend intentionally uses an insecure session cookie and does not emit HSTS because traffic is plain HTTP.

## HTTPS Production Stack QA

After starting a real production stack with a valid domain and TLS certificate, run:

```sh
API_BASE_URL=https://tasks.example.com \
TRUSTED_ORIGIN=https://tasks.example.com \
ADMIN_LOGIN=admin \
ADMIN_PASSWORD='<production-admin-password>' \
RATE_LIMIT_LOGIN_PER_MINUTE=10 \
EXPECT_SECURE_COOKIE=true \
EXPECT_HSTS=true \
make smoke-production
```

This mode additionally requires the session cookie to include `Secure` and requires the reverse proxy to emit `Strict-Transport-Security`.

Use the real configured `RATE_LIMIT_LOGIN_PER_MINUTE` value when it differs from the default.

## Full V3 Regression

With the localhost stack running:

```sh
make setup-db
make smoke-production
make smoke-api
make frontend-e2e
make verify
GOCACHE=/private/tmp/team-task-tracker-gocache make backend-integration-test
git diff --check
```

The browser suite includes the V1/V2 regression flows and the V3 invite create, accept, login, member visibility, and revoke scenarios.

## Safety Notes

- `make smoke-production` does not create projects, issues, members, invites, labels, saved filters, or notifications.
- It creates one temporary authenticated session and deletes it through a valid CSRF-protected logout.
- It consumes login attempts only for a unique nonexistent login identifier.
- The HTTPS mode must target an instance where using the configured admin account for QA is acceptable.
