#!/usr/bin/env python3
"""Exercise the bundled coturn client; never print local test credentials."""

import ipaddress
from pathlib import Path
import re
import subprocess
import time

ROOT = Path(__file__).resolve().parents[2]
COMPOSE = [
    "docker", "compose", "--env-file", ".env", "-f",
    "infra/compose/compose.yaml", "--profile", "turn", "exec", "-T", "coturn",
]


def run(args):
    try:
        return subprocess.run(
            COMPOSE + args, cwd=ROOT, text=True, stdout=subprocess.PIPE,
            stderr=subprocess.STDOUT, timeout=15, check=False,
        )
    except subprocess.TimeoutExpired:
        raise SystemExit("Local coturn test timed out; inspect container health.") from None


def main():
    values = dict(
        line.split("=", 1) for line in (ROOT / ".env").read_text().splitlines()
        if line and not line.startswith("#")
    )
    password = values["TURN_TEST_PASSWORD"]
    # Wait for the fixture, without assuming Compose start means TURN is ready.
    for _ in range(10):
        binding = run(["turnutils_stunclient", "127.0.0.1"])
        if binding.returncode == 0 and "reflexive addr:" in binding.stdout:
            break
        time.sleep(1)
    else:
        raise SystemExit("Local STUN binding failed.")
    address_result = run(["hostname", "-i"])
    addresses = address_result.stdout.split() if address_result.returncode == 0 else []
    address = next(
        (value for value in addresses
         if ipaddress.ip_address(value).version == 4
         and not ipaddress.ip_address(value).is_loopback), None,
    )
    if address is None:
        raise SystemExit("No container-local IPv4 relay address found.")
    args = ["turnutils_uclient", "-u", "local-test", "-y", "-c", "-n", "2", "-m", "2"]
    valid = run(args + ["-w", password, address])
    totals = re.findall(r"tot_send_msgs=(\d+), tot_recv_msgs=(\d+)", valid.stdout)
    if valid.returncode != 0 or not totals or totals[-1] != ("4", "4"):
        raise SystemExit("Authenticated local TURN relay test failed (expected 4/4 packets).")
    invalid = run(args + ["-w", "intentionally-invalid", address])
    if invalid.returncode == 0 or "Cannot complete Allocation" not in invalid.stdout:
        raise SystemExit("Invalid-credential TURN allocation was not rejected as expected.")
    print("STUN binding passed; local TURN relayed 4/4 packets; invalid credentials rejected.")
    print("This verifies container-local interoperability only, not phone connectivity or NAT traversal.")


if __name__ == "__main__":
    main()
