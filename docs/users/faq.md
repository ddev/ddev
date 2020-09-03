## Frequently-Asked Questions (FAQ)

* **What operating systems will DDEV-Local work with?** DDEV-Local works nearly anywhere Docker will run, including macOS, Windows 10 Pro/Enterprise,  Windows 10 Home, and every Linux variant we've ever tried. It also runs in many Linux-like environments, for example ChromeOS (in Linux machine) and Windows 10's WSL2. In general, DDEV works the same on each of these platforms, as all the important work is done inside identical Docker containers.

* **Do I lose my data when I do a `ddev poweroff` or `ddev stop` or `ddev restart`?** No, you don't lose data in your database or code with any of these commands. Your database is safely stored on a docker volume.

* **How does my project connect to the database?** `ddev describe` gives full details of how to connect to the database. *Inside* the container the hostname is 'db' (NOT 127.0.0.1). User/password/database are all 'db'. For connection from the *host*, see `ddev describe`.

* **How can I troubleshoot what's going wrong?** See the [troubleshooting](troubleshooting.md) and [Docker troubleshooting](docker_installation.md#troubleshooting) sections of the docs.

* **Do I need to install PHP or Composer or Nginx on my computer?** Absolutely *not*. All of these tools live inside DDEV's docker containers, so you need only Docker and DDEV. This is especially handy for Windows users where there's a bit more friction installing those tools.

* **How do I get support?** See the (many) [support options](../index.md#support), including Slack, Gitter, Stack Overflow and others.

* **How can I get the best performance?** Docker's normal mounting can be slow, especially on macOS. See the [Performance](performance.md) section for speed-up options including NFS mounting.

* **How can I check that Docker is working?** The Docker Installation docs have a [full Docker troubleshooting section](docker_installation.md#troubleshooting), including a single `docker run` command that will verify whether everything is set up.

* **Can I run DDEV and also other Docker or non-Docker development environments at the same time?** Yes, you can, as long as they're configured with different ports. But it's easiest to shut down one before using the other. For example, if you use Lando for one project, do a `lando poweroff` before using DDEV, and then do a `ddev poweroff` before using Lando again. If you run nginx or apache locally, just stop them before using DDEV. More information is in the [troubleshooting](troubleshooting.md) section.

* **How can I contribute to DDEV-Local?** We love contributions of knowledge, support, docs, and code, and invite you to all of them. Make an issue or PR to the [main repo](https://github.com/drud/ddev). Add your external resource to [awesome-ddev](https://github.com/drud/awesome-ddev). Add your recipe or HOWTO to [ddev-contrib](https://github.com/drud/ddev-contrib). Help others in [Stack Overflow](https://stackoverflow.com/tags/ddev) or [Slack](../index.md#support) or [gitter](https://gitter.im/drud/ddev).

* **How can I show my local project to someone else?** It's often the case that we want a customer or coworker to be able to view our local environment, even if they're on a different machine, network, etc. `ddev share` (requires [ngrok](https://ngrok.com)) provides a link that anyone can view and so they can interact with your local project while you allow it. See `ddev share -h` for more information.

* **Can I use additional databases with DDEV?** Yes, you can create additional databases and manually do whatever you need on them. They are automatically created if you use `ddev import-db` with `--target-db`, for example `ddev import-db --target-db=extradb --src=.tarballs/extradb.sql.gz`. You can use `ddev mysql` for random queries, or also use the mysql client within `ddev ssh` or `ddev ssh -s db` as well.

* **Can different projects communicate with each other?** Yes, this is commonly required for situations like Drupal migrations. For the web container to access the db container of another project, use `ddev-<projectname>-db` as the hostname of the other project. For example, in project1, use `mysql ddev-project2-db` to access the db server of project2. For HTTP/S communication you can 1) access the web container of project2 directly with the hostname `ddev-<project2>-web` and port 80 or 443: `curl https://ddev-project2-web` or 2) Access via the ddev router with the official hostname: `curl https://ddev-router -H Host:d7git.ddev.site`.

* **How do I make DDEV match my production webserver environment?** You can change the PHP major version (currently 5.6 through 7.4) and choose between nginx+fpm (default and apache+fpm and choose the MariaDB version add [extra services like solr and memcached](extend/additional-services.md). You will not be able to make every detail match your production server, but with PHP version and webserver type you'll be close.

* **How do I completely destroy a project?** Use `ddev delete <project>` to destroy a project. (Also, `ddev stop --remove-data` will do the same thing.) By default, a `ddev snapshot` of your database is taken, but you can skip this, see `ddev delete -h` for options.

* **I don't like the settings files or gitignores that DDEV creates. What can I do?**  You have a couple of options that work well for most people:
    * Use project type "php" instead of the type of your CMS. "php" just means "Don't try to create settings files and such for me.". The "php" type works great for experienced developers.
    * "Take over" the settings file or .gitignore by deleting the line "#ddev-generated" in it (and then check in the file). If that line is removed, ddev will not try to replace or change the file.

* **I see "Internet connection not detected" and DDEV-Local wants to add a hostname to /etc/hosts. But my internet is working!** ddev uses the ability to do a DNS lookup of <somethingrandom>.ddev.site as a proxy for whether you're connected to the internet, and it guesses that it ought to be able to complete that in about 750ms or maybe the internet isn't available. If it finds no internet, it will try to create entries in /etc/hosts, since it's assuming you can't use *.ddev.site via DNS. If your internet is actually working, but DNS is slow, just `ddev config global --internet-detection-timeout-ms=3000` to set the timeout to 3 seconds. See[issue link](https://github.com/drud/ddev/issues/2409#issuecomment-662448025) for more details.
