## DDEV Version History

This version history has been driven by what we hear from our wonderful community of users. If you have lobbying for a favorite item or think things should be re-prioritized, just lobby in the [issue queue](https://github.com/drud/ddev/issues). We listen. Or talk to us in any of the [support locations](https://ddev.readthedocs.io/en/stable/#support).

### v1.18 (Released 2021-09-28)

- [x] Mutagen support results in a huge speedup for macOS and traditional Windows users
- [x] Support docker-compose v1 and v2
- [x] Support MariaDB 10.6
- [x] Support PHP 8.1
- [x] Improved integration with PhpStorm on all platforms, including WSL
- [x] xhprof support for performance profiling alongside blackfire.io support
- [x] Base image for the ddev-webserver is now Debian 11 Bullseye

### v1.17 (Released 2021-04-07)

- [x] Composer v2 is now the default composer version
- [x] Brand new provider integration system, with user-configurable and extensible techniques, Acquia, Platform.sh, DDEV-Live, Pantheon.io integration
- [x] Excellent improvements to `ddev snapshot`, including `ddev snapshot restore --latest`, prompted `ddev snapshot restore`, `ddev snapshot --list`, `ddev snapshot --cleanup`, `ddev snapshot --all`
- [x] `ddev snapshot` restore now shows progress as it goes
- [x] Built-in support for Blackfire.io profiling
- [x] New ddev config --auto option that configures a project with detected defaults
- [x] Web container environment variables can be set in `config.yaml` or `global_config.yaml` with the `web_environment key`
- [x] `ddev heidisql` command provides a nice database browser on Windows and Windows WSL2
- [x] The PHP default for new projects is now PHP 7.4
- [x] The MariaDB default for new projects is 10.3
- [x] New docs theme

### v1.16 (Released 2020-11-12)

- [x] Support Shopware 6
- [x] Remove support for docker toolbox on Win10 Home (in favor of new docker desktop)
- [x] Remove apache-cgi webserver_type
- [x] Per-project-type commands like `ddev drush`, `ddev typo3`
- [x] Build hardened ddev with hardened images for open-source production hosting

### v1.15 (Released 2020-07-08)

- [x] Laravel support
- [x] Global custom commands
- [x] Global homeadditions
- [x] WSL2 support
- [x] Reworked Nginx/Apache configurations
- [x] zsh completions
- [x] MariaDB 10.5 support
- [x] Remove obsolete support for drud-aws.

### v1.14 (Released 2020-04-21)

- [x] Drupal 9 support
- [x] Global NFS configuration
- [x] `ddev xdebug` command
- [x] Improve `ddev describe` to show information about additional services, <https://github.com/drud/ddev/issues/788>
- [x] Competitive analysis with similar products both within the spaces we usually work and outside them.
- [x] GUI evaluations

### v1.13 (Released 2020-02-04)

- [x] Updated support of pantheon via terminus instead of undocumented API
- [x] Support for Magento and Magento 2, #1802
- [x] Remove deprecated support for webcache
- [ ] Develop an advisory board of interested users to determine product focus
- [x] Review/experimentation with GUI options [#2110](https://github.com/drud/ddev/issues/2110)

### v1.12 (Released 2019-12-04)

- [x] Support for multiple versions of Oracle MySQL as well as MariaDB
- [x] Improved WordPress support (several open WordPress bugs)
- [x] Custom command improvements

### v1.11 (Released 2019-09-19)

- [x] PHP 7.4 support
- [x] [Allow omitting the db container](https://github.com/drud/ddev/issues/1490)

### v1.10 (released 2019-08-02)

- [x] Improved instrumentation with [segment](https://segment.com/): @unn is advocating for segment as better than Sentry (or in addition to Sentry). Statistics: Monthly active users, Conversion ratio. [#1640](https://github.com/drud/ddev/issues/1640)
- [x] [Add custom ddev commands](https://github.com/drud/ddev/issues/1372) - See [docksal's approach](https://docs.docksal.io/fin/custom-commands/)
- [x] [Allow user additions to .bashrc, store bash history, copy gitconfig](https://github.com/drud/ddev/issues/926): These are intended to make it more comfortable for the user to work inside the web container.
- [x] [Add "ddev mysql" command](https://github.com/drud/ddev/issues/1551)
- [x] [Add delete, poweroff, cleanup commands and hints](https://github.com/drud/ddev/issues/1588)
- [x] Sign macOS binary #1626
- [x] Make sure exposed ports are not exposed on local subnet, #1662
- [x] [Rework containers to provide a "real" user inside container](https://github.com/drud/ddev/issues/1403)

### v1.9 (released 2019-06-26)

- [x] [Contrib-pointers for additional Services and techniques](https://github.com/drud/ddev/issues/1474): We want to make another place for the outstanding content and pointers and applications that our users are developing. This will probably be a contrib repository for ddev.
- [x] [NFS Setup Security Review](#1474): More docs and improved NFS setup scripts so people can think clearly and plan carefully for how they're using NFS with ddev.
- [x] [Use DNS to provide name resolution when internet available](https://github.com/drud/ddev/issues/416)
- [x] [Manage ddev project list in ~/.ddev/global_config.yaml](https://github.com/drud/ddev/issues/642): Since the beginning of ddev `ddev list` and everything that depended on it couldn't work if the project was shut down. This should fix that.
- [x] [Allow specifying a target container for hook execution](https://github.com/drud/ddev/issues/1038)
- [x] [Support ngrok to allow web access remotely](https://github.com/drud/ddev/issues/375)
- [x] [Hook system overhaul](https://github.com/drud/ddev/issues/1372)
