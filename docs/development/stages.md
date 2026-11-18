# Stage boundaries

Current stage: **2 — backend implemented and automated checks passed; awaiting
human review and native integration**.
Stage 1 was accepted on 2026-10-28. On 2026-11-18 the reviewer approved the
[Stage 2 backend design](stage-2.md) by directing implementation in response to
the explicit security-design approval request. Registration, device WebAuthn
authentication, profile access and revocable sessions are now implemented.
QA corrections require a standard signed enrollment proof before the first
session, patch Go to 1.26.11 and add CI vulnerability scanning. The reviewer has
requested mobile UI/UX work next, with integration afterward; later identity and
calling implementation remains gated.
See [implementation and validation](stage-2-results.md) and the
[manual acceptance checklist](manual-testing.md).

Android authentication integration remains unimplemented: RP/domain/origin/app
association and physical-device compatibility must be resolved before that work.
The Flutter app still shows its development shell. Review Stage 2 before Stage 3;
recovery, transfer, signaling, calling and media security remain outside this
increment.

## Stage 1 implementation and review

Backend connectivity, migrations, readiness, metrics and isolated integration
tests are implemented. See `stage-1.md` for commands and validation results.
The following list records the completed stage's authorized scope.

1. Approve backend configuration conventions and migration runner selection.
2. Add a pinned PostgreSQL driver, bounded connection pool, connection/startup
   deadlines and graceful pool shutdown. Keep secrets out of errors and logs.
3. Add reviewed up/down migrations for infrastructure bookkeeping only; defer
   identity schema choices until the Account & Identity specification is agreed.
   Define transaction, lock and migration failure behavior explicitly.
4. Keep `/healthz` as process liveness; add dependency readiness that fails closed
   when PostgreSQL is unavailable, plus bounded startup/retry behavior.
5. Add privacy-safe structured logs and minimal operational metrics with no user
   identifiers or high-cardinality labels. Define safe errors and config validation.
6. Test clean migration, rollback/reapply, invalid configuration, database outage,
   startup/shutdown and timeouts against an isolated database. Add integration CI.
7. Justify Redis before integrating it; do not add it merely because Compose has it.

No account registration, authentication, device keys or calling in Stage 1.

## Later review sequence

2: account registration and device authentication; 3: recovery/transfer/revocation;
4: Flutter onboarding/dialer/settings; 5: signaling and call state machine;
6: native local audio/framing/Opus; 7: reviewed packet format and UDP network tests;
8: approved authenticated session establishment and E2EE integration;
9: NAT orchestration/STUN/TURN; 10: jitter/loss/reconnection/quality;
11: Android incoming-call lifecycle/FCM; 12: device/performance/security hardening;
13: approved staging/beta/Play preparation; 14: future receiver-side detector.

Refinement recommendation: plan NAT/path and congestion requirements alongside
the Stage 7 specification so Stage 8 does not assume a transport that later needs
redesign. Implementation remains staged and review-gated.
