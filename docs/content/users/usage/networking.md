# Special Network Configurations

!!!tip "The networking configurations described here are not typical"
    Don't spend time with this page unless you know you have a global TLS trust or proxy configuration issue.

There are a few networking situations which occasionally cause trouble for some users, especially in corporate environments. These typically fall into two categories:

1. **TLS interception** (e.g., Zscaler, GlobalProtect) causes SSL verification errors in `docker pull` or inside containers.
2. **Proxy-only internet access**, which blocks Docker or container tools unless proxy settings are configured.

In these situations, multiple layers may require configuration:

- **Docker Engine**: Trust external registries
- **Docker CLI**: Use system or explicit proxy settings
- **Containers (e.g., DDEV web)**: Trust HTTPS inside the container
- **Package Managers (e.g., apt)**: If used inside containers, may also need proxy/CA configuration

## Corporate Packet-inspection VPNs (including Zscaler and Global Protect)

!!!note "Your IT Department or Vendor May Have Easier Ways to Solve These Problems"

    In many cases, your IT department or VPN vendor may be able to whitelist certain internet resources, and you may not have to do extensive configuration. Check with them (if possible) before you start making configuration changes.

Packet-inspecting VPNs like **Zscaler**, **GlobalProtect**, and similar products intercept HTTPS traffic using a corporate-controlled TLS Certificate Authority (CA). These systems act as a "man-in-the-middle" proxy, decrypting and re-encrypting HTTPS traffic. As a result, systems and applications that are not explicitly configured to trust the corporate CA will experience SSL/TLS verification errors.

This creates two separate problems in Docker-based workflows:

| Layer                  | Problem                                                                 | Solution |
|------------------------|-------------------------------------------------------------------------|----------|
| Docker Engine          | `docker pull` fails with certificate errors when connecting to Docker registries like `hub.docker.com` | Configure Docker Engine to trust the corporate CA |
| Inside Containers      | Tools like `curl` or `composer` inside containers fail to connect to the internet | Install the corporate CA in the container image |

### 🧩 Docker Engine SSL Trust (for `docker pull`)

The Docker Provider itself must trust the corporate CA to pull images from remote registries. The method of adding this trust varies by platform and Docker engine.

Often, though, the easiest way to solve this particular problem is for your IT department or VPN vendor to whitelist your registry (usually `registry-1.docker.io`) so you don't have to deal with this problem in the first place. If you can't do that, a variety of solutions are provided below.

#### macOS

- **Docker Desktop**, **Orbstack**, and **Rancher Desktop** automatically use the macOS system keychain, so you likely don’t need to configure SSL trust.
- **Colima**: Colima copies `~/.docker/certs.d` into the VM at startup. To trust a CA, place it in `~/.docker/certs.d/` **before** starting Colima.
- **Lima**: Copy the CA certificate into `/etc/docker/certs.d/` using `limactl shell default`.

#### Windows

Install the trusted certificate in the system:

1. Search "Settings" for "Manage Computer Certificates", which runs `certlm`.
2. Navigate to "Certificates - Local Computer" -> "Trusted Root Certification" -> "Certificates".
3. Right-click -> "All tasks" -> "Import" to import the `crt` file.

- **Docker Desktop**: Uses the Windows system certificate store. If the CA is trusted by the system, Docker will trust it too.
- **WSL2 with Docker Desktop**: Behaves like Docker Desktop (Windows trust store).
- **WSL2 with `docker-ce`**: Requires manual installation of the CA cert as with native Linux.

#### Linux

For native Linux or WSL2 with `docker-ce`:

```bash
sudo mkdir -p /etc/docker/certs.d/
sudo cp mycorp-ca.crt /etc/docker/certs.d/
sudo systemctl restart docker
```

#### ✅ **Test Docker Engine Trust**

To test:

```bash
docker pull alpine
# ✅ Should succeed if CA is trusted
# ❌ If not trusted: x509: certificate signed by unknown authority
```

If it works without SSL errors, the CA is trusted properly.

