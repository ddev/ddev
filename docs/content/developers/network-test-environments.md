---
search:
  boost: .1
---
# Network Test Environments: Packet-Inspection VPN Simulation

!!!tip "Basic HTTP Proxy Testing"
    For simpler HTTP proxy testing without SSL interception, see [HTTP Proxy Test Environments](http-proxy-test-environments.md). Start there for basic proxy scenarios before moving to SSL interception testing.

## Simulating SSL Interception with Squid (Simplified via `HTTPS_PROXY`)

A straightforward way to simulate a packet-inspecting VPN is by using **Squid** with SSL bumping and configuring your environment to use it via `HTTPS_PROXY`. While it's less transparent than a full MITM router setup, it closely replicates the behavior of Zscaler and similar tools from the perspective of apps like Docker and `curl`.

## Setup Overview

- Squid listens on port 3128.
- HTTPS traffic routed via the proxy is intercepted and re-signed with a custom CA.
- Clients that trust this CA will succeed; others will fail SSL validation.
- You simulate VPN-like interception by exporting `HTTPS_PROXY=http://localhost:3128`.

## Step-by-Step Instructions

These instructions are for Debian/Ubuntu but can be adapted for container-based setup or for another environment.

1. **Install Squid**

    ```bash
    sudo apt-get install squid-openssl ssl-cert
    ```

2. **Generate a Root CA for Signing**
  This `mitm.crt` is the CA certificate used by Squid to re-sign intercepted traffic, and it must be trusted by any client interacting through the proxy (e.g., Docker, curl, system-wide tools).

    ```bash
    sudo openssl req -new -newkey rsa:2048 -days 365 -nodes -x509 \
      -keyout /etc/squid/mitm.key \
      -out /etc/squid/mitm.crt \
      -subj "/CN=SquidMITMTest"
    ```

3. **Configure Squid for SSL Bumping**

    Edit `/etc/squid/squid.conf`, replacing or appending the following:

    ```conf
    http_port 3128 ssl-bump cert=/etc/squid/mitm.crt key=/etc/squid/mitm.key generate-host-certificates=on dynamic_cert_mem_cache_size=4MB
    
    sslcrtd_program /usr/lib/squid/security_file_certgen -s /var/lib/ssl_db -M 4MB
    sslcrtd_children 5
    
    ssl_bump server-first all
    
    http_access allow all
    ```

4. **Initialize Squid’s SSL Certificate Store**

    ```bash
    sudo /usr/lib/squid/security_file_certgen -c -s /var/lib/ssl_db -M 4MB
    sudo chown -R proxy: /var/lib/ssl_db
    ```

5. **Restart Squid**

    ```bash
    sudo systemctl restart squid
    ```

6. **Export HTTPS Proxy for Testing**

    ```bash
    export HTTPS_PROXY=http://squid.host-only:3128
    ```

---

## Testing Proxy Behavior

Once the proxy is running and your environment is configured, test both Docker registry access and in-container HTTPS access.

### Test Docker Pull

```bash
export HTTPS_PROXY=http://squid.host-only:3128
docker pull alpine
```

- If Docker trusts the Squid CA, the pull will succeed.
- If not, you’ll see x509 or certificate verification errors.

## Trusting the CA for Docker Pulls

Docker does not use the system trust store. To allow `docker pull` to work when HTTPS is intercepted by Squid, you must explicitly trust the Squid CA by placing it in Docker’s certificate directory:

```bash
sudo mkdir -p /etc/docker/certs.d/
sudo cp /etc/squid/mitm.crt /etc/docker/certs.d/
sudo systemctl restart docker
```

You can confirm Docker is using the proxy by watching the Squid logs while pulling:

```bash
docker pull alpine
```

```bash
sudo tail -f /var/log/squid/access.log | grep docker
```

This setup is sufficient for testing purposes. Docker will then trust any server certificates signed by the Squid CA.

### Test with OpenSSL (Raw Certificate Check)

```bash
openssl s_client -connect www.google.com:443 -proxy squid.host-only:3128 -CAfile /etc/squid/mitm.crt
```

### Test from Another Host (Linux or macOS)

You can verify Squid’s behavior from a different host using `curl`. These examples test HTTPS interception and validate that your CA is trusted.

#### Option 1: Explicit Proxy with `curl`

```bash
curl -I https://www.google.com --proxy http://squid.host-only:3128
```

- Replace `squid.host-only` with the IP address or hostname of your Squid proxy host.
- You should receive a `200 OK` response if the CA is trusted. Otherwise, you'll get a certificate error.

#### Option 2: Using `HTTPS_PROXY` Environment Variable

```bash
export HTTPS_PROXY=http://squid.host-only:3128
curl -I https://www.google.com
```

This has the same effect as `--proxy` but applies to all tools that honor `HTTPS_PROXY`.

---

## Trusting the CA Certificate

If you receive certificate errors, install the Squid CA (`mitm.crt`) on the client system:

### On Linux

```bash
sudo cp mitm.crt /usr/local/share/ca-certificates/
sudo update-ca-certificates
```

### On macOS

1. Copy `mitm.crt` to your local system.
2. Open **Keychain Access**.
3. Select **System** in the sidebar.
4. Drag `mitm.crt` into the window.
5. Double-click the cert → expand **Trust** → set **"When using this certificate"** to **Always Trust**.
6. Close and enter your password when prompted.

### On WSL2

WSL2 behaves like native Linux. Use the same instructions as for Linux to trust the CA inside your WSL2 distro.

After installing the cert, re-run the `curl` test — you should no longer see SSL errors.

### Converting Exported PEM or CER Files to CRT

When exporting a CA certificate from your browser or OS, it might have a `.pem` or `.cer` extension. These formats are usually identical to `.crt`. You can rename them safely:

```bash
mv my-cert.pem my-cert.crt
mv my-cert.cer my-cert.crt
```

Just ensure the file begins with:

```
-----BEGIN CERTIFICATE-----
```

If so, it can be used with `update-ca-certificates`, Docker, or as a trusted CA in testing.

---

## Optional: Verify CA Without Installing It

To test the Squid CA without installing it, you can use:

```bash
curl -I https://www.google.com --proxy http://squid.host-only:3128 --cacert /etc/squid/mitm.crt
```

This helps confirm that the proxy and CA work before trusting the cert system-wide.

---

### Quick Recap

| Environment      | Where to install CA                 | Trust command or method                     |
|------------------|-------------------------------------|---------------------------------------------|
| Linux (host)     | `/usr/local/share/ca-certificates`  | `update-ca-certificates`                    |
| macOS            | System Keychain                     | Keychain Access → "Always Trust"            |
| Docker Engine    | `/etc/docker/certs.d/ca.crt`        | `systemctl restart docker`                  |
| Inside container | `/usr/local/share/ca-certificates`  | `update-ca-certificates` inside container   |
| WSL2             | Same as Linux                       | `update-ca-certificates`                    |

---

## Monitoring Squid Logs to Verify Traffic Path

You can monitor the Squid log in another terminal to confirm proxy use:

```bash
sudo tail -f /var/log/squid/access.log
```

or for example

```bash
sudo tail -f /var/log/squid/access.log | grep docker
```

---

This `HTTPS_PROXY`-based setup is simpler and provides a very effective way to simulate real-world TLS inspection without needing DNS or firewall redirection.
