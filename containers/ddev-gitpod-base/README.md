# ddev-gitpod-base docker image

## Overview

ddev/ddev-gitpod-base is a base image for Gitpod integration with [DDEV](https://github.com/ddev/ddev). 

Details about how it is used are in https://ddev.readthedocs.io/en/stable/users/install/ddev-installation/#gitpod

### Features

All the packages you expect to find to use DDEV on gitpod.

## Instructions

Use [DDEV on Gitpod](https://ddev.readthedocs.io/en/stable/users/install/ddev-installation/#gitpod)

### Building and pushing to Docker Hub

Use [push.sh](https://github.com/ddev/ddev/blob/master/.gitpod/images/push.sh)

### Running

To run the container by itself:

```bash
docker run -it --rm ddev/ddev-gitpod-base:<tag> bash
```

## Source:

[https://github.com/ddev/ddev/blob/master/.gitpod/images/Dockerfile](https://github.com/ddev/ddev/blob/master/.gitpod/images/Dockerfile)

## Maintained by:

The [DDEV Docker Maintainers](https://github.com/ddev)

## Where to get help:

* [DDEV Community Discord](https://discord.gg/5wjP76mBJD)
* [Stack Overflow](https://stackoverflow.com/questions/tagged/ddev)

## Where to file issues:

https://github.com/ddev/ddev/issues

## Documentation:

* https://ddev.readthedocs.io/
* https://ddev.com/

## What is DDEV?

[DDEV](https://github.com/ddev/ddev) is an open source tool for launching local web development environments in minutes. It supports PHP, Node.js, and Python (experimental).

These environments can be extended, version controlled, and shared, so you can take advantage of a Docker workflow without Docker experience or bespoke configuration. Projects can be changed, powered down, or removed as easily as they’re started.

## License

View [license information](https://github.com/ddev/ddev/blob/master/LICENSE) for the software contained in this image.

As with all Docker images, these likely also contain other software which may be under other licenses (such as Bash, etc from the base distribution, along with any direct or indirect dependencies of the primary software being contained).

As for any pre-built image usage, it is the image user's responsibility to ensure that any use of this image complies with any relevant licenses for all software contained within.