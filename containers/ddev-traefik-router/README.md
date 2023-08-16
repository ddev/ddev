## Information

This is a simple wrapper on the upstream [traefik](https://hub.docker.com/_/traefik) image.

## Usage

This container is used to allow all [DDEV](https://github.com/ddev/ddev) sites to exist side by side on shared ports (typically 80, 443, etc.). It serves as a reverse proxy to those sites, and forwards traffic to the appropriate site depending on the hostname.
