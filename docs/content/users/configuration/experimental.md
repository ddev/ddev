# Experimental Configurations

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
