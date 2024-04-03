# arch-aur-builder docker image

## Overview

This obsolete image was once used for pushing DDEV releases to AUR. It is no longer used, as we use goreleaser now.

## Instructions

This can be pushed with

```bash
cat aur-checker-Dockerfile | docker buildx build  --push --platform linux/amd64 -t "ddev/arch-aur-builder:latest" -
```
`
- Edit PKGBUILD to change the version and hash or anything else
- Then run it with

```bash
# docker run --rm --mount type=bind,source=$(pwd),target=/tmp/ddev-bin --workdir=/tmp/ddev-bin ddev/arch-aur-builder bash -c "makepkg --printsrcinfo > .SRCINFO && makepkg -s"
```

- Then `git add -u` and commit and push


## Source:

[https://github.com/ddev/ddev/tree/master/.circleci/aur-checker-Dockerfile](https://github.com/ddev/ddev/tree/master/.circleci/aur-checker-Dockerfile)

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