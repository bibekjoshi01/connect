# Connect

Android-first, privacy-first app-to-app voice project. **Stage 0 only.** This
repository contains a launchable Flutter development screen, a Go process
liveness endpoint, a portable C++20 library build, and local infrastructure.
It does not implement accounts, authentication, calling, audio, NAT traversal or
E2EE. No WebRTC engine is used. Human review is required before Stage 1.

## App versus backend

These are independent projects in one repository, with separate build artifacts
and release lifecycles:

- **Android app:** [`apps/mobile/`](apps/mobile/README.md). Flutter UI and Kotlin
  Android host; installed on the user's phone as an APK (later distributed through
  Google Play using a signed app bundle). Entry point: `lib/main.dart`.
- **Backend service:** [`services/api/`](services/api/README.md). Go executable
  running on a server; future account and signaling control plane. Entry point:
  `cmd/api/main.go`. It currently exposes process liveness only.
- **Native app library:** `engine/rtc/`. C++ code intended to be packaged inside
  the Android app in a later stage; it is not another backend service.
- **Server infrastructure:** `infra/`. Local PostgreSQL, optional Redis and coturn
  fixtures; production deployment configuration is not implemented.
- **Shared definitions:** `contracts/`. Versioned API/protocol definitions shared
  by app and backend development; this directory is not deployed as a service.

The app can be checked without starting the backend. The backend can be built and
tested without Flutter or the RTC engine. App and server secrets/configuration
must remain separate. A shared repository keeps coordinated contract changes
reviewable; it does not require deploying everything together.

See [deployment boundaries and proposed release flow](docs/architecture/deployment.md).

## Repository

```text
connect/
├── apps/mobile/                 Flutter app and generated Kotlin Android host
│   ├── lib/app/connect_app.dart
│   ├── lib/main.dart
│   ├── android/                 Gradle wrapper, manifests, MainActivity, resources
│   ├── test/app_test.dart
│   ├── pubspec.yaml
│   └── pubspec.lock
├── engine/rtc/
│   ├── include/connect/rtc/build_info.hpp
│   ├── src/build_info.cpp
│   ├── tests/build_smoke.cpp
│   └── CMakeLists.txt
├── services/api/
│   ├── cmd/api/main.go
│   ├── internal/config/         Validated listener configuration and tests
│   ├── internal/httpserver/     Liveness HTTP transport and tests
│   └── go.mod
├── contracts/rest/health-v1.openapi.yaml
├── infra/
│   ├── compose/compose.yaml
│   ├── coturn/                  Local-only template and deployment distinction
│   ├── scripts/                 Secret generation and infrastructure checks
│   └── tooling/requirements.txt
├── docs/
│   ├── architecture/overview.md
│   ├── decisions/0001-stage-0-scope.md
│   ├── development/             Tools, stage plan, validation and full file tree
│   ├── protocols/README.md
│   └── security/review-gates.md
├── .github/workflows/validate.yml
├── .env.example
├── .flutter-version
├── AGENTS.md
└── Makefile
```

Full source inventory: [repository-tree.txt](docs/development/repository-tree.txt).
Build artifacts, SDK caches, local credentials and database volumes are excluded.
Directories for future domains, iOS, migrations, signaling and media are added only
when their stage has real content.

## Tooling and setup

