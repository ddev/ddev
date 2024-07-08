# Casual Hosting

!!!warning "Experimental Feature!"
    This is not a replacement for scalable, managed hosting. It’s unknown how much traffic it can handle in a given environment.

One of DDEV’s experimental features is lightweight hosting with Let’s Encrypt for HTTPS support. You can run DDEV on a public web server, point DNS to it, and use it as a limited hosting environment.

This may be appropriate for small or abandoned sites that have special requirements like old versions of PHP that aren’t supported elsewhere.

Here’s how to try it for yourself:

1. Install DDEV on an internet-connected Linux server. (You’re responsible for your firewall and maintenance of the server!)
2. On Debian/Ubuntu, you can set up a simple firewall with  
`ufw allow 80 && ufw allow 443 && ufw allow 22 && ufw enable`.
3. Point DNS for the site you’re going to host to the server.
4. Before proceeding, your system and your project must be accessible on the internet on port 80 and your project DNS name (`myproject.example.com`) must resolve to the appropriate server.
5. Configure your project with [`ddev config`](../usage/commands.md#config).
6. Import your database and files using [`ddev import-db`](../usage/commands.md#import-db) and [`ddev import-files`](../usage/commands.md#import-files).
7. Tell DDEV to listen on all network interfaces, omit the SSH agent, use hardened images, and enable Let’s Encrypt:

    ```
    ddev config global --router-bind-all-interfaces --omit-containers=ddev-ssh-agent --use-hardened-images --performance-mode=none --use-letsencrypt --letsencrypt-email=you@example.com
    ```

8. Create your DDEV project as you normally would, but `ddev config --project-name=<yourproject> --project-tld=<your-top-level-domain>`. If your website responds to multiple hostnames (e.g., with and without `www`), you’ll need to add `additional_hostnames`.
9. Redirect HTTP to HTTPS. Edit the `.ddev/traefik/config/<projectname>.yaml` to remove the `#ddev-generated` and uncomment the `middlewares:` and `- "redirectHttps"` lines in the HTTP router section.
10. Run [`ddev start`](../usage/commands.md#start) and visit your site. With some CMSes, you may also need to clear your cache.
11. If you see trouble with Let's Encrypt `ACME` failures, you can temporarily switch to the `ACME` staging server, and avoid getting rate-limited while you are experimenting. The certificates it serves will not be valid, but you'll see that they're coming from Let's Encrypt anyway. Add a `~/.ddev/traefik/static_config.staging.yaml` with the contents:

    ```yaml
    certificatesResolvers:
    acme-tlsChallenge:
        acme:
            caServer: "https://acme-staging-v02.api.letsencrypt.org/directory"
    ```

You may have to restart DDEV with `ddev poweroff && ddev start --all` if Let’s Encrypt has failed for some reason or the DNS name is not yet resolving. (Use `docker logs -f ddev-router` to see Let’s Encrypt activity.)

## Additional Server Setup

* Depending on how you’re using this, you may want to set up automated database and file backups—ideally off-site—like you would on any production system. Many CMSes have modules/plugins to allow this, and you can use `ddev export-db` or `ddev snapshot` as you see fit and do the backup on the host.
* You may want to allow your host system to send email. On Debian/Ubuntu `sudo apt-get install postfix`. Typically you’ll need to set up reverse DNS for your system, and perhaps SPF and/or DKIM records to for more reliable delivery to other mail systems.
* You may want to generally tailor your PHP settings for hosting rather than local development. Error-reporting defaults in `php.ini`, for example, may be too verbose and expose too much information publicly. You may want something less:

    ```ini
    ; Error handling and logging ;
    error_reporting = E_ALL
    display_errors = On
    display_startup_errors = On
    log_errors = On
    ```

* To make DDEV start sites on system boot, you’ll want to set up a `systemd` unit on systems like Debian/Ubuntu and Fedora. For example, a file named `/etc/systemd/system/ddev.service` containing:

    ```
    # Start DDEV when system starts (after Docker)
    # Stop DDEV when Docker shuts down
    # Start with `sudo systemctl start ddev`
    # Enable on boot with `sudo systemctl enable ddev`
    # Make sure to edit the User= for your user and the
    # full path to `ddev` on your system.
    # Optionally give a list of sites instead of --all
    [Unit]
    Description=DDEV sites
    After=multi-user.target
    Requires=docker.service
    PartOf=docker.service
    [Service]
    User=rfay
    Type=oneshot
    ExecStart=/usr/local/bin/ddev start --all
    RemainAfterExit=true
    ExecStop=/usr/local/bin/ddev poweroff

    [Install]
    WantedBy=multi-user.target
    ```

* You’ll likely want to turn off PHP errors to screen in a `.ddev/php/noerrors.ini`:

    ```ini
    display_errors = Off
    display_startup_errors = Off
    ```

## Caveats

* It’s unknown how much traffic a given server and Docker setup can sustain, or what the results will be if the traffic is more than the server can handle.
* DDEV does not provide outgoing SMTP mail handling service, and the development-focused Mailpit feature is disabled if you’re using `use_hardened_images`. You can provide SMTP service a number of ways, but the recommended way is to use SMTP in your application via a third-party transactional email service such as [SendGrid](https://sendgrid.com), [Postmark](https://postmarkapp.com), or [Mailgun](https://www.mailgun.com). This is the best way to ensure mail is actually delivered.
* You may need an external cron trigger for some CMSes.
* Debugging Let’s Encrypt failures requires viewing the `ddev-router` logs with `docker logs ddev-router`.
* A malicious attack on a website hosted with `use_hardened_images` will likely not be able to do anything significant to the host, but it can certainly change your code, which is mounted on the host.

When `use_hardened_images` is enabled, Docker runs the web image as an unprivileged user, and the container does not have sudo. However, any Docker server hosted on the internet is a potential vulnerability. Keep your packages up to date and make sure your firewall does not allow access to ports other than (normally) 22, 80, and 443.

There are no warranties implied or expressed.
