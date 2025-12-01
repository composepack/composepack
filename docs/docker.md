# Docker Image Publishing

ComposePack automatically publishes Docker images to Docker Hub on every tagged release.

## Image Details

- **Registry:** Docker Hub (`docker.io`)
- **Image Name:** `garearc/composepack`
- **Architectures:** `linux/amd64`, `linux/arm64`
- **Tags:**
  - `latest` - Always points to the latest release
  - `v1.0.0` - Specific version (multi-arch manifest)
  - `v1.0.0-amd64` - AMD64-specific image
  - `v1.0.0-arm64` - ARM64-specific image

## Usage

### Pull and Run

```bash
# Pull the latest image
docker pull garearc/composepack:latest

# Run a command
docker run --rm garearc/composepack:latest --version

# Run with volume mounts for CI/CD
docker run --rm \
  -v $(pwd):/workspace \
  -w /workspace \
  garearc/composepack:latest \
  install charts/myapp --name prod
```

### GitHub Actions Example

```yaml
name: Deploy

on:
  push:
    branches: [main]

jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      
      - name: Install chart
        run: |
          docker run --rm \
            -v ${{ github.workspace }}:/workspace \
            -w /workspace \
            garearc/composepack:latest \
            install charts/myapp --name prod --auto-start
```

### GitLab CI Example

```yaml
deploy:
  image: garearc/composepack:latest
  script:
    - composepack install charts/myapp --name prod --auto-start
```

## Required Secrets

To enable Docker image publishing in CI, you need to configure the following secrets in your GitHub repository:

### Setting Up Secrets

1. Go to your repository on GitHub
2. Navigate to **Settings** → **Secrets and variables** → **Actions**
3. Add the following secrets:

#### `DOCKERHUB_USERNAME`
- **Description:** Your Docker Hub username
- **Example:** `garearc`
- **Required:** Yes

#### `DOCKERHUB_TOKEN`
- **Description:** Docker Hub access token (not your password)
- **How to create:**
  1. Go to [Docker Hub](https://hub.docker.com/)
  2. Navigate to **Account Settings** → **Security**
  3. Click **New Access Token**
  4. Give it a name (e.g., "GitHub Actions")
  5. Set permissions to **Read & Write** (or **Read, Write & Delete**)
  6. Copy the token and add it as a secret
- **Required:** Yes

### Token Permissions

The Docker Hub token needs the following permissions:
- **Read** - To pull images (for testing)
- **Write** - To push images and manifests

## Image Size

The Docker image is minimal, built from `scratch` and contains only the `composepack` binary:
- **Size:** ~10-15 MB (compressed)
- **Base:** `scratch` (no OS layer)
- **Dependencies:** None (statically linked Go binary)

## Multi-Architecture Support

The image automatically supports both architectures:
- **linux/amd64** - For Intel/AMD processors
- **linux/arm64** - For ARM processors (Apple Silicon, AWS Graviton, etc.)

Docker will automatically select the correct architecture when you pull the image.

## Troubleshooting

### Image Not Found

If you get an error that the image doesn't exist:
1. Check that a release has been tagged (images are only published on releases)
2. Verify the image name: `garearc/composepack`
3. Check Docker Hub: https://hub.docker.com/r/garearc/composepack

### Authentication Errors

If the CI job fails with authentication errors:
1. Verify `DOCKERHUB_USERNAME` secret is set correctly
2. Verify `DOCKERHUB_TOKEN` secret is set correctly
3. Check that the token has not expired
4. Ensure the token has **Write** permissions

### Build Failures

If the Docker build job fails:
1. Check the GitHub Actions logs for specific error messages
2. Verify that the Linux binaries were built successfully in the `build` job
3. Ensure the Dockerfile is present in the repository root

## Manual Publishing

If you need to publish manually (outside of CI):

```bash
# Build for your current architecture
docker build -t garearc/composepack:local .

# Or build for specific architecture
docker buildx build --platform linux/amd64 -t garearc/composepack:v1.0.0-amd64 .
docker buildx build --platform linux/arm64 -t garearc/composepack:v1.0.0-arm64 .

# Login to Docker Hub
docker login

# Push images
docker push garearc/composepack:v1.0.0-amd64
docker push garearc/composepack:v1.0.0-arm64

# Create and push manifest
docker buildx imagetools create -t garearc/composepack:v1.0.0 \
  garearc/composepack:v1.0.0-amd64 \
  garearc/composepack:v1.0.0-arm64
```

