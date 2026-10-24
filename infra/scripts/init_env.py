#!/usr/bin/env python3
"""Generate local-only secrets without overwriting an existing environment."""

import os
from pathlib import Path
import secrets

ROOT = Path(__file__).resolve().parents[2]


def main():
    env = ROOT / ".env"
    if not env.exists():
        template = (ROOT / ".env.example").read_text()
        for marker in (
            "replace-with-local-random-value",
            "replace-with-another-local-random-value",
            "replace-with-a-third-local-random-value",
        ):
            template = template.replace(marker, secrets.token_hex(32))
        # Exclusive creation avoids clobbering another process/user's local config.
        with open(env, "x", opener=lambda p, f: os.open(p, f, 0o600)) as output:
            output.write(template)
        print("Created .env with local-only random credentials (mode 0600).")
    values = {}
    for line in env.read_text().splitlines():
        if line.strip() and not line.lstrip().startswith("#"):
            key, _, value = line.partition("=")
            values[key.strip()] = value.strip()
    password = values.get("TURN_TEST_PASSWORD", "")
    # This file is deliberately a simple KEY=value format, not a shell program.
    if len(password) < 32 or not all(c in "0123456789abcdef" for c in password):
        raise SystemExit("TURN_TEST_PASSWORD must be at least 32 lowercase hex characters.")
    for key in ("POSTGRES_PASSWORD", "REDIS_PASSWORD"):
        if not values.get(key) or values[key].startswith("replace-with-"):
            raise SystemExit(f"Set a local-only {key} in .env first.")
    target = ROOT / ".build/coturn/turnserver.conf"
    target.parent.mkdir(parents=True, exist_ok=True)
    rendered = (ROOT / "infra/coturn/turnserver.conf.example").read_text().replace(
        "@TURN_TEST_PASSWORD@", password
    )
    target.write_text(rendered)
    # Container user must be able to read this bind-mounted LOCAL fixture.
    target.chmod(0o644)
    print("Rendered .build/coturn/turnserver.conf; do not publish local fixtures.")


if __name__ == "__main__":
    main()
