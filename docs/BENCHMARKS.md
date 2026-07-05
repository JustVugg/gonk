# Benchmarks

GONK benchmarks are run locally. The project intentionally does not rely on GitHub Actions for performance checks.

## Quick Run

```bash
make bench
```

Use a longer benchmark window when comparing changes:

```bash
BENCH_TIME=10s make bench
```

Run only matching benchmarks:

```bash
BENCH_PATTERN=BenchmarkHTTPProxy BENCH_TIME=10s make bench
```

The current benchmark suite covers the HTTP reverse proxy hot path, including single-upstream proxying and route-prefix stripping. It prints `ns/op`, `B/op`, and `allocs/op`.

## Release Baseline

Before a manual release, run:

```bash
make release-check
```

This runs tests, local benchmarks, builds release binaries, packages archives, verifies checksums, and builds the Docker image unless `SKIP_DOCKER=1` is set.

For a performance-sensitive release, save the benchmark output with the release notes. Compare results only on the same machine, OS, Go version, and power profile.

## What To Watch

- `ns/op`: proxy overhead per request in the local benchmark harness.
- `B/op`: heap allocation pressure per request.
- `allocs/op`: number of allocations per request.
- Large changes after touching proxy, middleware, auth, cache, or load-balancing code.

## Next Benchmark Targets

- End-to-end HTTP latency with a running `gonk` binary and a local backend.
- Rate-limit, cache hit, and auth middleware microbenchmarks.
- mTLS handshake and steady-state request overhead.
- WebSocket and gRPC smoke benchmarks.
