# Using the `ddev` command

Type `ddev` or `ddev -h`in a terminal window to see the available ddev commands. There are commands to configure a project, start, stop, describe, etc. Each command also has help using `ddev help <command>` or `ddev command -h`. For example, `ddev help snapshot` will show help and examples for the snapshot command.

* `ddev config` configures a project's type and docroot.
* `ddev start` starts up a project.
* `ddev launch` opens a web browser showing the project.
* `ddev list` shows current projects and their state.
* `ddev describe` gives all the info about the current project.
* `ddev ssh` takes you into the web container.
* `ddev exec <command>` lets you execute any command inside the web container.
* `ddev stop` stops a project and removes its memory usage (but does not throw away any data).
* `ddev poweroff` stops all resources that DDEV is using. 
* `ddev delete` will destroy the database and DDEV's knowledge of the project, but does nothing to your code.

## Lots of other commands

* `ddev mysql` gives direct access to the mysql client and `ddev psql` to the PostgreSQL `psql` client.
* `ddev sequelpro`, `ddev sequelace`, and `ddev tableplus` (macOS only, if the app is installed) give access to the Sequel Pro, Sequel Ace, or TablePlus database browser GUIs.
* `ddev heidisql` (Windows/WSL2 only, if installed) gives access to the HeidiSQL database browser GUI.
* `ddev import-db` and `ddev export-db` let you import or export a sql or compressed sql file.
* `ddev composer` lets you run composer (inside the container), for example `ddev composer install` will do a full composer install for you without even needing composer on your computer. See [developer tools](developer-tools.md#ddev-and-composer). 
* `ddev snapshot` makes a very fast snapshot of your database that can be easily and quickly restored with `ddev snapshot restore`.
* `ddev share` requires ngrok and at least a free account on [ngrok.com](https://ngrok.com) so you can let someone in the next office or on the other side of the planet see your project and what you're working on. `ddev share -h` gives more info about how to set up ngrok.
* `ddev xdebug` enables xdebug, `ddev xdebug off` disables it, `ddev xdebug status` shows status
* `ddev xhprof` enables xhprof, `ddev xhprof off` disables it, `ddev xhprof status` shows status
* `ddev drush` (Drupal and Backdrop only) gives direct access to the drush CLI
* `ddev artisan` (Laravel only) gives direct access to the Laravel artisan CLI
* `ddev magento` (Magento2 only) gives access to the magento CLI
* `ddev yarn` and `ddev npm` give direct access to the yarn/npm CLIs

## Node.js, npm, nvm, and yarn

`nodejs`, `npm`, `nvm` and `yarn` are preinstalled in the web container. You can configure the default value of the installed nodejs version with the `nodejs_version` option in `.ddev/config.yaml` or with `ddev config --nodejs_version`. You can also override that with any value using the built-in `nvm` in the web container or with `ddev nvm`, for example `ddev nvm install 6`. There is also a `ddev yarn` command.

## More bundled tools

In addition to the *commands* listed above, there are loads and loads of tools included inside the containers:

* `ddev describe` tells how to access **MailHog**, which captures email in your development environment.
* `ddev describe` tells how to use the built-in **phpMyAdmin** and `ddev launch -p` gives direct access to it.
* Composer, git, node, npm, nvm, and dozens of other tools are installed in the web container, and you can access them via `ddev ssh` or `ddev exec`. 
* `ddev logs` gets you webserver logs; `ddev logs -s db` gets dbserver logs.
* `sqlite3` and the `mysql` and `psql` clients are inside the web container (and `mysql` or `psql` client is also in the db container).

## Exporting a Database

You can export a database with `ddev export-db`, which outputs to stdout or with options to a file:

```bash
ddev export-db --file=/tmp/db.sql.gz
ddev export-db --gzip=false --file=/tmp/db.sql
ddev export-db >/tmp/db.sql.gz
```

## `ddev import-files`

To import static file assets for a project, such as uploaded images and documents, use the command `ddev import-files`. This command will prompt you to specify the location of your import asset, then import the assets into the project's upload directory. To define a custom upload directory, set the `upload_dir` key in your project's `config.yaml`. If no custom upload directory is defined, the default will be used:

* For Drupal projects, this is the `sites/default/files` directory.
* For WordPress projects, this is the `wp-content/uploads` directory.
* For TYPO3 projects, this is the `fileadmin` directory.
* For Backdrop projects, this is the `files` .
* For Magento 1 projects, this is the `media` directory.
* For Magento 2 projects, this is the `pub/media` directory.

```bash
ddev import-files
Provide the path to the directory or archive you wish to import. Please note, if the destination directory exists, it will be replaced with the import assets specified here.
Import path:
~/Downloads/files.tar.gz
Successfully imported files for drupal8
```

`ddev import-files` supports the following file types:  `.tar`, `.tar.gz`, `.tar.xz`, `.tar.bz2`, `.tgz`, or `.zip`.

It can also import directory containing static assets.

If you want to use import-files without answering prompts, you can use the `--src` flag to provide the path to the import asset. If you are importing an archive, and wish to specify the path within the archive to extract, you can use the `--extract-path` flag in conjunction with the `--src` flag. Example:

`ddev import-files --src=/tmp/files.tgz`

See `ddev help import-files` for more examples.

## Snapshotting and restoring a database

The project database is stored in a docker volume, but can be snapshotted (and later restored) with the `ddev snapshot` command. A snapshot is automatically taken when you do a `ddev stop --remove-data`. For example:

```bash
ddev snapshot
Creating database snapshot d9_20220107124831-mariadb_10.3.gz
Created database snapshot d9_20220107124831-mariadb_10.3.gz

ddev snapshot restore d9_20220107124831-mariadb_10.3.gz
Stopping db container for snapshot restore of 'd9_20220107124831-mariadb_10.3.gz'...
Restored database snapshot d9_20220107124831-mariadb_10.3.gz
```

Snapshots are stored as gzipped files in the project's .ddev/db_snapshots directory, and the file created for a snapshot can be renamed as necessary. For example, if you rename the above d9_20220107124831-mariadb_10.3.gz file to "working-before-migration-mariadb_10.3.gz", then you can use `ddev snapshot restore working-before-migration-mariadb_10.3.gz`. (Note that the description of the database type and version (`mariadb_10.3` for example) must remain intact.)
To restore the latest snapshot add the `--latest` flag (`ddev snapshot restore --latest`).

All snapshots of a project can be removed with `ddev snapshot --cleanup`. A single snapshot can be removed by `ddev snapshot --cleanup --name <snapshot-name>`.

To see all existing snapshots of a project use `ddev snapshot --list`.
All existing snapshots of all projects can be listed by adding the `--all` option to the command (`ddev snapshot --list --all`).

Note that with very large snapshots or perhaps with slower systems, the default timeout to wait for the snapshot restore to complete may not be adequate. In these cases you can increase the timeout by setting `default_container_timeout` to a higher value. Also, if it does time out on you, that doesn't mean it has actually failed. You can watch the snapshot restore complete with `ddev logs -s db`.

## Interacting with your project

DDEV provides several commands to facilitate interacting with your project in the development environment. These commands can be run within the working directory of your project while the project is running in ddev.

### Executing Commands in Containers

The `ddev exec` command allows you to run shell commands in the container for a ddev service. By default, commands are executed on the web service container, in the docroot path of your project. This allows you to use [the developer tools included in the web container](developer-tools.md). For example, to run the "ls" in the web container, you would run `ddev exec ls` or `ddev . ls`.

To run a shell command in the container for a different service, use the `--service` (or `-s`) flag at the beginning of your exec command to specify the service the command should be run against. For example, to run the mysql client in the database, container, you would run `ddev exec --service db mysql`. To specify the directory in which a shell command will be run, use the `--dir` flag. For example, to see the contents of the `/usr/bin` directory, you would run `ddev exec --dir /usr/bin ls`.

To run privileged commands, sudo can be passed into `ddev exec`. For example, to update the container's apt package lists, use `ddev exec sudo apt-get update`.

Commands can also be executed using the shorter `ddev . <cmd>` alias.

Normally, `ddev exec` commands are executed in the container using bash, which means that environment variables and redirection and pipes can be used. For example, a complex command like `ddev exec 'ls -l ${DDEV_FILES_DIR} | grep x >/tmp/junk.out'` will be interpreted by bash and will work. However, there are cases where bash introduces too much complexity and it's best to just run the command directly. In those cases, something like `ddev exec --raw ls -l "dir1" "dir2"` may be useful. with `--raw` the ls command is executed directly instead of the full command being interpreted by bash. But you cannot use environment variables, pipes, redirection, etc.

### SSH Into Containers

The `ddev ssh` command will open an interactive bash or sh shell session to the container for a ddev service. The web service is connected to by default. The session can be ended by typing `exit`. To connect to another service, use the `--service` flag to specify the service you want to connect to. For example, to connect to the database container, you would run `ddev ssh --service db`. To specify the destination directory, use the `--dir` flag. For example, to connect to the database container and be placed into the `/home` directory, you would run `ddev ssh --service db --dir /home`.

If you want to use your personal ssh keys within the web container, that's possible. Use `ddev auth ssh` to add the keys from your ~/.ssh directory and provide a passphrase, and then those keys will be usable from within the web container. You generally only have to `ddev auth ssh` one time per computer reboot. This is a very popular approach for accessing private composer repositories, or for using drush aliases against remote servers.

### `ddev logs`

The `ddev logs` command allows you to easily view error logs from the web container (both nginx/apache and php-fpm logs are concatenated). To follow the log (watch the lines in real time), run `ddev logs -f`. When you are done, press CTRL+C to exit from the log trail. Similarly, `ddev logs -s db` will show logs from a running or stopped db container.

## Stopping a project

To remove a project's containers run `ddev stop` in the working directory of the project. To remove any running project's containers, providing the project name as an argument, e.g. `ddev stop <projectname>`.

`ddev stop` is *not* destructive. It removes the docker containers but does not remove the database for the project, and does nothing to your codebase. This allows you to have many configured projects with databases loaded without wasting docker containers on unused projects. **`ddev stop` does not affect the project code base and files.**

To remove the imported database for a project, use the flag `--remove-data`, as in `ddev stop --remove-data`. This command will destroy both the containers and the imported database contents.

