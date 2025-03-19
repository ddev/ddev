# Configuring Browsers for DDEV projects

DDEV generates SSL certificates to make local projects work natively with HTTPS protocol. It uses a custom root Certificate Authority to generate SSL certificates for `*.ddev.site` domains.

But, because this is a custom Certificate Authority, web browsers produce a "net::ERR_CERT_AUTHORITY_INVALID" warning when you try to open a local DDEV website with HTTPS protocol.

To get rid of this warning, you can install the custom root Certificate Authority into the browser.

For default browsers in Linux systems, it works out of the box using the [mkcert](https://github.com/FiloSottile/mkcert) utility.

For other operating systems and custom browsers manual steps are required to fix this.

## Adding the DDEV root Certificate Authority into browsers

The concrete steps to install depend on the browser, but generally they are like this:

1. Find a directory with the generated root certificate by [mkcert](https://github.com/FiloSottile/mkcert) utility using a command `mkcert -CAROOT` and find there a `rootCA.pem` file. DDEV should generate it during installation. If not - you can generate it manually using `mkcert -install` command.
2. Open a web browser window and open the browser preferences.
3. Find the Certificate Manager somewhere in the preferences and open it. Usually it is located in the "Security" section.
4. Click on the "View Certificates" button.
5. Select the tab "Authorities".
6. Click to the "Import" button to import a custom authority certificate.
7. Import the root certificate authority file.
