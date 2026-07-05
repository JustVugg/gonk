#!/usr/bin/env sh
set -eu

ROOT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)
cd "$ROOT_DIR"

BENCH_TIME="${BENCH_TIME:-3s}"
BENCH_PATTERN="${BENCH_PATTERN:-Benchmark}"

echo "Running GONK local benchmarks"
echo "  pattern: ${BENCH_PATTERN}"
echo "  time:    ${BENCH_TIME}"
echo

go test -run '^$' -bench "${BENCH_PATTERN}" -benchtime "${BENCH_TIME}" -benchmem \
	./internal/proxy \
	./internal/cache \
	./internal/middleware
