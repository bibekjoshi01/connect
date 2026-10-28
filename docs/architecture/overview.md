# Architecture review — Stage 0

Status: Stage 0 reviewed; Stage 1 backend foundation authorized and implemented.
Security-specific protocol proposals still require their separate approvals.
The repository was empty. No existing Account & Identity specification was found.
The supplied project requirements remain authoritative.

## Technology choices

- Flutter/Dart: UI, navigation, and application state. Pin the SDK and commit
  `pubspec.lock`. No UDP or continuous media processing in the UI isolate.
- Kotlin: Android lifecycle, Keystore, audio focus/routing, FCM delivery, and
  Telecom integration through explicit platform interfaces in later stages.
  Android recommends self-managed Core-Telecom for standalone calling apps:
  https://developer.android.com/develop/connectivity/telecom
- C++20/CMake: portable native RTC library. Stage 0 compiles build metadata only;
  it is not an audio or networking implementation. Opus, Android audio, reviewed
  crypto, and STUN/TURN will be integrated separately after approval.
- Go: standard-library HTTP process in a modular monolith; no framework required.
  `connect.local/api` is a local module name, not an internet domain dependency.
- PostgreSQL: bounded pgx/database/sql connectivity, explicit Goose migrations,
  and read-only dependency readiness. The first migration creates only the
  application namespace. Account/device/recovery tables remain deferred.
- Redis: optional local Compose profile. Its production use must be justified
  (e.g. bounded ephemeral coordination/rate limiting); it cannot be the source of
  truth for device revocation. Not connected to the API yet.
- coturn: standard STUN/TURN, isolated local configuration only. No custom relay.
- FCM and HTTPS/WSS ingress: deferred until their stages. Caddy versus Nginx is
  undecided; Stage 0 HTTP is loopback development only.

## Boundaries and dependencies

`apps/mobile/lib/app` owns presentation. Future feature modules depend on narrow
application/native interfaces. Kotlin platform code owns OS concerns; a future
FFI/JNI bridge delegates media to `engine/rtc`. The engine has no Flutter,
account database, or backend dependency. The bridge and packaging are deferred.

`services/api/cmd/api` composes configuration and HTTP transport. Future domain
modules own their repository interfaces; infrastructure adapters implement them.
REST and authenticated WebSockets form the control plane. Direct UDP/TURN carries
encrypted media; the server and relay must never obtain media decryption keys.

Versioned shared contracts live in `contracts/`. Add domain directories and
protocol contracts only with real implementations. No empty speculative modules.
See `../decisions/0002-backend-foundation.md` for Stage 1 configuration, timeouts,
pooling, migration locking, error redaction and operational endpoint boundaries.

## Identity invariants carried forward

Immutable internal account ID; unique username; random permanent Call ID; exactly
one active trusted device; device-bound private authentication key remains on the
device. No mandatory SIM, email, SMS, or password. Enrollment verifies proof of
possession. Challenge-response uses bounded, expiring, single-use challenges and
revocable sessions. Transfer/recovery atomically replace the active device and
revoke old sessions. Recovery delay, provisionally 12–24 hours, is configurable
and cancellable by the trusted device. No administrator takeover bypass.

## Risks and approval gates

- Custom RTC increases engineering and validation effort: pacing, audio deadlines,
  jitter, loss, NAT, congestion, battery, and Android lifecycle require real-device
  measurements. No WebRTC engine is allowed.
- App-to-app lookup needs abuse prevention without turning random Call IDs into
  an account enumeration API. Identifier formats remain unapproved.
- An untrusted signaling server can substitute first-contact identity keys unless
  users authenticate them independently. See `../security/review-gates.md`.
- Android force-stop, disabled notifications, OEM restrictions, and unavailable
  networks prevent delivery guarantees. FCM is a coordination aid, not delivery
  proof. Review current platform requirements again before Stage 11.
- Audio reconstruction will eventually expose a receiver-local observer interface
  with bounded work outside the playback deadline. No AI or detector is added now.
- Minimum Android version and final package identifier need product review. The
  generated identifier is provisional; release signing is not configured.

No paid infrastructure or external account is required for Stage 0. Later FCM
needs a Firebase project; GitHub-hosted CI needs a repository/account and may use
billable minutes beyond allowances. Public TURN needs reachable infrastructure,
bandwidth, DNS/TLS and monitoring; staging and Play distribution need separate
cost/account approval. Nothing is provisioned or purchased by this bootstrap.
