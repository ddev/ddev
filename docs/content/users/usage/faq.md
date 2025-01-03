---
search:
  boost: 2
---

# FAQ

Frequently-asked questions organized into high-level functionality, investigating issues, daily usage, and connecting with our community.

## Features & Requirements

### What operating systems will DDEV work with?

DDEV works nearly anywhere Docker will run, including macOS, WSL2, Windows 10/11 Pro/Enterprise and Home, and every Linux variant we’ve ever tried. It also runs in many Linux-like environments, like ChromeOS (in Linux machine). DDEV works the same on each of these platforms since the important work is done inside identical Docker containers. This means that a team using diverse environments can share everything without trouble.

### Does DDEV change or deploy my code?

You are responsible for your code and its deployment. DDEV does not alter any code or fix any bugs in it. DDEV *does* add DDEV-specific settings for some CMSes if the [settings management](cms-settings.md) is enabled. These items are excluded by `.gitignore` so they won't affect a deployed project, but in most cases they would do no harm if deployed, because they check to see if they're running in DDEV context.

### Why do I have to type `ddev` in front of so many commands?

When you are using commands like `ddev composer`, `ddev drush`, `ddev npm`, or `ddev yarn`, you are telling DDEV to execute that very command inside the web container. That is where the exact tool for the exact environment required by your project lives. It's possible to execute `composer install` without  prepending `ddev` in your project folder, but often you won't have the same PHP version on your host computer as your project is configured to use inside the container, or perhaps you'll have a different version of `composer` even. This can lead into workarounds like having to use `composer --ignore-platform-reqs` or even introducing incompatibilities  into your project. With tools like `ddev composer` you are able to run several projects at the same time, each with different configurations, but when you use the tool inside the container, you get the exact configuration for the project you've configured. You can run any tool inside the web container with `ddev exec`, but many commands like `ddev composer` have two-word shortcuts.

### Where is my database stored in my DDEV project?

The MariaDB, MySQL, or PostgreSQL database for your project lives in a Docker volume, which means it does not appear in your DDEV project's filesystem, and is not checked in. This configuration is for performance and portability reasons, but it means that if you change Docker providers or do a factory reset on your Docker provider, you will lose databases. By default many Docker providers do not keep Docker volumes where they are backed up by normal backup solutions. Remember to keep backups using `ddev export-db` or `ddev snapshot`. See [How can I migrate from one Docker provider to another](#how-can-i-migrate-from-one-docker-provider-to-another).

### What Docker providers can I use?

We have automated testing and support for a staggering range of Docker providers.

| Docker Provider            | Support Level                                                            |
|----------------------------|--------------------------------------------------------------------------|
| OrbStack (macOS)           | officially tested and supported on macOS                                 |
| Docker Desktop for Mac     | officially tested and supported on both Intel and Apple Silicon          |
| Docker Desktop for Windows | officially tested and supported on WSL2 and traditional Windows          |
| Colima (macOS)             | officially tested and supported, no longer recommended                   |
| Colima (Linux)             | reported working in DDEV v1.22+, but poor solution compared to docker-ce |
| Docker-ce (Linux/WSL2)     | officially supported with automated tests on WSL2/Ubuntu. Recommended.   |
| Rancher Desktop (macOS)    | officially tested and supported on macOS                                 |

* Docker Desktop for Linux does *not* work with DDEV because it mounts all files into the container owned as root.
* Rancher Desktop for Windows does not work with DDEV.

### How can I migrate from one Docker provider to another?

There are many Docker providers on DDEV’s supported platforms. For example, on macOS people use Docker Desktop and OrbStack along with Colima and Rancher Desktop. On Windows WSL2, people may use Docker Desktop or Docker CE inside WSL2. In all cases, if you want to switch between Docker providers, save your database and make sure the Docker providers don't interfere with each other:

1. Save away your projects' databases. You can run `ddev snapshot --all` to make snapshots of all *registered* projects (that show up in `ddev list`). If you prefer a different way of saving database dumps, that works too!
2. Stop the Docker provider you're moving from. For example, exit Docker Desktop.
3. Start the Docker provider you're moving to.
4. Start projects and restore their databases. For example, you could run `ddev snapshot restore --latest` to load a snapshot taken in step one.

### Do I need to install PHP, Composer, nginx, or Node.js/npm on my workstation?

