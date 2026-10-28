# Stage 1 — backend foundation

Scope authorized: backend configuration, PostgreSQL pooling, explicit migrations,
readiness, operational metrics and tests. Account/authentication/calling remain
out of scope. No hosting, remote repository or paid infrastructure is provisioned.

## Local commands

From the repository root:

```bash
make env
make infra-up
make api-migrate
make api-migrate ARGS=status
make api-run
```

In another terminal, check `http://127.0.0.1:8080/healthz`, `/readyz`, and `/metrics`.
`make api-run` loads the ignored local `.env` privately and derives `DATABASE_URL`
from `POSTGRES_USER`, `POSTGRES_PASSWORD`, `POSTGRES_DB` and `POSTGRES_PORT`.
An explicitly supplied process `DATABASE_URL` takes precedence. The executable
itself reads only process environment; it never loads a dotenv file automatically.

Development PostgreSQL binds to loopback **55432** by default. A different,
unrelated PostgreSQL was already reachable on the host's 5432 during validation;
it was not modified. Change `POSTGRES_PORT` in `.env` if needed and recreate only
this project's service with `make infra-up`.

## Configuration

- `API_LISTEN_ADDR`: default `127.0.0.1:8080`; valid nonzero host:port.
- `APP_ENV`: `development` (default), `test`, or `production`.
- `DATABASE_URL`: required PostgreSQL URL with explicit user/host/database and
  `sslmode`. The local helper constructs it without printing credentials. Remote
  connections require `verify-full`; `disable` is allowed only on loopback in
  development/test. Host overrides and unknown/duplicate URL options are rejected.
- `DB_MAX_OPEN_CONNS`: default 8; allowed 1–64.
- `DB_MAX_IDLE_CONNS`: default 4; allowed 0–64 and no larger than max open.
- `DB_CONNECT_TIMEOUT`: default 2s; allowed 100ms–30s per attempt.
- `DB_STARTUP_TIMEOUT`: default 10s; allowed 100ms–2m total startup wait.
- `DB_READY_TIMEOUT`: default 1s; allowed 50ms–5s per readiness probe.
- `DB_MIGRATION_TIMEOUT`: default 30s; allowed 1s–5m per migration command.

Untrusted URL/driver errors are not echoed to logs or HTTP clients. Do not log
configuration structs, which contain database credentials. Production roles,
secret delivery and ingress still require deployment review.

## Migrations and failure behavior

`cmd/migrate` embeds the SQL files; no working-directory SQL lookup or external
Goose executable is needed. `up` applies pending migrations, `status` reports the
current version, and `down` rolls back one version. PostgreSQL advisory locks
serialize migration runners; lock acquisition is bounded. SQL runs transactionally.

The initial migration creates only the `connect` application namespace. No
identity schema is invented. The down operation uses `RESTRICT`: future tables
prevent accidental cascading data deletion. On the current empty namespace,
local rollback/reapply is available explicitly:

```bash
make api-migrate ARGS=down
make api-migrate ARGS=up
```

Migrations never run during API startup. PostgreSQL unavailability causes bounded
startup failure. Once running, `/healthz` stays live through database outages;
`/readyz` returns 503 for unavailable/outdated schema and returns 200 after recovery.
Missing migration tables are not created by readiness probes.

`/metrics` contains fixed counters and pool gauges, without dynamic request paths
or user identifiers. It and the health endpoints must remain private at ingress.

## Tests

```bash
make api-lint api-test api-build
make api-integration
```

The integration target starts a dedicated PostgreSQL container with temporary
storage and database `connect_stage1_test` on port 15432, runs race-enabled tests,
and removes only that test stack. It does not mount or reset development volumes.
`TEST_POSTGRES_PORT` can override the test port. Direct integration test execution
requires `TEST_DATABASE_URL` to name that dedicated test database.

Validated: fresh and repeat migrations, concurrent runners, advisory lock
contention, protected rollback/failure atomicity, rollback/reapply, readiness query
deadline and recovery, closed-database failure, startup retry deadline, config/TLS
validation/error redaction, concurrent bounded metrics, and live outage/restart
behavior. See the validation record below for final build and preview results.

## Android build and preview

