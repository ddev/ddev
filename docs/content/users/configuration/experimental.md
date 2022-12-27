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
* If the Docker host is reachable on the internet, you can actually enable real HTTPS for it using Let’s Encrypt as described in [Casual Webhosting](../topics/hosting.md). Just make sure port 2375 is not available on the internet.

## Rancher Desktop on macOS

[Rancher Desktop](https://rancherdesktop.io/) is another quickly-maturing Docker Desktop alternative for macOS. You can install it for many target platforms from the [release page](https://github.com/rancher-sandbox/rancher-desktop/releases).

Rancher Desktop integration currently has no automated testing for DDEV integration.

* By default, Rancher Desktop will provide a version of the Docker client if you don’t have one on your machine.
* Rancher changes over the “default” context in Docker, so you’ll want to turn off Docker Desktop if you’re using it.
* Rancher Desktop does not provide bind mounts, so use `ddev config global --no-bind-mounts` which also turns on Mutagen.
* Use a non-`ddev.site` name, `ddev config --additional-fqdns=rancher` for example, because the resolution of `*.ddev.site` seems to make it not work.
* Rancher Desktop does not seem to currently work with `mkcert` and `https`, so turn those off with `mkcert -uninstall && rm -r "$(mkcert -CAROOT)"`. This does no harm and can be undone with just `mkcert -install` again.

## Traefik Router

DDEV’s router plays an important role in its [container architecture](../usage/architecture.md#container-architecture), receiving most HTTP and HTTPS traffic for requests like `*.ddev.site` and delivering them to the relevant project’s web container.

`ddev-router` has been based on a forked, poorly-documented nginx reverse proxy. Versions after DDEV v1.21.3 add a new router based on the popular [Traefik Proxy](https://traefik.io/traefik/), available as an experimental feature until it becomes the default in a future release. Run the following to enable it:

```
ddev poweroff && ddev config global --use-traefik
```

Most DDEV projects will work fine out of the box, with the benefit of vastly more configuration options and ways to work with the router. (This will likely lead to more features in the future, and we’d love your feedback if you’re trying this out now!)

### Traefik Configuration

You can fully customize the router’s [Traefik configuration](https://doc.traefik.io/traefik/).

All Traefik configuration uses the *file* provider, not the *docker* provider. Even though the Traefik daemon itself is running inside the `ddev-router` container, it uses mounted files for configuration, rather than listening to the Docker socket.

!!!tip
    Like other DDEV configuration, any file with `#ddev-generated` will be overwritten unless you choose to “take over” it yourself. You can do this by removing the `#ddev-generated` line. DDEV will stop making changes to that file and you’ll be responsible for updating it.

#### Global Traefik Configuration

Global configuration is automatically generated in the `~/.ddev/traefik` directory:

* `static_config.yaml` is the base configuration.
* `certs/default_cert.*` files are the default DDEV-generated certificates.
* `config/default_config.yaml` contains global dynamic configuration, including pointers to the default certificates.

#### Project Traefik Configuration

Project configuration is automatically generated in the project’s `.ddev/traefik` directory.

* The `certs` directory contains the `<projectname>.crt` and `<projectname>.key` certificate generated for the project.
* The `config/<projectname>.yaml` file contains the configuration for the project, including information about routers, services, and certificates.

### Debugging Traefik Routing

Traefik provides a dynamic description of its configuration you can visit at `http://localhost:9999`.
When things seem to be going wrong, run [`ddev poweroff`](../usage/commands.md#poweroff) and then start your project again by running [`ddev start`](../usage/commands.md#start). Examine the router’s logs to see what the Traefik daemon is doing (or failing at) by running `docker logs ddev-router` or `docker logs -f ddev-router`.
