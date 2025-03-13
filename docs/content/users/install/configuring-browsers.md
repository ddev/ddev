# Configuring Browsers for DDEV projects

DDEV generates SSL certificates to make local projects work natively with HTTPS protocol. It uses a custom root Certificate Authority to generate SSL certificates for `*.ddev.site` domains.

But, because this is a custom Certificate Authority, web browsers produce a "net::ERR_CERT_AUTHORITY_INVALID" warning when you try to open a local DDEV website with HTTPS protocol.

To get rid of this warning, you can install the custom root Certificate Authority into the browser.

For default browsers in Linux system it works out of the box using [mkcert](https://github.com/FiloSottile/mkcert) utility. 

For other operation systems and custom browsers manual steps are needed to fix this.

## Adding the DDEV root Certificate Authority into browsers

The concrete steps to install depend on the browser, but generally they are like this:

1. Download the custom DDEV root Certificate Authority certificate [here](#)
2. Open the browser preferences.
3. Find the Certificate Manager there.
4. Click on the "View Certificates" button.
5. Select the tab "Authorities".
6. Click to the "Import" button to import a custom authority cerificate.
7. Import the root certificate authority file.

