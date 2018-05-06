# Docker NGINX PHP7 FPM for Local Development

## Introduction
This is a Dockerfile to build a container image for NGINX and PHP7 in FPM with development configurations and tools. This container starts from the production [nginx-php-fpm](https://github.com/drud/docker.nginx-php-fpm) container.

## Developer Tools

* [Composer](https://getcomposer.org/) (from the production container)
* [Drush](http://www.drush.org) (from the production container)
* [WP-CLI](http://www.wp-cli.org) (from the production container)
* [Mailhog](https://github.com/mailhog/MailHog)

## Building and pushing to dockerhub

```
make container
make push
make VERSION=0.3.0 container
make VERSION=0.3.0 push
make clean
```

## Running
To simply run the container:
```
sudo docker run -d drud/nginx-php-fpm7
```
