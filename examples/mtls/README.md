# mTLS Demo

This demo runs GONK with TLS enabled and requires client certificates for the protected route.

Prerequisite: Docker Compose v2.

## Run

```bash
sh examples/mtls/run.sh
```

The script:

- builds the demo images;
- generates a local CA;
- generates server and client certificates signed by that CA;
- starts GONK on `https://localhost:8443`;
- calls `/device/ping` with the generated client certificate.

## Manual Request

```bash
curl -ks \
  --cert examples/mtls/certs/client.crt \
  --key examples/mtls/certs/client.key \
  https://localhost:8443/device/ping
```

Without the client certificate, the TLS handshake is rejected.

## Stop

```bash
docker compose -f examples/mtls/docker-compose.yml down --remove-orphans
```
