# Homebrew Tap Automation

This repository publishes a Homebrew formula on every tagged release (`v*`).

What the CI does:

- Builds multi-arch binaries and packages macOS builds into tarballs
- Computes SHA256 checksums for macOS tarballs
- Generates a `composepack.rb` formula pointing at the release assets
- Optionally pushes the formula to your tap repository

## One-time setup

1) Create a tap repository on GitHub (suggested):

- `composepack/homebrew-tap` (public)
- It must contain a `Formula/` directory (CI will create it if absent)

2) Add repository variables and secrets in this repo (Settings → Secrets and variables → Actions):

- Variable: `HOMEBREW_TAP_REPO` → `composepack/homebrew-tap`
- Secret: `HOMEBREW_TAP_GITHUB_TOKEN` → a PAT with `repo` scope that can push to the tap

That’s it. On the next tag push (`git tag vX.Y.Z && git push --tags`), the CI will update `Formula/composepack.rb` in your tap.

## Install via Homebrew

```bash
brew tap composepack/tap
brew install composepack
```

Once the formula is accepted into homebrew-core, `brew install composepack` will work without tapping.

