# Special Network Configurations

There are a few networking situations which occasionally cause trouble for some users. We can't explain every permutation of these, and since most happen in a corporate environment, you may need to confer with your IT department to sort them out.

In these situations there are often at least three configurations that need to be changed:

1. Docker Engine configuration (allowing pull of Docker images from `hub.docker.com`)
2. Docker client configuration (Allowing the `docker` client to interact with VPN or proxy)
3. DDEV web image configuration (Allowing processes inside the `web` container to access internet locations)
4. (Debian/Ubuntu) Configuration for the `apt` subsystem to allow `apt update`, etc.

## Corporate Packet-inspection VPNs (including Zscaler and Global Protect)

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
- **Colima**, **Lima**: You must configure the CA manually inside the Linux VM used by the Docker engine.

  **Colima**: Create the directory `~/.docker/certs.d` if it doesn't exist, and put the CA certificate in that directory. Colima then automatically copies that into `/etc/docker/certs.d` inside the Colima VM.

  **Lima**: Copy the CA certificate into `/etc/docker/certs.d/hub.docker.com/` using `limactl shell default`.

#### Windows

Install the trusted certificate in the system:

1. Search `Settings` for "Manage Computer Certificates", which runs `certlm`.
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
sudo systemctl daemon-reload
```

To test:

```bash
docker pull alpine
```

If it works without SSL errors, the CA is trusted properly.

---

### 📦 Container-Level SSL Trust (for `curl`, `composer`, etc.)

Applications running inside containers do **not** inherit trust from the host system. If the container makes outbound HTTPS connections, you must install the corporate CA inside the container image.

The standard approach:

1. Export the corporate CA certificate (`.crt`) as described in the section below.
2. Place the `.crt` file in your `.ddev/web-build` directory.
3. Add a `Dockerfile.vpn` like this:

    ```Dockerfile
    COPY mycorp-ca.crt /usr/local/share/ca-certificates/
    RUN update-ca-certificates
    ```

4. Run:

    ```bash
    ddev restart
    ddev exec curl -I https://www.google.com
    ```

    You should see a `200 OK` response if the CA is trusted correctly.

This method works across all OS platforms and all Docker providers, because you're explicitly modifying the container's certificate store.

---

### 🔍 Where to Get the Corporate CA Certificate

#### Option 1: Ask IT

Request the "TLS root certificate" or "SSL inspection CA" used by your company’s VPN or proxy.

#### Option 2: Extract from Your Own System

- **macOS**: Use Keychain Access to export the cert from the “System” keychain.
- **Windows**: Use `certmgr.msc` and export the cert from “Trusted Root Certification Authorities”.
- **Linux**: Locate and copy certs from `/etc/ssl/certs/` or use Firefox + `certutil`.

You can also visit a site like `https://example.com` in Chrome or Firefox, inspect the certificate chain, and export the root CA manually.

For detailed instructions, see the [Stack Overflow reference](https://stackoverflow.com/a/71642712/215713).

## Corporate or Internet Provider Proxy

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

See [Docker documentation](https://docs.docker.com/engine/daemon/proxy/#daemon-configuration) to configure the Docker daemon so that you can do things that involve the Docker registry (actions like `docker pull`).

For example. `/etc/docker/daemon.json` might be:

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
sudo systemctl daemon-reload
```

When this is working, you should be able to successfully `docker pull alpine`.

## Restrictive DNS servers, especially Fritzbox routers

The normal use of DDEV involves project URLs (and hostnames) like `*.ddev.site`. So a project with the name `mytypo3` will have the default hostname `mytypo3.ddev.site` and the default URL `https://mytypo3.ddev.site`. The way this works is that `*.ddev.site` is a Domain Name System (DNS) entry which always resolves to `127.0.0.1`, or `localhost`.

There are a few DNS servers, mostly local Fritzbox routers, which do not allow a DNS lookup to result in `127.0.0.1`. In this situation, DDEV will ask you to use superuser (`sudo`) privileges to add the hostname to the system `hosts` file, often `/etc/hosts` or `C:\Windows\system32\drivers\etc\hosts`. **This is not the preferred behavior, as DDEV does not want to edit your system files.**

Instead, if DDEV is asking you to do this and add hostnames, it's best to solve the underlying problem by adding configuration to the DNS server (often Fritzbox router) or by using a less-restrictive DNS server like the Cloudflare `1.1.1.1` public DNS server.

These options are explained in the [Troubleshooting - DNS Rebinding](troubleshooting.md#dns-rebinding-prohibited-mostly-on-fritzbox-routers) section of the documentation.

## Resources

- [Configuring Rancher Desktop Proxy](https://github.com/rancher-sandbox/rancher-desktop/issues/2259#issuecomment-1136833849)
- [Colima proxy setup](https://gist.github.com/res0nat0r/e182f23272a331f20b83195156eef83f)
