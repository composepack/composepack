# Drift Detection (Diff Command)

## Overview

The `composepack diff` command enables **drift detection** by comparing your currently deployed release with what would be deployed if you ran `install` or `up` now.

This is similar to Helm's `helm diff` plugin and gives you confidence before making changes.

## Purpose

**Key Question:** *"If I run install now, will it restart my database?"*

The diff command answers this by showing:

* What will change in your docker-compose configuration
* Which services will be affected (added, removed, or modified)
* What file assets will change
* Whether the changes will cause container restarts

## Usage

### Basic Usage

```bash
composepack diff <release> --chart <chart-source>
```

### Common Scenarios

#### 1. Preview changes before upgrading

```bash
# Check what would change if you upgrade to a new chart version
composepack diff myapp --chart charts/myapp-v2.0
```

#### 2. Preview value changes

```bash
# See impact of changing configuration values
composepack diff myapp --chart charts/myapp --set image.tag=v1.5.0
```

#### 3. Check file changes

```bash
# Show detailed diffs for changed files
composepack diff myapp --chart charts/myapp --show-files
```

#### 4. Use custom values file

```bash
# Compare with a new values file
composepack diff myapp --chart charts/myapp -f values-prod.yaml
```

## Command Flags

| Flag            | Short | Description                                         |
| --------------- | ----- | --------------------------------------------------- |
| `--chart`       |       | Chart directory or archive to compare (required)    |
| `--values`      | `-f`  | Values files to include (can specify multiple)      |
| `--set`         |       | Direct value overrides (key=value)                  |
| `--show-files`  |       | Show diffs for changed files in addition to compose |
| `--context`     | `-C`  | Number of context lines in diff output (default: 3) |
| `--runtime-dir` |       | Path to existing release directory                  |

## Output Format

### No Changes

When there are no differences:

```
✓ No changes detected in docker-compose.yaml
```

### Changes Detected

When differences are found:

```
📝 Docker Compose Changes:

-     image: myapp:v1.0
+     image: myapp:v2.0

⚠️  Affected Services:
  • myapp-api (modified)
  • myapp-worker (modified)
```

### File Changes

When using `--show-files`:

```
📁 File Changes:
  Added:
    + config/new-config.yaml
  Modified:
    ~ config/app.env
  Removed:
    - config/old-config.yaml

📄 Detailed File Diffs:

--- a/config/app.env
+++ b/config/app.env
- DATABASE_URL=postgres://old-host:5432/db
+ DATABASE_URL=postgres://new-host:5432/db
```

## Understanding the Impact

### Service Status Indicators

* **`(added)`** - New service will be created
* **`(removed)`** - Existing service will be stopped and removed
* **`(modified)`** - Service configuration changed; may cause restart

### When Will Containers Restart?

Containers will typically restart when:

* **Image changes** - Different image or tag
* **Environment variables change** - New or modified env vars
* **Volume mounts change** - Different paths or configurations
* **Network settings change** - Ports, networks, etc.
* **Command/entrypoint changes** - Different startup commands

Containers will NOT restart when:

* **Only comments change** - Comments in compose file
* **Unrelated services change** - Changes to other services
* **File content changes** - Unless the container reads them on startup

## Integration with Workflow

### Recommended Workflow

```bash
# 1. Check what would change
composepack diff myapp --chart charts/myapp --set version=2.0

# 2. Review the output carefully

# 3. If satisfied, apply the changes
composepack install charts/myapp --name myapp --set version=2.0 --auto-start
```

### CI/CD Integration

```bash
#!/bin/bash
# In your deployment pipeline

# Generate diff and capture output
DIFF_OUTPUT=$(composepack diff production --chart charts/myapp)

# Review or post to PR/Slack
echo "$DIFF_OUTPUT"

# Proceed with deployment if approved
composepack install charts/myapp --name production --auto-start
```

## Error Handling

### Release Not Found

```
release myapp not found (run 'composepack install' first)
```

**Solution:** Install the release first before running diff.

### Chart Required

```
--chart is required to specify what to compare against
```

**Solution:** Always specify `--chart` flag with the chart source.

### Chart Not Found

```
load chart: chart not found at path/to/chart
```

**Solution:** Verify the chart path is correct.

## Examples

### Example 1: Upgrading Application Version

```bash
$ composepack diff prod-app --chart ./charts/app --set app.version=2.1.0

📝 Docker Compose Changes:

-     image: myapp:2.0.0
+     image: myapp:2.1.0

⚠️  Affected Services:
  • prod-app-api (modified)
```

### Example 2: Changing Database Configuration

```bash
$ composepack diff prod-app --chart ./charts/app --set db.host=new-db-host

📝 Docker Compose Changes:

-       DB_HOST: old-db-host
+       DB_HOST: new-db-host

⚠️  Affected Services:
  • prod-app-api (modified)
```

### Example 3: Adding New Service

```bash
$ composepack diff prod-app --chart ./charts/app-with-cache

📝 Docker Compose Changes:

+   prod-app-cache:
+     image: redis:7
+     ports:
+       - "6379:6379"

⚠️  Affected Services:
  • prod-app-cache (added)
```

## Best Practices

1. **Always diff before install** - Make it a habit to preview changes
2. **Review affected services** - Pay special attention to services marked as "modified"
3. **Check file changes** - Use `--show-files` when config files are involved
4. **Communicate changes** - Share diff output with your team before deploying
5. **Test in staging first** - Run diff in staging environment before production
6. **Document decisions** - Keep diff outputs for audit trails

## Limitations

* **Runtime state not detected** - Diff shows configuration changes, not actual container state
* **No volume data comparison** - Content of volumes is not compared
* **No network traffic analysis** - Runtime behavior changes aren't predicted
* **Template logic complexity** - Complex templates may be hard to interpret in diffs

For actual container state, use:

```bash
composepack ps myapp
docker compose ps
```

## Related Commands

* `composepack install` - Apply changes after reviewing diff
* `composepack template` - Render templates without applying
* `composepack ps` - Check current container status
* `composepack up` - Start containers after confirming changes

## Troubleshooting

### Diff shows no changes but I expect changes

**Possible causes:**

1. Values haven't actually changed
2. Using the same chart version
3. Template logic filtering out changes

**Debug steps:**

```bash
# Check current release metadata
cat .cpack-releases/myapp/release.json

# Re-render and inspect
composepack template myapp --chart charts/myapp
cat .cpack-releases/myapp/docker-compose.yaml
```

### Diff is too verbose

**Solution:** Reduce context lines:

```bash
composepack diff myapp --chart charts/myapp -C 1
```

### Need to see full compose files

**Solution:** Use template command:

```bash
composepack template myapp --chart charts/myapp
cat .cpack-releases/myapp/docker-compose.yaml
```
