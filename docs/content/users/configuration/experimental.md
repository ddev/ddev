# Experimental Configurations

## Remote Docker Instances

You can use remote Docker instances, whether on the internet, inside your network, or running in a virtual machine.

* On the remote machine, the Docker port must be exposed if it’s not already. See [instructions](https://gist.github.com/styblope/dc55e0ad2a9848f2cc3307d4819d819f) for how to do this on a systemd-based remote server. **Be aware that this has serious security implications and must not be done without taking those into consideration.** In fact, `dockerd` will complain:

    > Binding to IP address without --tlsverify is insecure and gives root access on this machine to everyone who has access to your network.  host="tcp://0.0.0.0:2375".

* If you do not already have the Docker client installed (like you would from Docker Desktop), install *just* the client with `brew install docker`.
* Create a Docker context that points to the remote Docker instance. For example, if the remote hostname is `debian-11`, then `docker context create debian-11 --docker host=tcp://debian-11:2375 && docker use debian-11`. Alternately, you can use the `DOCKER_HOST` environment variable, e.g. `export DOCKER_HOST=tcp://debian-11:2375`.
* Make sure you can access the remote machine using `docker ps`.
* Bind mounts cannot work on a remote Docker setup, so you must use `ddev config global --no-bind-mounts`. This will cause DDEV to push needed information to and from the remote Docker instance when needed. This also automatically turns on Mutagen caching.
* You may want to use a FQDN other than `*.ddev.site` because the DDEV site will *not* be at `127.0.0.1`. For example, `ddev config --fqdns=debian-11` and then use `https://debian-11` to access the site.
* If the Docker host is reachable on the internet, you can actually enable real HTTPS for it using Let’s Encrypt as described in [Casual Webhosting](../details/alternate-uses.md#casual-project-webhosting-on-the-internet-including-lets-encrypt). Just make sure port 2375 is not available on the internet.

## Rancher Desktop on macOS

[Rancher Desktop](https://rancherdesktop.io/) is another quickly-maturing Docker Desktop alternative for macOS. You can install it for many target platforms from the [release page](https://github.com/rancher-sandbox/rancher-desktop/releases).

Rancher Desktop integration currently has no automated testing for DDEV integration.

* By default, Rancher Desktop will provide a version of the Docker client if you don’t have one on your machine.
* Rancher changes over the “default” context in Docker, so you’ll want to turn off Docker Desktop if you’re using it.
* Rancher Desktop does not provide bind mounts, so use `ddev config global --no-bind-mounts` which also turns on Mutagen.
* Use a non-`ddev.site` name, `ddev config --additional-fqdns=rancher` for example, because the resolution of `*.ddev.site` seems to make it not work.
* Rancher Desktop does not seem to currently work with `mkcert` and `https`, so turn those off with `mkcert -uninstall && rm -r "$(mkcert -CAROOT)"`. This does no harm and can be undone with just `mkcert -install` again.

## Traefik Router

The job of the `ddev-router` is to receive most HTTP or HTTPS traffic, like requests to `*.ddev.site`, and deliver it to the correct project's web container, see [Container Architecture](../basics/architecture.md#container-architecture). The router as of DDEV v1.21.3 was always a forked, poorly documented nginx reverse proxy container.

The very popular [Traefik Proxy](https://traefik.io/traefik/) will replace the current `ddev-router` in the future, but is available as an experimental configuration today. Feedback is welcome. To enable it:

```
ddev poweroff && ddev config global --use-traefik
```

Most DDEV projects will work fine out of the box, but there is a vast array of possible configuration changes and options, and new ways to view what is going on with the router. You don't need to know any of these to use the Traefik router, but they will offer new options in the future.

### Traefik Configuration

All configuration for the new router is intended to be customizable where needed. As with other files throughout the DDEV ecosystem, if the file has `#ddev-generated` in it, it can and will be overwritten by DDEV. If you want to "take over" the configuration, you remove the `#ddev-generated` and become responsible for the file's content.

All Traefik configuration is described at [docs.traefik.io](https://doc.traefik.io/traefik/).

All Traefik configuration uses the *file* provider, not the *docker* provider. Even though the Traefik daemon itself is running inside the `ddev-router` container, it uses mounted files for configuration, rather than listening to the Docker socket.

#### Global Traefik Configuration

Global configuration for Traefik is automatically generated in ~/.ddev/traefik.

* `static_config.yaml` which is the base configuration.
* `certs/default_cert.*` is the default DDEV-generated certificates.
* `config/default_config.yaml` contains global dynamic configuration, including pointers to the default certificates.

#### Project Traefik Configuration

Project configuration is automatically generated in the project's .ddev/traefik directory.

* The `certs` directory contains the `<projectname>.crt` and `<projectname>.key` certificate generated for this project.
* The `config/<projectname>.yaml` file contains the configuration for the project, including information about routers, services, and certificates.

### Debugging Traefik Routing

* Traefik provides a dynamic description of its configuration that you can visit at `http://localhost:9999`.
* When things seem to be going wrong, do a `ddev poweroff` and then start your project and see what the Traefik daemon is doing or failing at with `docker logs ddev-router` or `docker logs -f ddev-router`.