No. Tools like PHP, Composer, nginx, and Node.js/npm live inside DDEV’s Docker containers, so you only need to [install Docker](../install/docker-installation.md) and [install DDEV](../install/ddev-installation.md).

For most users we recommend that you do *not* install PHP or composer on your workstation, so you get in the habit of using `ddev composer`, which will use the configured composer and PHP versions for your project, which can be different for each project. See [DDEV and Composer](developer-tools.md#ddev-and-composer).

### Do I lose data when I run `ddev poweroff`, `ddev stop`, or `ddev restart`?

No. Your code continues to live on your workstation, and your database is safely stored on a Docker volume—both unaffected by these commands.

### How can I connect to my database?

The answer depends on where you’re connecting *from*.

The [`ddev describe`](../usage/commands.md#describe) command includes database connection details in a row like this:

```
│ db         │ OK   │ InDocker: ddev-mysite-db:3306 │ mariadb:10.3       │
│            │      │ Host: localhost:63161         │ User/Pass: 'db/db' │
│            │      │                               │ or 'root/root'     │
```

Inside your project container, where the app itself is running, the database hostname is `db`
(**not** `127.0.0.1`) and the port is the default for your database engine—`3306` for MySQL/MariaDB, `5432` for PostgreSQL.

Outside your project’s web container, for example a database GUI on your workstation, the hostname is `localhost` and the port is unique to that project. In the example above, it’s `63161`.

The username, password, and database are each `db` regardless of how you connect.

### Can I use additional databases with DDEV?

Yes, you can create additional databases and manually do whatever you need on them. They’re created automatically if you use `ddev import-db` with the `--database` option. In this example, `extradb.sql.gz` is extracted and imported to a newly-created database named `extradb`:

```
ddev import-db --database=extradb --file=.tarballs/extradb.sql.gz
```

You can use [`ddev mysql`](../usage/commands.md#mysql) or `ddev psql` to execute queries, or use the MySQL/PostgreSQL clients within `ddev ssh` or `ddev ssh -s db`. See the [Database Management](database-management.md) page.

### Can different projects communicate with each other?

Yes, this is commonly required for situations like Drupal migrations or server-side API calls between projects.

#### Communicate with database of other project

For the `web` container to access the `db` container of another project, use `ddev-<projectname>-db` as the hostname of the other project.

Let’s say we have two projects, for example: project A, and project B.

In project A, use `mysql -h ddev-projectb-db` to access the database server of project B.

#### Communicate via HTTP/S

Let’s say we have two projects, for example: project A, and project B.

To enable server-side HTTP/S communication (i.e. server-side API calls) between projects you can:

1. Either access the web container of project B directly with the hostname `ddev-<projectb>-web` and port 80 or 443 from project A:

    ```bash
    # call from project A web container to project B's web container
    curl https://ddev-projectb-web
    ```

2. Or add a `.ddev/docker-compose.communicate.yaml` to project A:

    ```yaml
    # add this to project A, allows connection to project B
    services:
      web:
        external_links:
          - "ddev-router:projectb.ddev.site"
    ```

    This lets the `ddev-router` know that project A can access the web container on project B's official FQDN.

    You can now make calls to project B via the regular FQDN `https://projectb.ddev.site` from project A:

    ```bash
    # call from project A web container to project B's web container
    curl https://projectb.ddev.site
    ```

    If you are using other hostnames or `project_tld`, you will need to adjust the `projectb.ddev.site` value.

### Can I run DDEV with other development environments at the same time?

Yes, as long as they’re configured with different ports. It doesn’t matter whether your other environments use Docker or not, it should only be a matter of avoiding port conflicts.

It’s probably easiest, however, to shut down one before using the other.

For example, if you use Lando for one project, do a `lando poweroff` before using DDEV, and then run [`ddev poweroff`](../usage/commands.md#poweroff) before using Lando again. If you run nginx or Apache locally, stop them before using DDEV. The [troubleshooting](troubleshooting.md) section goes into more detail about identifying and resolving port conflicts.

## Performance & Troubleshooting

### How can I get the best performance?

Docker’s normal mounting can be slow, especially on macOS. See the [Performance](../install/performance.md) section for speed-up options including Mutagen and NFS mounting.

### How can I troubleshoot what’s going wrong?

See the [troubleshooting](troubleshooting.md), [Docker troubleshooting](../install/docker-installation.md#testing-and-troubleshooting-your-docker-installation) and [Xdebug troubleshooting](../debugging-profiling/step-debugging.md#troubleshooting-xdebug) sections.

### How can I check that Docker is working?

See the [troubleshooting section](../install/docker-installation.md#troubleshooting) on the Docker Installation page.

### Why do I get a 403 or 404 on my project after `ddev launch`?

Most likely because the docroot is misconfigured, or there’s no `index.php` or `index.html` in it. Open your `.ddev/config.yaml` file and check the [`docroot`](../configuration/config.md#docroot) value, which should be a relative path to the directory containing your project’s `index.php`.

### Why do I see nginx headers when I’ve set `webserver_type: apache-fpm`?

Apache runs in the web container, but when you use the `https://*.ddev.site` URL, it goes through `ddev-router`, which is an nginx reverse proxy. That’s why you see nginx headers even though your web container’s using Apache. Read more in [this Stack Overflow answer](https://stackoverflow.com/a/52780601/215713).

### Why does `ddev start` fail with “error while mounting volume, Permission denied”?

This almost always means NFS is enabled in your project, but NFS isn’t working on your machine.

Start by completely turning NFS off for your projects with `ddev config --performance-mode=none && ddev config global --performance-mode=none`. Then later, [get NFS working](../install/performance.md#using-nfs-to-mount-the-project-into-the-web-container). NFS can improve macOS and traditional Windows performance, but is never needed on Linux or Windows WSL2. Most people on macOS and Windows use Mutagen instead of NFS because of its vastly improved performance, so instead of trying to fix this you can use Mutagen which is enabled by default. On Linux you can enable Mutagen for the project by running `ddev config --performance-mode=mutagen` or globally `ddev config global --performance-mode=mutagen`.

### Why are my Apache HTTP → HTTPS redirects stuck in an infinite loop?

It’s common to set up HTTP-to-TLS redirects in an `.htaccess` file, which leads to issues with the DDEV proxy setup. The TLS endpoint of a DDEV project is always the `ddev-router` container and requests are forwarded through plain HTTP to the project’s web server. This results in endless redirects, so you need to change the root `.htaccess` file for Apache correctly handles these requests for your local development environment with DDEV. The following snippet should work for most scenarios—even outside of DDEV—and could replace an existing redirect:

```apache
# http:// -> https:// plain or behind proxy for Apache 2.2 and 2.4
# behind proxy
RewriteCond %{HTTP:X-FORWARDED-PROTO} ^http$
RewriteRule (.*) https://%{HTTP_HOST}/$1 [R=301,L]

# plain
RewriteCond %{HTTP:X-FORWARDED-PROTO} ^$
RewriteCond %{REQUEST_SCHEME} ^http$ [NC,OR]
RewriteCond %{HTTPS} off
RewriteRule (.*) https://%{HTTP_HOST}/$1 [R=301,L]
```

### My browser redirects `http` URLs to `https`

Several browsers want you to use `https`, so they will automatically redirect you to the `https` version of a site. This may not be what you want, and things may break on redirect. For example, the Apache SOLR web UI often doesn't work with `https`, and when it redirects it things might break.

To solve this for your browser, see:

* [Google Chrome](https://stackoverflow.com/q/73875589)
* [Mozilla Firefox](https://stackoverflow.com/q/30532471)
* [Safari](https://stackoverflow.com/q/46394682)

### Why is `ddev-webserver` such a huge Docker image?

When you update DDEV you'll see it pull a `ddev-webserver` image which is almost half a gigabyte compressed, and this can be an inconvenient thing to wait for when you're doing an upgrade, especially if you have a slow internet connection.

The reason that `ddev-webserver` is so big is that it's built for your daily requirements for a local development environment. It lets you switch PHP versions or switch between `nginx` and `apache` web servers with a simple `ddev restart`, rather than a lengthy build process. It lets you use Xdebug with a simple `ddev xdebug on`. It has many, many features and tools that make it easy for you as a developer, but that one would not include in a production image.

## Workflow

### How can I update/upgrade DDEV?

See **[Upgrading DDEV](../install/ddev-upgrade.md)** for your operating system and installation technique.

You can use the [`ddev self-upgrade`](../usage/commands.md#self-upgrade) command for quick instructions tailored to your installation.

### How can I install a specific version of DDEV?

#### Debian, Ubuntu, or WSL2

For Debian/Ubuntu/WSL2 with DDEV installed via apt, you can run `sudo apt-get update && sudo apt-get install ddev=<version>`, for example `sudo apt-get install ddev=1.23.4` to run a previous, older version of DDEV.

#### Homebrew

If you’re using Homebrew, first run `brew unlink ddev` to get rid of the version you have there. Then use one of these options:

1. Download the version you want from the [releases page](https://github.com/ddev/ddev/releases) and place it in your `$PATH`.
2. Use the [install_ddev.sh](https://raw.githubusercontent.com/ddev/ddev/main/scripts/install_ddev.sh) script with the version number argument. For example, if you want `v1.23.5`, run `curl -fsSL https://ddev.com/install.sh | bash -s v1.23.5`.
3. If you want the very latest, unreleased version of DDEV, run `brew unlink ddev && brew install ddev/ddev/ddev --HEAD`.

### Why do I have an old DDEV?

You may have installed DDEV several times using different techniques. Use `which -a ddev` to find all installed binaries. For example, you could install a DDEV in WSL2 with Homebrew, forget about it for a while, install it manually, and then install it again with `apt`:

```bash
$ which -a ddev
/home/linuxbrew/.linuxbrew/bin/ddev # installed with Homebrew
/usr/local/bin/ddev # installed manually with install_ddev.sh script
/usr/bin/ddev # installed with apt or yum/rpm
/bin/ddev # don't touch it, it's a link to /usr/bin/ddev
```

You can check each binary version by its full path (`/usr/bin/ddev --version`) to find old versions. Remove them preferably in the same way you installed them, i.e. `/home/linuxbrew/.linuxbrew/bin/ddev` should be removed with Homebrew: `brew uninstall ddev`. A manually installed DDEV can be removed by deleting the `ddev` binary.

Restart the terminal (or run `hash -r`) after uninstalling other versions of DDEV for the changes to take effect.

If you see duplicates in the `which -a ddev` output, it means that some directories are added to your `$PATH` more than once. You can either ignore this or remove the extra directory from your `$PATH`.

### Should I check in the `.ddev` directory? How about add-ons?

Most teams check in the project `.ddev` directory. That way all team members will have the exact same configuration for the project, even if they're on different operating systems or architectures or Docker providers.

DDEV [add-ons](../extend/additional-services.md) are installed via the `.ddev` directory, so checking things in will get them as well, and that's also recommended practice.

Do *not* alter or check in the `.ddev/.gitignore` as it is automatically generated to DDEV and does its best to figure out what files you "own" (like the `.ddev/config.yaml`) and which files DDEV "owns", so do not have to be committed.

### How can I back up or restore all project databases?

You can back up all projects that show in `ddev list` with `ddev snapshot -a`. This only snapshots projects displayed in `ddev list`; any projects not shown there will need to be started so they’re be registered in `ddev list`.

### How can I share my local project with someone?

See [Sharing Your Project](../topics/sharing.md).

### How do I make DDEV match my production environment?

You can change the major PHP version and choose between nginx+fpm (default) and Apache+fpm and choose the MariaDB/MySQL/PostgreSQL version add [extra services like Solr and Memcached](../extend/additional-services.md). You won’t be able to make every detail match your production server, but with database server type and version, PHP version and web server type you’ll be close.

The [lightly maintained rfay/ddev-php-patch-build add-on](https://github.com/rfay/ddev-php-patch-build) may allow you to use a specific PHP patch version.

### How do I completely destroy a project?

Use [`ddev delete <project>`](../usage/commands.md#delete) to destroy a project. By default, a [`ddev snapshot`](../usage/commands.md#snapshot) of your database is taken, but you can skip this using `ddev delete --omit-snapshot` or `ddev delete --omit-snapshot -y`. See `ddev delete -h` for options. It’s up to you to then delete the code directory.

### What if I don’t like the settings files or gitignores DDEV creates?

You have several options:

* Use the [`disable_settings_management: true`](../configuration/config.md#disable_settings_management) option in the project’s `.ddev/config.yaml` file. This disables DDEV from updating CMS-related settings files.
* Use the more generic “php” project type rather than a CMS-specific one; it basically means “don’t try to create settings files for me”. The “php” type works great for experienced developers.
* Take over the settings file or `.gitignore` by deleting the line `#ddev-generated` in it, then check in the file. If that line is removed, DDEV will not try to replace or change the file.

### How can I change a project’s name?

Delete it and migrate it to a new project with your preferred name:

1. Export the project’s database: `ddev export-db --file=/path/to/db.sql.gz`.
2. Delete the project: `ddev delete <project>`. (This takes a snapshot by default for safety.)
3. Rename the project: `ddev config --project-name=<new_name>`.
4. Start the new project with `ddev start`.
5. Import the database dump from step one: `ddev import-db --file=/path/to/db.sql.gz`.

### How can I move a project to another directory?

Run [`ddev stop --unlist`](../usage/commands.md#stop), then move the directory, then run [`ddev start`](../usage/commands.md#start) in the new directory.

### How can I move a project to another workstation?

Take a snapshot, move the project files, and restore the snapshot in a new project on the target workstation:

1. `ddev start && ddev snapshot`.
2. `ddev stop --unlist`.
3. Move the project directory to another computer any way you want.
4. On the new computer, run `ddev start && ddev snapshot restore --latest`.
5. Optionally, on the old computer, run `ddev delete --omit-snapshot` to remove its copy of the database.

### How can I move a project from traditional Windows to WSL2?

This is exactly the same as moving a project from one computer to another described above. Make sure you move the project into a native filesystem in WSL2, most likely `/home`.

### Why does DDEV want to edit `/etc/hosts`?

If you see “The hostname <hostname> is not currently resolvable” and you can successfully `ping <hostname>`, it may be that DNS resolution is slow.

DDEV doesn’t have control over your computer’s name resolution, so it doesn’t have any way to influence how your browser gets an IP address from a hostname. It knows you have to be connected to the internet to do that, and uses a test DNS lookup of `<somethingrandom>.ddev.site` as a way to guess whether you’re connected to the internet. If it’s unable to do a name lookup, or if the hostname associated with your project is not `*.ddev.site`, it will try to create entries in `/etc/hosts`, since it’s assuming you can’t look up your project’s hostname(s) via DNS. If your internet (and name resolution) is actually working, but DNS is slow, run `ddev config global --internet-detection-timeout-ms=3000` to set the timeout to 3 seconds (or higher). See [this GitHub issue](https://github.com/ddev/ddev/issues/2409#issuecomment-662448025) for more. (If DNS rebinding is disallowed on your network/router, this won’t be solvable without network/router changes. Help [here](https://github.com/ddev/ddev/issues/2409#issuecomment-675083658) and [here](https://github.com/ddev/ddev/issues/2409#issuecomment-686718237).) For more detailed troubleshooting information, please see the [troubleshooting section](troubleshooting.md#ddev-starts-but-browser-cant-access-url).

### How can I configure a project with the defaults without hitting <kbd>RETURN</kbd> a bunch of times?

Use `ddev config --auto` to set the docroot and project type based on the discovered code.
If anything in `.ddev/config.yaml` is wrong, you can edit that directly or use [`ddev config`](../usage/commands.md#config) commands to update settings.

## Getting Involved

### How do I get support?

See the [support options](../support.md), including [Discord](https://ddev.com/s/discord), [Stack Overflow](https://stackoverflow.com/questions/tagged/ddev) and the [issue queue](https://github.com/ddev/ddev/issues).

### How can I contribute to DDEV?

We love and welcome contributions of knowledge, support, docs, and code:

* Submit an issue or pull request to the [main repository](https://github.com/ddev/ddev).
* Add your external resource to [awesome-ddev](https://github.com/ddev/awesome-ddev).
* Help others in [Discord](https://ddev.com/s/discord) and on [Stack Overflow](https://stackoverflow.com/tags/ddev).
* Contribute financially via [GitHub Sponsors](https://github.com/sponsors/rfay).
* Get involved with DDEV governance and the [Advisory Group](https://github.com/ddev/ddev/discussions/categories/ddev-advisory-group).

### How do financial contributions support DDEV?

Thanks for asking! Contributions made via [GitHub Sponsors](https://github.com/sponsors/ddev) go to the [DDEV Foundation](https://ddev.com/foundation) and are used for infrastructure and supporting development.
