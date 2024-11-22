# Custom TLS Certificates

It’s possible to use “real” TLS certificates issued by a CA rather than the local-development-oriented `mkcert` command.

1. Obtain a certificate and key from Let’s Encrypt or another source.
2. Install the certificate and key in your project’s `.ddev/custom_certs` directory.
   * The files should be named `<projectname>.crt` and `<projectname>.key`, for example `exampleproj.crt` and `exampleproj.key`.
3. Run [`ddev start`](../usage/commands.md#start) and verify using a browser that you’re using the right certificate.
