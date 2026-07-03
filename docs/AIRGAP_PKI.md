# Airgapped PKI

GONK is designed to run without a public CA, public ACME endpoint, or hosted control plane. In airgapped environments, treat certificate management as a local trust-anchor workflow.

## Model

There are two supported operating models:

1. Bring your own internal PKI.
   Mount certificates issued by your internal CA, Vault, HSM-backed CA, or internal ACME service. Configure GONK with `server.tls.cert_file`, `server.tls.key_file`, and `server.tls.client_ca`.

2. Bootstrap a local offline CA.
   Use `gonk-cli certs bootstrap` to create a local CA, server certificate, and client certificate for smaller deployments, demos, labs, or disconnected edge boxes.

GONK does not need to phone home for either model.

## Bootstrap Order

The trust anchor must exist before mTLS traffic starts:

1. Generate or import the root/intermediate CA.
2. Distribute the CA bundle to every host, container, device, or client that must trust GONK.
3. Generate or import the GONK server certificate.
4. Generate or import client/device certificates.
5. Validate the full bundle with `gonk-cli certs doctor`.
6. Start GONK.

If the CA is not trusted first, services will fail with TLS verification errors even if the leaf certificates are otherwise valid.

## Offline Bootstrap

```bash
gonk-cli certs bootstrap \
  --cn edge-gateway.local \
  --client Device-001 \
  --output ./certs
```

This writes:

- `certs/ca.crt`
- `certs/ca.key`
- `certs/server.crt`
- `certs/server.key`
- `certs/client.crt`
- `certs/client.key`

Server certificates include a DNS or IP SAN based on `--cn`.

## GONK Config

```yaml
server:
  listen: ":8443"
  tls:
    enabled: true
    cert_file: "/certs/server.crt"
    key_file: "/certs/server.key"
    client_ca: "/certs/ca.crt"
    client_auth: "require"
```

Use `client_auth: "require"` when every caller must present a client certificate. Use `client_auth: "request"` for mixed JWT/API-key plus optional client certificate deployments.

## Validate Before Start

```bash
gonk-cli certs doctor \
  -c gonk.yaml \
  --client-cert ./certs/client.crt \
  --server-name edge-gateway.local
```

The doctor checks:

- server certificate file exists and parses;
- private key matches the server certificate;
- client CA parses;
- server certificate verifies against the configured CA;
- optional client certificate verifies against the configured CA;
- certificates are currently valid;
- certificates close to expiry are reported;
- routes requiring mTLS are consistent with `server.tls.client_auth`.

## Rotation

Single GONK instance:

1. Generate new server and/or client certificates.
2. Replace files atomically on disk.
3. Restart GONK.
4. Run `gonk-cli certs doctor` and a smoke test.

Multiple replicas behind a load balancer:

1. Generate new certs from the same trusted CA, or deploy a CA bundle that includes old and new intermediates.
2. Drain one GONK replica.
3. Replace certificate files.
4. Restart that replica.
5. Run `gonk-cli certs doctor`.
6. Return the replica to service.
7. Repeat for the remaining replicas.

This gives zero user-visible downtime when enough replicas are available. A single process still needs a restart for server certificate replacement.

## Internal PKI Integration

For internal ACME, Vault, or cert-manager style setups, let that system own issuance and renewal, then mount the resulting files into GONK. GONK only needs stable file paths in YAML.

Recommended layout:

```text
/etc/gonk/certs/
  ca.crt
  server.crt
  server.key
```

Run `gonk-cli certs doctor` from the same host or container namespace that GONK uses, so file paths match the real deployment.
