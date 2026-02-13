#ddev-generated

This ~/.ddev/traefik/config directory is a STAGING DIRECTORY ONLY.

DDEV uses this directory to temporarily stage Traefik configuration files
before copying them into the ddev-global-cache Docker volume. Files in this
directory are generated during `ddev start` and then copied to the volume.

After being copied to the volume, these staging files are automatically
cleaned up on `ddev poweroff` to prevent issues when downgrading DDEV versions.

DO NOT manually edit files in this directory. They will be overwritten or
deleted. Instead:

* For project-specific Traefik configuration, edit files in your project's
  .ddev/traefik/config/ directory (and remove #ddev-generated if needed).

* For global Traefik configuration that applies to all projects, add YAML
  files to ~/.ddev/traefik/custom-global-config/ directory.

For more information, see:
https://docs.ddev.com/en/stable/users/extend/traefik-router/
