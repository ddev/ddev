# Docker NGINX PHP7 FPM for Local Development

## Introduction
This is a Dockerfile to build a container image for NGINX and PHP7 in FPM with development configurations and tools. 

## Developer Tools

* [Composer](https://getcomposer.org/) (from the production container)
* [Drush](http://www.drush.org) (from the production container)
* [WP-CLI](http://www.wp-cli.org) (from the production container)
* [Mailhog](https://github.com/mailhog/MailHog)
* npm
* yarn

## Building and pushing to dockerhub

```
make container
make push
make VERSION=0.3.0 container
make VERSION=0.3.0 push
make clean
```

## Running
To run the container by itself:

```
docker run -it drud/nginx-php-fpm-local bash
```
