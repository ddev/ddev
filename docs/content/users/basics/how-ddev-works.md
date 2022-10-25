# How DDEV Works

It’s easiest to think of DDEV as a set of little networked computers (Docker containers) that are in a different network from your workstation but still reachable from it.

When you install or upgrade DDEV you’re mostly installing a single `ddev` binary. When you use it, it downloads the Docker images it needs, and then starts them based on what’s needed for your projects.

* The `ddev-webserver` container (one per project) runs `nginx` or `apache` and `php-fpm` for a single site, so it does all the basic work of a PHP-interpreting web server.
* The `ddev-dbserver` container (one per project) handles MariaDB/MySQL/PostgreSQL database management. It can be reached from the web server by the hostname `db` or with the more explicit name `ddev-<projectname>-db`.
* The optional `dba` container runs phpMyAdmin for projects with MySQL or MariaDB.
* Additional add-on services may be there for a given project, for example `solr` or `elasticsearch` or `memcached`.

Although it’s not common usage, different projects can communicate with each other as described in the [FAQ](faq.md#can-different-projects-communicate-with-each-other).

Now for the two oddball global containers (there’s only one of each):

* The `ddev-router` container is a “reverse proxy”. It takes incoming HTTP/S requests, looks up the hostname in the incoming URL, and routes it to the correct project’s `ddev-webserver`. Depending on the project’s configuration with `additional_hostnames` and `additional_fqdns`, it can route many different URLs to a single project’s `ddev-webserver`. If, like most people, you use the named URLs like `https://something.ddev.site`, your request goes through the router. When you use the `127.0.0.1` URLs, the requests go directly to the `ddev-webserver`.
* The `ddev-ssh-agent` container runs an `ssh-agent` inside the Docker network so that after you do a `ddev auth ssh` all the different projects can use your SSH keys for outgoing requests—like private Composer access or SCP from a remote host.

Here’s a basic diagram of how it works inside the Docker network:

![DDEV Docker Network Architecture](../topics/images/DDEV%20Container%20Architecture.svg)
