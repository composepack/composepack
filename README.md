# ComposePack

ComposePack packages Docker Compose applications the way Helm packages charts. You point it at a chart directory (templated Compose fragments + assets) and ComposePack renders a release, writes a self-contained runtime directory, and runs `docker compose` for you.

## Why use ComposePack?

* **Templating + overrides** – system defaults live in the chart, end-users provide `values.yaml` overlays or `--set` flags so there’s a clean separation between product and customer configs.
* **Safe runtimes** – each release lives in `.cpack-releases/<name>/` with a merged `docker-compose.yaml` and rendered files.
* **Simple CLI** – thin wrapper over `docker compose` (`install`, `up`, `down`, `logs`, `ps`, `version`).

## Installation

### macOS / Linux

```bash
curl -fsSL https://raw.githubusercontent.com/GareArc/composepack/main/scripts/install.sh | bash
```

* Installs to `/usr/local/bin/composepack` when writable, otherwise falls back to `~/.local/bin/composepack`. Override with `COMPOSEPACK_INSTALL_DIR`.
* Set `COMPOSEPACK_REPO` if releases live under a different GitHub org/repo.
* Requires `curl` (and `python3` only when discovering the “latest” tag automatically). If you haven’t published a release yet, pass a specific tag/version to the script once it exists.

Uninstall with:

```bash
./scripts/uninstall.sh
```

### Windows (PowerShell)

```powershell
./scripts/install.ps1 -Version v1.0.0 -InstallDir "$env:ProgramFiles\ComposePack"
```

Uninstall:

```powershell
./scripts/uninstall.ps1
```

### Build from source

```bash
git clone https://github.com/GareArc/composepack.git
cd composepack
make build   # go build ./...
```

`make generate` runs `go generate ./...` (Wire DI, etc.) if you change providers.

## Create a chart

1. Choose a directory for your chart (e.g., `charts/example`).
2. Run the init command:

   ```bash
   composepack init charts/example --name example --version 0.1.0
   ```

3. You’ll get this starter structure:

   ```
   charts/example/
     Chart.yaml
     values.yaml
     templates/
       compose/00-app.tpl.yaml
       files/config/app.env.tpl
       helpers/_helpers.tpl
     files/
       config/
   ```

4. Edit `values.yaml`, `templates/compose/*.tpl.yaml`, and `templates/files/*.tpl` to match your services and runtime assets. Anything under `templates/helpers/` contains reusable snippets for `{{ include }}`.

See [`docs/PRD.md`](docs/PRD.md) for full details on chart layout, helper functions, and runtime expectations.

To package the chart for distribution, run:

```bash
composepack package charts/example --destination dist
# creates dist/example-0.1.0.cpack.tgz
```

You can optionally provide `--output mychart.cpack.tgz` or `--force` to overwrite existing archives.

You can also manually create a `.cpack.tgz` (or `.tgz/.tar.gz`) archive if needed:

```bash
tar -czf example.cpack.tgz -C charts/example .
```

## Quick start

1. **Prepare a chart** following the structure in [`PRD.md`](docs/PRD.md) (`Chart.yaml`, `templates/`, `files/`, etc.).
2. **Install the chart** into a release:

   ```bash
   composepack install ./charts/example --name my-release -f values-prod.yaml --auto-start
   ```

   This renders `.cpack-releases/my-release/` and runs `docker compose up -d`.

3. **Iterate**:

   ```bash
   composepack template my-release --chart ./charts/example   # render only
   composepack up my-release --chart ./charts/example -f overrides.yaml
   composepack logs my-release --follow
   composepack down my-release --volumes
   composepack ps my-release
   composepack version
   ```

All runtime files live under `.cpack-releases/<release>/`. You can `cd` into that directory and run `docker compose` manually if needed. `composepack install` accepts either a chart directory or a packaged archive (`.cpack`, `.tgz`, `.tar.gz`, or even an HTTPS/HTTP URL pointing to one), so customers can run `composepack package`, upload the archive somewhere, and their users can do `composepack install https://example.com/mychart.cpack.tgz --name prod --auto-start`.

## Runtime layout (for reference)

```
.cpack-releases/<release>/
  docker-compose.yaml   # merged compose file
  files/                # rendered scripts/configs referenced in templates
  release.json          # metadata describing chart + values
```

You rarely need to edit these manually, but it helps to know where ComposePack keeps things.

## Contributing

* Use `make fmt`, `make test`, `make build`, and `make generate` when developing.
* CI (`.github/workflows/ci.yml`) runs gofmt, go vet, and go test on PRs and pushes to `main`.
* Tag releases (`git tag vX.Y.Z && git push --tags`) to trigger the release workflow, which cross-compiles binaries and uploads them to GitHub Releases.
