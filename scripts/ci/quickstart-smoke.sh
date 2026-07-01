#!/usr/bin/env bash
set -euo pipefail

COMPOSE_FILE="${COMPOSE_FILE:-examples/quickstart/docker-compose.yml}"
BASE_URL="${BASE_URL:-http://localhost:8080}"

if docker compose version >/dev/null 2>&1; then
  COMPOSE=(docker compose)
elif command -v docker-compose >/dev/null 2>&1; then
  COMPOSE=(docker-compose)
else
  echo "Docker Compose v2 or docker-compose is required" >&2
  exit 127
fi

cleanup() {
  "${COMPOSE[@]}" -f "$COMPOSE_FILE" down --remove-orphans -v >/dev/null 2>&1 || true
}

dump_logs() {
  "${COMPOSE[@]}" -f "$COMPOSE_FILE" logs --no-color || true
}

trap cleanup EXIT
trap 'dump_logs' ERR

"${COMPOSE[@]}" -f "$COMPOSE_FILE" up -d --build

echo "Waiting for public route..."
for _ in $(seq 1 60); do
  if curl -fsS "$BASE_URL/public/ping" >/dev/null 2>&1; then
    break
  fi
  sleep 1
done

curl -fsS "$BASE_URL/public/ping" >/dev/null

token_output="$(
  "${COMPOSE[@]}" -f "$COMPOSE_FILE" run --rm --entrypoint gonk-cli gonk \
    auth jwt generate \
    --role user \
    --scopes read:api \
    --user-id ci-smoke \
    --expiry 10m
)"
token="$(printf '%s\n' "$token_output" | awk '/^eyJ/{print; exit}')"

if [ -z "$token" ]; then
  echo "failed to extract JWT from gonk-cli output" >&2
  printf '%s\n' "$token_output" >&2
  exit 1
fi

curl -fsS -H "Authorization: Bearer $token" "$BASE_URL/api/ping" >/dev/null
curl -fsS "$BASE_URL/metrics" | grep -q "gonk_http_requests_total"

echo "Quickstart smoke test passed"
