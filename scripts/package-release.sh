#!/usr/bin/env sh
set -eu

ROOT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
cd "$ROOT_DIR"

VERSION="${VERSION:-1.2.1}"
BIN_DIR="${BIN_DIR:-bin}"
DIST_DIR="${DIST_DIR:-dist}"

require_file() {
	if [ ! -f "$1" ]; then
		echo "missing release binary: $1" >&2
		exit 1
	fi
}

write_readme() {
	package_dir="$1"
	platform="$2"
	exe_suffix="$3"

	{
		echo "GONK ${VERSION} (${platform})"
		echo
		echo "Included files:"
		echo "- gonk${exe_suffix}: gateway server"
		echo "- gonk-cli${exe_suffix}: operator CLI"
		echo "- gonk.example.yaml: complete example configuration"
		echo
		echo "Quick check:"
		if [ "$exe_suffix" = ".exe" ]; then
			echo "  .\\gonk.exe -version"
			echo "  .\\gonk-cli.exe --version"
			echo
			echo "Run:"
			echo "  .\\gonk.exe -config gonk.yaml"
		else
			echo "  chmod +x gonk gonk-cli"
			echo "  ./gonk -version"
			echo "  ./gonk-cli --version"
			echo
			echo "Run:"
			echo "  ./gonk -config gonk.yaml"
		fi
		echo
		echo "Documentation: https://github.com/JustVugg/gonk"
	} > "$package_dir/README.txt"
}

copy_common_files() {
	package_dir="$1"

	cp LICENSE "$package_dir/LICENSE"
	cp configs/gonk.example.yaml "$package_dir/gonk.example.yaml"
}

package_tar() {
	os="$1"
	arch="$2"
	server_bin="$BIN_DIR/gonk-${os}-${arch}"
	cli_bin="$BIN_DIR/gonk-cli-${os}-${arch}"
	package_name="gonk_${VERSION}_${os}_${arch}"
	package_dir="$DIST_DIR/$package_name"

	require_file "$server_bin"
	require_file "$cli_bin"

	mkdir -p "$package_dir"
	cp "$server_bin" "$package_dir/gonk"
	cp "$cli_bin" "$package_dir/gonk-cli"
	copy_common_files "$package_dir"
	write_readme "$package_dir" "${os}/${arch}" ""

	tar -czf "$DIST_DIR/${package_name}.tar.gz" -C "$DIST_DIR" "$package_name"
	rm -rf "$package_dir"
}

package_zip() {
	os="$1"
	arch="$2"
	server_bin="$BIN_DIR/gonk-${os}-${arch}.exe"
	cli_bin="$BIN_DIR/gonk-cli-${os}-${arch}.exe"
	package_name="gonk_${VERSION}_${os}_${arch}"
	package_dir="$DIST_DIR/$package_name"

	if ! command -v zip >/dev/null 2>&1; then
		echo "zip is required to create Windows release archives" >&2
		exit 1
	fi

	require_file "$server_bin"
	require_file "$cli_bin"

	mkdir -p "$package_dir"
	cp "$server_bin" "$package_dir/gonk.exe"
	cp "$cli_bin" "$package_dir/gonk-cli.exe"
	copy_common_files "$package_dir"
	write_readme "$package_dir" "${os}/${arch}" ".exe"

	(
		cd "$DIST_DIR"
		zip -qr "${package_name}.zip" "$package_name"
	)
	rm -rf "$package_dir"
}

rm -rf "$DIST_DIR"
mkdir -p "$DIST_DIR"

package_tar linux amd64
package_tar linux arm64
package_tar darwin amd64
package_tar darwin arm64
package_zip windows amd64
package_zip windows arm64

(
	cd "$DIST_DIR"
	sha256sum * > checksums.txt
)

echo "Release packages written to $DIST_DIR"
