#!/usr/bin/env python3
"""Local Go commands with private environment loading; never print DSNs."""

import os
from pathlib import Path
import subprocess
import sys
from urllib.parse import quote

ROOT = Path(__file__).resolve().parents[2]


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
        if key in ("API_LISTEN_ADDR", "APP_ENV", "DATABASE_URL") or key.startswith("DB_"):
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
        raise SystemExit("Usage: api_local.py run|migrate up|down|status|integration")
    action = sys.argv[1]
    if action not in ("run", "migrate", "integration"):
        raise SystemExit("Unsupported local backend command.")
    if action == "migrate" and sys.argv[2:] not in (["up"], ["down"], ["status"]):
        raise SystemExit("Usage: api_local.py migrate up|down|status")
    env, values = local_environment()
    if action == "integration":
        port = local_port(env.get("TEST_POSTGRES_PORT", "15432"), "TEST_POSTGRES_PORT")
        compose = ["docker", "compose", "--env-file", str(ROOT / ".env"), "-f", str(ROOT / "infra/compose/test.yaml")]
        env["TEST_DATABASE_URL"] = database_url(values, port, "connect_stage1_test")
        try:
            subprocess.run(compose + ["up", "-d", "--wait"], check=True, cwd=ROOT, env=env)
            result = subprocess.run(["go", "test", "-tags=integration", "-race", "-count=1", "-timeout=90s", "./tests/integration"], cwd=ROOT / "services/api", env=env)
        finally:
            # Only the dedicated test stack is removed, never connect-dev data.
            subprocess.run(compose + ["down", "--volumes"], check=True, cwd=ROOT, env=env)
        raise SystemExit(result.returncode)
    port = local_port(env.get("POSTGRES_PORT", values.get("POSTGRES_PORT", "55432")), "POSTGRES_PORT")
    env.setdefault("DATABASE_URL", database_url(values, port, values["POSTGRES_DB"]))
    args = ["go", "run", "./cmd/api"] if action == "run" else ["go", "run", "./cmd/migrate", sys.argv[2]]
    os.chdir(ROOT / "services/api")
    os.execvpe("go", args, env)


if __name__ == "__main__":
    main()
