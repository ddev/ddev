# Install DDEV

This devcontainer feature installs DDEV in GitHub Codespaces and other devcontainer environments.

## Usage

Add the feature to your `.devcontainer/devcontainer.json`:

```json
{
  "image": "mcr.microsoft.com/devcontainers/base:debian-12",
  "features": {
    "ghcr.io/devcontainers/features/docker-in-docker:2": {
      "version": "latest"
    },
    "ghcr.io/ddev/ddev/install-ddev:latest": {}
  }
}
```

The feature automatically:
- Installs DDEV from the apt repository
- Configures environment variables (`XDG_CONFIG_HOME`, `IN_DEVCONTAINER`)
- Sets up mkcert for SSL certificates
- Fixes `/workspaces` permissions for config storage
- Verifies installation with `ddev version`

## Testing the Feature

To test local changes to this feature before publishing:

1. Create a test devcontainer configuration that references the local feature:

```json
{
  "image": "mcr.microsoft.com/devcontainers/base:debian-12",
  "features": {
    "ghcr.io/devcontainers/features/docker-in-docker:2": {
      "version": "latest"
    },
    "./path/to/install-ddev": {}
  }
}
```

2. The feature directory must be within the `.devcontainer/` folder (VS Code security requirement)
3. Copy or symlink the feature files into `.devcontainer/install-ddev/`
4. Rebuild the container to test changes

## Development Builds

To test with a development build of DDEV (from source) instead of the released apt package:

1. Add the Go feature and mount your DDEV source:

```json
{
  "image": "mcr.microsoft.com/devcontainers/base:debian-12",
  "features": {
    "ghcr.io/devcontainers/features/docker-in-docker:2": {
      "version": "latest"
    },
    "ghcr.io/devcontainers/features/go:1": {
      "version": "latest"
    },
    "./install-ddev": {}
  },
  "containerEnv": {
    "DDEV_BUILD_FROM_SOURCE": "/workspaces/ddev"
  },
  "mounts": [
    "source=/path/to/ddev/source,target=/workspaces/ddev,type=bind"
  ]
}
```

2. When `DDEV_BUILD_FROM_SOURCE` is set and points to a valid DDEV source directory:
   - The stable DDEV is installed from apt during build (fast)
   - During post-create, DDEV is built from source and replaces the apt version
   - Architecture is automatically detected (`amd64`, `arm64`, etc.)

3. To rebuild DDEV manually after making changes:

```bash
cd /workspaces/ddev && make && sudo cp .gotmp/bin/linux_$(dpkg --print-architecture)/ddev /usr/local/bin/ddev
```

## Files

- `devcontainer-feature.json` - Feature metadata and configuration
- `install.sh` - Executed during container build to install DDEV and dependencies
- `post-create.sh` - Executed after container creation to verify installation and optionally build from source
