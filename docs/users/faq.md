## Frequently-Asked Questions (FAQ)

What operating systems will DDEV-Local work with?
: DDEV-Local works nearly anywhere Docker will run, including macOS, Windows 10/11 Pro/Enterprise,  Windows 10/11 Home, and every Linux variant we've ever tried. It also runs in many Linux-like environments, for example ChromeOS (in Linux machine) and Windows 10/11's WSL2. In general, DDEV works the same on each of these platforms, as all the important work is done inside identical Docker containers.

Do I lose my data when I do a `ddev poweroff` or `ddev stop` or `ddev restart`?
: No, you don't lose data in your database or code with any of these commands. Your database is safely stored on a docker volume.

How does my project connect to the database?
: `ddev describe` gives full details of how to connect to the database. *Inside* the container the hostname is 'db' (**NOT** 127.0.0.1). User/password/database are all 'db'. For connection from the *host*, see `ddev describe`.

How can I troubleshoot what's going wrong?
: See the [troubleshooting](troubleshooting.md), [Docker troubleshooting](docker_installation.md#troubleshooting) and [Xdebug troubleshooting](step-debugging.md#troubleshooting-xdebug) sections of the docs.

Do I need to install PHP or Composer or Nginx or nodejs/npm on my computer?
: Absolutely *not*. All of these tools live inside DDEV's docker containers, so you need only Docker and DDEV. This is especially handy for Windows users where there's a bit more friction installing those tools.

How do I get support?
: See the (many) [support options](../index.md#support), including [Discord](https://discord.gg/kDvSFBSZfs), Stack Overflow and others.

How can I get the best performance?
: Docker's normal mounting can be slow, especially on macOS. See the [Performance](performance.md) section for speed-up options including Mutagen and NFS mounting.

How can I check that Docker is working?
: The Docker Installation docs have a [full Docker troubleshooting section](docker_installation.md#troubleshooting), including a single `docker run` command that will verify whether everything is set up.

Can I run DDEV and also other Docker or non-Docker development environments at the same time?
: Yes, you can, as long as they're configured with different ports. But it's easiest to shut down one before using the other. For example, if you use Lando for one project, do a `lando poweroff` before using DDEV, and then do a `ddev poweroff` before using Lando again. If you run nginx or apache locally, just stop them before using DDEV. More information is in the [troubleshooting](troubleshooting.md) section.

How can I contribute to DDEV-Local?
: We love contributions of knowledge, support, docs, and code, and invite you to all of them. Make an issue or PR to the [main repo](https://github.com/drud/ddev). Add your external resource to [awesome-ddev](https://github.com/drud/awesome-ddev). Add your recipe or HOWTO to [ddev-contrib](https://github.com/drud/ddev-contrib). Help others in [Discord](https://discord.gg/kDvSFBSZfs) and [Stack Overflow](https://stackoverflow.com/tags/ddev). Follow the [governance issue](https://github.com/drud/ddev/issues/3268) to learn about financial support possibilities.

How can I show my local project to someone else?
: It's often the case that we want a customer or coworker to be able to view our local environment, even if they're on a different machine, network, etc. There are [several ways to do this](topics/sharing.md). `ddev share` is one of the ways - it provides a link that anyone can view and so they can interact with your local project while you allow it. See `ddev share -h` for more information. It does require an account on [ngrok.com](https://ngrok.com).

Can I use additional databases with DDEV?
: Yes, you can create additional databases and manually do whatever you need on them. They are automatically created if you use `ddev import-db` with `--target-db`, for example `ddev import-db --target-db=extradb --src=.tarballs/extradb.sql.gz`. You can use `ddev mysql` or `ddev psql` for random queries, or also use the mysql/psql clients within `ddev ssh` or `ddev ssh -s db` as well.

<a name="projects-communicate-with-each-other"></a>
Can different projects communicate with each other?
: Yes, this is commonly required for situations like Drupal migrations. For the web container to access the db container of another project, use `ddev-<projectname>-db` as the hostname of the other project. For example, in project1, use `mysql -h ddev-project2-db` to access the db server of project2. For HTTP/S communication you can 1) access the web container of project2 directly with the hostname `ddev-<project2>-web` and port 80 or 443: `curl https://ddev-project2-web` or 2) Add a .ddev/docker-compose.communicate.yaml which will allow you to access the other project via the official FQDN.

    ```yaml
      version: '3.6'
      services:
        web:
            external_links:
              - "ddev-router:project2.ddev.site"
    ```

How do I make DDEV match my production webserver environment?
: You can change the PHP major version and choose between nginx+fpm (default) and apache+fpm and choose the MariaDB/MySQL/Postgresql version add [extra services like solr and memcached](extend/additional-services.md). You will not be able to make every detail match your production server, but with database server type and version, PHP version and webserver type you'll be close.

How do I completely destroy a project?
: Use `ddev delete <project>` to destroy a project. By default, a `ddev snapshot` of your database is taken, but you can skip this using `ddev delete --omit-snapshot` or `ddev delete --omit-snapshot -y`, see `ddev delete -h` for options. It's up to you to then delete the code directory.

I don't like the settings files or gitignores that DDEV creates. What can I do?
: You have a couple of options that work well for most people:

    * Use project type "php" instead of the type of your CMS. "php" just means "Don't try to create settings files and such for me.". The "php" type works great for experienced developers.

    * "Take over" the settings file or .gitignore by deleting the line "#ddev-generated" in it (and then check in the file). If that line is removed, ddev will not try to replace or change the file.
  
How can I change the name of a project?
: Use this process:
     1. Export the database of the project: `ddev export-db --file=/path/to/db.sql.gz`
     2. `ddev delete <project>`. By default this will make a snapshot, which is a nice safety valve.
     3. Rename the project, `ddev config --project-name=<new_name>`
     4. `ddev start`
     5. `ddev import-db --src=/path/to/db.sql.gz`

How can I move a project from one directory to another?
: `ddev stop --unlist`, then move the directory, then `ddev start` in the new directory.
  
How can I move a project from one computer to another?
: Follow this procedure:
     1. `ddev start && ddev snapshot`
     2. `ddev stop --unlist`
     3. Move the project directory to another computer any way you want.
     4. On the new computer, `ddev start && ddev snapshot restore --latest`
     5. Optionally, on the old computer, `ddev delete --omit-snapshot` to get rid of the database there.

How can I move a project from traditional Windows to WSL2?
:  This is exactly the same as moving a project from one computer to another, see above. Make sure you move the project into a native filesystem in WSL2, most likely /home.
  
DDEV-Local wants to add a hostname to /etc/hosts but I don't think it should need to.
: If you see "The hostname <hostname> is not currently resolvable" and you *can* `ping <hostname>`, it may be that DNS resolution is slow. DDEV doesn't have any control of your computer's name resolution, so doesn't have any way to influence how your browser gets an IP address from a hostname. It knows you have to be connected to the Internet to do that, and uses a test DNS lookup of <somethingrandom>.ddev.site as a way to guess whether you're connected to the internet. If it is unable to do a name lookup, or if the hostname associated with your project is *not* \*.ddev.site, it will try to create entries in /etc/hosts, since it's assuming you can't look up your project's hostname(s) via DNS. If your internet (and name resolution) is actually working, but DNS is slow, just `ddev config global --internet-detection-timeout-ms=3000` to set the timeout to 3 seconds (or higher). See[issue link](https://github.com/drud/ddev/issues/2409#issuecomment-662448025) for more details. (If DNS rebinding is disallowed on your network/router, this won't be solvable without network/router changes. Help [here](https://github.com/drud/ddev/issues/2409#issuecomment-675083658) and [here](https://github.com/drud/ddev/issues/2409#issuecomment-686718237).) For more detailed troubleshooting information please see the [troubleshooting section](troubleshooting.md#ddev-starts-fine-but-my-browser-cant-access-the-url-url-server-ip-address-could-not-be-found-or-we-cant-connect-to-the-server-at-url).

How can I configure a project with the defaults without hitting <RETURN> a bunch of times?
: Just use `ddev config --auto` and it will choose docroot and project type based on the discovered code. If it gets anything wrong (just look at `.ddev/config.yaml`) you can change that at any time using various `ddev config` commands or just by editing the `.ddev/config.yaml`.

Why do I get a 403 or 404 on my project after `ddev launch`?
: The most likely reason for this is that the docroot is misconfigured, or there's no index.php or index.html file in the docroot. Take a look at your .ddev/config.yaml and see what is there for the docroot. It should be a relative path to where your index.php is.

Why do I see nginx headers when I'm configured to use `webserver_type: apache-fpm`?
: Apache runs in the web container but when you use the `http://*.ddev.site` URL, it goes through ddev-router, which is an nginx reverse proxy, and that's why you see the nginx headers. But rest assured you are using Apache. More detail in [Stack Overflow answer](https://stackoverflow.com/a/52780601/215713)

Why does `ddev start` fail with "error while mounting volume, Permission denied"?
: This almost always means that you have NFS enabled in your project, but NFS isn't working on your machine. Start by completely turning NFS off for your projects with `ddev config --nfs-mount-enabled=false && ddev config global --nfs-mount-enabled=false`. Then later, [go get NFS working](performance.md#using-nfs-to-mount-the-project-into-the-web-container). NFS can be a big performance help on macOS and traditional Windows, and not needed on Linux or Windows WSL2.

How can I install a specific version of DDEV?
: If you want to use a different version of DDEV, you easily get a different version. If you're using homebrew, `brew unlink ddev` first, to get rid of the version you have there. Then use one of these options:

    1. Download the version you want from the [releases page](https://github.com/drud/ddev/releases) and place it somewhere in your `$PATH`.
    2. Use the [install_ddev.sh](https://raw.githubusercontent.com/drud/ddev/master/scripts/install_ddev.sh) script with the version number argument. For example, if you want v1.18.3-alpha1, use `curl -LO https://raw.githubusercontent.com/drud/ddev/master/scripts/install_ddev.sh && bash install_ddev.sh v1.18.3-alpha1`
    3. If you want the very latest, unreleased version of ddev, use `brew unlink ddev && brew install drud/ddev/ddev --HEAD`.

How can I back up or restore all databases of all projects?
: You can back up all projects that show in `ddev list` with `ddev snapshot -a`. This only snapshots projects that are shown in `ddev list` though, so if you have other projects that aren't shown, you'd need to start them so they'd be registered in `ddev list`.

How can I contribute financially to the DDEV project?
: Thanks for asking! Contributions can be done via [GitHub Sponsors](https://github.com/sponsors/rfay). They go to the [Localdev Foundation](https://localdev.foundation) and are used for infrastructure and supporting development.
