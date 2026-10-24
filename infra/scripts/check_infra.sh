#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/../.."
compose=(docker compose --env-file .env -f infra/compose/compose.yaml)
"${compose[@]}" exec -T postgres sh -ec 'PGPASSWORD="$POSTGRES_PASSWORD" psql -h 127.0.0.1 -U "$POSTGRES_USER" -d "$POSTGRES_DB" -v ON_ERROR_STOP=1 -c "SELECT 1;"'
if [[ "${1:-}" == "cache" ]]; then
  # redis-cli can exit successfully while reporting an authentication error.
  "${compose[@]}" --profile cache exec -T redis sh -ec 'REDISCLI_AUTH="$REDIS_PASSWORD" redis-cli ping | grep -qx PONG'
  echo 'Redis authenticated PING passed.'
fi
