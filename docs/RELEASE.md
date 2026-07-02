# Manual Release Process

GONK releases are created manually to avoid GitHub Actions usage.

## Checklist

1. Run local verification:

```bash
make test
make build
make demo-smoke
make docker-build
```

2. Build release archives locally:

```bash
make package-release VERSION=1.2.1
```

3. Verify checksums locally:

```bash
cd dist
sha256sum -c checksums.txt
cd ..
```

4. Create and push a version tag:

```bash
git tag v1.2.1
git push origin v1.2.1
```

5. Create the GitHub release manually and upload every file from `dist/`.

6. Verify the GitHub release contains:

- archived Linux, macOS, and Windows packages;
- `checksums.txt`.

## Artifacts

`make package-release` builds Linux, macOS, and Windows binaries, then writes SHA-256 checksums and archives:

- `gonk_<version>_linux_amd64.tar.gz`;
- `gonk_<version>_linux_arm64.tar.gz`;
- `gonk_<version>_darwin_amd64.tar.gz`;
- `gonk_<version>_darwin_arm64.tar.gz`;
- `gonk_<version>_windows_amd64.zip`;
- `gonk_<version>_windows_arm64.zip`;
- `checksums.txt`.

## Optional Container Publishing

Publishing container images is optional and should be done manually only when needed.

Build and push a local image:

```bash
docker build -f deployments/docker/Dockerfile -t ghcr.io/justvugg/gonk:v1.2.1 .
docker push ghcr.io/justvugg/gonk:v1.2.1
```

Run `make package-release VERSION=1.2.1` locally to inspect the exact archives before tagging.
