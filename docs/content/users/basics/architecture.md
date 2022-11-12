# How DDEV Works

DDEV is a [Go](https://go.dev) application that stores its configuration in [files on your workstation](#directory-tour).  
It uses those blueprints to mount your project files into [Docker containers](#container-architecture) that facilitate the operation of a local development environment.

DDEV writes and uses [docker-compose](https://docs.docker.com/compose/) files for you, which is a detail you can cheerfully ignore unless you’re Docker-curious or [defining your own services](../extend/custom-compose-files.md).

## Directory Tour

A project’s `.ddev` directory can be intimidating at first, so let’s take a look at what lives in there.

!!!tip "Yours May Differ Slightly"
    You may have some directories or files that aren’t listed here, likely added by custom services.  
    For example, if you see a `solr` directory, it probably pertains to a custom Solr [add-on service](../extend/additional-services.md).

* `apache` directory: Apache configuration for those using `webserver_type: apache-fpm`. There are docs and the default configuration in there. See [Apache customization docs](../extend/customization-extendibility.md#providing-custom-apache-configuration).
* `commands` subdirectories: Contains DDEV shell commands, both built-in and custom, that can run on the host or inside any container. See [docs](../extend/custom-commands.md).
* `config.yaml` file: This is the basic configuration file for the project. Take a look at the comments below for suggestions about things you can do, or look in [docs](../configuration/config.md)).
* `config.*.yaml` files: You can add configuration here that overrides parts of `config.yaml`. This is nice for situations where one developer’s project needs one-off configuration. For example, you could turn on `nfs-mount-enabled` or `mutagen-enabled` or use a different database type. By default, these are gitignored and not get checked in. See [docs](../extend/customization-extendibility.md#extending-configyaml-with-custom-configyaml-files).
* `db-build` directory: Can be used to provide a custom Dockerfile for the database container.
* `db_snapshots` directory: This is where snapshots go when you `ddev snapshot`. If you don’t need these backups, you can delete anything there at any time. See [snapshot docs](../basics/cli-usage.md#snapshotting-and-restoring-a-database).
* `docker-compose.*.yaml` files: Advanced users can provide their own services or service overrides using `docker-compose.*.yaml` files. See [custom compose files](../extend/custom-compose-files.md) and [additional services](../extend/additional-services.md). Also see the many examples in [ddev-contrib](https://github.com/drud/ddev-contrib).
* `homeadditions` directory: Anything you put in the `homeadditions` directory (including both files and directories) will be copied into the web container on startup. This lets you easily override the default home directory contents (`.profile`, `.bashrc`, `.composer`, `.ssh`) or anything you want to put in there. It could also include scripts that you want to have easily available inside the container. (You can do the same thing globally in `~/.ddev/homeadditions`.) See [homeadditions docs](../extend/in-container-configuration.md).
* `mutagen` directory: contains `mutagen/mutagen.yml` where you can override the default Mutagen configuration. See [mutagen docs](../install/performance.md#advanced-mutagen-configuration-options).
* `mysql` directory: contains optional `mysql` or `mariadb` configuration. See [MySQL docs](../extend/customization-extendibility.md#providing-custom-mysqlmariadb-configuration-mycnf).
* `nginx` directory: (deprecated) can be used for add-on nginx snippets.
* `nginx_full` directory: Contains the nginx configuration used by the web container, which can be customized following the instructions there. See [providing custom nginx configuration](../extend/customization-extendibility.md#providing-custom-nginx-configuration).
* `postgres` directory: contains `postgres/postgresql.conf`, which can be edited if needed. Remove the `#ddev-generated` line at the top to take it over.
* `providers` directory: Contains examples and implementations showing ways to configure DDEV so `ddev pull` can work. You can use `ddev pull` with [hosting providers](../providers/index.md) like Acquia, Platform.sh, or Pantheon, as well as with local files or custom database/files sources.
* `web-build` directory: You can add a custom Dockerfile that adds things into the Docker image used for your web container. See [Customizing images](../extend/customizing-images.md).
* `xhprof` directory: Contains the `xhprof_prepend.php` file that can be used to customize [xhprof](../debugging-profiling/xhprof-profiling.md) behavior for different types of websites.

### Look But Don’t Touch!

The hidden files (beginning with `.`) are not intended to be fiddled with, and are hidden for that reason. Most are regenerated, and thus overwritten, on every `ddev start`:

* `.dbimageBuild` directory: The generated Dockerfile used to customize the `db` container on first start.
* `.ddev-docker-compose-base.yaml`: The base docker-compose file used to describe a project.
* `.ddev-docker-compose-full.yaml`: This is the result of preprocessing `.ddev-docker-compose-base.yaml` using `docker-compose config`. Mostly it replaces environment variables with their values.
* `.gitignore`: The `.gitignore` is generated by DDEV and should generally not be edited or checked in. (It gitignores itself to make sure you don’t check it in.) It’s generated on every `ddev start` and will change as DDEV versions change, so if you check it in by accident it will always be showing changes that you don’t need to see in `git status`.
* `.global_commands`: This is a temporary directory used to get global commands available inside a project. You shouldn’t ever have to look there.
* `.homeadditions`: This is a temporary directory used to consolidate global `homeadditions` with project-level `homeadditions`. You shouldn’t ever have to look here.
* `.webimageBuild` directory: The generated Dockerfile used to customize the web container on first start.

## Container Architecture

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

![DDEV Docker Network Architecture](../../images/container-diagram.png)
