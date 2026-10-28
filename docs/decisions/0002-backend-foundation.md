# ADR 0002 — Stage 1 backend foundation

Status: implemented within the reviewer's authorization to begin the next stage.
No account/authentication or cryptographic protocol decisions are included.

Use pinned pgx v5.11.0 through its `database/sql` adapter, with Go's bounded
connection pool. Use pinned Goose v3.29.0 through its provider API for embedded
SQL migrations, PostgreSQL session advisory locking, and transactional execution.
This shares one database adapter and avoids an ORM or a separately installed CLI.

Migrations run through an explicit `cmd/migrate` command, never during API startup.
The first migration creates the application-owned `connect` namespace only;
Goose's version table remains `public.goose_db_version`. Identity tables are
deferred to Stage 2. Down migration uses `RESTRICT`, so a nonempty application
namespace prevents rollback instead of cascading through future user data.
Failed transactional migrations leave the prior version intact. A later deployed
schema needs its own forward/rollback and backup review; a down command is not
a substitute for that review.

The API waits for PostgreSQL only within `DB_STARTUP_TIMEOUT`. Each connection
attempt is bounded and retries are paced at 250 ms. Unavailable PostgreSQL makes
startup fail with a sanitized error. After startup, `/healthz` is process liveness;
`/readyz` performs a bounded, read-only query checking the migration version and
namespace. Database failure makes readiness return 503 while liveness remains 200;
subsequent probes can recover without restarting the API. Missing migrations are
not created by readiness probes.

The pool defaults to eight open/four idle connections, 30-minute maximum lifetime
and five-minute idle lifetime. Configurable pool sizes and operation deadlines
have validation limits. `DATABASE_URL` is required by the executable. Local Make
commands load ignored `.env` values in Python and derive a loopback URL without
printing it. Remote PostgreSQL must use verified TLS (`sslmode=verify-full`);
plaintext is limited to development/test loopback. URL overrides that could
change the connection host or weaken the explicit TLS policy are rejected.
Never log configuration structs or raw driver/migration errors.

Metrics use fixed names and no dynamic labels: HTTP request counts/time,
readiness checks/failures, and connection-pool gauges/wait count. Structured logs
cover lifecycle and sanitized failure categories. Operational endpoints must
remain private when ingress is introduced. No request headers, query strings,
account IDs, credentials or media are logged.

Redis is not integrated: this stage has no concrete need for a second state store.
Integration tests use a dedicated, disposable `connect_stage1_test` database on
loopback port 15432; development volumes are not mounted or reset.

Sources: [pgx](https://github.com/jackc/pgx/releases),
[Goose provider and locking](https://pressly.github.io/goose/documentation/provider/).
