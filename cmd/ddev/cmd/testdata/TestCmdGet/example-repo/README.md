[![tests](https://github.com/drud/ddev-memcached/actions/workflows/tests.yml/badge.svg)](https://github.com/drud/ddev-memcached/actions/workflows/tests.yml)

## What is this?

This repository allows you to quickly install memcached into a [Ddev](https://ddev.readthedocs.io) project using just `ddev service get drud/ddev-memcached`.

## Installation

1.`ddev service get drud/ddev-memcached && ddev restart`
5. `ddev restart`

## Explanation

This memcached recipe for [ddev](https://ddev.readthedocs.io) installs a [`.ddev/docker-compose.memcached.yaml`](docker-compose.memcached.yaml) using the `memcached` docker image.

## Interacting with Memcached

* The Memcached instance will listen on TCP port 11211 (the Memcached default).
* Configure your application to access Memcached on the host:port `memcached:11211`.
* To reach the Memcached admin interface, run ddev ssh to connect to the web container, then use nc or telnet to connect to the Memcached container on port 11211, i.e. nc memcached 11211. You can then run commands such as `stats` to see usage information.
