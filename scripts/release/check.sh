#!/usr/bin/env sh
set -eu

ROOT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)
cd "$ROOT_DIR"

VERSION="${VERSION:-1.2.1}"
SKIP_DOCKER="${SKIP_DOCKER:-0}"

echo "Running manual GONK release checks for ${VERSION}"
echo

make test
make bench
make build
make package-release VERSION="${VERSION}"

(
	cd dist
	sha256sum -c checksums.txt
)

if [ "$SKIP_DOCKER" = "1" ]; then
	echo "Skipping Docker image build because SKIP_DOCKER=1"
else
	make docker-build VERSION="${VERSION}"
fi

echo
echo "Release checks passed for ${VERSION}"
