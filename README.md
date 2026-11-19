# Connect

An Android-first app-to-app voice project with a Flutter app, Go backend and
portable C++ engine. The mobile app and backend are built and run separately.

Currently, the backend supports accounts and device authentication. The mobile
app has a local UI preview; account integration and calling are still pending.

## Repository structure

```text
mobile-app/       Flutter UI and Kotlin Android host
backend/          Go API, database migrations and backend tests
native-engine/    Portable C++ engine foundation
infrastructure/   Local PostgreSQL, Redis and TURN services
contracts/        Shared API definitions for backend and mobile
docs/            Architecture, setup and development guides
```

## Requirements

- **Backend:** Go 1.26.11, Python 3.10+, Docker with Compose v2 and Make.
- **Mobile:** Flutter 3.47.8, JDK 21, Android SDK and a device or emulator.
- **Native engine:** C++20 compiler and CMake.

See [toolchain setup](docs/development/toolchains.md) for details.

## Run locally

Run these commands from the repository root.

**Start the backend:**

```bash
make env infra-up
make backend-migrate
make backend-run
```

The local API runs at `http://127.0.0.1:8080`. Check `/healthz` and `/readyz`.
Account APIs require additional HTTPS configuration; see the
[backend README](backend/README.md).

**Run the Android app** in another terminal with a device or emulator connected:

```bash
make mobile-run
```

Explore welcome, profile setup, calls, Call ID entry and appearance settings.
Preview data resets on app restart; no account or call is created.

To build a debug APK, use `make mobile-apk`. The APK is written to
`mobile-app/build/app/outputs/flutter-apk/app-debug.apk`.

Stop the backend with `Ctrl+C`, then run `make infra-down` to stop local services.
The generated `.env` contains local credentials and must stay out of Git.

## Checks

```bash
make backend-check backend-integration
make mobile-check
make native-test
```

Each component has its own README and Makefile. Run `make help` for more commands.
