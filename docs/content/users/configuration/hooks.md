# Hooks

Most DDEV commands provide hooks to run tasks before or after the main command executes. To automate setup tasks specific to your project, define them in the project’s `config.yaml` file.

To define command tasks in your configuration, specify the desired command hook as a subfield to `hooks`, then a list of tasks to run:

```yaml
hooks:
  post-start:
    - exec: "simple command expression"
    - exec: "ls >/dev/null && touch /var/www/html/somefile.txt"
    - exec-host: "simple command expression"
  post-import-db:
    - exec: "drush uli"
```

## Global Hooks

Hooks that should run for every project on your machine go in the `hooks` section of `$HOME/.ddev/global_config.yaml` (see [global configuration directory](../usage/architecture.md#global-files)), with the same syntax as above:

```yaml
hooks:
  post-start:
    - exec-host: "echo a global hook ran"
```

For each hook, global tasks run first, then the project's tasks. If a project has a task identical to a global one, the global copy is dropped and the project's task runs in its place, so a hook you already copied into your projects does not run twice.

* A global `exec` task whose `service` is [omitted](config.md#omit_containers) or not running in a project is skipped with a notice, and does not count as a failure even with [`fail_on_hook_fail`](config.md#fail_on_hook_fail).
* Set [`skip_global_hooks: true`](config.md#skip_global_hooks) in a project's configuration to skip all global hooks there, and use `ddev --skip-hooks` to skip all hooks, global and project.
* `exec-host` tasks run in the project directory, so use absolute paths or `$HOME` when running a script from your home directory.
* Global `pre-exec` and `post-exec` hooks run on every [`ddev exec`](../usage/commands.md#exec) and on the commands DDEV runs in containers itself, for example during `ddev start` and `ddev import-db`, so keep them fast. They do not run around `exec` tasks in other hooks.
* An invalid hook name or task in `global_config.yaml` stops every `ddev` command with an error naming the file, until it is fixed.
* [`ddev utility check-custom-config`](../usage/commands.md#utility-check-custom-config) and the warning at `ddev start` list global hooks.

## Supported Command Hooks

* `pre-start`: Hooks into [`ddev start`](../usage/commands.md#start). Execute tasks before the project environment starts.

    !!!tip
        Only `exec-host` tasks can run during `pre-start` because the containers are not yet running. See [Supported Tasks](#supported-tasks) below.

* `post-start`: Execute tasks after the project environment has started.
* `pre-import-db` and `post-import-db`: Execute tasks before or after database import.
* `pre-import-files` and `post-import-files`: Execute tasks before or after files are imported.
* `pre-composer` and `post-composer`: Execute tasks before or after the `composer` command.
* `pre-share` and `post-share`: Execute tasks before or after the `share` command.
* `pre-stop`, `pre-config`, `post-config`, `pre-describe`, `post-describe`, `pre-exec`, `post-exec`, `pre-pull`, `post-pull`, `pre-push`, `post-push`, `pre-snapshot`, `post-snapshot`, `pre-delete-snapshot`, `post-delete-snapshot`, `pre-restore-snapshot`, `post-restore-snapshot`: Execute as the name suggests.
* `post-stop`: Hooks into [`ddev stop`](../usage/commands.md#stop). Execute tasks after the project environment stopped.

    !!!tip
        Only `exec-host` tasks can run during `post-stop`. See [Supported Tasks](#supported-tasks) below.

## Supported Tasks

DDEV currently supports these tasks:

* `exec` to execute a command in any service/container.
* `exec-host` to execute a command on the host.
* `composer` to execute a Composer command in the web container.

### `exec`: Execute a shell command in a container (defaults to web container)

Value: string providing the command to run. Commands requiring user interaction are not supported.

**Optional keys:**

* `service`: Specify which container to run the command in (defaults to `web`)
* `user`: Specify which user to run the command as (username or UID, defaults to container's default user)
* `exec_raw`: Array of command arguments for direct execution without shell interpretation (alternative to string command)

Example: _Use Drush to rebuild all caches and get a user login link after database import_.

```yaml
hooks:
  post-import-db:
    - exec: drush cache:rebuild
    - exec: drush user:login
```

Example: _Use wp-cli to replace the production URL with development URL in a WordPress project’s database_.

```yaml
hooks:
  post-import-db:
    - exec: wp search-replace https://www.myproductionsite.com http://mydevsite.ddev.site
```

Example: _Use Drush to sanitize a database by removing or obfuscating user data_.

```yaml
hooks:
  post-import-db:
    - exec: drush sql:sanitize
```

Example: _Add an extra database before `import-db`, executing in `db` container_.

```yaml
hooks:
  pre-import-db:
    - exec: mysql -uroot -proot -e "CREATE DATABASE IF NOT EXISTS some_new_database;"
      service: db
```

Example: _Execute a command as root user in the `db` container_.

```yaml
hooks:
  post-start:
    - exec: ls -la /root
      service: db
      user: root
```

Example: _Add the common `ll` alias into the `web` container’s `.bashrc` file_.

```yaml
hooks:
  post-start:
    - exec: sudo echo alias ll=\"ls -lhA\" >> ~/.bashrc
```

!!!tip
    This could be done more efficiently via `.ddev/web-build/Dockerfile` as explained in [Customizing Images](../extend/customizing-images.md).

Advanced usages may require running commands directly with explicit arguments. This approach is useful when Bash interpretation is not required (no environment variables, no redirection, etc.).

```yaml
hooks:
  post-start:
    - exec:
      exec_raw: [ls, -lR, /var/www/html]
```

### `exec-host`: Execute a shell command on the host system

Value: string providing the command to run. Commands requiring user interaction are not supported.

```yaml
hooks:
  pre-start:
    - exec-host: "command to run"
```

### `composer`: Execute a Composer command in the web container

Value: string providing the Composer command to run.

**Optional keys:**

* `exec_raw`: Array of Composer command arguments (alternative to string command)

Example:

```yaml
hooks:
  post-start:
    - composer: config discard-changes true
```

Example with `exec_raw`:

```yaml
hooks:
  post-start:
    - composer:
      exec_raw: [install, --no-dev]
```

## WordPress Example

```yaml
hooks:
  post-start:
    # Install WordPress after start
    - exec: "wp config create --dbname=db --dbuser=db --dbpass=db --dbhost=db"
    - exec: "wp core install --url=http://mysite.ddev.site --title=MySite --admin_user=admin --admin_email=admin@mail.test"
  post-import-db:
    # Update the URL of your project throughout your database after import
    - exec: "wp search-replace https://www.myproductionsite.com http://mydevsite.ddev.site"
```

## Drupal 7 Example

```yaml
hooks:
  post-start:
    # Install Drupal after start if not installed already
    - exec: "(drush status bootstrap | grep -q Successful) || drush site-install -y --db-url=db:db@db/db"
    # Generate a one-time login link for the admin account
    - exec: "drush uli"
  post-import-db:
    # Set the project name
    - exec: "drush vset site_name MyDevSite"
    # Enable the environment indicator module
    - exec: "drush en -y environment_indicator"
    # Clear the cache
    - exec: "drush cc all"
```

## Drupal 10 Example

```yaml
hooks:
  post-start:
    # Install Composer dependencies from the web container
    - composer: install
    # Generate a one-time login link for the admin account
    - exec: "drush user:login"
  post-import-db:
    # Sanitize the database
    - exec: "drush sql:sanitize"
    # Apply any database updates
    - exec: "drush updatedb"
    # Rebuild all caches
    - exec: "drush cache:rebuild"
```

## TYPO3 Example

```yaml
hooks:
  post-start:
    - composer: install
```
