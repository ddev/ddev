# Configuring Browsers for DDEV projects
**Most DDEV users can ignore this page. The standard instructions in DDEV Installation do everything that is needed. These instructions are for unusual browsers or OS environments.**
DDEV generates SSL certificates to enable local projects to use the HTTPS protocol. It uses a custom root Certificate Authority (CA) to generate SSL certificates for `*.ddev.site` domains.

However, since this is a custom CA, web browsers display an ERR_CERT_AUTHORITY_INVALID warning when trying to access a local DDEV site over HTTPS.

To eliminate this warning, you can install the custom root CA into your browser.

For default browsers, this works automatically using the [mkcert](https://github.com/FiloSottile/mkcert) utility.

For custom browsers (such as Firefox Developer Edition, Vivaldi, etc.), manual steps may be required.

!!!tip "Want to learn more about HTTPS in DDEV?"
    See [Hostnames and Wildcards and DDEV, Oh My!](https://ddev.com/blog/ddev-name-resolution-wildcards/) for more information on DDEV hostname resolution.

## Adding the DDEV Root Certificate Authority to Browsers

The steps to install the root CA depend on the browser, but they generally follow this process:

1. Use `mkcert -CAROOT` to locate the directory with the generated root certificate. Inside, you should find a `rootCA.pem` file. If it's missing, run `mkcert -install` command.
2. Open your browser and navigate to the preferences or settings.
3. Find the Certificate Manager, typically located in the "Security" section of the preferences.
4. Click the "View Certificates" button.
5. Go to the "Authorities" tab.
6. Click the "Import" button to add a custom authority certificate.
7. Import the `rootCA.pem` file to install the root certificate authority.

!!!note "Still having issues?"
    Check out [this specific mkcert thread](https://github.com/FiloSottile/mkcert/issues/370) for additional troubleshooting.
