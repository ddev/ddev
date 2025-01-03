# ddev-{{ CONTAINER NAME }} docker image

## Overview

Docker container image for DDEV's {{ CONTAINER NAME }} container.

This container image is part of DDEV, and not typically used stand-alone.

### Features

{{ FEATURES GO HERE }}

## Instructions

Use [DDEV](https://ddev.readthedocs.io)

### Building and pushing to Docker Hub

See [DDEV docs](https://ddev.readthedocs.io/en/stable/developers/release-management/#pushing-docker-images-with-the-github-actions-workflow)

### Running
To run the container by itself:

```bash
{{ INSTRUCTIONS GO HERE }}
docker run -it --rm ddev/ddev-{{ CONTAINER NAME }}:<tag> bash
```

## Source:

[https://github.com/ddev/ddev/tree/main/containers/ddev-{{ CONTAINER NAME }}](https://github.com/ddev/ddev/tree/main/containers/ddev-{{ CONTAINER NAME }})

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
