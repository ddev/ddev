#ddev-generated

This ~/.ddev/traefik/custom-global-config directory is for USER-MANAGED
global Traefik dynamic configuration.

Add YAML files here to define custom Traefik configuration that applies to
all DDEV projects. This is useful for:

* Global middleware definitions (rate limiting, authentication, etc.)
* Custom routers or services that span multiple projects
* Advanced Traefik features that need to be available globally

Files in this directory are copied to the router's configuration volume on
each `ddev start`. You are responsible for maintaining these files - DDEV
will not modify or delete them.

Example: Create custom-global-config/global_middlewares.yaml with:

  http:
    middlewares:
      global-ratelimit:
        rateLimit:
          average: 100
          burst: 50

For more information, see:
https://docs.ddev.com/en/stable/users/extend/traefik-router/
