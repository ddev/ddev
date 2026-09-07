---
paths:
  - "containers/ddev-webserver/**"
  - "pkg/ddevapp/webserver_config_assets/**"
  - "pkg/ddevapp/ddevapp.go"
---

# Webserver config has two independent copies — keep them in sync

Apache and nginx site configs are baked into the `ddev-webserver` image at
`containers/ddev-webserver/ddev-webserver-base-files/etc/{apache2,nginx}/`, but
`start.sh` replaces them wholesale from the *project's* `.ddev/` directory when
it exists:

- `.ddev/nginx_full/` replaces `/etc/nginx/sites-enabled`
- `.ddev/apache/` replaces `/etc/apache2/sites-enabled`

Those project files come from the `ddev` **CLI binary**, not the image: a
`//go:embed`ded copy under `pkg/ddevapp/webserver_config_assets/`, written by
`(*DdevApp).GenerateWebserverConfig()` in `pkg/ddevapp/ddevapp.go`. They carry
a `#ddev-generated` marker and are refreshed on `ddev start` and `ddev config`,
so they always win over the image, however often you rebuild it.

**When changing a real (non-fallback) apache or nginx site template, check
whether `pkg/ddevapp/webserver_config_assets/` holds the same content and
update it too, then rebuild the binary with `make` — not just the image.**

nginx site templates `include /etc/nginx/common.d/*.conf`, so prefer putting
fixes in `common.d`: image-only, no duplication. Apache has no such include, so
apache fixes usually need both copies.

To pick up a `webserver_config_assets` change in an existing project, rebuild
the binary and run `ddev start`. A project that removed the `#ddev-generated`
line owns its file and will not be touched.
