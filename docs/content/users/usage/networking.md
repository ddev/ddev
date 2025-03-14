# Special Network Configurations

There are a few networking situations which occasionally cause trouble for some users. We can't explain every permutation of these, and since most happen in a corporate environment, you may need to confer with your IT department to sort them out.

## Some Corporate VPNs (including Zscaler)

Although DDEV (and Docker) work fine with most VPN systems, there are a number of VPNs (and similar products like Zscaler or Global Protect) which perform SSL/TLS interception and filtering. In other words, instead of connecting directly to HTTPS servers on the internet, they send traffic to the corporate networking infrastructure which can intercept HTTPS traffic and do SSL/TLS inspection. In a normal network, a client application can use HTTPS to connect directly to a server on the internet and can determine whether the server is what it says it is by examining its certificate and where the certificate was issued (the "CA" or "Certificate Authority"). Zscaler and similar products actually present their own certificates (which are trusted only because of corporate configuration) and are able to inspect traffic that would normally be encrypted, and then pass the traffic on to the end system if it is approved. This gives corporate networks extensive control over HTTPS traffic and its contents.

It gives Docker and DDEV quite a lot of trouble, though, because Docker containers do not inherit host machine trust settings and configuration, so they don't recognize or trust corporate CA, leading to SSL validation errors.

To fix this so that applications inside the web container (or other containers) can access the internet, the web image must be adjusted to trust the alternate CA that the VPN provides, so the intermediate system is not rejected as invalid.

Several specific ways to sort this out are listed in the related [Stack Overflow](https://stackoverflow.com/questions/71595327/corporate-network-vpn-ddev-composer-create-results-in-ssl-certificate-proble) question, but the basic answer is:

1. Obtain the CA `.crt` files from your IT department, vendor, or other source.
2. Place the `.crt` files in your `.ddev/web-build` directory.
3. Use a `.ddev/web-build/Dockerfile.vpn` to install the `.crt` files, as shown in this example `.ddev/web-build/Dockerfile.vpn`:

    ```Dockerfile
    COPY <yourcert>*.crt /usr/local/share/ca-certificates/
    RUN update-ca-certificates --fresh
    ```

4. To test for success,

  ```bash
  ddev restart
  ddev exec curl -I https://www.google.com # Or any URL you need
  ```

  and you expect a "200 OK" response.

### Additional Resources

* For more approaches to resolving this, see [this Stack Overflow discussion](https://stackoverflow.com/questions/71595327/corporate-network-vpn-ddev-composer-create-results-in-ssl-certificate-proble).

## Corporate or Internet Provider Proxy

Some network environments, including some corporate networks, require a "proxy" system be used to access the outside network. In these environments, most systems do not have direct access to the public internet, but instead must use a configured proxy host to access the public internet. A proxy is a system that receives HTTP and HTTPS traffic and then sends and receives traffic on behalf of the client that requests it.

In most environments, the proxy will be configured at a system level. For example, on macOS, it can be configured at `Settings -> Wi-Fi -> Connection -> Details -> Proxies`. On Ubuntu it's at `Settings -> Network -> Network Proxy`.

In each of these situations the configuration required is essentially this:

* HTTP Proxy or "Web Proxy (HTTP)"
* HTTPS Proxy or "Secure web proxy (HTTPS)"
* "Ignore Hosts" or "Bypass proxy settings for these hosts"

Given a proxy with the hostname `yourproxy.intranet` with the IP address `192.168.1.254` and a port of `8888`, you would usually configure the HTTP and HTTPS Proxies as `yourproxy.intranet` with port `8888`. But it's usually important to tell your system *not* to proxy some hostnames and IP addresses, including `localhost`, `*.ddev.site`, `127.0.0.1`, and `::1`. These exclusions ensure that local development domains (such as `*.ddev.site`) and local network addresses (`127.0.0.1`, `::1`) are not mistakenly routed through the proxy, which could disrupt DDEV’s functionality.

System configuration in many systems results in environment variables like these examples:

* `HTTP_PROXY=http://yourproxy.intranet:8888`
* `HTTPS_PROXY=http://yourproxy.intranet:8888`
* `NO_PROXY=localhost,127.0.0.1,::1,*.ddev.site`

If they are not set automatically, they can be set manually in your `.bash_profile` or similar configuration file.

### Configuring Proxy Information using Docker's Configuration

The user's `~/.docker/config.json` is one way to tell the Docker CLI (and DDEV) how to use a required proxy. See [Docker docs](https://docs.docker.com/engine/cli/proxy/). It might have a stanza like this:

```json
{
    "proxies": {
        "default": {
            "httpProxy": "http://username:pass@yourproxy.intranet:8888",
            "httpsProxy": "http://username:pass@yourproxy.intranet:8888",
            "noProxy": "localhost,127.0.0.1/8,::1,*.ddev.site"
        }
    }
}
```

Docker Desktop and other providers also have proxy configuration of the same form.

After updating proxy settings, restart your Docker provider to make the configuration take effect.

### Configuring Proxy Information with DDEV's `ddev-proxy-support` Add-on

If your system is already configured and working with a proxy, you don't have to configure Docker explicitly and can instead use DDEV's [ddev-proxy-support](https://github.com/ddev/ddev-proxy-support) add-on to help DDEV and its web container to use the proxy correctly. This technique will only help DDEV and the project that the add-on is installed in, and won't solve problems with other Docker containers.

If your system is already working correctly with your proxy, you can often use

```bash
ddev add-on get ddev/ddev-proxy-support && ddev restart
```

to make the DDEV project work correctly with the proxy.

If you are working with multiple DDEV projects, you will need to install [ddev-proxy-support](https://github.com/ddev/ddev-proxy-support) into each project where a proxy is required.

## Restrictive DNS servers, especially Fritzbox routers

The normal use of DDEV involves project URLs (and hostnames) like `*.ddev.site`. So a project with the name `mytypo3` will have the default hostname `mytypo3.ddev.site` and the default URL `https://mytypo3.ddev.site`. The way this works is that `*.ddev.site` is a Domain Name System (DNS) entry which always resolves to `127.0.0.1`, or `localhost`.

There are a few DNS servers, mostly local Fritzbox routers, which do not allow a DNS lookup to result in `127.0.0.1`. In this situation, DDEV will ask you to use superuser (`sudo`) privileges to add the hostname to the system `hosts` file, often `/etc/hosts` or `C:\Windows\system32\drivers\etc\hosts`. **This is not the preferred behavior, as DDEV does not want to edit your system files.**

Instead, if DDEV is asking you to do this and add hostnames, it's best to solve the underlying problem by adding configuration to the DNS server (often Fritzbox router) or by using a less-restrictive DNS server like the Cloudflare `1.1.1.1` public DNS server.

These options are explained in the [Troubleshooting - DNS Rebinding](troubleshooting.md#dns-rebinding-prohibited-mostly-on-fritzbox-routers) section of the documentation.
