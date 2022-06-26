## Alternate Uses for DDEV-Local

### Continuous Integration (CI) for a project

Although it has not a primary goal of DDEV-Local, a number of people have found it easy to use DDEV-Local on a CI system like GitHub Actions or TravisCI or CircleCI to test out their projects. Instead of setting up a hosting environment for testing, they just start the project using DDEV and run their tests.

Examples of this approach are shown in [Codeception tests in Travis CI with DDEV and Selenium](https://dev.to/tomasnorre/codeception-tests-in-travis-ci-with-ddev-and-selenium-1607) and [GitHub Action Setup Ddev](https://github.com/jonaseberle/github-action-setup-ddev)

### Integration of DDEV-Local Docker Images Into Other Projects

It is possible to use DDEV-Local Docker images outside the context of the DDEV-Local environment. People have used the ddev-webserver image for running tests in PhpStorm, for example.

### Casual Project Webhosting on the Internet (including Let's Encrypt)

An experimental feature of DDEV-local is simplified small-project hosting on the internet. One can run DDEV-Local on an internet server and point their DNS to it and use it as a regular (though limited) hosting environment.

This may be completely appropriate for small or abandoned sites that have special requirements like old versions of PHP that aren't supported elsewhere.

**Note that this is no replacement for a scalable managed hosting offering. It's unknown how much traffic it can handle in a given environment. And it's EXPERIMENTAL. And it will never replace managed hosting.**

1. Install DDEV-Local on a regular Linux server that is directly connected to the Internet. You're responsible for your firewall and maintenance of the server, of course.  
2. On Debian/Ubuntu, you can set up a simple firewall with `ufw allow 80 && ufw allow 443 && ufw allow 22 && ufw enable`
3. Point DNS for the site you're going to host to the server.
4. Before proceeding, your system and your project must be accessible on the internet on port 80 and your project DNS name (myproject.example.com) must resolve to the appropriate server.
5. Configure your project with `ddev config`
6. Import your database and files using `ddev import-db` and `ddev import-files`.
7. Use `ddev config global --router-bind-all-interfaces --omit-containers=dba,ddev-ssh-agent --use-hardened-images --use-letsencrypt --letsencrypt-email=you@example.com` to tell DDEV to listen to all network interfaces (not just localhost), not provide phpMyAdmin or ddev-ssh-agent, use the hardened images, and turn on Let's Encrypt.
8. Create your DDEV-Local project as you normally would, but `ddev config --additional-fqdns=<internet_fqdn`. If your website responds to multiple hostnames (for example, with "www" and without it) then you'll need to add each hostname.
9. `ddev start` and visit your site. Clear your cache (on some CMSs).

You may have to restart ddev with `ddev poweroff && ddev start --all` if Let's Encrypt has failed due to port 80 not being open or the DNS name not yet resolving. (Use `docker logs ddev-router` to see Let's Encrypt activity.)

#### Additional Server Setup

* Depending on how you're using this, you may want to set up automated database and files backups (preferably offsite) as on all production systems. Many CMSs have modules/plugins to allow this, or you can use `ddev export-db` or `ddev snapshot` as you see fit and do the backup on the host.
* You may want to allow your host system to send email (for notifications from the host itself). On Debian/Ubuntu `sudo apt-get install postfix`. Typically you'll need to set up reverse DNS for your system, and perhaps an SPF record in order for other systems to accept the email.
* You may want to update your php settings to use other than the defaults. For example, the error-reporting defaults in php.ini are very aggressive and you may want something less:

```ini
; Error handling and logging ;
error_reporting = E_ALL
display_errors = On
display_startup_errors = On
log_errors = On
```

* To make ddev start sites on system boot, you'll want to set up a systemd unit on systemd systems like Debian/Ubuntu and Fedora. For example, a file named /etc/systemd/system/ddev.service containing:

```
# Start ddev when system starts (after docker)
# Stop ddev when docker shuts down
# Start with `sudo systemctl start ddev`
# Enable on boot with `sudo systemctl enable ddev`
# Make sure to edit the User= for your user and the
# full path to ddev on your system.
# Optionally give a list of sites instead of --all
[Unit]
Description=DDEV-Local sites
After=network.target
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

* You will probably want to regularly renew the Let's Encrypt certificates. This is often done on a system reboot, but that may not be soon enough. A cron with the command `docker exec ddev-router bash -c "certbot renew && nginx -s reload"` will do the renewals.
* You'll likely want to turn off PHP errors to screen in a .ddev/php/noerrors.ini:

```ini
display_errors = Off
display_startup_errors = Off
```

Caveats:

* It's unknown how much traffic a given server and docker setup can sustain, or what the results will be if the traffic is more than the server can handle.
* DDEV-Local does not provide outgoing SMTP mailhandling service, and the development-focused MailHog feature is disabled if you're using `use_hardened_images`. You can provide SMTP service a number of ways, but the recommended way is to enable SMTP mailsending in your application and leverage a third-party transactional email service such as SendGrid, Mandrill, or Mailgun. This is the best way to make sure your mail actually gets delivered.
* You may need an external cron trigger for some types of CMS.
* Debugging Let's Encrypt failures requires viewing the ddev-router logs with `docker logs ddev-router`
* A malicious attack on a website hosted with `use_hardened_images` will likely not be able to do anything significant to the host, but it can certainly change your code, which is mounted on the host.

When using `use_hardened_images` docker runs the webimage as an unprivileged user, and the container does not have sudo. However, any docker server hosted on the internet is a potential vulnerability. Keep your packages up-to-date. Make sure that your firewall does not allow access to ports other than (normally) 22, 80, and 443.

There are no warranties implied or expressed.
