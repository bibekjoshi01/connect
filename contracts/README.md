# Shared contracts

Only process liveness is specified in Stage 0: `rest/health-v1.openapi.yaml`.
The operational `/healthz` path is stable; its schema version is tracked in the contract.

Future reviewed contracts go in `rest/`, `signaling/`, and `media/` when needed.
Use explicit protocol/schema versions, compatibility rules, and golden vectors.
Signaling and media are separate protocols. No session media keys or plaintext audio
may be placed in ordinary signaling. No media wire format is approved yet.
