#ddev-generated

The global `traefik` directory usually contains only DDEV-generated files.

For example, the hidden traefik/.static_config.yaml gives Traefik its static configuration,
which is generated during `ddev start`.

The `certs` directory contains default certs, and the `config` directory normally contains only
default dynamic configuration in the `config/default_config.yaml` file, which is available
to all projects.

Additional static configuration can be added to the DDEV-generated .static_config.yaml by 
adding `static_config.*.yaml` files. For example, a `static_config.test.yaml` with the contents:

```
# there is nothing here
```

would be appended to the ``.static_config.yaml`

A more significant example (a Traefik plugin) is shown in the `static_config.fail2ban.yaml.example`.
