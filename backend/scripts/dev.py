#!/usr/bin/env python3
"""Local Go commands with private environment loading; never print DSNs."""

import os
import secrets
from pathlib import Path
import subprocess
import sys
from urllib.parse import quote

BACKEND = Path(__file__).resolve().parents[1]
ROOT = BACKEND.parent


def local_environment():
    path = ROOT / ".env"
    if not path.is_file():
        raise SystemExit("Run make env before local backend commands.")
    values = {}
    for line in path.read_text().splitlines():
        if line.strip() and not line.lstrip().startswith("#"):
            key, separator, value = line.partition("=")
            if not separator:
                raise SystemExit("Local .env must contain simple KEY=value entries.")
            values[key.strip()] = value.strip()
    env = dict(os.environ)
    for key, value in values.items():
        if key in ("API_LISTEN_ADDR", "APP_ENV", "DATABASE_URL") or key.startswith(("DB_", "AUTH_", "API_TLS_")):
            env.setdefault(key, value)
    env.setdefault("APP_ENV", "development")
    return env, values


def database_url(values, port, database):
    user = quote(values["POSTGRES_USER"], safe="")
    password = quote(values["POSTGRES_PASSWORD"], safe="")
    return f"postgres://{user}:{password}@127.0.0.1:{port}/{quote(database, safe='')}?sslmode=disable"


def local_port(raw, name):
    try:
        port = int(raw)
    except ValueError:
        raise SystemExit(f"{name} must be a valid port.") from None
    if not 1 <= port <= 65535:
        raise SystemExit(f"{name} must be a valid port.")
    return port


def main():
    if len(sys.argv) < 2:
        raise SystemExit("Usage: dev.py run|migrate up|down|status|integration")
    action = sys.argv[1]
    if action not in ("run", "migrate", "integration"):
        raise SystemExit("Unsupported local backend command.")
    if action == "migrate" and sys.argv[2:] not in (["up"], ["down"], ["status"]):
        raise SystemExit("Usage: dev.py migrate up|down|status")
    if action == "integration":
        # Test credentials and storage are independent of development services.
        env = dict(os.environ)
        values = {"POSTGRES_USER": "connect_test", "POSTGRES_PASSWORD": secrets.token_hex(32)}
        env.update(values)
        port = local_port(env.get("TEST_POSTGRES_PORT", "15432"), "TEST_POSTGRES_PORT")
        env["TEST_POSTGRES_PORT"] = str(port)
        compose = ["docker", "compose", "--env-file", os.devnull, "-f", str(BACKEND / "tests/integration/compose.yaml")]
        env["TEST_DATABASE_URL"] = database_url(values, port, "connect_stage1_test")
        try:
            subprocess.run(compose + ["up", "-d", "--wait"], check=True, cwd=ROOT, env=env)
            result = subprocess.run(["go", "test", "-tags=integration", "-race", "-count=1", "-timeout=90s", "./tests/integration"], cwd=BACKEND, env=env)
        finally:
            # Only the dedicated test stack is removed, never connect-dev data.
            subprocess.run(compose + ["down", "--volumes"], check=True, cwd=ROOT, env=env)
        raise SystemExit(result.returncode)
    env, values = local_environment()
    port = local_port(env.get("POSTGRES_PORT", values.get("POSTGRES_PORT", "55432")), "POSTGRES_PORT")
    env.setdefault("DATABASE_URL", database_url(values, port, values["POSTGRES_DB"]))
    args = ["go", "run", "./cmd/api"] if action == "run" else ["go", "run", "./cmd/migrate", sys.argv[2]]
    os.chdir(BACKEND)
    os.execvpe("go", args, env)


if __name__ == "__main__":
    main()
