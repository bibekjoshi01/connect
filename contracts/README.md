# Shared contracts

Operational endpoints are specified in `rest/health-v1.openapi.yaml`:
`/healthz` (liveness), `/readyz` (PostgreSQL/schema readiness), and `/metrics`.
These paths are stable; their schema version is tracked in the contract. Keep
operational endpoints on private ingress; no public operational access is approved.

Future reviewed contracts go in `rest/`, `signaling/`, and `media/` when needed.
Use explicit protocol/schema versions, compatibility rules, and golden vectors.
Signaling and media are separate protocols. No session media keys or plaintext audio
may be placed in ordinary signaling. No media wire format is approved yet.
