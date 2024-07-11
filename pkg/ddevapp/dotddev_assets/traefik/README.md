#ddev-generated

The project `traefik` directory usually contains only DDEV-generated files.

The `certs` directory contains default certs, and the `config` directory normally contains only
default dynamic configuration in the `config/<project>.yaml` file.

Additional dynamic configuration can be merged into the DDEV-generated `<project>.yaml` by
adding `dynamic_config.*.yaml` files to the `traefik` directory. As with the global `static_config.*.yaml` files,
merging is done with an **override** strategy, meaning that the final file in alphanumeric sort to touch a
particular element of the YAML structure wins.

The `dynamic_config.middleware.yaml.example` can be used as a starting point for adding
middlewares - be they built-in (https://doc.traefik.io/traefik/middlewares/overview/) or 3rd party plugins
(https://plugins.traefik.io/plugins). For plugins, it is important to click Install Plugin on any plugin page
to receive the plugin's configuration, as the configuration provided in the body of the page tends to be outdated.

As is always the case with YAML, syntax (indents, spacing, hypens etc...) is very important. This is especially the
case with the middleware definitions, which can be finicky. If it isn't working, please check the routers and
middlewares at http://localhost:10999/dashboard/, where they will show you an error if something was ill-defined.

You can also add other files to override any other aspect of the `<project>.yaml`
