# Connect

An Android-first voice project organized as independent components in one
repository. Each component has its own source, README and build commands.

```text
mobile-app/       Flutter UI and Kotlin Android host; built as the Android app
backend/          Go API, account/auth logic, migrations and backend tests
native-engine/    Portable C++20 library; packaged into the app in a later stage
infrastructure/   Independent PostgreSQL, Redis and TURN service configuration
contracts/        Shared, versioned API definitions for backend and mobile
docs/            Architecture, decisions, setup and manual testing
```

## Develop a component

**Mobile app** — Flutter 3.47.8, JDK 21 and the Android SDK are required:

```bash
cd mobile-app
make run
# Other commands: make check, make apk
```

The app currently shows a development screen; backend integration and calling
are not implemented in the app. See [mobile-app/README.md](mobile-app/README.md).

**Backend** — Go 1.26.10; Docker is needed for isolated integration tests:

```bash
cd backend
make check
make integration
```

These commands need neither Flutter nor the native engine. Integration tests
create their own temporary database credentials and storage. See
[backend/README.md](backend/README.md) for API and authentication configuration.

To run the backend against the shared local development database, from the root:

```bash
make env infra-up
make backend-migrate
make backend-run
```

**Native engine** — C++20 and CMake, with no Flutter or Go dependency:

```bash
cd native-engine
make test
```

It currently exposes build metadata only; audio/network processing and app
packaging are later stages. See [native-engine/README.md](native-engine/README.md).

**Infrastructure** — independent dependency services:

```bash
cd infrastructure
make env
make up
# Stop services, preserving development data: make down
```

See [infrastructure/README.md](infrastructure/README.md). The ignored root `.env`
holds local server/dependency configuration; never commit credentials. Mobile
builds do not need database credentials.

## Build and check everything together

The root Makefile delegates to each component's Makefile:

```bash
make build       # Backend binaries, native static library, Android debug APK
make check       # Backend, native and mobile checks
make backend-integration  # Separate disposable PostgreSQL tests
```

Each command requires the corresponding [pinned toolchains](docs/development/toolchains.md).
Component commands also remain available from the root, such as `make mobile-apk`
and `make native-test`. Existing `api-*` commands remain compatibility aliases for
`backend-*` commands. No command deploys the project.

Outputs are `.build/backend/`, `.build/native-engine/` and
`mobile-app/build/app/outputs/flutter-apk/app-debug.apk`. Ignored `.build/`,
`.cache/` and `.tools/` contain artifacts, caches and local tool installations.

## How the components fit

Flutter presentation and Kotlin OS integration belong to the same Android app.
The portable native library will be packaged into that app through a later native
bridge. The Go backend is built and run as a separate server; it is never bundled
into the APK. `contracts/` defines their versioned API boundary. Infrastructure
provides independently managed services. The app will contact TURN over the
network when calling is implemented; TURN is not embedded in the app.

Account registration, WebAuthn device authentication and sessions are implemented
in the backend. Account routes require explicit RP configuration and verified
HTTPS; default loopback HTTP serves operational endpoints only. The app has no
login flow yet, and calling is not implemented.

See [Stage 2 results](docs/development/stage-2-results.md), the
[manual test checklist](docs/development/manual-testing.md) and the
[repository separation decision](docs/decisions/0003-component-layout.md).
