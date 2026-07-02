# Deployment

This guide covers production-style deployment with GitHub Releases, optional GHCR images, Docker Compose, and systemd.

## Release Artifacts

Tagged releases publish archived binaries for Linux, macOS, and Windows:

- `gonk_<version>_linux_amd64.tar.gz`
- `gonk_<version>_linux_arm64.tar.gz`
- `gonk_<version>_darwin_amd64.tar.gz`
- `gonk_<version>_darwin_arm64.tar.gz`
- `gonk_<version>_windows_amd64.zip`
- `gonk_<version>_windows_arm64.zip`
- `checksums.txt`

Each archive contains:

- `gonk`, the gateway server;
- `gonk-cli`, the operator CLI;
- `gonk.example.yaml`;
- `LICENSE`;
- `README.txt`.

Verify a download:

```bash
sha256sum -c checksums.txt --ignore-missing
```

For OS-specific install commands, see [INSTALL.md](INSTALL.md).

## Container Image

When a container image has been published manually, pull it from GitHub Container Registry:

```bash
docker pull ghcr.io/justvugg/gonk:latest
docker pull ghcr.io/justvugg/gonk:v1.2.1
```

Images are built for `linux/amd64` and `linux/arm64`.

## Docker Compose Production Template

Create an environment file:

```bash
cp deployments/docker/prod.env.example .env
```

Edit `.env`, then start GONK:

```bash
docker compose --env-file .env -f deployments/docker/docker-compose.prod.yml up -d
```

The default compose file mounts `configs/gonk.production.yaml`. Replace `UPSTREAM_URL`, `JWT_SECRET`, `API_KEY_1`, and `GONK_ADMIN_TOKEN` before running it outside a test environment.

## systemd

Install the release binaries:

```bash
sudo install -m 0755 gonk /usr/local/bin/gonk
sudo install -m 0755 gonk-cli /usr/local/bin/gonk-cli
```

Create directories and a service user:

```bash
sudo useradd --system --home /var/lib/gonk --shell /usr/sbin/nologin gonk
sudo install -d -o gonk -g gonk /etc/gonk /var/lib/gonk /var/log/gonk
```

Install configuration:

```bash
sudo install -m 0640 -o root -g gonk configs/gonk.production.yaml /etc/gonk/gonk.yaml
sudo install -m 0640 -o root -g gonk deployments/systemd/gonk.env.example /etc/gonk/gonk.env
```

Edit `/etc/gonk/gonk.env`, then install and start the unit:

```bash
sudo install -m 0644 deployments/systemd/gonk.service /etc/systemd/system/gonk.service
sudo systemctl daemon-reload
sudo systemctl enable --now gonk
sudo systemctl status gonk
```

For privileged ports such as `:443`, the unit already grants `CAP_NET_BIND_SERVICE`.
