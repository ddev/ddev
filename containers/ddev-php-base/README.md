# ddev-php-base

This repository provides the build techniques for the webserving/php DDEV Docker images and provides the base for DDEV to build ddev-webserver images:

* *ddev-php-base* is the base for ddev-php-prod, and will be used by DDEV to build ddev-webserver images.

![Block Diagram](docs-pics/ddev-images-block-diagram.png)

## Building

To build, use `make VERSION=<versiontag>` or `make images`. To push, use `make push`

Individual images can be built using `make ddev-php-prod VERSION=<versiontag>`

## Testing

Each image is intended to have a robust set of tests. The tests should be included in the `tests/<imagename>` directory, and should be launched with a `test.sh` in that directory.
