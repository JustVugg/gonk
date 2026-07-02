# Installation

GONK can run as a downloaded binary, a Docker container, or a locally built Go binary.

## Download A Release

Use the latest GitHub Release for prebuilt binaries:

https://github.com/JustVugg/gonk/releases/latest

Release archives include `gonk`, the gateway server, and `gonk-cli`, the operator CLI.

## Linux

Download the archive for your CPU:

- `gonk_<version>_linux_amd64.tar.gz` for most servers and desktops.
- `gonk_<version>_linux_arm64.tar.gz` for ARM edge devices and Raspberry Pi-class hardware.

Install:

```bash
VERSION=1.2.1
tar -xzf "gonk_${VERSION}_linux_amd64.tar.gz"
cd "gonk_${VERSION}_linux_amd64"
chmod +x gonk gonk-cli
./gonk -version
./gonk-cli --version
```

Optional system install:

```bash
sudo install -m 0755 gonk /usr/local/bin/gonk
sudo install -m 0755 gonk-cli /usr/local/bin/gonk-cli
```

## Windows

Download:

- `gonk_<version>_windows_amd64.zip` for most Windows machines.
- `gonk_<version>_windows_arm64.zip` for Windows on ARM.

PowerShell:

```powershell
$Version = "1.2.1"
Expand-Archive ".\gonk_${Version}_windows_amd64.zip"
cd ".\gonk_${Version}_windows_amd64"
.\gonk.exe -version
.\gonk-cli.exe --version
```

Run with a config file:

```powershell
.\gonk.exe -config .\gonk.example.yaml
```

## macOS

Download:

- `gonk_<version>_darwin_amd64.tar.gz` for Intel Macs.
- `gonk_<version>_darwin_arm64.tar.gz` for Apple Silicon.

```bash
VERSION=1.2.1
tar -xzf "gonk_${VERSION}_darwin_arm64.tar.gz"
cd "gonk_${VERSION}_darwin_arm64"
chmod +x gonk gonk-cli
./gonk -version
./gonk-cli --version
```

macOS may require allowing the downloaded binaries in System Settings because they are not notarized yet.

## Docker

If a container image has been published for the release:

```bash
docker pull ghcr.io/justvugg/gonk:v1.2.1
```

Run with a local config:

```bash
docker run --rm -p 8080:8080 \
  -v "$PWD/configs/gonk.production.yaml:/etc/gonk/gonk.yaml:ro" \
  --env-file .env \
  ghcr.io/justvugg/gonk:v1.2.1
```

## Build From Source

Requirements: Go 1.21+ and Make.

```bash
git clone https://github.com/JustVugg/gonk
cd gonk
make build
./bin/gonk -version
./bin/gonk-cli --version
```

Build all release packages locally:

```bash
make package-release VERSION=1.2.1
```

The archives are written to `dist/`.
