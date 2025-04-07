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

1. **Install Squid**

    ```bash
    sudo apt install squid ssl-cert
    ```

2. **Generate a Root CA for Signing**

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
    export HTTPS_PROXY=http://localhost:3128
    ```

---

### Testing Proxy Behavior

Once the proxy is running and your environment is configured, test both Docker registry access and in-container HTTPS access.

#### Test Docker Pull

```bash
export HTTPS_PROXY=http://localhost:3128
docker pull alpine
```

- If Docker trusts the Squid CA: Pull will succeed.
- If not: You’ll see x509 or certificate verification errors.

To trust the CA for Docker:

```bash
sudo mkdir -p /etc/docker/certs.d/hub.docker.com/
sudo cp /etc/squid/mitm.crt /etc/docker/certs.d/hub.docker.com/ca.crt
sudo systemctl restart docker
```

#### Test with OpenSSL (Raw Certificate Check)

```bash
openssl s_client -connect www.google.com:443 -proxy localhost:3128 -CAfile /etc/squid/mitm.crt
```

#### Test Inside Container

If you're using DDEV or a custom container image:

```bash
ddev exec curl -I https://www.google.com
```

If it fails, copy `mitm.crt` into `.ddev/web-build/`, then in your Dockerfile:

```Dockerfile
COPY mitm.crt /usr/local/share/ca-certificates/
RUN update-ca-certificates
```

Restart DDEV:

```bash
ddev restart
```

Re-run the curl command and expect a `200 OK` response.

---

This `HTTPS_PROXY`-based setup is simpler and provides a very effective way to simulate real-world TLS inspection without needing DNS or firewall redirection.
