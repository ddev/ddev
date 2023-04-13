# Experimental Configurations

## Traefik Router

DDEV’s router plays an important role in its [container architecture](../usage/architecture.md#container-architecture), receiving most HTTP and HTTPS traffic for requests like `*.ddev.site` and delivering them to the relevant project’s web container.

`ddev-router` has been based on a forked, poorly-documented nginx reverse proxy. Versions after DDEV v1.21.3 add a new router based on the popular [Traefik Proxy](https://traefik.io/traefik/), available as an experimental feature until it becomes the default in a future release. Run the following to enable it:

```
ddev poweroff && ddev config global --use-traefik
```

Most DDEV projects will work fine out of the box, with the benefit of vastly more configuration options and ways to work with the router. (This will likely lead to more features in the future, and we’d love your feedback if you’re trying this out now!)

### Traefik Configuration

!!!note
Traefik will become the default router in DDEV v1.22+.

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

## Django4 and Python Project Types

DDEV v1.22+ supports Python-based projects, including those built with Django 4 and Flask.

`ddev config --project-type=django4` will by default a project to use the `nginx-gunicorn` `webserver_type` and the `postgres` database type.

Community feedback is essential for Django/Python support to improve, thank you for participating!
