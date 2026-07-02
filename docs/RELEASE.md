# Release Process

GONK releases are tag-driven. Tags that match `v*` trigger the GitHub Actions release workflow.

## Checklist

1. Ensure CI is green on `main`.
2. Run local verification:

```bash
make test
make build
make demo-smoke
```

3. Update user-facing docs if commands or config changed.
4. Create and push a version tag:

```bash
git tag v1.2.0
git push origin v1.2.0
```

5. Verify the GitHub release contains:

- archived Linux, macOS, and Windows packages;
- `checksums.txt`.
- a GHCR image tagged as `vX.Y.Z`, `X.Y.Z`, and `latest`.

## Artifacts

The release workflow builds Linux, macOS, and Windows binaries using `make package-release`, then publishes SHA-256 checksums and archives:

- `gonk_<version>_linux_amd64.tar.gz`;
- `gonk_<version>_linux_arm64.tar.gz`;
- `gonk_<version>_darwin_amd64.tar.gz`;
- `gonk_<version>_darwin_arm64.tar.gz`;
- `gonk_<version>_windows_amd64.zip`;
- `gonk_<version>_windows_arm64.zip`;
- `checksums.txt`.

It also publishes a multi-architecture image to GHCR:

```bash
docker pull ghcr.io/justvugg/gonk:v1.2.0
docker pull ghcr.io/justvugg/gonk:1.2.0
docker pull ghcr.io/justvugg/gonk:latest
```

Run `make package-release VERSION=1.2.0` locally to inspect the exact archives before tagging.
