# Custom TLS Certificates

It’s possible to use “real” TLS certificates issued by a CA rather than the local-development-oriented `mkcert` command.

1. Obtain a certificate and key from Let’s Encrypt or another source.
2. Install the certificate and key in your project’s `.ddev/custom_certs` directory. Each certificate must be named with the pattern `fqdn.crt` and `fqdn.key`. If you’re working with a project `example.ddev.site`, you would have `example.ddev.site.crt` and `example.ddev.site.key` in `.ddev/custom_certs`. There must be one cert-set for each FQDN handled by the project.
3. `ddev start` and verify using a browser that you’re using the right certificate.
