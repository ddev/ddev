## Custom TLS Certificates

It is possible to use "real" TLS certificates that have been issued by a CA other than the local-development-oriented mkcert command.

1. Obtain a certificate and key from a Let's Encrypt or another source.
2. Install the certificate and key in your project's .ddev/custom_certs directory. Each cert must be named with the pattern `f.q.d.n.crt` and `f.q.d.n.key`. In other words, if you're working with a project "example.ddev.site", you would have "example.ddev.site.crt" and "example.ddev.site.key" in .ddev/custom_certs.  There must be one cert-set for each FQDN handled by the project.
3. `ddev start` and verify using a browser that you are using the right certificate.
