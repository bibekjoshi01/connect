# Project rules

- Work stage by stage. Stage 0 is the current scope; request human review before Stage 1.
- Read `docs/architecture/overview.md` and `docs/development/stages.md` before changing architecture.
- No WebRTC engine, authentication, calling, media protocol, or AI in Stage 0.
- Keep Flutter presentation in `apps/mobile`; continuous audio/network work belongs in native code.
- Keep `engine/rtc` portable C++20, independent of Flutter, Android UI, and account storage.
- Keep Android lifecycle, Keystore, Telecom, and audio routing behind Kotlin platform interfaces.
- Keep the Go backend a modular monolith; add domain modules only when implemented.
- Account/auth code must not depend on the media engine. Storage uses module-owned interfaces.
- Version shared contracts in `contracts`; signaling must never carry plaintext audio or media keys.
- Never invent crypto or compose an ad hoc handshake. Protocols, trust models, and security changes require documented human approval before implementation.
- Never commit secrets, private keys, credentials, audio, or production configuration. Never disable TLS verification.
- Preserve the one-active-device invariant and atomic revocation in future identity work.
- Do not claim guaranteed incoming-call delivery under Android force-stop or platform restrictions.
- Pin dependencies, keep logs free of sensitive data, validate inputs, bound resources, and use timeouts.
- Run the relevant Makefile checks; report checks that could not run honestly.
- Avoid placeholder modules and speculative abstractions. Document non-obvious decisions.
- Do not provision paid services, deploy, or add account-takeover bypasses.
