# Stage boundaries

Current stage: **0 — implemented foundation, awaiting human review**.
Do not start Stage 1 until the reviewer approves it.

## Recommended Stage 1 increment

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
