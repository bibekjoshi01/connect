# Security decisions requiring approval

No authentication, key establishment, encrypted-media protocol, or recovery
implementation is authorized by the Stage 0 bootstrap.

## Peer key establishment proposal

Recommendation for evaluation: **mutually authenticated DTLS 1.3**, as specified
in [RFC 9147](https://www.rfc-editor.org/rfc/rfc9147.html), through a maintained
library with an independently reviewed implementation. First prove C++/NDK
support, nonblocking operation, datagram overhead, replay behavior, key lifecycle,
licensing, and peer-certificate verification. This is a proposal, not a selection
or a claim that a suitable library has been validated. Do not silently downgrade
to an older protocol if implementation support is inadequate.

Trust assumptions: endpoints and their OS/key stores are trusted while calls run;
signaling, directory storage, and TURN may be malicious. Peer certificate/key
fingerprints must be authenticated through an independently verified mechanism
(such as an explicit QR/fingerprint comparison). A fingerprint delivered only by
the same untrusted server does not authenticate the first contact. Without prior
independent verification, the UI must disclose an unverified identity and the
first-contact substitution risk. The exact device-auth-key to call-identity
binding and change/transfer/recovery semantics require their own reviewed spec;
do not invent a signature-based handshake to bridge the gap.

Alternatives for review:

- A fixed, published Noise handshake pattern using an audited implementation.
  Noise defines patterns, not the application's identity trust policy or a UDP
  replay/reordering protocol. Library maturity, Android Keystore interoperability,
  peer identity verification, and datagram integration still need review.
  [Noise specification](https://noiseprotocol.org/noise.html).
- ZRTP plus SRTP is purpose-built for voice key agreement and supports a short
  authentication string, but brings RTP/SRTP coupling and requires a maintained,
  suitable implementation. It conflicts with assuming a fully custom media wire
  format unless that constraint is explicitly revised.
  [RFC 6189](https://www.rfc-editor.org/rfc/rfc6189.html).

Do not combine algorithm primitives into a new handshake. No comparative
security claim against WebRTC or Signal is supported by this project.

## Required decisions before dependent implementation

1. Android authentication key algorithm, enrollment proof, challenge binding,
   replay/expiry policy, secure credential storage and revocation semantics.
2. Recovery authorization format/storage, security delay and cancellation rules;
   transfer authorization and atomic replacement. No hidden bypass.
3. Peer identity trust model, reviewed protocol and library, verification UX,
   key-change behavior and account/device binding.
4. Media packet specification and security review before custom encrypted media:
   version/header, byte order, sequence width/rollover, timestamps, authenticated
   fields, replay window, MTU/fragmentation, Opus frame duration/configuration,
   jitter/PLC, congestion behavior, path changes, key rotation/erasure, termination
   and quality measurements. These values are intentionally not chosen in Stage 0.
5. Metadata retention/deletion, directory abuse protection, push payload privacy,
   TURN credential lifetime and access/rate-limit policies.

Device authentication and E2EE identity are distinct responsibilities. Android
Keystore availability does not by itself authenticate a remote account. A server
can deny service and observe routing/timing metadata even with sound E2EE.
