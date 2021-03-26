# Docker ddev-webserver (webserver and php)

## Introduction

This is a Dockerfile to build a container image for ddev's web container.

## Developer Tools

* [Composer](https://getcomposer.org/) (from the production container)
* [Drush](http://www.drush.org) (from the production container)
* [WP-CLI](http://www.wp-cli.org) (from the production container)
* [Blackfire CLI](https://blackfire.io/docs/cookbooks/profiling-http-via-cli)
* [Mailhog](https://github.com/mailhog/MailHog)
* npm
* yarn

## Building and pushing to dockerhub

```
make container
make push # Pushes the git committish as version
make VERSION=20190101_test_version container
make VERSION=20190101_test_version push
make clean
```

## Running

To run the container by itself:

```
docker run -it drud/ddev-webserver bash
```
