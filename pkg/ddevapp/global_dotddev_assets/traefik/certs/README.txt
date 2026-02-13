#ddev-generated

This ~/.ddev/traefik/certs directory is a STAGING DIRECTORY ONLY.

DDEV uses this directory to stage SSL/TLS certificates before copying them
into the ddev-global-cache Docker volume. Certificates are generated during
`ddev start` and then copied to the volume.

DO NOT manually edit files in this directory. They will be overwritten.
Instead:

* For project-specific certificates, place them in your project's
  .ddev/traefik/certs/ directory or .ddev/custom_certs/ directory.

* Default certificates are managed automatically by DDEV using mkcert.

For more information, see:
https://docs.ddev.com/en/stable/users/extend/traefik-router/
