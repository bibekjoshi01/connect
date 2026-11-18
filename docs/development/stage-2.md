# Stage 2 — accounts and device authentication

Status: **backend design approved and implemented on 2026-11-18; automated
backend checks passed, manual/native acceptance pending**.
Prepared 2026-10-28 following the reviewer's Stage 1 acceptance and request to
begin the next stage. The backend behavior below is now implemented; native
Android integration remains pending. See [implementation and validation](stage-2-results.md).
Updated 2026-11-18 after the reviewer requested the next phase and a subsequent
manual review. That request authorizes preparation; the separate security-design
approval required by `AGENTS.md` was subsequently given: after reviewing the
backend approval request, the reviewer instructed, "implement them, then we will
develop the app and run it". This authorizes the backend scope below. Android
RP/app configuration and compatibility remain prerequisites for native work.

## Scope and account behavior

- Create an account without a phone number, email, password, or Google sign-in.
- Give it an immutable random UUIDv4 account ID and a permanent random Call ID:
  16 random bytes encoded as 32 lowercase hexadecimal characters. Neither ID
  authorizes access. Retry uniqueness collisions at most three times.
- Usernames are 3–32 ASCII characters matching `[a-z][a-z0-9_]{2,31}`;
  normalize ASCII uppercase to lowercase, reject whitespace and other input,
  and enforce uniqueness in PostgreSQL. Renaming is deferred.
- Enroll exactly one trusted device. Another device cannot claim an existing
  account by entering its username or Call ID.
- Support authentication, reading one's own profile, and current-session logout.
  No public username search, Call ID lookup, or availability endpoint yet.

Recovery and device transfer belong to Stage 3; product onboarding screens to
Stage 4. In this increment, losing the authenticator or required local credential
metadata can make the account inaccessible. There is no administrator bypass.

## Authentication design

