# Local infrastructure

This independent folder owns local dependency services: PostgreSQL, optional
Redis and coturn. The backend uses PostgreSQL; future mobile/native calling will
contact TURN over the network. Service configuration stays separate from app and
backend application source. Docker Engine with Compose v2 and Python 3.10+ are required.

- `compose/compose.yaml`: PostgreSQL by default; optional Redis and coturn profiles.
- `coturn/`: local TURN configuration template and its limits.
- `scripts/`: private environment generation and service smoke checks.
- `Makefile`: independent service lifecycle commands.

From this folder:

```bash
make env
make config
make up
make check
make down
```

`make env` creates the ignored root `.env` only if missing and renders a local
TURN fixture under `../.build/coturn/`. It preserves existing local configuration.
`make down` stops this development stack while preserving its database volume.

Optional Redis and coturn checks:

```bash
make up-cache up-turn
make check-cache check-turn
```

PostgreSQL binds to loopback port 55432 by default. Redis remains unconnected to
the API. coturn tests prove container-local interoperability only. These are local
development fixtures, not a production stack or proof of phone NAT traversal.
No paid service, hosting or deployment is provisioned.

The backend's disposable integration database belongs to
`../backend/tests/integration/compose.yaml` and is managed by `make integration`
in `../backend/`. It uses its own credentials/storage and does not modify this
stack. Build/run the API separately; this Compose file contains no API service.
