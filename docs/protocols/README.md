# Protocol documentation policy

Executable schemas belong in `contracts/`; rationale, state machines, compatibility
and test-vector explanations belong here. Stage 1 defines only HTTP operational
endpoints (liveness, PostgreSQL/schema readiness, and bounded metrics).
There is no media packet format, signaling protocol or cryptographic handshake.
Require specification and security review before custom encrypted-media code.