Use standard WebAuthn registration and assertion ceremonies through Android's
FIDO2 API, verified by a maintained Go WebAuthn library. Do not create our own
signed-challenge format or handshake. Selected backend verifier:
[`go-webauthn/webauthn` v0.18.2](https://github.com/go-webauthn/webauthn/releases/tag/v0.18.2).
Its resolved dependency graph is pinned in `go.mod`/`go.sum` and builds under
Go 1.26.10. Android library selection remains deferred until native integration.

Approved credential policy: ES256 only, platform authenticator,
`residentKey=discouraged`, and user verification required. Reject backup-eligible
or backed-up credentials at enrollment and authentication. A selection hint alone
does not prove that a credential is device-bound. Use the verifier's standard
challenge, origin, RP ID, signature, user-verification, and backup-flag checks.
Approved application policy: reject an assertion when the verifier reports
`CloneWarning`; do not issue a replacement session or update the stored counter.
The library reports this warning without necessarily returning an error, so a
successful verifier call alone is insufficient. Zero counters alone do not prove
cloning. Persist successful counter/flag updates with session rotation in the
same account-locked transaction; revalidate against the locked credential state
if verification began from an earlier snapshot.

Google documents a device-bound FIDO2 flow separately from synced passkeys.
Its provider manages the private credential; Flutter must never receive it.
This design trusts the Android OS/provider and makes no remotely verified
hardware-security claim. Request no identifying attestation (`none`); requiring
hardware attestation would need a separate trust-root and privacy decision.

Before end-to-end Android work, demonstrate the policy on a supported phone:
registration, assertion, rejection of synced credentials, cancellation, and
screen-lock/key loss. Confirm Google Play services requirements and supported
Android versions. Unsupported devices fail explicitly; no software-key fallback.
This compatibility work remains part of native Stage 2 acceptance, not completed QA.

RP ID, allowed origins, app package, signing-certificate association, and a
development test arrangement must be agreed before native integration. Android
Digital Asset Links may require a reachable HTTPS domain; no hosting or domain
purchase is authorized. Backend fixtures cannot establish Android compatibility.
Never accept wildcard origins or disable origin/TLS verification for testing.

## Frontend and backend separation

- `backend`: account behavior and module-owned storage interfaces, with a
  PostgreSQL adapter and WebAuthn verifier integration. No media-engine dependency.
- `mobile-app/android`: Kotlin interface for FIDO2 and private local storage.
  Keep credential metadata out of backups and device-transfer exports. Approved
  session storage is memory-only, so process restart requires authentication.
- `mobile-app/lib`: later consumes the Kotlin interface for presentation.
  Stage 2 can validate native integration without building Stage 4 product UI.
- `contracts/rest`: publish the reviewed versioned API before implementing routes.

## API surface

All new routes use `/v1`. Options responses contain an opaque `ceremony_id` plus
standard WebAuthn options; finish requests contain that ID and the standard
credential response. The server stores all ceremony context and never trusts
client-supplied account IDs, origins, or configuration as authoritative.

- `POST /v1/accounts/registration/options`: validate username; create pending
  ceremony context. Do not create an account or reserve the username yet.
- `POST /v1/accounts/registration/finish`: verify registration; atomically consume
  the ceremony and create account, device, and initial session. Only one racing
  request may win a username; return a generic registration conflict otherwise.
- `POST /v1/authentication/options`: accept the device's random 32-byte auth handle
  saved at enrollment, rather than expose username-to-credential lookup.
  Unknown handles receive equivalent-shaped options and cannot authenticate.
- `POST /v1/authentication/finish`: verify assertion; atomically consume its
  ceremony and replace the device's previous session with a new one.
- `GET /v1/me`: return only the authenticated account's ID, username, and Call ID.
- `DELETE /v1/sessions/current`: revoke the current session; retain the device
  credential so the same device can authenticate again.

Bearer credentials travel only in the Authorization header over HTTPS. Return
`Cache-Control: no-store`. Use generic 401 authentication failures, bounded 400
validation errors, 409 registration conflicts, 429 throttling, and sanitized 503
dependency failures. Username uniqueness conflicts necessarily reveal limited
availability information; rate limits reduce abuse but do not eliminate it.

## Storage, concurrency, and sessions

Implemented records: accounts, devices, ceremonies, and sessions. Accounts own their
username and Call ID and have a non-null active-device pointer. A deferred
composite foreign key must ensure the pointed device belongs to that account.
The pointer is the sole active-device authority; no separate active-device flag.
Devices store the verifier credential record and a digest of the random auth
handle. Credential IDs are globally unique; each device has one credential.

Ceremonies use 32 random challenge bytes, expire after two minutes, and bind the
operation, pending username/user handle or enrolled device, and verifier session
context. A transaction rechecks expiry and account/device state when consuming
the ceremony. Replay and concurrent finish attempts must not issue two sessions.
Invalid responses never create partial accounts or sessions.

Sessions are opaque 32-byte random bearer tokens with 15-minute absolute
expiry, no refresh token, and at most one unrevoked session per device. Store only
SHA-256 token digests. Generate randomness with the OS-backed standard library;
never persist raw tokens or log tokens, challenges, handles, or credential bodies.

Every authenticated operation checks the session and active-device pointer in
PostgreSQL and fails closed during database failure. Mutations lock the account
row through commit; future replacement/revocation must use that same ordering.
Requests authorized before a revocation commits may already be in flight.
Do not use a cache or standalone JWT as the revocation authority.

Bearer-token theft allows impersonation until expiry or revocation. This design
enforces one enrolled device and one live session, not proof of the physical
device on every HTTP request. Stronger request binding needs separate review.
FIDO2 authentication requires user interaction, so the approved short sessions
also mean repeated unlock prompts. Review this before later background signaling.

## Resource and privacy limits

Implemented limits: 32 KiB JSON bodies, strict decoding, two-second database
operations, at most 1,000 stored ceremonies (including consumed state awaiting
cleanup), and three pending login ceremonies per device. Enforce ceremony capacity atomically across API instances. Remove
expired/consumed ceremony state within five minutes with bounded cleanup batches.

Per-process source throttling: five registration starts and twenty authentication
starts per minute, plus twenty finish attempts per minute, with a 10,000-entry
cap and ten-minute idle eviction. Allow at most three failed finishes per
ceremony and eight simultaneous verification operations per API process; reject
excess work rather than queue without bounds. Reject new sources when the limiter
is full. Use the socket peer address; do not trust forwarded IP headers until
proxy configuration is reviewed. This is a development safeguard, not sufficient
distributed abuse prevention for public signup.

Keep source addresses in limiter memory only; no identifier-bearing metrics or
request bodies in logs. Delete expired/revoked session rows within 24 hours.
Keep account/device records for account operation; account deletion and retention
after closure require a later policy before public signup. Down migrations must
refuse to remove nonempty identity tables. No production rollout is authorized.

## Implementation order and acceptance

1. Record human approval of the choices and limitations in this design (recorded below).
2. Review and pin verifier dependencies, add versioned contracts, then implement
   the module and transactional migrations together. Add no empty module stubs.
3. Test races for usernames, credential reuse and ceremony consumption; wrong
   operation/origin/RP ID, invalid signatures, expiry/replay, missing verification,
   backup eligibility, session rotation/logout, and database failure. Exercise
   limits and inspect errors/logs for credential disclosure.
4. Complete the Android compatibility check before claiming an app login flow.
   Keep tests synthetic; never commit device credentials or captured responses.
5. Run `make api-lint api-test api-build api-integration`; run `make mobile-check`
   and `make mobile-apk` when Android code is added. Report unavailable checks explicitly.
   Review Stage 2 results before Stage 3.

## Concrete review scope — 2026-11-18

The approved implementation increment is the **backend portion of Stage 2**: the six routes above, account/device/ceremony/session storage,
WebAuthn verification, limits, and synthetic plus real-PostgreSQL tests. Android
integration follows only after its RP/app configuration and compatibility checks
are resolved. This split does not change the Stage 2 acceptance requirement for
native compatibility and does not authorize Stage 3.

The reviewer approved these product/security tradeoffs before implementation:

- Device authentication uses standard WebAuthn ES256 with required user
  verification and rejection of backup-eligible/backed-up credentials. No custom
  handshake, software-key fallback, or hardware-attestation guarantee is proposed.
- Exactly one enrolled active device and one unrevoked session per device.
  Successful login rotates the session; logout revokes it. Sessions are bearer
  credentials, expire after 15 minutes, and are kept only in app memory.
- No recovery or transfer in this increment. Loss of the credential or local
  authentication handle can permanently prevent access until a later approved
  recovery design exists. A registration that commits but whose response or
  local metadata is lost can also strand the account. Do not recover by username
  or disclose the authentication handle through an unauthenticated retry.
- Identifiers, username rules, expiry/resource limits, generic failures and
  source throttling are as specified above. Signup remains local development
  work; its availability leakage and per-process throttling are unsuitable as
  the sole protections for a public service.

Before native work, record the chosen RP domain, exact allowed browser/native
origins, app identifier and signing-certificate association, supported phone/API
and Play services version, pinned Android FIDO2 dependency, and an HTTPS test
arrangement. None is silently selected by backend fixtures. No domain purchase,
hosting, release signing, or deployment is included in this approval.

Dependency source inspection on 2026-11-18 found that the candidate verifier's
[`go.mod`](https://raw.githubusercontent.com/go-webauthn/webauthn/v0.18.2/go.mod)
declares `go 1.26.0` and suggests `toolchain go1.27.1`. The consuming backend
subsequently resolved its graph and passed builds/tests under the project's
pinned Go 1.26.10. Go distinguishes
[minimum language requirements from toolchain suggestions](https://go.dev/doc/toolchain).
The verifier and resolved dependencies were added; the project toolchain was not changed.

The candidate's
[`Authenticator.UpdateCounter`](https://github.com/go-webauthn/webauthn/blob/v0.18.2/webauthn/authenticator.go)
sets the clone-warning flag, leaving the rejection policy to the application.
Its [configuration](https://github.com/go-webauthn/webauthn/blob/v0.18.2/webauthn/types.go)
also distinguishes native opaque origins from browser origins; use exact native
origin allowlists rather than treating an Android origin as an HTTPS URL.

Manual acceptance steps are tracked in [manual-testing.md](manual-testing.md).
Backend automated results are recorded separately; unchecked manual and native
cases remain acceptance criteria, not passing test results.

## Sources and approval record

Source review: 2026-10-28. Sources support standard/platform behavior, not approval
of this application's chosen limits, identifiers, session policy, or trust model.

- [WebAuthn standard](https://www.w3.org/TR/webauthn-3/): ceremonies and credential verification.
- [Android FIDO2](https://developers.google.com/identity/fido/android/native-apps): native API and app association.
- [Device-bound FIDO2 guidance](https://android-developers.googleblog.com/2024/09/attestation-format-change-for-android-fido2-api.html): distinction from synced passkeys.
- [Resident-key requirement](https://developers.google.com/android/reference/com/google/android/gms/fido/fido2/api/common/ResidentKeyRequirement): a preference does not guarantee credential type.
- [OWASP session guidance](https://cheatsheetseries.owasp.org/cheatsheets/Session_Management_Cheat_Sheet.html): token lifecycle and transport considerations.

Stage 1 acceptance and Stage 2 specification authorization: reviewer, 2026-10-28.
Next-phase preparation and later manual testing requested: reviewer, 2026-11-18.
Stage 2 backend authentication/security design: **approved by the reviewer,
2026-11-18**, in response to the explicit backend design approval request.
Backend verifier dependencies: **pinned and validated under Go 1.26.10**.
Android compatibility/dependency and deployed RP/app configuration: **not validated/selected**.
