# Custom TLS Certificates

It’s possible to use “real” TLS certificates issued by a CA rather than the local-development-oriented `mkcert` command.

1. Obtain a certificate and key from Let’s Encrypt or another source.
2. Install the certificate and key in your project’s `.ddev/custom_certs` directory.
   * If you’re using the default [Traefik](traefik-router.md) (`router: traefik`), the files should be named `<projectname>.crt` and `<projectname>.key`, for example `exampleproj.crt` and `exampleproj.key`.
   * If you’re using the deprecated nginx-proxy router (`router: nginx-proxy`) , each certificate must be named with the pattern `fqdn.crt` and `fqdn.key`. A project named `example.ddev.site`, for example, would need `example.ddev.site.crt` and `example.ddev.site.key` in `.ddev/custom_certs`. There must be one cert-set for each FQDN handled by the project.
3. Run [`ddev start`](../usage/commands.md#start) and verify using a browser that you’re using the right certificate.