The app remains the Stage 0 development screen; Stage 1 adds backend capability,
not onboarding/dialer/account UI. Local Android tools were installed under ignored
`.tools/`: Temurin JDK 21.0.13, Android API 36/build-tools 36.0.0, NDK
28.2.13676358, emulator and a Google APIs arm64 API 36 image. Official tool
downloads were checked against their published checksums. No shell startup files
were changed. Flutter 3.47.8 remains under `/tmp/connect-flutter-sdk`.

For this machine's current local installation, run from the repository root:

```bash
export JAVA_HOME="$(find "$PWD/.tools/jdk21" -type d -path '*/Contents/Home' -print -quit)"
export ANDROID_HOME="$PWD/.tools/android-sdk"
export ANDROID_USER_HOME="$PWD/.tools/android-user"
export ANDROID_AVD_HOME="$ANDROID_USER_HOME/avd"
export GRADLE_USER_HOME="$PWD/.cache/gradle"
export PUB_CACHE=/tmp/connect-pub-cache
export PATH="/tmp/connect-flutter-sdk/bin:$ANDROID_HOME/platform-tools:$ANDROID_HOME/emulator:$JAVA_HOME/bin:$PATH"
make mobile-apk
```

The APK is `apps/mobile/build/app/outputs/flutter-apk/app-debug.apk`. Launch the
existing preview emulator and install it with:

```bash
emulator -avd Connect_API_36 -no-snapshot -no-boot-anim -gpu swiftshader -memory 2048 -no-audio
# In another terminal with the same environment, once Android has booted:
adb -s emulator-5554 install -r apps/mobile/build/app/outputs/flutter-apk/app-debug.apk
adb -s emulator-5554 shell am start -n dev.connect.connect_mobile/.MainActivity
```

The preview uses a local 2 GB data partition and no saved snapshot. Its runtime
files, signing/debug keys, tools, build caches, APK and screenshots are ignored.
For a durable Flutter installation or a different machine, use the root README's
official setup instructions instead of depending on `/tmp` contents.

## Validation record — 2026-10-24

- Full host `make check`: Go formatting/vet/race tests and both executable builds,
  native format/build/CTest, Flutter lockfile/format/analyzer/two widget tests passed.
- `make api-integration`: all isolated real-PostgreSQL scenarios listed above passed.
- `go mod verify`: all module checksums verified. Workflow actionlint and helper
  syntax checks passed; GitHub Actions itself has not executed remotely.
- Live API through a development database outage/restart: readiness 200 → 503 →
  200, liveness stayed 200, metrics recorded failure, and SIGTERM exited cleanly.
  Captured lifecycle logs contained no database password or connection URL.
- Android debug APK built and installed on the API 36 arm64 emulator; its rendered
  development screen was inspected and captured to `.build/qa/app-preview.png`.
  The first cold debug launch exceeded `am start`'s wait timeout; the screen then
  rendered, a subsequent launch returned status OK, and Android reported no ANR.
  These observations are not a performance or incoming-call reliability benchmark.
- Android arm64 C++ cross-compilation passed using the NDK.
- Android lint passed. The generated API-21-only launch resource was consolidated
  into the base resource because minimum API is 24. The pinned toolchain is retained;
  lint's newer-tool-version advisory does not authorize an independent upgrade.
  A clean rebuild after consolidation passed, with zero lint errors and only the
  tool-version advisory remaining. SDK XML/Gradle deprecation advisories are noted
  for a separately reviewed toolchain update.

The debug APK is also copied to `.build/artifacts/Connect-debug.apk` for review.
QA database containers are stopped and their development data is preserved; the
emulator is left open with the app for inspection. No source, SDK, credential or
artifact was pushed to a remote repository; the reviewer requested local commits.

Known limits: no physical-device test, release signing, account UI/authentication,
native audio bridge, calling, FCM or E2EE. The visible app is a real debug build
of the development shell. Stage 1 operational endpoints remain a local backend
increment; no production deployment is claimed.

## Next review gate

Stop before Stage 2. Agree the Account & Identity specification, enrollment and
device-key challenge semantics, key storage/algorithm support, credential/session
revocation, directory identifiers and abuse controls before authentication code.
Existing E2EE protocol/library proposals remain separately unapproved.
