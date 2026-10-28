# Stage 0 validation record

Date: 2026-10-24 (Asia/Kathmandu). Repository started empty; a local `main` Git
repository is now initialized. No remote, external account, paid infrastructure
or deployment was created. Follow-up QA fixes are included in the initial commit.

## Implemented increment

- Android-only Flutter/Kotlin scaffold with a development screen and widget test.
  Flutter 3.47.8 / Dart 3.13.7; no production app dependencies beyond Flutter.
- Go standard-library HTTP process with validated listener config, JSON logs,
  bounded HTTP resources, `/healthz`, and graceful signal shutdown. The module
  pins Go 1.26.10. There is no database/auth/signaling implementation.
- Standalone C++20 static library and CTest linkage smoke test. It exposes build
  metadata only; no media functionality or Flutter bridge exists yet.
- Digest-pinned PostgreSQL, optional Redis, local-only coturn profiles, generated
  ignored local credentials, and repeatable infrastructure smoke checks.
- Root commands, formatting rules, shared liveness contract, architecture/security
  review documents, repository instructions and commit-pinned CI jobs.

## Checks actually executed

Passed the complete `make check` suite using:

```bash
export CI=true
export FLUTTER_SUPPRESS_ANALYTICS=true
export PUB_CACHE=/tmp/connect-pub-cache
export XDG_CONFIG_HOME=/tmp/connect-xdg
make check \
  FLUTTER=/tmp/connect-flutter-sdk/bin/flutter \
  DART=/tmp/connect-flutter-sdk/bin/dart \
  CLANG_FORMAT=.tools/format/bin/clang-format
```

- Go: gofmt, vet, race-enabled config and HTTP contract tests, executable build
  with patched Go 1.26.10. An initial sandbox-only attempt failed because socket
  binding was denied; rerun with local socket permission passed.
- Native: clang-format 21.1.8, CMake 4.2.1, Apple Clang 21, warnings as errors,
  static library build, one CTest smoke test. This tests build/linkage, not RTC.
- Flutter: lockfile enforcement, Dart formatting, analyzer with fatal infos,
  two widget tests including small-screen/enlarged-text accessibility; all passed.
- Live Go process: loopback `GET /healthz`, SIGTERM shutdown with exit 0, and
  invalid listener configuration with structured error/exit 1; passed.
- Docker Compose: all-profile config validation, healthy PostgreSQL and Redis,
  SQL `SELECT 1`, authenticated Redis `PING`; passed.
- coturn: STUN binding, authenticated container-local client-to-client relay
  (four sent, four received), and invalid-password rejection; passed.
- GitHub Actions workflow: actionlint 1.7.10; shell/Python syntax checks; passed.

The Docker runtime was local OrbStack (Linux arm64 containers). Service tests do
not establish production suitability, load behavior, external NAT reachability
or security hardening. Source secrets/ignore rules are checked before handoff.
Test containers are stopped after validation; the PostgreSQL volume is preserved.
Restart with `make infra-up-cache infra-up-turn` when needed.

## Follow-up QA before commit

The reviewer requested a local commit and additional QA. Re-ran the host suite,
infrastructure checks and workflow linting, and fixed three reproduced issues:

- The C++ `connect` namespace collided with the POSIX socket `connect()` function.
  Renamed it to `connect_rtc`; the native smoke test now includes the socket header
  before the public engine header, and compilation/linkage pass.
- `redis-cli ping` returned exit 0 even with invalid credentials. The infrastructure
  smoke check now requires the exact `PONG` response. Verified bad credentials
  fail the stricter check and the generated local password still succeeds.
- The Flutter screen overflowed at 320×480 with 3× text scaling. Added scrolling
  and a regression test that verifies no layout exception and that the final
  notice can be reached; both widget tests and analysis pass.

Also verified isolated environment generation creates `.env` with mode 0600,
preserves an existing file on rerun, rejects an invalid TURN password, and leaves
generated local passwords out of source files. Live API QA passed health
responses, SIGINT/SIGTERM cleanup, invalid configuration and occupied-port failure.
Corrected the tooling notes to describe Flutter's Kotlin compatibility flags.

Available Stage 0 checks pass after these fixes. Android APK/lint/cross-compilation
and device tests remain unverified locally for the reasons below; workflow syntax
validation does not establish that those CI steps have executed successfully.

## Checks unavailable and known limitations

- `flutter build apk --debug --no-pub` was attempted and failed with **No Android
  SDK found**. Only JDK 25 is installed globally; the configured Android build
  baseline uses JDK 21. No APK or Android native cross-build is claimed locally.
- CI contains debug APK, Android lint and NDK arm64 compile steps, but **GitHub
  Actions has not executed**. Local workflow linting is not a CI pass.
- There is no Android device/emulator validation, release signing, native engine
  bridge, audio processing, network media, account/auth, FCM, TLS proxy or E2EE.
- Flutter was downloaded to `/tmp/connect-flutter-sdk` for validation and is not
  installed permanently or added to shell startup files. `/tmp` may be cleared;
  follow the root README for a durable SDK installation.
- Generated local credentials are ignored. Database data is local-only. Public
  TURN and device testing require a different, approved network configuration.

## Review required before continuation

Review the requested stack/boundaries and bootstrap ADR, provisional
`dev.connect.connect_mobile` identifier and Android minimum API 24, and the
recommended Stage 1 plan. Security proposals in `../security/review-gates.md`
require separate approval before dependent implementation; DTLS 1.3 is an
evaluation recommendation, not an approved protocol/library choice.

Stop here. Stage 1 is not started.