Use Flutter **3.47.8** / bundled Dart **3.13.7**, Go **1.26.10**, CMake >=3.22,
a C++20 compiler, Python >=3.10, Bash, Git, Docker Engine and Compose v2+.
For Android builds, install JDK **21** and Android SDK API **36**, build-tools
**36.0.0**, and NDK **28.2.13676358**. See
[toolchain details](docs/development/toolchains.md) and
[official Android setup](https://docs.flutter.dev/platform-integration/android/setup).

One reproducible Flutter installation option (choose your own SDK directory):

```bash
git clone --depth 1 --branch 3.47.8 https://github.com/flutter/flutter.git "$HOME/development/flutter-3.47.8"
export PATH="$HOME/development/flutter-3.47.8/bin:$PATH"
flutter --version
dart --version
go version
docker compose version
```

From this repository root, prepare formatting and local environment files:

```bash
python3 -m venv .tools/format
.tools/format/bin/python -m pip install -r infra/tooling/requirements.txt
make env
make mobile-deps
```

`make env` creates ignored `.env` credentials with mode 0600 without overwriting
an existing file. It renders an ignored coturn fixture. `.env.example` has no real
secrets. If you copy it manually, replace every password; TURN's local password
must have at least 32 lowercase hex characters. Keep `.env` in simple `KEY=value`
format (no shell expansions or quotes). Local files are not production config.

The API currently reads only `API_LISTEN_ADDR` from its **process environment**;
it does not automatically load `.env`. Default is `127.0.0.1:8080`. The `.env`
file configures Compose. Do not put production secrets in the mobile app.

## Build and test

All commands below run from the repository root:

```bash
# Go: formatting, vet, race tests and executable
make api-lint api-test api-build
make api-run
# In another terminal:
curl --fail http://127.0.0.1:8080/healthz
# {"status":"ok"}; Ctrl-C stops the API gracefully.
```

```bash
# Native: warnings as errors, host library and CTest smoke test
make native-format-check native-test CLANG_FORMAT=.tools/format/bin/clang-format
# CMake-only equivalents:
cmake -S engine/rtc -B .build/rtc -DCMAKE_BUILD_TYPE=Debug -DBUILD_TESTING=ON
cmake --build .build/rtc --parallel
ctest --test-dir .build/rtc --output-on-failure
```

```bash
# Flutter: enforced lockfile, Dart format check, analyzer and widget test
make mobile-check
# Inspect Android prerequisites and review licenses interactively:
flutter doctor -v
flutter doctor --android-licenses
# With sdkmanager on PATH and ANDROID_HOME pointing to the installed SDK:
sdkmanager 'platforms;android-36' 'build-tools;36.0.0' 'ndk;28.2.13676358'
make mobile-apk
# Debug APK: apps/mobile/build/app/outputs/flutter-apk/app-debug.apk
cd apps/mobile
flutter run -d <android-device-id>
```

For an independent Android arm64 native build (no Flutter integration yet):

```bash
cmake -S engine/rtc -B .build/rtc-android \
  -DCMAKE_TOOLCHAIN_FILE="$ANDROID_HOME/ndk/28.2.13676358/build/cmake/android.toolchain.cmake" \
  -DANDROID_ABI=arm64-v8a -DANDROID_PLATFORM=android-24 -DBUILD_TESTING=OFF
cmake --build .build/rtc-android --parallel
```

```bash
# Entire host validation suite; no Android SDK needed for widget tests
make check CLANG_FORMAT=.tools/format/bin/clang-format
# Apply Go, Dart and native formatting
make format CLANG_FORMAT=.tools/format/bin/clang-format
```

Go's `toolchain` directive downloads the pinned patched toolchain if needed. First
Flutter/Go/Gradle builds need network access for official SDK artifacts/packages.
Compiler output goes to ignored `.build/`, `.cache/` and Flutter build paths.
Release signing is deliberately unconfigured; only debug APKs are part of Stage 0.

## Local services

```bash
make env
make infra-config              # Validate all profiles without printing credentials
make infra-up                 # PostgreSQL only, wait for health
make infra-up-cache           # PostgreSQL + optional Redis, wait for health
make infra-check-cache        # SQL query + authenticated Redis PING
make infra-up-turn            # Optional local-only coturn fixture
make infra-check-turn         # STUN + 4/4 relayed packets + bad-password rejection
make infra-down               # Stop all profiles; preserve PostgreSQL volume
```

All host ports bind to **127.0.0.1**: PostgreSQL 5432, Redis 6379, coturn 3478
TCP/UDP and relay UDP 49160–49169. PostgreSQL data persists in the project-named
Docker volume; Redis is intentionally ephemeral. Password changes do not change
an already initialized PostgreSQL user's password: alter it explicitly in the
database or reset only disposable data. No destructive reset target is provided.

The API does not connect to PostgreSQL or Redis yet. `/healthz` proves process
liveness only. coturn relay validation is inside its container namespace; this
configuration cannot prove physical-device connectivity, external address mapping
or real NAT traversal. Read [local versus public TURN](infra/coturn/README.md).
Development HTTP and TURN fixtures do not provide production TLS ingress.

## Validation and review

GitHub Actions validates Go, native C++, Flutter/debug APK, Android native
cross-compilation and local infrastructure in separate jobs. Actions are pinned
to commits and have read-only repository permissions. Workflows have not run on
GitHub until the repository is pushed; there is no configured remote.

See [actual local results and limitations](docs/development/stage-0-results.md).
This machine passed host Go/native/Flutter and local service checks; the local
APK attempt is blocked by the missing Android SDK. No device audio, calling or
E2EE behavior has been tested or claimed.

Review [architecture and rationale](docs/architecture/overview.md),
[Stage 0 decision](docs/decisions/0001-stage-0-scope.md),
[security approval gates](docs/security/review-gates.md), and
[recommended Stage 1 plan](docs/development/stages.md) before proceeding.
The Android package/minimum version and future crypto/identity decisions remain
pending review. No paid services, external project accounts, remote repository,
Firebase configuration, deployment or production secrets were provisioned.
