#!/usr/bin/env bash
set -euo pipefail

# Generate a Homebrew formula for composepack that installs prebuilt macOS binaries.
#
# Usage:
#   gen-brew-formula.sh <version> <owner/repo> <output_path> <darwin_amd64_sha256> <darwin_arm64_sha256> [<linux_amd64_sha256> <linux_arm64_sha256>]
# Examples:
#   # macOS only
#   ./scripts/gen-brew-formula.sh v0.1.0 composepack/composepack /tmp/composepack.rb \
#     DEADBEEF... C0FFEE...
#   # macOS + Linux
#   ./scripts/gen-brew-formula.sh v0.1.0 composepack/composepack /tmp/composepack.rb \
#     DEADBEEF... C0FFEE... L1NUXAMD64... L1NUXARM64...

if [[ $# -ne 5 && $# -ne 7 ]]; then
  echo "Usage: $0 <version> <owner/repo> <output_path> <darwin_amd64_sha256> <darwin_arm64_sha256> [<linux_amd64_sha256> <linux_arm64_sha256>]" >&2
  exit 1
fi

VERSION="$1"
VERSION_NO_V="${VERSION#v}"
REPO_SLUG="$2" # OWNER/REPO
OUT="$3"
SHA_DARWIN_AMD64="$4"
SHA_DARWIN_ARM64="$5"
SHA_LINUX_AMD64="${6:-}"
SHA_LINUX_ARM64="${7:-}"

OWNER="${REPO_SLUG%%/*}"
REPO="${REPO_SLUG##*/}"

cat >"$OUT" <<RUBY
class Composepack < Formula
  desc "Helm-style templating and packaging for Docker Compose"
  homepage "https://github.com/${OWNER}/${REPO}"
  version "${VERSION_NO_V}"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/${OWNER}/${REPO}/releases/download/${VERSION}/composepack_${VERSION}_darwin_arm64.tar.gz"
      sha256 "${SHA_DARWIN_ARM64}"
    else
      url "https://github.com/${OWNER}/${REPO}/releases/download/${VERSION}/composepack_${VERSION}_darwin_amd64.tar.gz"
      sha256 "${SHA_DARWIN_AMD64}"
    end
  end

  # Linux support when shas are provided
  on_linux do
RUBY

if [[ -n "$SHA_LINUX_AMD64" && -n "$SHA_LINUX_ARM64" ]]; then
  cat >>"$OUT" <<'RUBY'
    if Hardware::CPU.arm?
RUBY
  cat >>"$OUT" <<RUBY
      url "https://github.com/${OWNER}/${REPO}/releases/download/${VERSION}/composepack_${VERSION}_linux_arm64.tar.gz"
      sha256 "${SHA_LINUX_ARM64}"
RUBY
  cat >>"$OUT" <<'RUBY'
    else
RUBY
  cat >>"$OUT" <<RUBY
      url "https://github.com/${OWNER}/${REPO}/releases/download/${VERSION}/composepack_${VERSION}_linux_amd64.tar.gz"
      sha256 "${SHA_LINUX_AMD64}"
RUBY
  cat >>"$OUT" <<'RUBY'
    end
RUBY
else
  # Fallback to building from source on Linux if shas were not provided
  cat >>"$OUT" <<'RUBY'
    odie "Linux tarball checksums not provided in formula; try curl install or build from source"
RUBY
fi

cat >>"$OUT" <<'RUBY'
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