See [Docker Engine certificate configuration](https://docs.docker.com/engine/security/certificates/) for more background.

---

### 📦 Container-Level SSL Trust (for `curl`, `composer`, `Node.js`, etc.)

Applications running inside containers do **not** inherit trust from the host system. If the container makes outbound HTTPS connections, you must install the corporate CA inside the container image.

The standard approach:

1. Export the corporate CA certificate (`.crt`) as described in the section below. Use the name of your `crt` file, not `mycorp-ca.crt`.
2. Place the `.crt` file in both your `.ddev/web-build` and `.ddev/db-build` directories.
3. Add a `.ddev/web-build/pre.Dockerfile.vpn` and `.ddev/db-build/pre.Dockerfile.vpn` like this:

    ```Dockerfile
    COPY mycorp-ca.crt /usr/local/share/ca-certificates/
    RUN update-ca-certificates
    ```

4. If you're using Node.js/npm make it trust both the DDEV `mkcert` CA and your corporate CA by combining the two into a single file and then making the environment variable `NODE_EXTRA_CA_CERTS` point to that file. Add a post-start hook to concatenate the required files into the needed `/usr/local/share/ca-certificates/node_ca_certs.pem`. For example, in a `.ddev/config.vpn.yaml` add `post-start` hook:

    ```yaml
    web_environment:
      - NODE_EXTRA_CA_CERTS=/usr/local/share/ca-certificates/node_ca_certs.pem
    hooks:
      post-start:
        - exec: "cat /mnt/ddev-global-cache/mkcert/rootCA.pem /usr/local/share/ca-certificates/mycorp-ca.crt > /usr/local/share/ca-certificates/node_ca_certs.pem"
    ```

5. Run:

    ```bash
    ddev restart
    ddev exec curl -I https://www.google.com
    # ✅ Expect: HTTP/2 200
    # ❌ If not trusted: curl: (60) SSL certificate problem
    ddev npm install --verbose
    # ✅ Expect: Successful install
    # ❌ If not trusted: attempt 2 failed with SELF_SIGNED_CERT_IN_CHAIN
    ddev yarn install
    # ✅ Expect: Successful install
    # ❌ If not trusted: code: 'SELF_SIGNED_CERT_IN_CHAIN'

    ```

This method works across all OS platforms and all Docker providers, because you're explicitly modifying the container's certificate store.

!!!note "Clear or recreate buildx contexts"
    During the testing process, we found that an old `docker buildx` context can make all of the `ddev start` builds fail in this situation. We had to remove buildx contexts (`docker buildx ls` and `docker buildx rm <context>` may help, or removing the `~/.docker/buildx` directory.)

---

### 🔍 Where to Get the Corporate CA Certificate

#### Option 1: Ask Your IT Department

Request the "TLS root certificate" or "SSL inspection CA" used by your company’s VPN or proxy.

#### Option 2: Extract from Your Own System

- **macOS**: Use Keychain Access to export the cert from the “System” keychain.
- **Windows**: Use `certmgr.msc` and export the cert from “Trusted Root Certification Authorities”.
- **Linux**: Locate and copy certs from `/etc/ssl/certs/` or use Firefox + `certutil`.

You can also visit a site like `https://www.google.com` in Chrome or Firefox, inspect the certificate chain, and export the root CA manually ([example method](https://stackoverflow.com/a/71642712/215713)).

### Converting Exported PEM or CER Files to CRT

When you export a CA certificate from your system (e.g., Keychain Access on macOS or certmgr on Windows), the file may be saved with a `.pem` or `.cer` extension. These are usually the same format as `.crt`: a base64-encoded X.509 certificate.

For Docker or Linux tools that expect a `.crt` file, you can simply rename the exported file:

```bash
mv your-cert.pem your-cert.crt
# or
mv your-cert.cer your-cert.crt
```

To confirm it is the correct format, the file should begin with:

```
-----BEGIN CERTIFICATE-----
```

If your file contains this header, renaming it to `.crt` is sufficient for use with Docker and container trust configurations.

### Corporate Packet-inspection VPN Resources

- [Adding Self-signed Registry Certs to Docker & Docker for Mac](https://blog.container-solutions.com/adding-self-signed-registry-certs-docker-mac)
- [Docker Docs: How Do I Add TLS Certificates](https://docs.docker.com/desktop/troubleshoot-and-support/faqs/macfaqs/#how-do-i-add-tls-certificates)

### VPN Trust Summary by Provider

| Provider           | Engine Trust Needed? | Auto-Inherits Host Trust? | Notes                                 |
|-------------------|----------------------|----------------------------|---------------------------------------|
| Docker Desktop     | No                   | ✅                          | Uses macOS/Windows system keychain    |
| Orbstack           | No                   | ✅                          | Fully integrated with macOS keychain  |
| Colima             | Yes                  | 🚫                          | Requires pre-start copy of certs      |
| Rancher Desktop    | Yes (dockerd)        | Partial                    | Depends on Lima config and backend    |
| WSL2 + docker-ce   | Yes                  | 🚫                          | Must configure trust inside WSL       |

<!-- textlint-disable -->
## Corporate or Internet Provider Proxy
<!-- textlint-enable -->

Some network environments, including some corporate networks, require a "proxy" system be used to access the outside network. In these environments, most systems do not have direct access to the public internet, but instead must use a configured proxy host to access the public internet. A proxy is a system that receives HTTP and HTTPS traffic and then sends and receives traffic on behalf of the client that requests it.

In most environments, the proxy will be configured at a system level. For example, on macOS, it can be configured at `Settings -> Wi-Fi -> Connection -> Details -> Proxies`. On Ubuntu it's at `Settings -> Network -> Network Proxy`.

In each of these situations the configuration required is essentially this:

- HTTP Proxy or "Web Proxy (HTTP)"
- HTTPS Proxy or "Secure web proxy (HTTPS)"
- "Ignore Hosts" or "Bypass proxy settings for these hosts"

Given a proxy with the hostname `yourproxy.intranet` with the IP address `192.168.1.254` and a port of `8888`, you would usually configure the HTTP and HTTPS Proxies as `yourproxy.intranet` with port `8888`. But it's usually important to tell your system *not* to proxy some hostnames and IP addresses, including `localhost`, `*.ddev.site`, `127.0.0.1`, and `::1`. These exclusions ensure that local development domains (such as `*.ddev.site`) and local network addresses (`127.0.0.1`, `::1`) are not mistakenly routed through the proxy, which could disrupt DDEV’s functionality.

System configuration in many systems results in environment variables like these examples:

- `HTTP_PROXY=http://yourproxy.intranet:8888`
- `HTTPS_PROXY=http://yourproxy.intranet:8888`
- `NO_PROXY=localhost,127.0.0.1,::1,*.ddev.site`

If they are not set automatically, they can be set manually in your `.bash_profile` or similar configuration file.

### Configuring Docker Daemon for Proxy

See the [Docker documentation](https://docs.docker.com/engine/daemon/proxy/#daemon-configuration) to configure the Docker daemon for access to the Docker registry (for actions like `docker pull`).

For example, `/etc/docker/daemon.json` might be:

```json
{
  "proxies": {
    "http-proxy": "http://squid.host-only:3128",
    "https-proxy": "http://squid.host-only:3128",
    "no-proxy": "localhost,127.0.0.0/8"
  }
}
```

After configuration,

```bash
sudo systemctl restart docker
```

When this is working, you should be able to successfully `docker pull alpine`.

### Configuring Docker Client for Proxy

See the [Docker documentation](https://docs.docker.com/engine/cli/proxy/#configure-the-docker-client) for how to configure the Docker client to pass proxy variables during container build (`ddev start`) and runtime (`ddev exec`).

For example, `~/.docker/config.json` might be:

```json
{
  "proxies": {
    "default": {
      "httpProxy": "http://squid.host-only:3128",
      "httpsProxy": "http://squid.host-only:3128",
      "noProxy": "localhost,127.0.0.0/8"
    }
  }
}
```

After configuration, restart the DDEV project if it is already running.

### Proxy Resources

- [Configuring Rancher Desktop Proxy](https://github.com/rancher-sandbox/rancher-desktop/issues/2259#issuecomment-1136833849)
- [Colima proxy setup](https://gist.github.com/res0nat0r/e182f23272a331f20b83195156eef83f)
- [Linux Docker Daemon Proxy Configuration](https://docs.docker.com/engine/daemon/proxy/#daemon-configuration)
- [Linux Docker Client Proxy Configuration](https://docs.docker.com/engine/cli/proxy/#configure-the-docker-client)
- [Colima Proxy Configuration](https://gist.github.com/res0nat0r/e182f23272a331f20b83195156eef83f)

## Restrictive DNS servers, especially Fritz!Box (FritzBox) routers

The normal use of DDEV involves project URLs (and hostnames) like `*.ddev.site`. So a project with the name `mytypo3` will have the default hostname `mytypo3.ddev.site` and the default URL `https://mytypo3.ddev.site`. The way this works is that `*.ddev.site` is a Domain Name System (DNS) entry which always resolves to `127.0.0.1`, or `localhost`.

There are a few DNS servers, mostly local Fritz!Box routers, which do not allow a DNS lookup to result in `127.0.0.1`. In this situation, DDEV will ask you to use superuser (`sudo`) privileges to add the hostname to the system `hosts` file, often `/etc/hosts` or `C:\Windows\system32\drivers\etc\hosts`. **This is not the preferred behavior, as DDEV does not want to edit your system files.**

Instead, if DDEV is asking you to do this and add hostnames, it's best to solve the underlying problem by adding configuration to the DNS server (often Fritz!Box router) or by using a less-restrictive DNS server like the Cloudflare `1.1.1.1` public DNS server. Full details for Fritz!Box are at [Fritz!Box Routers and DDEV](https://ddev.com/blog/fritzbox-routers-and-ddev/).

These options are explained in the [Troubleshooting - DNS Rebinding](troubleshooting.md#dns-rebinding-prohibited-mostly-on-fritzbox-routers) section of the documentation.
