# Installation/Upgrade

See the [installation instructions](https://github.com/drud/ddev/blob/master/docs/index.md) for details, but it's easy:

- macOS Homebrew: `brew upgrade ddev`
- Linux or macOS via script:
`curl https://raw.githubusercontent.com/drud/ddev/master/install_ddev.sh | bash`
- Windows: Download the ddev Windows installer above.

And anywhere, you can just download the tarball or zipball, un-tar or un-zip it, and place the executable in your path where it belongs.

To upgrade to this release with each existing project, please be cautious and:

1. Temporarily remove any docker-compose.*.yaml customizations you’ve made, and any nginx, apache, php or mariadb overrides.
2. Run `ddev config` in your project directory to update your .ddev/config.yaml
3. After you've verified basic operation, add your customizations back in.

# Caveats

* Snapshots from previous versions of ddev cannot be restored with v1.3+. There's an easy workaround [explained in the docs](https://ddev.readthedocs.io/en/latest/users/troubleshooting/#cant-restore-snapshot-created-before-ddev-v13) 
* Databases from versions before v1.1 (bind-mounted, stored in `~/.ddev/<project>/mysql`) cannot be migrated to Docker volumes by this version because that process uses snapshots. However, you can migrate with v1.2.0 and then use v1.3; the easiest thing to do with those old ones is to `mv ~/.ddev/<project>/mysql ~/.ddev/<project/mysql.saved` and then use `ddev import-db` to load from a sql dump file.
* Before v1.3, containers ran as root for Windows users, now they run as a normal user. This means that if you have hooks (like post-start hooks) in your config.yaml that do things that need extra privileges, you'll have to add "sudo" to them. 

# Key changes in _VERSION_:

*

# Commits since _PREVIOUS VERSION_

```
$ git log --oneline --decorate=no $PREVIOUS_VERSION..$VERSION
```
