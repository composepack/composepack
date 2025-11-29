# ComposePack Naming System

This document clarifies the different types of names used in ComposePack and how they relate to each other.

## Three Types of Names

### 1. Chart Name

**What it is:** The name of the chart package itself, defined in `Chart.yaml`

**Where it comes from:** The `name` field in `Chart.yaml`

**Example:**

```yaml
# Chart.yaml
name: example
version: 0.1.0
```

**Used for:**

- Identifying the chart package
- Metadata in `release.json`
- Template context (`.Chart.Name`)

**Key point:** This is a property of the chart, not the deployment.

---

### 2. Release Name

**What it is:** The name you give to a specific deployment instance

**Where it comes from:** The `--name` flag (install) or positional argument (up, diff, etc.)

**Examples:**

- `myapp` - your production deployment
- `staging` - your staging environment
- `dev` - your development instance
- `prod-api` - a specific production service

**Used for:**

- Creating the runtime directory: `.cpack-releases/<release-name>/`
- Docker Compose project name
- Identifying which deployment you're operating on

**Key point:** You can deploy the same chart multiple times with different release names.

---

### 3. Chart Source

**What it is:** The path or URL where the chart is located

**Where it comes from:** Positional argument (install) or `--chart` flag

**Examples:**

- `charts/example` - local directory
- `example.cpack.tgz` - packaged archive
- `https://example.com/charts/app-1.0.0.cpack.tgz` - remote URL

**Used for:**

- Loading the chart
- Stored in `release.json` for auto-resolution

**Key point:** This tells ComposePack where to find the chart files.

---

## How They Work Together

### Example 1: Installing a Chart

```bash
composepack install charts/example --name myapp
```

**Breakdown:**

- **Chart Source:** `charts/example` (where the chart is)
- **Chart Name:** `example` (from Chart.yaml inside that directory)
- **Release Name:** `myapp` (what you call this deployment)

**Result:**

- Creates `.cpack-releases/myapp/` directory
- Stores chart source `charts/example` in metadata
- Chart metadata shows name `example`

### Example 2: Operating on a Release

```bash
composepack up myapp
composepack diff myapp
composepack logs myapp
```

**Breakdown:**

- **Release Name:** `myapp` (identifies which deployment)
- Chart source is auto-resolved from `.cpack-releases/myapp/release.json`

**Key point:** You only need the release name - the chart source is remembered.

### Example 3: Comparing Against Different Chart

```bash
composepack diff myapp --chart charts/example-v2
```

**Breakdown:**

- **Release Name:** `myapp` (existing deployment)
- **Chart Source:** `charts/example-v2` (override - compare against different chart)

**Key point:** You can override the auto-resolved chart source.

---

## Common Patterns

### Pattern 1: Same Chart, Multiple Environments

```bash
# Deploy the same chart to different environments
composepack install charts/myapp --name prod -f values-prod.yaml
composepack install charts/myapp --name staging -f values-staging.yaml
composepack install charts/myapp --name dev -f values-dev.yaml
```

- **Chart Source:** Same (`charts/myapp`)
- **Chart Name:** Same (`myapp` from Chart.yaml)
- **Release Names:** Different (`prod`, `staging`, `dev`)

### Pattern 2: Different Charts, Same Release Name

```bash
# Upgrade a release to a new chart version
composepack install charts/myapp-v1 --name myapp
composepack diff myapp --chart charts/myapp-v2  # Preview changes
composepack install charts/myapp-v2 --name myapp  # Upgrade
```

- **Release Name:** Same (`myapp`)
- **Chart Sources:** Different (`charts/myapp-v1` vs `charts/myapp-v2`)

### Pattern 3: Preview Before Install

```bash
# See what would be created
composepack diff new-release --chart charts/myapp

# Then install it
composepack install charts/myapp --name new-release
```

- **Release Name:** `new-release` (doesn't exist yet)
- **Chart Source:** Required (`--chart`) since release doesn't exist

---

## Command Reference

### `install <chart-source> --name <release-name>`

- **Positional arg:** Chart source (required)
- **`--name` flag:** Release name (required)
- **Result:** Creates a new release

### `up <release-name> [--chart <chart-source>]`

- **Positional arg:** Release name (required)
- **`--chart` flag:** Optional - if provided, re-renders from that chart
- **Result:** Starts/updates the release

### `diff <release-name> [--chart <chart-source>]`

- **Positional arg:** Release name (required)
- **`--chart` flag:** Optional - auto-resolved from release if omitted
- **Result:** Shows what would change

### `down <release-name>`

- **Positional arg:** Release name (required)
- **Result:** Stops the release

### `logs <release-name>`

- **Positional arg:** Release name (required)
- **Result:** Shows logs for the release

### `ps <release-name>`

- **Positional arg:** Release name (required)
- **Result:** Shows container status for the release

---

## Quick Reference Table

| Term             | Definition                  | Example          | Where Defined                    |
| ---------------- | --------------------------- | ---------------- | -------------------------------- |
| **Chart Name**   | Name of the chart package   | `example`        | `Chart.yaml`                     |
| **Release Name** | Name of deployment instance | `myapp`, `prod`  | `--name` flag or positional arg  |
| **Chart Source** | Path/URL to chart location  | `charts/example` | Positional arg or `--chart` flag |

---

## FAQ

### Q: Can I use the same release name twice?

**A:** Yes, but it will overwrite the existing release. Use `diff` first to see what would change.

### Q: Can I change the chart name after installing?

**A:** The chart name comes from Chart.yaml. If you want to use a different chart, install with a different chart source (or upgrade the existing release).

### Q: What if I move the chart directory?

**A:** The chart source is stored in `release.json`. If you move it, you'll need to either:

- Move it back
- Reinstall with the new path
- Use `--chart` flag to override

### Q: Can two releases use the same chart?

**A:** Yes! That's the whole point - one chart, multiple deployments.

### Q: What's the difference between chart name and release name?

**A:**

- **Chart name** = what the package is called (like a product name)
- **Release name** = what you call your deployment (like an instance name)

Think of it like: Chart = "WordPress", Release = "my-blog" or "company-news"

---

## Summary

- **Chart Name** = Product name (from Chart.yaml)
- **Release Name** = Instance name (your choice)
- **Chart Source** = Where to find the chart (path/URL)

When in doubt: **Release name** is what you use most often in commands after installation.
