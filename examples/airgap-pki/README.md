# Airgap PKI Example

This example runs GONK with an offline CA and required client certificates. It does not need internet access after images and binaries are already available locally.

## 1. Generate Offline Certificates

From the repository root:

```bash
./bin/gonk-cli certs bootstrap \
  --cn localhost \
  --client Device-001 \
  --output examples/airgap-pki/certs
```

If files already exist and you intentionally want to replace them:

```bash
./bin/gonk-cli certs bootstrap \
  --cn localhost \
  --client Device-001 \
  --output examples/airgap-pki/certs \
  --force
```

## 2. Validate The Bundle

```bash
docker compose -f examples/airgap-pki/docker-compose.yml run --rm --entrypoint gonk-cli gonk \
  certs doctor \
  -c /etc/gonk/gonk.yaml \
  --client-cert /certs/client.crt \
  --server-name localhost
```

`certs doctor` should run with the same file paths GONK will use. In this example, those paths exist inside the container as `/certs/...`.

## 3. Run

```bash
docker compose -f examples/airgap-pki/docker-compose.yml up --build
```

Legacy Compose:

```bash
docker-compose -f examples/airgap-pki/docker-compose.yml up --build
```

## 4. Call With Client Certificate

```bash
curl -k \
  --cert examples/airgap-pki/certs/client.crt \
  --key examples/airgap-pki/certs/client.key \
  https://localhost:8443/device/ping
```

Without the client certificate, TLS handshake verification should fail.

## 5. Stop

```bash
docker compose -f examples/airgap-pki/docker-compose.yml down --remove-orphans
```

## Rotation

For a single local instance, regenerate certificates with `--force`, rerun `certs doctor`, then restart the container. For multiple replicas, rotate one replica at a time behind a load balancer.
