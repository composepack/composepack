#!/usr/bin/env bash
set -euo pipefail

REPO="${COMPOSEPACK_REPO:-composepack/composepack}"
VERSION="${1:-latest}"
if [[ -n "${COMPOSEPACK_INSTALL_DIR:-}" ]]; then
  INSTALL_DIR="$COMPOSEPACK_INSTALL_DIR"
else
  INSTALL_DIR="/usr/local/bin"
  if [[ ! -d "$INSTALL_DIR" || ! -w "$INSTALL_DIR" ]]; then
    INSTALL_DIR="${HOME}/.local/bin"
    echo "Installing to $INSTALL_DIR (set COMPOSEPACK_INSTALL_DIR to override)." >&2
  fi
fi

if [[ "$VERSION" == "latest" ]]; then
  if ! command -v curl >/dev/null 2>&1; then
    echo "curl is required to download releases" >&2
    exit 1
  fi
  if ! command -v python3 >/dev/null 2>&1; then
    echo "python3 is required to discover the latest release automatically" >&2
    exit 1
  fi
  VERSION=$(
    REPO="$REPO" python3 - <<'PY'
import json
import os
import sys
import urllib.error
import urllib.request

repo = os.environ["REPO"]
url = f"https://api.github.com/repos/{repo}/releases/latest"
try:
    with urllib.request.urlopen(url) as resp:
        payload = json.load(resp)
except urllib.error.HTTPError as exc:
    if exc.code == 404:
        sys.stderr.write("No published releases found for repository.\n")
    else:
        sys.stderr.write(f"Failed to fetch release metadata: {exc}\n")
    sys.exit(1)
except Exception as exc:
    sys.stderr.write(f"Failed to parse release metadata: {exc}\n")
    sys.exit(1)

tag = payload.get("tag_name")
if not tag:
    sys.stderr.write("Latest release response did not include tag_name.\n")
    sys.exit(1)
print(tag)
PY
  ) || {
    echo "Unable to determine latest release. Specify a version (e.g., install.sh v0.1.0) or publish a release." >&2
    exit 1
  }
  VERSION="${VERSION//$'\n'/}"
fi

OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)
case "$ARCH" in
  x86_64|amd64)
    ARCH=amd64
    ;;
  arm64|aarch64)
    ARCH=arm64
    ;;
  *)
    echo "Unsupported architecture: $ARCH" >&2
    exit 1
    ;;
esac

TARBALL="composepack_${VERSION}_${OS}_${ARCH}.tar.gz"
URL="https://github.com/${REPO}/releases/download/${VERSION}/${TARBALL}"

TMP_DIR=$(mktemp -d)
trap 'rm -rf "$TMP_DIR"' EXIT

curl -fSL "$URL" -o "$TMP_DIR/$TARBALL"
tar -C "$TMP_DIR" -xzf "$TMP_DIR/$TARBALL"
chmod +x "$TMP_DIR/composepack"
mkdir -p "$INSTALL_DIR"
install -m 0755 "$TMP_DIR/composepack" "$INSTALL_DIR/composepack"

echo "composepack ${VERSION} installed to ${INSTALL_DIR}/composepack"
