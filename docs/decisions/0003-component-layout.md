# 0003 — Independent components and shared boundaries

Date: 2026-11-18. Accepted through the reviewer's request for independent,
accessible components and a local commit. This changes source/build organization;
it does not authorize a later stage or change authentication/media policy.

## Decision

Use these top-level folders:

- `mobile-app/`: Flutter presentation, Kotlin Android host, resources, dependency
  lockfile, widget tests, SDK pin and APK build. Kotlin stays with the Android app
  it ships in. Continuous native work stays outside Flutter presentation.
- `backend/`: independent Go module with API, implemented domains and storage
  adapters, embedded migrations, backend scripts and integration fixtures.
- `native-engine/`: portable C++20 library, headers, tests, CMake, formatting rules
  and pinned formatter tooling; independent of Flutter, accounts and Android UI.
- `infrastructure/`: independently managed local dependency services, Compose,
  TURN template, environment generation and service smoke checks.
- `contracts/`: shared versioned API definitions that keep mobile and backend
  compatible; one authoritative copy outside either application's source.
- `docs/`: architecture, stage approvals, decisions, setup and validation records.

Each buildable component and the infrastructure folder owns a README and
Makefile. The root Makefile coordinates their commands. Existing `api-*` commands
remain aliases for `backend-*`. No empty future modules are added.

## Runtime and release boundaries

Flutter and Kotlin ship in one Android app. The native library is built
independently and will later be packaged into that app through a native bridge;
the bridge is not implemented yet. The Go API is a separate server executable
with its own deployment lifecycle and is never linked into the APK.

PostgreSQL is used by the backend. Redis remains optional and unconnected to the
API. TURN is independent relay infrastructure that the mobile/native client will
contact over the network in a later stage. Its configuration is not mobile app
source. No production deployment is introduced by this layout.

Contracts are specifications, not a separately running service. Backend changes
and mobile integration must follow their shared versioned definitions; do not
maintain separate copies in each component.

## Development ownership

The backend owns its disposable PostgreSQL integration fixture under
`backend/tests/integration/`, since it exists only to test that module. Each run
generates test credentials in memory, uses temporary storage, bypasses dotenv
loading, and needs no root development `.env`, Flutter, native library or TURN.

The ignored root `.env` remains local backend/dependency configuration. Root
`.build/`, `.cache/` and `.tools/` hold common ignored artifacts/caches/tooling,
not source modules. Backend binaries live in `.build/backend/`; native host builds
live in `.build/native-engine/`. The new native build directory avoids CMake
caches containing the old source path. Existing generated artifacts and local
development data are preserved.

Component toolchain pins stay with their owner. Open a component folder for
focused work, or the repository root for coordinated changes. Historical stage
notes use current paths in commands/links; their recorded validation dates remain
historical. Stage 2 native authentication acceptance remains outstanding.

## Validation

Executed locally on 2026-11-18:

- `make backend-check backend-integration` passed formatting/vet, race tests,
  executable builds and real PostgreSQL/verified-HTTPS integration scenarios.
- `make native-check CLANG_FORMAT=.tools/format/bin/clang-format` passed native
  formatting, CMake build and CTest.
- `make infra-config` passed. The TURN mount resolves to the preserved root
  fixture; temporary-directory environment generation verified correct paths,
  mode 0600 and preservation of existing configuration.
- Documentation links, contract YAML/local schema references, workflow YAML,
  Python/shell syntax and preservation of all original tracked source files were
  checked. Mobile and legacy command paths passed Make dry runs.

Flutter is missing on this machine, so widget/APK validation remains unavailable.
The native engine's earlier Android arm64 compile passed after its directory
move; the native path and source have not changed since that check. Remote CI,
production deployment and physical-device acceptance were not performed.

Generated artifacts, private local configuration and caches remain ignored. The
integration stack was removed after testing; development data was preserved.
