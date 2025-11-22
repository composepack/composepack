#!/usr/bin/env bash
set -euo pipefail

# Generate a Homebrew formula for composepack that installs prebuilt macOS binaries.
#
# Usage:
#   gen-brew-formula.sh <version> <owner/repo> <output_path> <darwin_amd64_sha256> <darwin_arm64_sha256>
# Example:
#   ./scripts/gen-brew-formula.sh v0.1.0 composepack/composepack /tmp/composepack.rb \
#     0123...cafe 4567...beef

if [[ $# -ne 5 ]]; then
  echo "Usage: $0 <version> <owner/repo> <output_path> <darwin_amd64_sha256> <darwin_arm64_sha256>" >&2
  exit 1
fi

VERSION="$1"
REPO_SLUG="$2" # OWNER/REPO
OUT="$3"
SHA_DARWIN_AMD64="$4"
SHA_DARWIN_ARM64="$5"

OWNER="${REPO_SLUG%%/*}"
REPO="${REPO_SLUG##*/}"

cat >"$OUT" <<RUBY
class Composepack < Formula
  desc "Helm-style templating and packaging for Docker Compose"
  homepage "https://github.com/${OWNER}/${REPO}"
  version "${VERSION}"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/${OWNER}/${REPO}/releases/download/${VERSION}/composepack_${VERSION}_darwin_arm64.tar.gz"
      sha256 "${SHA_DARWIN_ARM64}"
    else
      url "https://github.com/${OWNER}/${REPO}/releases/download/${VERSION}/composepack_${VERSION}_darwin_amd64.tar.gz"
      sha256 "${SHA_DARWIN_AMD64}"
    end
  end

  def install
    bin.install "composepack"
  end

  test do
    assert_match "composepack", shell_output("#{bin}/composepack version")
  end
end
RUBY

echo "Wrote Homebrew formula to: $OUT"
