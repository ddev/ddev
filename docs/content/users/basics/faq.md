# Frequently-Asked Questions (FAQ)

What operating systems will DDEV work with?
: DDEV works nearly anywhere Docker will run, including macOS, Windows 10/11 Pro/Enterprise, Windows 10/11 Home, and every Linux variant we’ve ever tried. It also runs in many Linux-like environments, like ChromeOS (in Linux machine) and Windows 10/11’s WSL2. DDEV works the same on each of these platforms since the important work is done inside identical Docker containers.

Do I lose my data when I do a `ddev poweroff` or `ddev stop` or `ddev restart`?
: No, you don’t lose data in your database or code with any of these commands. Your database is safely stored on a Docker volume.

How does my project connect to the database?
: `ddev describe` outputs full details for connecting to the database. *Inside* the container the hostname is `db` (**NOT** `127.0.0.1`). User/password/database are all `db`. For connection from your *host machine*, see `ddev describe`.

How can I troubleshoot what’s going wrong?
: See the [troubleshooting](troubleshooting.md), [Docker troubleshooting](../install/docker-installation.md#testing-and-troubleshooting-your-docker-installation) and [Xdebug troubleshooting](../debugging-profiling/step-debugging.md#troubleshooting-xdebug) sections of the docs.

Do I need to install PHP, Composer, nginx, or Node.js/npm on my computer?
: Absolutely *not*. All of these tools live inside DDEV’s Docker containers, so you need only Docker and DDEV. This is especially handy for Windows users where there’s more friction installing those tools.

How do I get support?
: See the (many) [support options](../support.md), including [Discord](https://discord.gg/kDvSFBSZfs), [Stack Overflow](https://stackoverflow.com/questions/tagged/ddev) and the [issue queue](https://github.com/drud/ddev/issues).

How can I get the best performance?
: Docker’s normal mounting can be slow, especially on macOS. See the [Performance](../install/performance.md) section for speed-up options including Mutagen and NFS mounting.

How can I check that Docker is working?
: The Docker Installation docs have a [full Docker troubleshooting section](../install/docker-installation.md#troubleshooting), including a single `docker run` command that will verify whether everything is set up.

Can I run DDEV with other Docker or non-Docker development environments at the same time?
: Yes you can, as long as they’re configured with different ports—but it’s easiest to shut down one before using the other.

    For example, if you use Lando for one project, do a `lando poweroff` before using DDEV, and then do a `ddev poweroff` before using Lando again. If you run nginx or Apache locally, stop them before using DDEV. More information is in the [troubleshooting](troubleshooting.md) section.

How can I contribute to DDEV?
: We love and welcome contributions of knowledge, support, docs, and code. Make an issue or PR to the [main repo](https://github.com/drud/ddev). Add your external resource to [awesome-ddev](https://github.com/drud/awesome-ddev). Add your recipe or HOWTO to [ddev-contrib](https://github.com/drud/ddev-contrib). Help others in [Discord](https://discord.gg/kDvSFBSZfs) and [Stack Overflow](https://stackoverflow.com/tags/ddev). Contribute financially via [GitHub Sponsors](https://github.com/sponsors/rfay). Get involved with DDEV governance and the [Advisory Group](https://github.com/drud/ddev/discussions/categories/ddev-advisory-group).

How can I show my local project to someone else?
: We often want a customer or coworker to be able to view our local environment, even if they’re on a different machine or network. There are [several ways to do this](../topics/sharing.md). `ddev share` is one: it provides a link that anyone can view and so they can interact with your local project while you allow it. See `ddev share -h` for more information. It does require an account on [ngrok.com](https://ngrok.com).

Can I use additional databases with DDEV?
: Yes, you can create additional databases and manually do whatever you need on them. They are automatically created if you use `ddev import-db` with `--target-db`, for example `ddev import-db --target-db=extradb --src=.tarballs/extradb.sql.gz`. You can use `ddev mysql` or `ddev psql` for random queries, or also use the MySQL/PostgreSQL clients within `ddev ssh` or `ddev ssh -s db`. See the [database topic](database_management.md).

<a name="projects-communicate-with-each-other"></a>
Can different projects communicate with each other?
: Yes, this is commonly required for situations like Drupal migrations. For the `web` container to access the `db` container of another project, use `ddev-<projectname>-db` as the hostname of the other project.

    Let’s say we have two projects, for example: project A, and project B. In project A, use `mysql -h ddev-projectb-db` to access the database server of project B. For HTTP/S communication you can 1) access the web container of project B directly with the hostname `ddev-<projectb>-web` and port 80 or 443: `curl https://ddev-projectb-web` or 2) Add a `.ddev/docker-compose.communicate.yaml` for accessing the other project via the official FQDN.

    ```yaml
      services:
        web:
            external_links:
              - "ddev-router:projectb.ddev.site"
    ```

How do I make DDEV match my production environment?
: You can change the major PHP version and choose between nginx+fpm (default) and Apache+fpm and choose the MariaDB/MySQL/PostgreSQL version add [extra services like Solr and Memcached](../extend/additional-services.md). You won’t be able to make every detail match your production server, but with database server type and version, PHP version and web server type you’ll be close.

How do I completely destroy a project?
: Use `ddev delete <project>` to destroy a project. By default, a `ddev snapshot` of your database is taken, but you can skip this using `ddev delete --omit-snapshot` or `ddev delete --omit-snapshot -y`. See `ddev delete -h` for options. It’s up to you to then delete the code directory.

I don’t like the settings files or gitignores that DDEV creates. What can I do?
: You have a couple of options that work well for most people:

    * Use the `disable_settings_management: true` option in the `.ddev/config.yaml`. This disables DDEV from updating CMS-related settings files.
    * Use the more generic “php” project type rather than a CMS-specific one. “php” just means “don’t try to create settings files and such for me.”. The “php” type works great for experienced developers.

    * Take over the settings file or `.gitignore` by deleting the line `#ddev-generated` in it, then check in the file. If that line is removed, DDEV will not try to replace or change the file.
  
How can I change the name of a project?
: Delete it and migrate it to a new project with your preferred name:  

    1. Export the project’s database: `ddev export-db --file=/path/to/db.sql.gz`.
    2. Delete the project: `ddev delete <project>`. (By default this will make a snapshot for safety.)
    3. Rename the project: `ddev config --project-name=<new_name>`.
    4. Start thew new project with `ddev start`.
    5. Import the database dump from step one: `ddev import-db --src=/path/to/db.sql.gz`.

How can I move a project from one directory to another?
: `ddev stop --unlist`, then move the directory, then `ddev start` in the new directory.
  
How can I move a project from one computer to another?
: Follow this procedure:  

    1. `ddev start && ddev snapshot`.
    2. `ddev stop --unlist`.
    3. Move the project directory to another computer any way you want.
    4. On the new computer, `ddev start && ddev snapshot restore --latest`.
    5. Optionally, on the old computer, `ddev delete --omit-snapshot` to get rid of the database there.

How can I move a project from traditional Windows to WSL2?
:  This is exactly the same as moving a project from one computer to another described above. Make sure you move the project into a native filesystem in WSL2, most likely `/home`.

DDEV wants to add a hostname to `/etc/hosts` but I don’t think it should need to.
: If you see “The hostname <hostname> is not currently resolvable” and you *can* `ping <hostname>`, it may be that DNS resolution is slow. DDEV doesn’t have control over your computer’s name resolution, so it doesn’t have any way to influence how your browser gets an IP address from a hostname. It knows you have to be connected to the Internet to do that, and uses a test DNS lookup of `<somethingrandom>.ddev.site` as a way to guess whether you’re connected to the internet. If it’s unable to do a name lookup, or if the hostname associated with your project is not `*.ddev.site`, it will try to create entries in `/etc/hosts`, since it’s assuming you can’t look up your project’s hostname(s) via DNS. If your internet (and name resolution) is actually working, but DNS is slow, run `ddev config global --internet-detection-timeout-ms=3000` to set the timeout to 3 seconds (or higher). See [this GitHub issue](https://github.com/drud/ddev/issues/2409#issuecomment-662448025) for more. (If DNS rebinding is disallowed on your network/router, this won’t be solvable without network/router changes. Help [here](https://github.com/drud/ddev/issues/2409#issuecomment-675083658) and [here](https://github.com/drud/ddev/issues/2409#issuecomment-686718237).) For more detailed troubleshooting information, please see the [troubleshooting section](troubleshooting.md#ddev-starts-fine-but-my-browser-cant-access-the-url-url-server-ip-address-could-not-be-found-or-we-cant-connect-to-the-server-at-url).

How can I configure a project with the defaults without hitting <kbd>RETURN</kbd> a bunch of times?
: Use `ddev config --auto` to set the docroot and project type based on the discovered code. If anything in `.ddev/config.yaml` is wrong, you can edit that directly or use `ddev config` commands to update settings.

Why do I get a 403 or 404 on my project after `ddev launch`?
: Most likely because the docroot is misconfigured, or there’s no `index.php` or `index.html` file in the docroot. Open your `.ddev/config.yaml` file and check the `docroot` value, which should be a relative path to the directory containing your project’s `index.php`.

Why do I see nginx headers when I’m configured to use `webserver_type: apache-fpm`?
: Apache runs in the web container, but when you use the `https://*.ddev.site` URL, it goes through `ddev-router`, which is an nginx reverse proxy, and that’s why you see the nginx headers even though your web container’s using Apache. Read more in [this Stack Overflow answer](https://stackoverflow.com/a/52780601/215713).

Why does `ddev start` fail with “error while mounting volume, Permission denied”?
: This almost always means NFS is enabled in your project, but NFS isn’t working on your machine. Start by completely turning NFS off for your projects with `ddev config --nfs-mount-enabled=false && ddev config global --nfs-mount-enabled=false`. Then later, [get NFS working](../install/performance.md#using-nfs-to-mount-the-project-into-the-web-container). NFS can be a big performance help on macOS and traditional Windows, and not needed on Linux or Windows WSL2. Most people on macOS and Windows use Mutagen instead of NFS because of its vastly improved performance, so instead of trying to fix this you can disable NFS and enable Mutagen by running `ddev config --nfs-mount-enabled=false --mutagen-enabled`.

How can I update/upgrade DDEV?
: DDEV is easiest to think of as a single binary, and it can be installed many ways, so can be upgraded many ways depending on your operating system and environment. Since upgrading is basically the same as installing, you can follow [DDEV Installation](../install/ddev-installation.md) to upgrade as well.

    * On macOS you likely installed via Homebrew; run `brew update && brew upgrade ddev`.
    <!-- markdownlint-disable-next-line -->
    * On Linux + WSL2 using Debian/Ubuntu’s `apt install` technique, run `sudo apt update && sudo apt upgrade ddev` like any other package on your system.
    <!-- markdownlint-disable-next-line -->
    * On Linux + WSL2 with a Homebrew install, run `brew update && brew upgrade ddev`.
    * On macOS or Linux (including WSL2) if you installed using the [install_ddev.sh script](https://github.com/drud/ddev/blob/master/scripts/install_ddev.sh) you just run it again:
    <!-- markdownlint-disable -->
    ```
    curl -fsSL https://raw.githubusercontent.com/drud/ddev/master/scripts/install_ddev.sh | bash
    ```
    <!-- markdownlint-restore -->
    * On traditional Windows, you likely installed with Chocolatey or by downloading the installer package. You can upgrade with `choco upgrade ddev` or by visiting the [releases](https://github.com/drud/ddev/releases) page and downloading the installer. Both techniques will work.
    * On Arch-Linux based systems, use the standard upgrade techniques, e.g. `yay -Syu`.

How can I install a specific version of DDEV?
: If you’re using Homebrew, first run `brew unlink ddev` to get rid of the version you have there. Then use one of these options:

    1. Download the version you want from the [releases page](https://github.com/drud/ddev/releases) and place it somewhere in your `$PATH`.
    2. Use the [install_ddev.sh](https://raw.githubusercontent.com/drud/ddev/master/scripts/install_ddev.sh) script with the version number argument. For example, if you want v1.18.3-alpha1, use `curl -fsSL https://raw.githubusercontent.com/drud/ddev/master/scripts/install_ddev.sh | bash -s v1.18.3-alpha1`.
    3. On Debian/Ubuntu/WSL2 with DDEV installed via apt, you can `sudo apt update && sudo apt install ddev=<version>`, for example `sudo apt install ddev=1.21.1`.
    4. If you want the very latest, unreleased version of ddev, use `brew unlink ddev && brew install drud/ddev/ddev --HEAD`.

How can I back up or restore all databases of all projects?
: You can back up all projects that show in `ddev list` with `ddev snapshot -a`. This only snapshots projects that are shown in `ddev list` though, so if you have other projects that aren’t shown, you’d need to start them so they’d be registered in `ddev list`.

Why do you recommend Colima over Docker Desktop on macOS?
: [Colima](https://github.com/abiosoft/colima) (with its bundled [Lima](https://github.com/lima-vm/lima)) is similar to what Docker Desktop provides, with a great DDEV experience on Intel and Apple Silicon machines. We specifically recommend Colima because of some important differences:

    * It’s open source software with an MIT license, unlike Docker Desktop which is proprietary software. No license fee to Docker, Inc. and no paid Docker plan required for larger organizations.
    * It’s CLI-focused, unlike Docker Desktop’s GUI.
    * It’s focused directly on running containers.
    * It’s fast and stable.

How can I contribute financially to the DDEV project?
: Thanks for asking! Contributions can be done via [GitHub Sponsors](https://github.com/sponsors/rfay). They go to the [Localdev Foundation](https://localdev.foundation) and are used for infrastructure and supporting development.
