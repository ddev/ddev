# Router Customization and Debugging (Traefik)

Traefik is the default router in DDEV v1.22+.

DDEV’s router plays an important role in its [container architecture](../usage/architecture.md#container-architecture), receiving most HTTP and HTTPS traffic for requests like `*.ddev.site` and delivering them to the relevant project’s web container.

DDEV uses Traefik by default unless you configure the traditional router by running `ddev poweroff && ddev config global --router=nginx-proxy`.

## Traefik Configuration

You can fully customize the router’s [Traefik configuration](https://doc.traefik.io/traefik/).

All Traefik configuration uses the *file* provider, not the *Docker* provider. Even though the Traefik daemon itself is running inside the `ddev-router` container, it uses mounted files for configuration, rather than listening to the Docker socket.

!!!tip
    Like other DDEV configuration, any file with `#ddev-generated` will be overwritten unless you choose to “take over” it yourself. You can do this by removing the `#ddev-generated` line. DDEV will stop making changes to that file and you’ll be responsible for updating it.

### Global Traefik Configuration

Global configuration is automatically generated in the `~/.ddev/traefik` directory:

* `static_config.yaml` is the base configuration.
* `certs/default_cert.*` files are the default DDEV-generated certificates.
* `config/default_config.yaml` contains global dynamic configuration, including pointers to the default certificates.

### Project Traefik Configuration

Project configuration is automatically generated in the project’s `.ddev/traefik` directory.

* The `certs` directory contains the `<projectname>.crt` and `<projectname>.key` certificate generated for the project.
* The `config/<projectname>.yaml` file contains the configuration for the project, including information about routers, services, and certificates.

## Debugging Traefik Routing

Traefik provides a dynamic description of its configuration you can visit at `http://localhost:10999`.
When things seem to be going wrong, run [`ddev poweroff`](../usage/commands.md#poweroff) and then start your project again by running [`ddev start`](../usage/commands.md#start). Examine the router’s logs to see what the Traefik daemon is doing (or failing at) by running `docker logs ddev-router` or `docker logs -f ddev-router`.
