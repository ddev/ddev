# DDEV Local Roadmap

This roadmap for DDEV-Local is intended to answer what we hear from our wonderful community of users. If you have lobbying for a favorite item or think things should be re-prioritized, just lobby in the [issue queue](https://github.com/drud/ddev/issues). We listen. Or talk to us in any of the [support locations](https://ddev.readthedocs.io/en/stable/#support).

These items are listed in loose priority order, so you can expect the top items to show up in nearer releases. We regularly fix bugs that are annoying our users, so only new features show up on this list.

We try to flag issues that we definitely intend to get done with the [prioritized](https://github.com/drud/ddev/issues?q=is%3Aissue+is%3Aopen+label%3APrioritized) label. If you think something is super-important, please request that it be marked "prioritized". It doesn't mean it will get tagged, but your vote absolutely counts.

## v1.17 and beyond

- [ ] Improve visual appearance of docs on readthedocs; consider switching build process.
- [ ] Global --root and --project flags (#2128)
- [ ] [Provide "ddev debug dockercheck" command](https://github.com/drud/ddev/issues/1443)
- [ ] Improve docs tutorial so people can just start right up with it.
- [ ] Implement automated testing for WSL2 projects
- [ ] Explicit support for additional CMSs, including Sulu, Joomla, CraftCMS
- [ ] GUI! again.

## Historical Releases

### v1.16 (Released 2020-11-12)

- [x] Support Shopware 6
- [x] Remove support for docker toolbox on Win10 Home (in favor of new docker desktop)
- [x] Remove apache-cgi webserver_type
- [ ] Remove nginx snippet support (.ddev/nginx)
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
