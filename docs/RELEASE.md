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

- `gonk-*` server binaries;
- `gonk-cli-*` CLI binaries;
- `checksums.txt`.

## Artifacts

The release workflow builds Linux, macOS, and Windows binaries using `make build-all`, then publishes SHA-256 checksums. Docker images are still built in CI; pushing versioned registry images is a future release step.
