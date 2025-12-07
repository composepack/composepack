#!/usr/bin/env bash
set -euo pipefail

BUILD_DIR="${BUILD_DIR:-out}"
BIN_NAME="${BIN_NAME:-composepack}"
INSTALL_DIR="${INSTALL_DIR:-$HOME/.local/bin}"

# check if artifact is already built
if [[ ! -f "$BUILD_DIR/$BIN_NAME" ]]; then
    echo "artifact not found in $BUILD_DIR/$BIN_NAME"
    exit 1
fi

mkdir -p "$INSTALL_DIR"
cp -a "$BUILD_DIR/$BIN_NAME" "$INSTALL_DIR/$BIN_NAME"

echo "composepack installed to $INSTALL_DIR/$BIN_NAME"