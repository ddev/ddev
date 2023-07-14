# Using the `ddev` Command

Type `ddev` or `ddev -h` in a terminal window to see the available DDEV [commands](../usage/commands.md). There are commands to configure a project, start, stop, describe, etc. Each command also has help using `ddev help <command>` or `ddev command -h`. For example, `ddev help snapshot` will show help and examples for the snapshot command.

* [`ddev config`](../usage/commands.md#config) configures a project’s type and docroot.
* [`ddev start`](../usage/commands.md#start) starts up a project.
* [`ddev launch`](../usage/commands.md#launch) opens a web browser showing the project.
* [`ddev list`](../usage/commands.md#list) shows current projects and their state.
* [`ddev describe`](../usage/commands.md#describe) gives all the info about the current project.
* [`ddev ssh`](../usage/commands.md#ssh) takes you into the web container.
* [`ddev exec <command>`](../usage/commands.md#exec) executes a command inside the web container.
* [`ddev stop`](../usage/commands.md#stop) stops a project and removes its memory usage (but does not throw away any data).
* [`ddev poweroff`](../usage/commands.md#poweroff) stops all resources that DDEV is using and stops the Mutagen daemon if it’s running.
* [`ddev delete`](../usage/commands.md#delete) destroys the database and DDEV’s knowledge of the project without touching your code.
* [`ddev get`](../usage/commands.md#get) adds an add-on service.

## Lots of Other Commands

* [`ddev mysql`](../usage/commands.md#mysql) gives direct access to the MySQL client and `ddev psql` to the PostgreSQL `psql` client.
* `ddev sequelace`, [`ddev tableplus`](../usage/commands.md#tableplus), and `ddev querious` (macOS only, if the app is installed) give access to the Sequel Ace, TablePlus or Querious database browser GUIs.
* `ddev heidisql` (Windows/WSL2 only, if installed) gives access to the HeidiSQL database browser GUI.
* [`ddev import-db`](../usage/commands.md#import-db) and [`ddev export-db`](../usage/commands.md#export-db) import or export SQL or compressed SQL files.
* [`ddev composer`](../usage/commands.md#composer) runs Composer inside the container. For example, `ddev composer install` will do a full composer install for you without even needing Composer on your computer. See [developer tools](developer-tools.md#ddev-and-composer).
* [`ddev snapshot`](../usage/commands.md#snapshot) makes a very fast snapshot of your database that can be easily and quickly restored with [`ddev snapshot restore`](../usage/commands.md#snapshot-restore).
* [`ddev share`](../usage/commands.md#share) requires ngrok and at least a free account on [ngrok.com](https://ngrok.com) so you can let someone in the next office or on the other side of the planet see your project and what you’re working on. `ddev share -h` gives more info about how to set up ngrok.
* [`ddev xdebug`](../usage/commands.md#xdebug) enables Xdebug, `ddev xdebug off` disables it, and `ddev xdebug status` shows status.
* [`ddev xhprof`](../usage/commands.md#xhprof) enables xhprof, `ddev xhprof off` disables it, and `ddev xhprof status` shows status.
* `ddev drush` (Drupal and Backdrop only) gives direct access to the `drush` CLI.
* `ddev artisan` (Laravel only) gives direct access to the Laravel `artisan` CLI.
* `ddev magento` (Magento2 only) gives access to the `magento` CLI.
* [`ddev craft`](../usage/commands.md#craft) (Craft CMS only) gives access to the `craft` CLI.
* [`ddev yarn`](../usage/commands.md#yarn) and [`ddev npm`](../usage/commands.md#npm) give direct access to the `yarn` and `npm` CLIs.

## Node.js, npm, nvm, and Yarn

`nodejs`, `npm`, `nvm` and `yarn` are preinstalled in the web container. You can configure the default value of the installed Node.js version with the [`nodejs_version`](../configuration/config.md#nodejs_version) option in `.ddev/config.yaml` or with `ddev config --nodejs_version`. You can also override that with any value using the built-in `nvm` in the web container or with [`ddev nvm`](../usage/commands.md#nvm), for example `ddev nvm install 6`. There is also a [`ddev yarn`](../usage/commands.md#yarn) command.

## More Bundled Tools

In addition to the [*commands*](../usage/commands.md) listed above, there are lots of tools included inside the containers:

* [`ddev describe`](../usage/commands.md#describe) tells how to access **MailHog**, which captures email in your development environment.
* Composer, Git, Node.js, npm, nvm, and dozens of other tools are installed in the web container, and you can access them via [`ddev ssh`](../usage/commands.md#ssh) or [`ddev exec`](../usage/commands.md#exec).
* [`ddev logs`](../usage/commands.md#logs) gets you web server logs; `ddev logs -s db` gets database server logs.
* `sqlite3` and the `mysql` and `psql` clients are inside the web container (and `mysql` or `psql` client is also in the `db` container).

## Exporting a Database

You can export a database with [`ddev export-db`](../usage/commands.md#export-db), which outputs to stdout or with options to a file:

```bash
ddev export-db --file=/tmp/db.sql.gz
ddev export-db --gzip=false --file=/tmp/db.sql
ddev export-db >/tmp/db.sql.gz
```

## `ddev import-files`

To import static file assets for a project, such as uploaded images and documents, use the command [`ddev import-files`](../usage/commands.md#import-files). This command will prompt you to specify the location of your import asset, then import the assets into the project’s upload directory. To define a custom upload directory, set the [`upload_dirs`](../configuration/config.md#upload_dirs) config option. If no custom upload directory is defined, the default will be used:

* For Backdrop projects, this is the `files`.
* For Drupal projects, these are the `sites/default/files` and `../private` directories.
* For Magento 1 projects, this is the `media` directory.
* For Magento 2 projects, this is the `pub/media` directory.
* For Shopware projects, this is the `media` directory.
* For TYPO3 projects, this is the `fileadmin` directory.
* For WordPress projects, this is the `wp-content/uploads` directory.

Other project types need a custom configuration to be able to use this command.

```bash
ddev import-files
Provide the path to the directory or archive you wish to import. Please note, if the destination directory exists, it will be replaced with the import assets specified here.
Import path:
~/Downloads/files.tar.gz
Successfully imported files for drupal8
```

`ddev import-files` supports the following file types: `.tar`, `.tar.gz`, `.tar.xz`, `.tar.bz2`, `.tgz`, or `.zip`.

It can also import a directory containing static assets.

If you want to use `import-files` without answering prompts, use the `--source` or `-s` flag to provide the path to the import asset. If you’re importing an archive, and wish to specify the path within the archive to extract, you can use the `--extract-path` flag in conjunction with the `--source` flag. Example:

`ddev import-files --source=/tmp/files.tgz`

When multiple `upload_dirs` are defined and you want to import to another upload dir than the first one, use the `--target` or `-t` flag to provide the path to the desired upload dir:

`ddev import-files --target=../private --source=/tmp/files.tgz`

See `ddev help import-files` for more examples.

## Snapshotting and Restoring a Database

The project database is stored in a Docker volume, but can be snapshotted (and later restored) with the [`ddev snapshot`](../usage/commands.md#snapshot) command. A snapshot is automatically taken when you run [`ddev stop --remove-data`](../usage/commands.md#stop). For example:

```bash
ddev snapshot
Creating database snapshot d9_20220107124831-mariadb_10.3.gz
Created database snapshot d9_20220107124831-mariadb_10.3.gz

ddev snapshot restore d9_20220107124831
Stopping db container for snapshot restore of 'd9_20220107124831-mariadb_10.3.gz'...
Restored database snapshot d9_20220107124831-mariadb_10.3.gz
```

Snapshots are stored as gzipped files in the project’s `.ddev/db_snapshots` directory, and the file created for a snapshot can be renamed as necessary. For example, if you rename the above `d9_20220107124831-mariadb_10.3.gz` file to `working-before-migration-mariadb_10.3.gz`, then you can use `ddev snapshot restore working-before-migration`. (The description of the database type and version—`mariadb_10.3`, for example—must remain intact.)
To restore the latest snapshot add the `--latest` flag (`ddev snapshot restore --latest`).

List snapshots for an existing project with `ddev snapshot --list`. (Add the `--all` option for an exhaustive list; `ddev snapshot --list --all`.) You can remove all of them with `ddev snapshot --cleanup`, or remove a single snapshot with `ddev snapshot --cleanup --name <snapshot-name>`.

!!!tip
    The default 120-second timeout may be inadequate for restores with very large snapshots or slower systems. You can increase this timeout by setting [`default_container_timeout`](../configuration/config.md#default_container_timeout) to a higher value.

    A timeout doesn’t necessarily mean the restore failed; you can watch the snapshot restore complete by running `ddev logs -s db`.

## Interacting with Your Project

DDEV provides several commands to facilitate interacting with your project in the development environment. These commands can be run within the working directory of your project while the project is running in DDEV.

### Executing Commands in Containers

The [`ddev exec`](../usage/commands.md#exec) command allows you to run shell commands in the container for a DDEV service. By default, commands are executed on the web service container, in the docroot path of your project. This allows you to use [the developer tools included in the web container](developer-tools.md). For example, to run the `ls` command in the web container, you would run `ddev exec ls` or `ddev . ls`.

To run a shell command in the container for a different service, use the `--service` (or `-s`) flag at the beginning of your `exec` command to specify the service the command should be run against. For example, to run the MySQL client in the database, container, you would run `ddev exec --service db mysql`. To specify the directory in which a shell command will be run, use the `--dir` flag. For example, to see the contents of the `/usr/bin` directory, you would run `ddev exec --dir /usr/bin ls`.

To run privileged commands, sudo can be passed into `ddev exec`. For example, to update the container’s apt package lists, use `ddev exec sudo apt-get update`.

Commands can also be executed using the shorter `ddev . <cmd>` alias.

Normally, `ddev exec` commands are executed in the container using Bash, which means that environment variables and redirection and pipes can be used. For example, a complex command like `ddev exec 'ls -l ${DDEV_FILES_DIR} | grep x >/tmp/junk.out'` will be interpreted by Bash and will work. However, there are cases where Bash introduces too much complexity and it’s best to run the command directly. In those cases, something like `ddev exec --raw ls -l "dir1" "dir2"` may be useful. With `--raw`, the `ls` command is executed directly instead of the full command being interpreted by Bash. But you cannot use environment variables, pipes, redirection, etc.

### SSH Into Containers

The [`ddev ssh`](../usage/commands.md#ssh) command opens an interactive Bash or sh shell session to the container for a DDEV service. The web service is connected by default, and the session can be ended by typing `exit`. To connect to another service, use the `--service` flag to specify the service you want to connect to. For example, to connect to the database container, you would run `ddev ssh --service db`. To specify the destination directory, use the `--dir` flag. For example, to connect to the database container and be placed into the `/home` directory, you would run `ddev ssh --service db --dir /home`.

You can also use your personal SSH keys within the web container. Run `ddev auth ssh` to add the keys from your `~/.ssh` directory and provide a passphrase, and those keys will be usable from within the web container. You generally only have to `ddev auth ssh` one time per computer reboot. This is a very popular approach for accessing private Composer repositories, or for using `drush` aliases against remote servers.

### `ddev logs`

The [`ddev logs`](../usage/commands.md#logs) command allows you to easily view error logs from the web container (both nginx/Apache and php-fpm logs are concatenated). To follow the logs in real time, run `ddev logs -f`. When you’re done, press <kbd>CTRL</kbd> + <kbd>C</kbd> to exit the log trail. Similarly, `ddev logs -s db` will show logs from a running or stopped database container.

## Stopping a Project

To remove a project’s containers, run [`ddev stop`](../usage/commands.md#stop) in the project’s working directory. To remove any running project’s containers regardless of context, specify the project name as an argument: `ddev stop <projectname>`.

`ddev stop` is *not* destructive. It removes the Docker containers but does not remove the database for the project, and does nothing to your code. This allows you to have many configured projects with databases loaded without wasting Docker containers on unused projects. **`ddev stop` does not affect the project codebase and files.**

To remove the imported database for a project, use the flag `--remove-data`, as in `ddev stop --remove-data`. This command will destroy both the containers and the imported database contents.
