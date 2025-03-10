# Special Network Configurations

There are a few networking situations which occasionally cause trouble for some users. We can't explain every permutation of these, and since most happen in a corporate environment, you may need to confer with your IT department to sort them out.

## Some Corporate VPNs (including Zscaler)

Although DDEV (and Docker) work fine with most VPN systems, there are a number of VPNs (and similar products like Zscaler) which perform SSL/TLS interception and filtering. In other words, instead of connecting directly to HTTPS servers on the internet, they send traffic to the corporate networking infrastructure which can intercept HTTPS traffic and do SSL/TLS inspection. In a normal network, a client application can use HTTPS to connect directly to a server on the internet and can determine whether the server is what it says it is by examining its certificate and where the certificate was issued (the "CA" or "Certificate Authority"). Zscaler and similar products actually present their own certificates (which are trusted only because of corporate configuration) and are able to inspect traffic that would normally be encrypted, and then pass the traffic on to the end system if it is approved. This gives corporate networks extensive control over HTTPS traffic and its contents.

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
  and you should expect a "200 OK" response.

### Additional Resources

* For more approaches to resolving this, see [this Stack Overflow discussion](https://stackoverflow.com/questions/71595327/corporate-network-vpn-ddev-composer-create-results-in-ssl-certificate-proble).

## Corporate or Internet Provider Proxy



## Restrictive DNS servers
