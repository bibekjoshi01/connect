# Local TURN development

`make env` renders a private local config with generated long-term **test**
credentials. `make infra-up-turn` publishes STUN/TURN and ten UDP relay ports
only on host IPv4 loopback. TLS/DTLS and TCP relay are disabled in this local
fixture; this does not specify the future end-to-end media encryption protocol.
`make infra-check-turn` checks a STUN binding and authenticated TURN allocation
inside the container's network namespace, with a negative credential check.

Docker Desktop/OrbStack port publishing does not rewrite TURN's advertised relay
address. This fixture therefore does **not** provide a reachable relay for a
physical Android phone, does not demonstrate NAT traversal, and is not a public
TURN configuration. Host-side STUN can be tested on `127.0.0.1:3478`; relay tests
are explicitly container-local.

Before public TURN, approve infrastructure costs and threat model, choose a
reachable public address, configure `external-ip` correctly for any NAT, open
the intended listening/relay ranges, configure DNS/TLS certificates and renewal,
enable short-lived credential issuance with a server-only secret, restrict peers,
enforce quotas/rate limits, and review privacy-safe monitoring/retention. Do not
copy the local test password, disable peer safeguards, or expose this Compose file.

Reference: https://github.com/coturn/coturn/blob/master/docker/coturn/README.md
and https://github.com/coturn/coturn/wiki/turnserver
