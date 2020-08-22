## Alternate Uses for DDEV-Local

### Continuous Integration (CI) for a project

Although it has not a primary goal of DDEV-Local, a number of people have found it easy to use DDEV-Local on a CI system like Github Actions or TravisCI or CircleCI to test out their projects. Instead of setting up a hosting environment for testing, they just start the project using DDEV and run their tests.

Examples of this approach are shown in [Codeception tests in Travis CI with DDEV and Selenium](https://dev.to/tomasnorre/codeception-tests-in-travis-ci-with-ddev-and-selenium-1607) and [Github Action Setup Ddev](https://github.com/jonaseberle/github-action-setup-ddev)

### Integration of DDEV-Local Docker Images Into Other Projects

It is possible to use DDEV-Local Docker images outside the context of the DDEV-Local environment. In the future we plan to share "hardened" images that share most of the characteristics of the regular DDEV-Local images, but pay more attention to security. For example, they will not have sudo installed inside them.

### Project Hosting on the Internet (including Let's Encrypt)

An experimental feature of DDEV-local is simplified small-project hosting on the internet. One can run DDEV-Local on an internet server and point their DNS to it and use it as a regular (though limited) hosting environment.

1. Install DDEV-Local on a regular Linux server that is directly connected to the Internet. You're responsible for your firewall and maintenance of the server, of course.
2. Use `ddev config global --router-bind-all-interfaces` to tell DDEV to listen to all network interfaces, not just localhost.
3. Use `ddev config global --use-letencrypt --letsencrypt-email=you@example.com` to configure Let's Encrypt.
4. Use a DNS provider to point a DNS name at the server.
5. Create your DDEV-Local project as you normally would, but `ddev config --project-tld=your-tld`. For example, if the top-level domain you're using were "ddev.example.com" you might use `ddev config --project-tld=ddev.example.com`
6. `ddev start` and visit your site.

Caveats:

* It's unknown how much traffic a given server and docker setup can sustain, or what the results will be if the traffic is more than the server can handle.
* Debugging Let's Encrypt failures requires viewing the ddev-router logs with `docker logs ddev-router`
* You will need to set up a startup script to make sure that your sites are restarted after a server restart.
