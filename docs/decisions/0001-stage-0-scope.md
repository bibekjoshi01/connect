# ADR 0001 — Stage 0 foundation

Status: accepted for the bootstrap following Stage 0 review and authorization to
begin Stage 1. Release identifiers and security-specific proposals remain separate
review decisions.

Use the requested monorepo and stack with only real bootstrap components. Generate
an Android-only Flutter project, compile a portable C++20 library, serve a Go
liveness endpoint, and provide isolated local PostgreSQL/optional Redis/coturn.
Do not create placeholder account/media modules or empty iOS scaffolding.

Use standard-library Go checks, Flutter's analyzer/test runner, CTest and compiler
warnings, clang-format, and separate CI jobs. Pin SDKs, package locks and container
digests. Add no production application dependencies at this stage.

Keep PostgreSQL mandatory for local infra startup; Redis and TURN opt-in profiles
avoid consuming resources for services not used by the Stage 0 API. Compose is
development-only, with host bindings restricted to IPv4 loopback. No TLS ingress
or public TURN deployment is implied.

Approval requested after validation: stack/boundaries, toolchain baseline,
provisional Android identifier/minimum version, and Stage 1 scope. Security
protocol proposals remain separately pending approval.
