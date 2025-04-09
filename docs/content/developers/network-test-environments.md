---
search:
  boost: .1
---
# Network Test Environments

## Packet-Inspection VPN Simulation

### Simulating SSL Interception with Squid (Simplified via `HTTPS_PROXY`)

A straightforward way to simulate a packet-inspecting VPN is by using **Squid** with SSL bumping and configuring your environment to use it via `HTTPS_PROXY`. While it's less transparent than a full MITM router setup, it closely replicates the behavior of Zscaler and similar tools from the perspective of apps like Docker and `curl`.

#### Setup Overview

- Squid listens on port 3128.
- HTTPS traffic routed via the proxy is intercepted and re-signed with a custom CA.
- Clients that trust this CA will succeed; others will fail SSL validation.
- You simulate VPN-like interception by exporting `HTTPS_PROXY=http://localhost:3128`.

#### Step-by-Step Instructions

These instructions are for Debian/Ubuntu but can be adapted for container-based setup or for another environment.

1. **Install Squid**

    ```bash
    sudo apt install squid-openssl ssl-cert
    ```

2. **Generate a Root CA for Signing**
  This `mitm.crt` becomes the certificate which is used everywhere that we need the custom CA.

    ```bash
    openssl req -new -newkey rsa:2048 -days 365 -nodes -x509 \
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

### Testing Proxy Behavior

Once the proxy is running and your environment is configured, test both Docker registry access and in-container HTTPS access.

#### Test Docker Pull

```bash
export HTTPS_PROXY=http://squid.host-only:3128
docker pull alpine
```

- If Docker trusts the Squid CA, the pull will succeed.
- If not, you’ll see x509 or certificate verification errors.

To trust the CA for Docker:

```bash
sudo mkdir -p /etc/docker/certs.d/
sudo cp /etc/squid/mitm.crt /etc/docker/certs.d/
sudo systemctl restart docker
```

#### Test with OpenSSL (Raw Certificate Check)

```bash
openssl s_client -connect www.google.com:443 -proxy squit.host-only:3128 -CAfile /etc/squid/mitm.crt
```

#### Test from Another Host (Linux or macOS)

You can verify Squid’s behavior from a different host using `curl`. These examples test HTTPS interception and validate that your CA is trusted.

##### Option 1: Explicit Proxy with `curl`

```bash
curl -I https://www.google.com --proxy http://squid.host-only:3128
```

- Replace `squid.host-only` with the IP address or hostname of your Squid proxy host.
- You should receive a `200 OK` response if the CA is trusted. Otherwise, you'll get a certificate error.

##### Option 2: Using `HTTPS_PROXY` Environment Variable

```bash
export HTTPS_PROXY=http://squid.host-only:3128
curl -I https://www.google.com
```

This has the same effect as `--proxy` but applies to all tools that honor `HTTPS_PROXY`.

---

### Trusting the CA Certificate

If you receive certificate errors, install the Squid CA (`mitm.crt`) on the client system:

#### On Linux

```bash
sudo cp mitm.crt /usr/local/share/ca-certificates/
sudo update-ca-certificates
```

#### On macOS

1. Copy `mitm.crt` to your local system.
2. Open **Keychain Access**.
3. Select **System** in the sidebar.
4. Drag `mitm.crt` into the window.
5. Double-click the cert → expand **Trust** → set **"When using this certificate"** to **Always Trust**.
6. Close and enter your password when prompted.

After installing the cert, re-run the `curl` test — you should no longer see SSL errors.

---

This `HTTPS_PROXY`-based setup is simpler and provides a very effective way to simulate real-world TLS inspection without needing DNS or firewall redirection.
