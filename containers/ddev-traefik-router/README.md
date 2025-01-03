# ddev-traefik-router docker image

## Overview

ddev-traefik-router is a simple wrapper on the upstream [traefik](https://hub.docker.com/_/traefik) image.

This container is used to allow all [DDEV](https://github.com/ddev/ddev) sites to exist side by side on shared ports (typically 80, 443, etc.). It serves as a reverse proxy to those sites, and forwards traffic to the appropriate site depending on the hostname.

This container image is part of DDEV, and not typically used stand-alone.

### Features

* traefik
* A few extra packages and configuration
* A healthcheck

## Instructions

Use [DDEV](https://ddev.readthedocs.io)

### Building and pushing to Docker Hub

See [DDEV docs](https://ddev.readthedocs.io/en/stable/developers/release-management/#pushing-docker-images-with-the-github-actions-workflow)

### Running
To run the container by itself:

```bash
docker run -it --rm ddev/ddev-traefik-router:<tag> bash
```

## Source:

[https://github.com/ddev/ddev/tree/main/containers/ddev-traefik-router](https://github.com/ddev/ddev/tree/main/containers/ddev-traefik-router)

## Maintained by:

The [DDEV Docker Maintainers](https://github.com/ddev)

## Where to get help:

* [DDEV Community Discord](https://ddev.com/s/discord)
* [Stack Overflow](https://stackoverflow.com/questions/tagged/ddev)

## Where to file issues:

https://github.com/ddev/ddev/issues

## Documentation:

* https://ddev.readthedocs.io/
* https://ddev.com/

## What is DDEV?

[DDEV](https://github.com/ddev/ddev) is an open source tool for launching local web development environments in minutes. It supports PHP and Node.js.

These environments can be extended, version controlled, and shared, so you can take advantage of a Docker workflow without Docker experience or bespoke configuration. Projects can be changed, powered down, or removed as easily as they’re started.

## License

View [license information](https://github.com/ddev/ddev/blob/main/LICENSE) for the software contained in this image.

As with all Docker images, these likely also contain other software which may be under other licenses (such as Bash, etc from the base distribution, along with any direct or indirect dependencies of the primary software being contained).

As for any pre-built image usage, it is the image user's responsibility to ensure that any use of this image complies with any relevant licenses for all software contained within.

