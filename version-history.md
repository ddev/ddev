---
search:
  exclude: true
---
# DDEV Version History

This version history has been driven by what we hear from our wonderful community of users. Please share your comments or ideas in the [issue queue](https://github.com/ddev/ddev/issues). We listen. Or talk to us in any of the [support locations](https://ddev.readthedocs.io/en/stable/#support).

## v1.23 (Released 2024-04)

- [x] `ddev-webserver` upgraded to Debian 12 Bookworm
- [x] `nodejs_version` now defaults to 20
- [x] `php_version` defaults to 8.2
- [x] PHP 8.3
- [x] Corepack and Yarn v4 support
- [x] Drupal 11 support
- [x] New "drupal", "cakephp" and "silverstripe" project types.
- [x] Support for even more Docker providers, especially on macOS
- [x] Upsun by Platform.sh and Amazee Lagoon hosting provider support
- [x] `ddev_version_constraint` constraint for `.ddev/config.yaml`
- [x] `ddev dbeaver` support of DBeaver database browser
- [x] PostgreSQL 16 and MariaDB 10.11 support
- [x] `nodejs_version` can be fully-qualified version, like `18.2.4`
- [x] Mailhog has been replaced by Mailpit as developer

## v1.22 (Released 2023-07)

- [x] `ddev get` improvements, including `ddev get --version <add-on>`, `ddev get --installed` to show existing add-ons, `ddev get --remove` to remove. Removal action support added for add-ons.
- [x] MySQL upgraded to 8.0.33 and 5.7.42
- [x] `nodejs_version` now defaults to 18
- [x] Default PHP version is now 8.1
- [x] Global configuration for router_http_port and router_https_port
- [x] Python and Django4 project types and support
- [x] PhpMyAdmin (dba) has been removed from DDEV core, use `ddev get ddev/ddev-phpmyadmin` instead, or another database browser.
- [x] multiple `upload_dirs` now supported
- [x] Tip of the day implemented
- [x] Telemetry/Instrumentation direct to Amplitude added

## v1.21 (Released 2022-08 to 2023-03)

- [x] PHP 8.2.0 with all extensions
- [x] Full Drupal 10 support, with needed database extensions on PostgreSQL and MariaDB/MySQL
- [x] TYPO3 v12 support
- [x] FIG spec generation
- [x] Craft CMS explict project type
- [x] Improved WSL2 support, with install scripts
- [x] Optional new Traefik replacement for ddev-router
- [x] Support for `.ddev/.env` files to set environment variables
- [x] Improved `ddev pull` and `ddev pull platform`
- [x] `ddev self-upgrade`, `ddev debug migrate-database`, `ddev debug get-volume-db-version`, `ddev debug check-db-match`, `ddev querious` commands
- [x] Repositories in "drud" org moved to github.com/ddev, including github.com/ddev/ddev.
- [x] Container Start Hooks are now available using .ddev/web-entrypoint.d/*.sh
- [x] Support for GitHub Codespaces
- [x] MySQL 8.0 upgraded to 8.0.31

## v1.20 (Released 2022-08)

- [x] Improved Node.js support**. You can now run a Node.js daemon with simple `config.yaml` support (`web_extra_daemons`) and if it listens on a port you can expose it via `ddev-router` with `web_extra_exposed_ports`. (Both of these features can be used for daemons other than Node.js daemons.)
- [x] Default PHP version for new projects is now 8.0; the default database version is now MariaDB 10.4.
- [x] `config.*.yaml` are now *merged* instead of overwriting other values. For example, if `config.yaml` has hooks, and `config.something.yaml` has hooks, the values will be merged, getting all of the hooks. Scalar values (like `php_version`, for example) still are overwritten by the last value loaded.
- [x] `ddev craft` command is now included for projects of type `php`.
- [x] `ddev npm` and `ddev yarn` use the relative path in the project for execution inside the container.
- [x] [Platform.sh](https://platform.sh) provider integration has been updated so it doesn’t require any editing of files, only setting a couple of environment variables. This is the future for all of the provider integrations; there’s no reason why files should have to be modified.

## v1.19.0-v1.19.5 (Released 2022-03 through 2022-07)

- [x] `ddev get` and `ddev get --list` allow quick installation of maintained, tested add-ons.
- [x] PostgreSQL support alongside MariaDB and MySQL.
- [x] Run on any platform without Docker Desktop. Colima support for macOS and Docker-inside-WSL2 for Windows.
- [x] Database snapshots are now gzipped, resulting in perhaps 20x size difference. A snapshot that used to use 207MB on disk is now 5MB.
- [x] New `ddev service enable`, `ddev service disable`, `ddev php`, `ddev debug test`, `ddev debug dockercheck` commands.
- [x] Support for remote Docker instances.
- [x] `import-db`, `export-db`, and `import-files` now accept xz and bzip2 compression, thanks to [weareb13](https://twitter.com/weareb13) for sponsoring this feature.
- [x] `ddev clean` will clean up project usage like snapshots and old images thanks to @mikebarkas
- [x] mariadb:10.7 and 10.8.
- [x]  `ddev debug refresh` command to clear Docker’s cache for composer-not-updating or related build issues.
- [x] Node.js 18 is now supported in `nodejs_version`.
- [x] `ddev snapshot --all` can now snapshot running and non-running projects.
- [x] `ddev config --web-environment-add` and `ddev config global --web-environment-add` now allow *adding* to existing environment variables, instead of replacing them.
- [x] Interactive project selection on `ddev start -s`  and `ddev stop -s` - you can choose which project you want to start or stop from the listed projects.
- [x] Multiple Dockerfiles: The `.ddev/web-build` and `.ddev/db-build` directories can now contain multiple Dockerfiles named Dockerfile, `Dockerfile.<anything>`.
- [x] pre-Dockerfiles: v1.19.3 added `.ddev/web-build/Dockerfile.*` but there are some things people have to do in a build before anything else, like adding proxy information or installing a CA.
- [x] Container Wait Timeout is improved so a long `ddev snapshot restore` doesn’t tell you it timed out.
- [x] `ddev npm` command

## v1.18 (Released 2021-09-28) and v1.18.2 (2021-12-08)

- [x] Gitpod support
- [x] Integrated docker-compose so Docker updates don’t break things.
- [x] Mutagen support results in a huge speedup for macOS and traditional Windows users.
- [x] Support docker-compose v1 and v2
- [x] Support MariaDB 10.6
- [x] Support PHP 8.1
- [x] Improved integration with PhpStorm on all platforms, including WSL.
- [x] xhprof support for performance profiling alongside Blackfire support
- [x] Base image for the `ddev-webserver` is now Debian 11 Bullseye.
- [x] mysql5.7 and mysql8.0 support on ARM64 (Mac M1) machines.

## v1.17 (Released 2021-04-07)

- [x] Composer v2 is now the default Composer version.
- [x] Brand new provider integration system, with user-configurable and extensible techniques, Acquia, Platform.sh, DDEV-Live, Pantheon integration.
- [x] Excellent improvements to `ddev snapshot`, including `ddev snapshot restore --latest`, prompted `ddev snapshot restore`, `ddev snapshot --list`, `ddev snapshot --cleanup`, `ddev snapshot --all`
- [x] `ddev snapshot` restore now shows progress as it goes.
- [x] Built-in support for both xhprof and Blackfire profiling.
- [x] New `ddev config --auto` option that configures a project with detected defaults.
- [x] Web container environment variables can be set in `config.yaml` or `global_config.yaml` with the `web_environment key`.
- [x] `ddev heidisql` command provides a nice database browser on Windows and Windows WSL2.
- [x] The PHP default for new projects is now PHP 7.4.
- [x] The MariaDB default for new projects is 10.3.
- [x] New docs theme.

## v1.16 (Released 2020-11-12)

- [x] Support Shopware 6.
- [x] Remove support for Docker Toolbox on Win10 Home (in favor of new Docker Desktop).
- [x] Remove `apache-cgi` `webserver_type`.
- [x] Per-project-type commands like `ddev drush`, `ddev typo3`
- [x] Build hardened DDEV with hardened images for open-source production hosting.

## v1.15 (Released 2020-07-08)

- [x] Laravel support
- [x] Global custom commands
- [x] Global homeadditions
- [x] WSL2 support
- [x] Reworked Nginx/Apache configurations
- [x] Zsh completions
- [x] MariaDB 10.5 support
- [x] Remove obsolete support for `drud-aws`.

## v1.14 (Released 2020-04-21)

- [x] Drupal 9 support
- [x] Global NFS configuration
- [x] `ddev xdebug` command
- [x] Improve `ddev describe` to show information about additional services, <https://github.com/ddev/ddev/issues/788>
- [x] Competitive analysis with similar products both within the spaces we usually work and outside them.
- [x] GUI evaluations

## v1.13 (Released 2020-02-04)

- [x] Updated support of Pantheon via Terminus instead of undocumented API
- [x] Support for Magento and Magento 2, #1802
- [x] Remove deprecated support for webcache
- [x] Review/experimentation with GUI options [#2110](https://github.com/ddev/ddev/issues/2110)

## v1.12 (Released 2019-12-04)

- [x] Support for multiple versions of Oracle MySQL as well as MariaDB
- [x] Improved WordPress support (several open WordPress bugs)
- [x] Custom command improvements

## v1.11 (Released 2019-09-19)

- [x] PHP 7.4 support
- [x] [Allow omitting the db container](https://github.com/ddev/ddev/issues/1490)

## v1.10 (released 2019-08-02)

- [x] Improved instrumentation with [segment](https://segment.com/): @unn is advocating for segment as better than Sentry (or in addition to Sentry). Statistics: Monthly active users, Conversion ratio. [#1640](https://github.com/ddev/ddev/issues/1640)
- [x] [Add custom DDEV commands](https://github.com/ddev/ddev/issues/1372) - See [docksal's approach](https://docs.docksal.io/fin/custom-commands/)
- [x] [Allow user additions to `.bashrc`, store Bash history, copy gitconfig](https://github.com/ddev/ddev/issues/926): These are intended to make it more comfortable for the user to work inside the web container.
- [x] [Add `ddev mysql` command](https://github.com/ddev/ddev/issues/1551)
- [x] [Add `delete`, `poweroff`, `cleanup` commands and hints](https://github.com/ddev/ddev/issues/1588)
- [x] Sign macOS binary #1626
- [x] Make sure exposed ports are not exposed on local subnet, #1662
- [x] [Rework containers to provide a "real" user inside container](https://github.com/ddev/ddev/issues/1403)

## v1.9 (released 2019-06-26)

- [x] [Contrib-pointers for additional Services and techniques](https://github.com/ddev/ddev/issues/1474): We want to make another place for the outstanding content and pointers and applications that our users are developing. This will probably be a contrib repository for ddev.
- [x] [NFS Setup Security Review](https://github.com/ddev/ddev/issues/1474): More docs and improved NFS setup scripts so people can think clearly and plan carefully for how they’re using NFS with DDEV.
- [x] [Use DNS to provide name resolution when internet available](https://github.com/ddev/ddev/issues/416)
- [x] [Manage ddev project list in `~/.ddev/global_config.yaml`](https://github.com/ddev/ddev/issues/642): Since the beginning of DDEV, `ddev list` and everything that depended on it couldn’t work if the project was shut down. This should fix that.
- [x] [Allow specifying a target container for hook execution](https://github.com/ddev/ddev/issues/1038)
- [x] [Support ngrok to allow web access remotely](https://github.com/ddev/ddev/issues/375)
- [x] [Hook system overhaul](https://github.com/ddev/ddev/issues/1372)

## v1.8 (released 2019-05-14)

- **Browsers and host OSs now trust DDEV sites over HTTPS**. You do have to take the one-time action to install the CA keys into your operating system and browsers: `mkcert -install`. If you use one of the package installation methods or the Windows installer, you should already have `mkcert`. Otherwise, see the [mkcert](https://github.com/FiloSottile/mkcert) page for installation options. (Even curl and operating system tools generally trust the `mkcert` certs, and curl within the container also trusts them.)
- **Dynamic container updates**: If you need extra Debian packages in your `web` or `db` container (or need to make more sophisticated adjustments) you no longer have to wait for them, or install them every time you start a project. You can add [`webimage_extra_packages`](https://ddev.readthedocs.io/en/latest/users/extend/customizing-images/#adding-extra-debian-packages-with-webimage_extra_packages-and-dbimage_extra_packages) to your `config.yaml` or [build a free-form Dockerfile](https://ddev.readthedocs.io/en/latest/users/extend/customizing-images/#adding-extra-dockerfiles-for-webimage-and-dbimage) (`Dockerfile.example` provided in your `.ddev` folder)
- **nginx configuration in the web container has been reorganized** and simplified. This mostly won’t matter to you unless you use custom nginx configuration, in which case you’ll want to upgrade your configuration (see below). [docs](https://ddev.readthedocs.io/en/latest/users/extend/customization-extendibility/#providing-custom-nginx-configuration)
- **`ddev exec` and exec hooks now interpret commands using Bash**. This means you can have a hook like `sudo apt-get update && sudo apt-get install -y some-package` without putting `bash -c` in front of it. And you can `ddev exec "sudo apt-get update && sudo apt-get upgrade -y some-package"` as well, no `bash -c` required.
- **`ddev exec` can now work with interactive situations**. So for example, you can `ddev exec mysql` and interact with the MySQL program directly.  Or `ddev exec bash`, which is the same as `ddev ssh`.

## v1.7 (released 2019-03-28)

- `config.*.yaml` overrides: local configuration files can be used to override a team-standard checked-in `.ddev/config.yaml`. See [docs](https://ddev.readthedocs.io/en/latest/users/extend/customization-extendibility/#extending-configyaml-with-custom-configyaml-files).  #1504 and #1410.
- Optional static bind ports for db and web server containers. Those who want the db server or web server bound port to be static within a project can use `host_db_port` or `host_webserver_port` to specify it: #1502, #1491, #941

- Previous versions of Docker for Mac supported operating systems back to El Capitan, but Docker-for-Mac has dropped support for anything before macOS Sierra 10.12, so DDEV also has to drop support for everything before Sierra.
- The default PHP version changes to 7.2. This affects new Drupal 8 and Drupal 7 projects, as well as projects of type `php`.
- Fix regression in v1.5.0 where `mariadb_version` = 10.1 did not properly set the container version to the 10.1 container version.
- The `ddev-dbserver` images now use the official MariaDB Docker image (which uses Debian) as a base, instead of building a custom Alpine image. This *should* be invisible to all users, but we love to hear feedback.

## v1.6 (released 2019-02-11)

- DDEV now supports NFS mounting into the container on all platforms. This provides nearly the speed increase of the experimental webcache feature, but with far greater reliability, and it works on all platforms. In addition, it seems to solve perpetual symlink problems that Windows users had. It does require some configuration on the host side, so [please read the docs](https://ddev.readthedocs.io/en/latest/users/install/performance/#using-nfs-to-mount-the-project-into-the-web-container)
- Chocolatey installs on Windows: [Chocolatey](https://chocolatey.org/) is a leading package manager for Windows, and it makes so many packages so easy to install. On our Windows testbots we use choco to install all the key items that a testbot needs with `choco install -y git mysql-cli golang make docker-desktop nssm GoogleChrome zip jq composer cmder netcat ddev`, note that DDEV is in there now :).
- The `ddev-dbserver` container has been updated so that triggers work, for those of you who use triggers.
- Root/sudo usage of DDEV is prevented (#1407). We found that people kept getting themselves in trouble by trying to use sudo and then they would have files that could not be accessed by an ordinary user.

## v1.5 (released 2018-12-18)

- The newly released PHP 7.3 is now supported, `php_version: 7.3` or `ddev config --php-version=7.3`. As noted above, `php-memcached` is not yet available for 7.3.
- For macOS users, a new *experimental* webcaching strategy makes webserving way faster on large projects like TYPO3 or Drupal 8. `webcache_enabled: true` in the `config.yaml` will start a caching container so that actual webserving happens on a much faster filesystem. This is experimental and has some risks, we want to know your experience. It takes longer to do a `ddev start` because your entire project has to be pushed into the container, but after that hitting a page is way, way more satisfying. Note that `.git` directories are not copied into the webcache, Git won’t work inside the web container. It seemed too risky to combine 2-way file synchronization with your precious Git repository, so do Git operations on the host. Note that if you have a lot of files or big files in your repository, they have to be pushed into the container, and that can take time. I have had to clean up my `.ddev/db_snapshots` directory rather than wait for the `docker cp` to happen forever. A big shout out to Drud team member @cweagans for the original [docker-bg-sync](https://github.com/cweagans/docker-bg-sync) that we forked and used to implement this! Thanks!
- Important Windows symlink support in `ddev composer` (<https://github.com/ddev/ddev/pull/1323>). On the CIFS filesystem used by Docker-for-Windows, real Linux/Mac symlinks are *supported* but cannot be created, so Composer operations inside the container in some cases create simulated symlinks, which are actually files with XSym content; these work fine inside the container… but they’re not real symlinks and sometimes cause some issues. We’ve added a cleanup step after `ddev composer` that converts those XSym files into real symlinks. It only works on Docker for Windows, and it only works if you have “Developer mode” enabled on your Windows 10/11 Pro host. More info is in the [docs](https://ddev.readthedocs.io/en/latest/users/basics/developer-tools/#ddev-and-composer)

## v1.4 (released 2018-11-15)

- The `ddev composer` command now provides in-container Composer commands for nearly anything you’d want to do with Composer. We found that lots of people, and especially Windows users, were having trouble with fairly difficult workarounds to use Composer. However this has value to most DDEV users:
    - The Composer and PHP version used are the exact version configured for your project.
    - Your Composer project will be configured for the OS it’s actually running (Linux in the container). This was a serious problem for Windows users, as `composer install` on Windows OS did not result in the same results as `composer install` in Linux, even if symlinks were working.
    - Note that because of problems with symlinks on Windows, it is *not* recommended to use code from the host (Windows) side, or to check it in. That means it will not be appropriate to check in the vendor directory on the host (although it would be safe inside the container), but most people do a composer build anyway.
    - The `ddev composer create` command is almost the same as `composer create-project` but we couldn’t make it exactly the same. See [docs](https://ddev.readthedocs.io/en/latest/users/basics/developer-tools/#ddev-and-composer) for its usage.
- Composer caching: Composer downloads are now cached in a shared Docker volume, making in-container composer builds far faster.
- Shared SSH authentication in web container: You can now `ddev auth ssh` to authenticate your keys in the automatically-started `ddev-ssh-agent` container, which shares auth information with ever project’s web container. This allows access to private composer repositories without the pain of manually mounting SSH keys and authenticating each time you need them in each web container. It also allows easier use of facilities like `drush rsync` that need SSH auth. This means that the previous [manual workaround for mounting SSH keys](https://stackoverflow.com/questions/51065054/how-can-i-get-my-ssh-keys-and-identity-into-ddevs-web-container) is now obsolete. Please use `ddev auth ssh` instead.
- Configurable working and destination directories. You can now specify the container directory you land in with `ddev ssh`, `ddev exec`, and exec hooks in `config.yaml` (#1214). This also means that TYPO3 users will land in the project root by default; Drupal/Backdrop users land in the project root by default.
- [`ddev export-db`](https://ddev.readthedocs.io/en/stable/users/cli-usage/#exporting-a-database) makes textual exports of the database easier; Don’t forget about `ddev snapshot` as well.

## v1.3 (released 2018-10-11)

- Apache support now works on Windows and has full test coverage and is no longer experimental. It seems to work quite well.
- MariaDB is upgraded from 10.1 to 10.2. MariaDB 10.2 has a number of advantages including more flexible key lengths. (This does mean that pre-v1.1.0 databases which were bind-mounted in `~/.ddev` can no longer be migrated to Docker volumes with this release, see caveats above. Also, snapshots from previous releases cannot be restored with this release, in caveats above.)
- Automatic generation of WordPress `wp-config.php` has been improved and extra guidance provided for WordPress projects. (#1156)
- The web server container logs format is improved and all key logs are now provided in `ddev logs`.
- The web server container now gives significantly more information in its healthcheck information (use `docker inspect --format='{{json .State.Health }}' ddev-<project>-web`). It also exits when one of its components is unhealthy, so that we all don’t spend time debugging a broken container or broken add-on configuration.
- The dbserver container now provides full log information to `ddev logs -s db`.
- `php-imagick`, `locales`, and `php*-sqlite3` and `sqlite3` packages added to web container. You can now run a project with a sqlite3 database.
- It’s now possible to access web/HTTP/HTTPS URLs by name (including HTTPS) from within the web container. They get routed to the `ddev-router`, which sends them back to the right place. So `curl https://somesite.ddev.local:8443` will work inside the web container for a project configured with HTTPS on port 8443. (#1151)
- No PHP functions are now disabled in the PHP `disable_functions` configuration. (#1130)
- Drupal Drush commands do not automatically use `-y` when issued inside the web container. They still do when used in exec hooks, so this should not break any exec hooks. (#1120)
- `ddev pull` now adds flags so you can skip downloading either files or db. See `ddev help pull`.
- A number of bugfixes, docs fixes, and significant automated testing improvements.

## v1.2 (released 2018-09-11)

- Experimental Apache support has been added, but it doesn’t yet work on Windows. You can now run with `apache-with-php-fpm` (`apache-fpm`) or `apache-with-cgi` (`apache-cgi`). Change your `.ddev/config.yaml` to use `webserver_type: apache-fpm` or `webserver_type: apache-cgi`. (#1007)
- `ddev config` now has additional config flags: `--php-version`, `--http-port`, `--https-port`, `--xdebug-enabled`, `--additional-hostnames`, `--additional-fqdns` to allow command-line configuration instead of direct editing of the `.ddev/config.yaml` file. (#1092)
- The upload directory (Drupal public files directory, etc.) can now be specified in `config.yaml`. (#1093)

## v1.1 (released 2018-08-15)

- DDEV now requires Docker 18.06; a serious Docker bug in 18.03 caused lots and lots of crashes, so we moved it up to 18.06.
- You can now remove hostnames that DDEV has added to `/etc/hosts`.
    - `ddev remove --remove-data` removes the hostname(s) associated with the project
    - `sudo ddev hostname --remove-inactive` will remove from `/etc/hosts` all hostnames that are not currently active in a DDEV project.
- The docker-compose version has been updated to 3.6, so any customized `docker-compose.*.yaml` files in your project must be updated to read `version: '3.6'`
- The project database is now stored in a Docker volume instead of in the `~/.ddev/<project>/mysql` directory. This means that on your first `ddev start` it will be migrated from the `~/.ddev` file into a Docker volume. The old `~/.ddev/<project>/mysql` will be renamed to `~/.ddev/<project>/mysql.bak`.
- Database snapshotting is now available. At any time you can create a snapshot (in mariabackup format) using `ddev snapshot` or `ddev snapshot --name <somename>`. That db snapshot can easily be restored later with `ddev restore-snapshot <somename>`. These are stored in the project’s `.ddev/db_snapshots directory`.
- `ddev remove --remove-data` now creates a snapshot by default.
- For Drupal users, Drush now works on the host for many commands (after you’ve done a `ddev config` and `ddev start`). So, for example, you can run `drush sql-cli` or `drush cr` on the *host* when you need it, rather than using `ddev exec` or `ddev ssh` to do it in the web container. This assumes you have Drush available on the host.
- `ddev --import-files` now works on TYPO3 and Backdrop.
- DDEV now has integration with the Drud hosting service, so `ddev config drud-s3` works for users of the Drud hosting service.
- `php-redis` was added to web container.

## v1.0 (released 2018-07-19)

- Improvements of settings file management for Drupal and Backdrop (more below) (#468)
- Support for fully qualified domain names (FQDNs) (#868) - You can now add to your `.ddev/config.yaml` `additional_fqdns: ["mysite.example.com"]` and your site will be available at `http://mysite.example.com` after restart. More in [the docs](https://ddev.readthedocs.io/en/latest/users/extend/additional-hostnames/)
- Start, stop, and remove multiple (or all) projects at once (#952). You can `ddev rm project1 project2 project3` or `ddev rm --all`; it works with `ddev stop` as well, and with `ddev start` for running or stopped projects.
- Much better resilience when a project is running and the host is rebooted or Docker is restarted, etc. This used to result in database corruption regularly on nontrivial databases, and seems to be much improved.
