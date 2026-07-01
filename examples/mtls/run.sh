#!/usr/bin/env sh
set -eu

COMPOSE_FILE="examples/mtls/docker-compose.yml"
CERT_DIR="examples/mtls/certs"

if docker compose version >/dev/null 2>&1; then
  COMPOSE="docker compose"
elif command -v docker-compose >/dev/null 2>&1; then
  COMPOSE="docker-compose"
else
  echo "Docker Compose v2 or docker-compose is required" >&2
  exit 127
fi

mkdir -p "$CERT_DIR"

$COMPOSE -f "$COMPOSE_FILE" build gonk backend

$COMPOSE -f "$COMPOSE_FILE" run --rm --entrypoint gonk-cli gonk \
  certs generate --type ca --cn "GONK Demo CA" --output /certs

$COMPOSE -f "$COMPOSE_FILE" run --rm --entrypoint gonk-cli gonk \
  certs generate --type server --cn localhost --output /certs \
  --ca-cert /certs/ca.crt --ca-key /certs/ca.key

$COMPOSE -f "$COMPOSE_FILE" run --rm --entrypoint gonk-cli gonk \
  certs generate --type client --cn Device-001 --output /certs \
  --ca-cert /certs/ca.crt --ca-key /certs/ca.key

$COMPOSE -f "$COMPOSE_FILE" up -d

printf 'Waiting for GONK mTLS endpoint'
for _ in $(seq 1 40); do
  if curl -ks --cert "$CERT_DIR/client.crt" --key "$CERT_DIR/client.key" \
    https://localhost:8443/device/ping >/dev/null 2>&1; then
    printf '\n'
    break
  fi
  printf '.'
  sleep 1
done

curl -ks --cert "$CERT_DIR/client.crt" --key "$CERT_DIR/client.key" \
  https://localhost:8443/device/ping
printf '\n'
