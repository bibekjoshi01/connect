# Shared API contracts

This folder is the single source of truth for the API boundary between backend
and mobile client. The backend implements these definitions; mobile integration
must follow them. Keep the schemas here instead of duplicating them in either
component. Specifications have no separate running process or deployment.

Operational endpoints are specified in `rest/health-v1.openapi.yaml`:
`/healthz` (liveness), `/readyz` (PostgreSQL/schema readiness), and `/metrics`.
These paths are stable; their schema version is tracked in the contract. Keep
operational endpoints on private ingress; no public operational access is approved.

Stage 2 accounts are specified in `rest/accounts-v1.openapi.yaml`: WebAuthn
registration options, attestation finish, mandatory signed enrollment proof,
authentication options/finish, own-profile access and logout. Registration finish
returns 202/assertion options; only registration proof creates an account/session.
The routes require HTTPS and explicit RP configuration; sessions are opaque
15-minute bearer credentials. There is no public account lookup or recovery API.

Future reviewed contracts go in `rest/`, `signaling/`, and `media/` when needed.
Use explicit protocol/schema versions, compatibility rules, and golden vectors.
Signaling and media are separate protocols. No session media keys or plaintext audio
may be placed in ordinary signaling. No media wire format is approved yet.
