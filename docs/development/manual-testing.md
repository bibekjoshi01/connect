# Manual acceptance checklist

Prepared 2026-11-18. Run from the repository root unless stated otherwise.
Unchecked items have not been manually verified in this review. The existing
runtime includes the Stage 2 backend and Stage 0 mobile shell. Backend account
cases are now covered by synthetic HTTPS/PostgreSQL integration tests; their
manual checkboxes remain unexecuted. Android account cases remain future work.
See [Stage 2 setup and results](stage-2-results.md) for the opt-in RP/TLS settings.

## Prerequisites and automated baseline

- [ ] Docker Engine and Compose v2 are running; Go 1.26.11 is available.
- [ ] For mobile checks, Flutter 3.47.8, JDK 21 and the Android SDK/device are
  configured using [toolchains.md](toolchains.md) and [stage-1.md](stage-1.md).
  The previous temporary Flutter SDK is currently missing.
- [ ] Run the relevant baseline checks and record their exit results:

```bash
make backend-lint backend-test backend-build
make native-format-check native-test CLANG_FORMAT=.tools/format/bin/clang-format
make mobile-check mobile-apk
make backend-integration
```

Use `CLANG_FORMAT=clang-format` if the pinned formatter is installed on PATH
instead. The integration target uses its own disposable database; do not point
it at development or unrelated databases. Local HTTP tests need permission to
bind loopback ports. A blocked tool or skipped suite is not a passing check.

## Existing backend — Stage 1

Start the project's development database, apply migrations, and run the API:

```bash
make env
make infra-up
make api-migrate
make api-migrate ARGS=status
make api-run
```

`make env` preserves an existing `.env`. Never paste that file, connection URLs,
or credentials into review evidence. `api-run` occupies this terminal.

In a second terminal:

```bash
curl --max-time 5 -i http://127.0.0.1:8080/healthz
curl --max-time 5 -i http://127.0.0.1:8080/readyz
curl --max-time 5 -i http://127.0.0.1:8080/metrics
curl --max-time 5 -i http://127.0.0.1:8080/v1/me
```

- [ ] Liveness returns HTTP 200 and `{"status":"ok"}`.
- [ ] Readiness returns HTTP 200 and `{"status":"ready"}` after migration.
- [ ] Metrics returns fixed `connect_*` counters and database pool gauges,
  without request bodies, credentials or user identifiers.
- [ ] `/v1/me` returns 404 with authentication settings absent. With explicit RP
  configuration the server requires HTTPS, and this route requires a session.
- [ ] Applying migrations again succeeds; status remains version 3.

For the database outage test, stop only this project's development PostgreSQL:

```bash
docker compose --env-file .env -f infrastructure/compose/compose.yaml stop postgres
curl --max-time 5 -i http://127.0.0.1:8080/healthz
curl --max-time 5 -i http://127.0.0.1:8080/readyz
```

- [ ] Liveness stays 200; readiness returns 503 with the generic unavailable
  response, without a database URL or driver error.

Restore the database even if an outage assertion fails:

```bash
make infra-up
curl --max-time 5 -i http://127.0.0.1:8080/readyz
```

- [ ] Readiness returns to 200 without restarting the API.
- [ ] Pressing Ctrl+C in the API terminal exits cleanly; lifecycle logs contain
  no connection URL, password or request body.

Do not manually roll back a database containing data you want to retain.
Rollback/reapply and concurrent migrations belong in `make backend-integration`'s
isolated stack. Stop development services with `make infra-down` after testing;
omit `--volumes` to preserve development data.

## Existing Android app — Stage 0 shell

With an emulator or device selected:

```bash
cd mobile-app
flutter run
```

- [ ] The app shows `Connect`, `Development foundation`, and the notice that
  calling/account setup are unavailable. No registration or dialer is expected.
- [ ] On a small screen with enlarged system text, scroll to the complete notice;
  no overflow/crash occurs.
- [ ] Background and reopen the app; the shell remains usable.
- [ ] The same shell launches while the backend is stopped. Backend connectivity
  is not implemented; this does not prove offline account or calling behavior.

## Backend acceptance — Stage 2, implemented; manual review pending

Execute against a disposable test database using the [versioned contract](../../contracts/rest/accounts-v1.openapi.yaml)
and synthetic WebAuthn fixtures. Real private device credentials, bearer tokens,
authentication handles and captured ceremony bodies must not enter Git or test
reports. Do not add an API that bypasses real verification for manual testing.

- [ ] Register a new username; verify immutable account/Call IDs, exactly one
  active device, and one live session only after the signed proof step. Neither
  options nor attestation finish may create account/device/session rows. Finish
  must return 202/assertion options without token, profile or device handle.
- [ ] With that session, `/v1/me` returns only the account's profile. Missing,
  invalid, revoked and expired sessions receive generic 401 responses.
- [ ] Login rotates the session atomically; the previous token no longer works.
  Logout revokes the current token while permitting a new real authentication.
- [ ] Duplicate username, reused credential and concurrent registration attempts
  create no partial accounts; only one racing enrollment wins.
- [ ] Replayed, expired, exhausted or wrong-operation ceremonies fail. Concurrent
  finishes issue at most one session and update verifier state consistently.
- [ ] Invalid signatures, wrong RP/origin, missing user verification, backup
  eligibility/state and clone warnings fail under the approved policy.
- [ ] Unknown auth handles receive equivalent-shaped options and cannot log in.
  Knowing a username or Call ID grants no account access.
- [ ] Strict body/field validation, body size, ceremony capacity, verification
  concurrency and throttling limits reject excess work without unbounded growth.
- [ ] Database outage fails authenticated requests closed with sanitized errors;
  recovery does not restore revoked sessions.
- [ ] Logs, metrics and error responses contain no tokens, handles, challenges,
  credential bodies or database credentials. Responses use `Cache-Control: no-store`.

These race, expiry and negative cases need automated assertions as well as manual
inspection. Their request shapes are defined by the versioned contract. The integration
suite constructs standard WebAuthn responses in memory; there is no manual login
shortcut. Real Android enrollment is still pending native integration.

## Future Android compatibility — Stage 2, not implemented

Use a supported physical test phone, the agreed RP/domain/origin/app association,
and the actual pinned FIDO2 implementation. An emulator or backend fixture alone
cannot establish compatibility.

- [ ] Registration and login work with required screen-lock/user verification.
- [ ] Synced/backup-eligible credentials are rejected; unsupported devices report
  an explicit error and offer no silent software-key fallback.
- [ ] Cancellation and provider errors return control without enrolling an account
  or issuing a session unexpectedly.
- [ ] Restarting the process loses its in-memory session and requires login;
  private credential material never passes through Flutter.
- [ ] Backup/restore and transfer do not export the app's credential metadata.
- [ ] Test key invalidation, screen-lock changes and metadata loss only on a
  disposable test account/device. Document any inaccessible account; no recovery
  bypass is expected in Stage 2.

Stage 4 owns product onboarding screens. These Stage 2 native checks do not
establish a complete onboarding UI or any voice-call behavior.

## Evidence and review

For each executed case record date, source revision and local changes, tool/device
versions, expected result, observed result, and pass/fail/blocked. Keep evidence
free of credentials and production configuration. Preserve old validation records
as history; do not relabel their results as a new execution.

Review Stage 2 implementation and this evidence before authorizing Stage 3.
