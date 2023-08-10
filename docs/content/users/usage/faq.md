# FAQ

Frequently-asked questions organized into high-level functionality, investigating issues, daily usage, and connecting with our community.

## Features & Requirements

### What operating systems will DDEV work with?

DDEV works nearly anywhere Docker will run, including macOS, Windows 10/11 Pro/Enterprise and Home, and every Linux variant we’ve ever tried. It also runs in many Linux-like environments, like ChromeOS (in Linux machine) and Windows 10/11’s WSL2. DDEV works the same on each of these platforms since the important work is done inside identical Docker containers.

### Why do you recommend Colima over Docker Desktop on macOS?

[Colima](https://github.com/abiosoft/colima) (with its bundled [Lima](https://github.com/lima-vm/lima)) is similar to what Docker Desktop provides, with a great DDEV experience on Intel and Apple Silicon machines. We specifically recommend Colima because of some important differences:

* It’s open source software with an MIT license, unlike Docker Desktop which is proprietary software. No license fee to Docker, Inc. and no paid Docker plan required for larger organizations.
* It’s CLI-focused, unlike Docker Desktop’s GUI.
* It’s focused directly on running containers.
* It’s fast and stable.

### How can I migrate from one Docker provider to another?

There are many Docker providers on DDEV’s supported platforms. For example, on macOS people use Docker Desktop and Colima (both officially supported) and they also use [OrbStack](https://orbstack.dev/) and [Rancher Desktop](https://rancherdesktop.io/), which don't yet have official DDEV support with automated tests. On Windows WSL2, people may use Docker Desktop or Docker CE inside WSL2. In all cases, if you want to switch between Docker providers, save your database and make sure the Docker providers don't interfere with each other:

1. Save away your projects' databases. You can run `ddev snapshot --all` to make snapshots of all *registered* projects (that show up in `ddev list`). If you prefer a different way of saving database dumps, that works too!
2. Stop the Docker provider you're moving from. For example, exit Docker Desktop.
3. Start the Docker provider you're moving to.
4. Start projects and restore their databases. For example, you could run `ddev snapshot restore --latest` to load a snapshot taken in step one.

### Can I run DDEV on an older Mac?

Probably! You’ll need to install an older, unsupported version of Docker Desktop—but you can likely use it to run the latest DDEV version.

Check out [this Stack Overflow answer](https://stackoverflow.com/a/69964995/897279) for a walk through the process.

### Do I need to install PHP, Composer, nginx, or Node.js/npm on my workstation?

No. These tools live inside DDEV’s Docker containers, so you only need to [install Docker](../install/docker-installation.md) and [install DDEV](../install/ddev-installation.md). This is especially handy for Windows users where there’s more friction getting these things installed.

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

Yes, you can create additional databases and manually do whatever you need on them. They’re created automatically if you use `ddev import-db` with the `--target-db` option. In this example, `extradb.sql.gz` is extracted and imported to a newly-created database named `extradb`:

```
ddev import-db --target-db=extradb --file=.tarballs/extradb.sql.gz
```

You can use [`ddev mysql`](../usage/commands.md#mysql) or `ddev psql` to execute queries, or use the MySQL/PostgreSQL clients within `ddev ssh` or `ddev ssh -s db`. See the [Database Management](database-management.md) page.

### Can different projects communicate with each other?

Yes, this is commonly required for situations like Drupal migrations. For the `web` container to access the `db` container of another project, use `ddev-<projectname>-db` as the hostname of the other project.

Let’s say we have two projects, for example: project A, and project B. In project A, use `mysql -h ddev-projectb-db` to access the database server of project B. For HTTP/S communication (i.e. API calls) you can 1) access the web container of project B directly with the hostname `ddev-<projectb>-web` and port 80 or 443: `curl https://ddev-projectb-web` or 2) Add a `.ddev/docker-compose.communicate.yaml` to project A to access project B via the official FQDN.

```yaml
services:
  web:
    external_links:
      - "ddev-router:projectb.ddev.site"
```

This lets the `ddev-router` know that project A can access the web container on project B's DDEV URL. If you are using other hostnames or `project_tld`, you will need to adjust the `projectb.ddev.site` value.

### Can I run DDEV with other development environments at the same time?

Yes, as long as they’re configured with different ports. It doesn’t matter whether your other environments use Docker or not, it should only be a matter of avoiding port conflicts.

It’s probably easiest, however, to shut down one before using the other.

For example, if you use Lando for one project, do a `lando poweroff` before using DDEV, and then run [`ddev poweroff`](../usage/commands.md#poweroff) before using Lando again. If you run nginx or Apache locally, stop them before using DDEV. The [troubleshooting](troubleshooting.md) section goes into more detail about identifying and resolving port conflicts.

## Performance & Troubleshooting

### How can I get the best performance?

Docker’s normal mounting can be slow, especially on macOS. See the [Performance](../install/performance.md) section for speed-up options including Mutagen and NFS mounting.

### How can I troubleshoot what’s going wrong?

See the [troubleshooting](troubleshooting.md), [Docker troubleshooting](../install/docker-installation.md#testing-and-troubleshooting##-your-docker-installation) and [Xdebug troubleshooting](../debugging-profiling/step-debugging.md#troubleshooting-xdebug) sections.

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

## Workflow

### How can I update/upgrade DDEV?

You’ll want to update DDEV using the same method you chose to install it. Since upgrading is basically the same as installing, you can follow [DDEV Installation](../install/ddev-installation.md) to upgrade.

You can use the [`self-upgrade`](../usage/commands.md#self-upgrade) command for getting instructions tailored to your installation.

* On macOS you likely installed via Homebrew; run `brew update && brew upgrade ddev`.
<!-- markdownlint-disable-next-line -->
* On Linux + WSL2 using Debian/Ubuntu’s `apt install` technique, run `sudo apt update && sudo apt upgrade ddev` like any other package on your system.
<!-- markdownlint-disable-next-line -->
* On Linux + WSL2 with a Homebrew install, run `brew update && brew upgrade ddev`.
* On macOS or Linux (including WSL2) if you installed using the [install_ddev.sh script](https://github.com/ddev/ddev/blob/master/scripts/install_ddev.sh), run it again:
    <!-- markdownlint-disable -->
    ```
    curl -fsSL https://ddev.com/install.sh | bash
    ```
    <!-- markdownlint-restore -->
* On traditional Windows, you likely installed with Chocolatey or by downloading the installer package. You can upgrade with `choco upgrade ddev` or by visiting the [releases](https://github.com/ddev/ddev/releases) page and downloading the installer. Both techniques will work.
* On Arch-Linux based systems, use the standard upgrade techniques, e.g. `yay -Syu`.

### How can I install a specific version of DDEV?

If you’re using Homebrew, first run `brew unlink ddev` to get rid of the version you have there. Then use one of these options:

1. Download the version you want from the [releases page](https://github.com/ddev/ddev/releases) and place it in your `$PATH`.
2. Use the [install_ddev.sh](https://raw.githubusercontent.com/ddev/ddev/master/scripts/install_ddev.sh) script with the version number argument. For example, if you want v1.21.5, run `curl -fsSL https://ddev.com/install.sh | bash -s v1.21.5`.
3. On Debian/Ubuntu/WSL2 with DDEV installed via apt, you can run `sudo apt update && sudo apt install ddev=<version>`, for example `sudo apt install ddev=1.21.5`.
4. If you want the very latest, unreleased version of DDEV, run `brew unlink ddev && brew install ddev/ddev/ddev --HEAD`.

### How can I back up or restore all project databases?

You can back up all projects that show in `ddev list` with `ddev snapshot -a`. This only snapshots projects displayed in `ddev list`; any projects not shown there will need to be started so they’re be registered in `ddev list`.

### How can I share my local project with someone?

See [Sharing Your Project](../topics/sharing.md).

### How do I make DDEV match my production environment?

You can change the major PHP version and choose between nginx+fpm (default) and Apache+fpm and choose the MariaDB/MySQL/PostgreSQL version add [extra services like Solr and Memcached](../extend/additional-services.md). You won’t be able to make every detail match your production server, but with database server type and version, PHP version and web server type you’ll be close.

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
4. Start thew new project with `ddev start`.
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

DDEV doesn’t have control over your computer’s name resolution, so it doesn’t have any way to influence how your browser gets an IP address from a hostname. It knows you have to be connected to the internet to do that, and uses a test DNS lookup of `<somethingrandom>.ddev.site` as a way to guess whether you’re connected to the internet. If it’s unable to do a name lookup, or if the hostname associated with your project is not `*.ddev.site`, it will try to create entries in `/etc/hosts`, since it’s assuming you can’t look up your project’s hostname(s) via DNS. If your internet (and name resolution) is actually working, but DNS is slow, run `ddev config global --internet-detection-timeout-ms=3000` to set the timeout to 3 seconds (or higher). See [this GitHub issue](https://github.com/ddev/ddev/issues/2409#issuecomment-662448025) for more. (If DNS rebinding is disallowed on your network/router, this won’t be solvable without network/router changes. Help [here](https://github.com/ddev/ddev/issues/2409#issuecomment-675083658) and [here](https://github.com/ddev/ddev/issues/2409#issuecomment-686718237).) For more detailed troubleshooting information, please see the [troubleshooting section](troubleshooting.md#ddev-starts-fine-but-my-browser-cant-access-the-url-url-server-ip-address-could-not-be-found-or-we-cant-connect-to-the-server-at-url).

### How can I configure a project with the defaults without hitting <kbd>RETURN</kbd> a bunch of times?

Use `ddev config --auto` to set the docroot and project type based on the discovered code.
If anything in `.ddev/config.yaml` is wrong, you can edit that directly or use [`ddev config`](../usage/commands.md#config) commands to update settings.

## Getting Involved

### How do I get support?

See the [support options](../support.md), including [Discord](https://discord.gg/kDvSFBSZfs), [Stack Overflow](https://stackoverflow.com/questions/tagged/ddev) and the [issue queue](https://github.com/ddev/ddev/issues).

### How can I contribute to DDEV?

We love and welcome contributions of knowledge, support, docs, and code:

* Submit an issue or pull request to the [main repository](https://github.com/ddev/ddev).
* Add your external resource to [awesome-ddev](https://github.com/ddev/awesome-ddev).
* Add your recipe or HOWTO to [ddev-contrib](https://github.com/ddev/ddev-contrib).
* Help others in [Discord](https://discord.gg/kDvSFBSZfs) and on [Stack Overflow](https://stackoverflow.com/tags/ddev).
* Contribute financially via [GitHub Sponsors](https://github.com/sponsors/rfay).
* Get involved with DDEV governance and the [Advisory Group](https://github.com/ddev/ddev/discussions/categories/ddev-advisory-group).

### How do financial contributions support DDEV?

Thanks for asking! Contributions made via [GitHub Sponsors](https://github.com/sponsors/rfay) go to the [Localdev Foundation](https://localdev.foundation) and are used for infrastructure and supporting development.
