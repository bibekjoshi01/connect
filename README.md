# Connect

An Android-first, privacy-focused app-to-app voice project.

Currently includes a Flutter development screen and a Go/PostgreSQL backend
foundation. Accounts and calling are not implemented yet.

## Frontend — Android app

[`apps/mobile/`](apps/mobile/) contains the Flutter UI and Kotlin Android host.
It runs independently; backend integration is not implemented yet.

Requires Flutter **3.47.8**, JDK **21**, and an Android SDK with an emulator or
connected device. See [toolchain setup](docs/development/toolchains.md).

From the repository root:

```bash
make mobile-deps
cd apps/mobile
flutter run
```

To build a debug APK, run `make mobile-apk` from the repository root.
Output: `apps/mobile/build/app/outputs/flutter-apk/app-debug.apk`.

## Backend — API

[`services/api/`](services/api/) contains the Go API and PostgreSQL migrations.
It builds and runs separately from the app.

Requires Go **1.26.10**, Python **3.10+**, and Docker with Compose v2.

From the repository root:

```bash
make env
make infra-up
make api-migrate
make api-run
```

API: `http://127.0.0.1:8080`. Check `/healthz` for liveness and `/readyz` for
database readiness. Local configuration is generated in the ignored `.env` file.

Stop the API with `Ctrl+C` and PostgreSQL with `make infra-down`.
