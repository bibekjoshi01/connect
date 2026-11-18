# Backend

Independent Go modular monolith. This folder owns the API, domain logic,
PostgreSQL adapters, migrations, tests and backend development scripts. Building
it requires no Flutter, Android SDK or native engine.

```text
cmd/api/              HTTP process and application composition
cmd/migrate/          Explicit database migration executable
internal/accounts/    Registration, device authentication, sessions and profile
  postgres/           Account-owned storage adapter
internal/config/      Validated runtime configuration
internal/database/    Bounded PostgreSQL connectivity and readiness
internal/httpserver/  HTTP transport and operational endpoints
internal/observability/  Privacy-safe operational metrics
migrations/           Versioned embedded transactional SQL
scripts/dev.py        Local run/migrate and isolated integration orchestration
tests/integration/    Real PostgreSQL/HTTPS tests and their disposable Compose file
go.mod, go.sum        This component's pinned dependencies
Makefile              This component's build/check/run commands
```

## Build and test

From this folder, using Go 1.26.11:

```bash
make build
make test
make lint
# All three: make check
make security  # Pinned govulncheck; also runs in CI
```

Executables are written to `../.build/backend/connect-api` and
`../.build/backend/connect-migrate`. The API is a separate server executable,
not part of the mobile APK.

With Docker Engine and Compose v2 running:

```bash
make integration
```

The integration command builds the API, generates private temporary test
credentials in memory, starts only the dedicated PostgreSQL test stack, runs
race-enabled tests, and removes that test stack afterward. It does not require
the root `.env`, start development services or touch development volumes.

## Run against the development database

First, from the repository root:

```bash
make env infra-up
```

Then from `backend/`:

```bash
make migrate
make run
```

`make migrate ARGS=status` reports the version; `ARGS=down` rolls back one version
only when its data-protection checks allow it. The local helper reads the ignored
root `.env` privately. A directly launched executable reads process environment
only and requires `DATABASE_URL`.

Default address: `http://127.0.0.1:8080`. `/healthz`, `/readyz` and `/metrics` are
operational endpoints. Account routes are opt-in and require explicit RP/origin
configuration and verified HTTPS. See [Stage 2 setup](../docs/development/stage-2-results.md)
and the [account API contract](../contracts/rest/accounts-v1.openapi.yaml).

## Boundaries

Account logic owns its storage interfaces; PostgreSQL implements them. Keep new
implemented domains in this module, without speculative empty modules. Backend
code must not depend on the mobile project or native media engine. API schemas
belong in `../contracts/` so both consumers share the same versioned interface.

Android authentication integration, recovery, signaling and calling remain
outside this backend increment. No production container/deployment/release
pipeline is implemented. See [deployment boundaries](../docs/architecture/deployment.md)
and [Stage 1 operational behavior](../docs/development/stage-1.md).

Shared API schemas are in `../contracts/`; server dependency configuration is in
`../infrastructure/`. Neither folder is part of the Go module. Manage local
services from the repository root with `make env infra-up` and `make infra-down`,
or use [the infrastructure project's commands](../infrastructure/README.md).

Registration has three requests: `/v1/accounts/registration/options`, then
`/finish` (202 with standard assertion options), then `/proof` (201 with the first
session and device handle). An attestation response alone creates no account or
session. The contract records the exact request/response shapes.

Before upgrading an existing development database, stop the API, run `make migrate`
to reach migration 3, then restart it. Migration 3 revokes pre-existing sessions;
enrolled credentials remain valid for normal signed authentication. Migrations 1
and 2 are unchanged. Production deployment is not part of this update.
