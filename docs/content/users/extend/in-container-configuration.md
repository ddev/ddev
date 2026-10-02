# In-Container Home Directory and Shell Configuration

Custom shell configuration (Bash or your preferred shell), your usual Git configuration, a Composer `auth.json` and more can be achieved within your containers.

## Using `homeadditions` to Customize In-Container Home Directory

!!!tip "Finding Your Global DDEV Directory"
    The examples below automatically detect the correct global DDEV directory (including when `$DDEV_XDG_CONFIG_HOME` is set) using the `DDEV_DIR` variable. See [global configuration directory](../usage/architecture.md#global-files) for details.

Place all your dotfiles in your global `homeadditions` directory or your project's `.ddev/homeadditions` directory and DDEV will use these in your project's `web` containers.

!!!tip "Ignore `.ddev/.homeadditions`!"
    A hidden/transient `.ddev/.homeadditions`—emphasis on the leading `.`—is used for processing global `homeadditions` and should be ignored.

On [`ddev start`](../usage/commands.md#start), DDEV attempts to create a user inside the `web` and `db` containers with the same name and user ID as the one you have on the host machine.

DDEV looks for the `homeadditions` directory both in the global `homeadditions` directory and the project-level `.ddev/homeadditions` directory, and will copy their contents recursively into the in-container home directory during `ddev start`. Project `homeadditions` contents override the global `homeadditions`.

Usage examples:

### Git Configuration

If you use Git inside the container, you may want to symlink your `$HOME/.gitconfig` into the global `homeadditions` directory or the project's `.ddev/homeadditions` so that in-container `git` commands use whatever username and email you've configured on your host machine.

```bash
DDEV_DIR="$(ddev version -j | docker run -i --rm ddev/ddev-utilities jq -r ".raw.\"global-ddev-dir\" | select (.!=null) // \"$HOME/.ddev\"" 2>/dev/null)"
ln -s $HOME/.gitconfig $DDEV_DIR/homeadditions/.gitconfig
```

### SSH Configuration

If you use SSH inside the container and want to use your `.ssh/config`, you can symlink it into the homeadditions directory. Some people will be able to symlink their entire `.ssh` directory.

```bash
DDEV_DIR="$(ddev version -j | docker run -i --rm ddev/ddev-utilities jq -r ".raw.\"global-ddev-dir\" | select (.!=null) // \"$HOME/.ddev\"" 2>/dev/null)"
mkdir -p $DDEV_DIR/homeadditions/.ssh
ln -s $HOME/.ssh/config $DDEV_DIR/homeadditions/.ssh/config
```

Or symlink the entire directory:

```bash
DDEV_DIR="$(ddev version -j | docker run -i --rm ddev/ddev-utilities jq -r ".raw.\"global-ddev-dir\" | select (.!=null) // \"$HOME/.ddev\"" 2>/dev/null)"
ln -s $HOME/.ssh $DDEV_DIR/homeadditions/.ssh
```

If you provide your own `.ssh/config` though, please make sure it includes these lines:

```text
UserKnownHostsFile=/home/.ssh-agent/known_hosts
StrictHostKeyChecking=accept-new
```

Alternately, you may also place multiple SSH config files within the global or project `.ddev/homeadditions/.ssh/config.d` directory, and they'll be automatically included as part of the default DDEV SSH config. The files must have a `.conf` extension in order to be included.

### Custom Scripts and Executables

If you need to add a script or other executable component into the project (or global configuration), you can put it in the project or global `.ddev/homeadditions/bin` directory and `$HOME/bin/<script>` will be created inside the container. This is useful for adding a script to one project or every project, or for overriding standard scripts, as `$HOME/bin` is first in the `$PATH` in the `web` container.

For example, to add a custom script:

```bash
DDEV_DIR="$(ddev version -j | docker run -i --rm ddev/ddev-utilities jq -r ".raw.\"global-ddev-dir\" | select (.!=null) // \"$HOME/.ddev\"" 2>/dev/null)"
# Create the bin directory
mkdir -p $DDEV_DIR/homeadditions/bin
# Add your script
echo '#!/usr/bin/env bash' > $DDEV_DIR/homeadditions/bin/myscript
echo 'echo "Hello from custom script"' >> $DDEV_DIR/homeadditions/bin/myscript
chmod +x $DDEV_DIR/homeadditions/bin/myscript
```

### Composer Authentication

If you use private, password-protected Composer repositories with [Satis](https://composer.github.io/satis/), for example, and use a global `auth.json`, you can symlink it into `homeadditions`. Be careful to exclude it from getting checked in by using a `.gitignore` or equivalent.

```bash
DDEV_DIR="$(ddev version -j | docker run -i --rm ddev/ddev-utilities jq -r ".raw.\"global-ddev-dir\" | select (.!=null) // \"$HOME/.ddev\"" 2>/dev/null)"
mkdir -p "$DDEV_DIR/homeadditions/.composer"
# Find the correct location of the auth.json file if Composer is installed
COMPOSER_AUTH_FILE="$(composer config --global home 2>/dev/null)/auth.json"
# Otherwise default to the location on Debian
if [ ! -f "$COMPOSER_AUTH_FILE" ]; then
  COMPOSER_AUTH_FILE="$HOME/.composer/auth.json"
fi
ln -s "$COMPOSER_AUTH_FILE" $DDEV_DIR/homeadditions/.composer/auth.json
```

### Startup Scripts

You can add small scripts to the `.bashrc.d` directory, and they will be executed on [`ddev ssh`](../usage/commands.md#ssh).

For example, create a script that shows which container you're in:

```bash
DDEV_DIR="$(ddev version -j | docker run -i --rm ddev/ddev-utilities jq -r ".raw.\"global-ddev-dir\" | select (.!=null) // \"$HOME/.ddev\"" 2>/dev/null)"

# Create the .bashrc.d directory
mkdir -p $DDEV_DIR/homeadditions/.bashrc.d

# Add a script that runs on ddev ssh
echo 'echo "I am in the $(hostname) container"' > $DDEV_DIR/homeadditions/.bashrc.d/whereami
```

After `ddev restart`, when you `ddev ssh` this script will be executed.

### Custom Bashrc

If you have a favorite `.bashrc`, copy it into either the global or project `homeadditions`:

```bash
DDEV_DIR="$(ddev version -j | docker run -i --rm ddev/ddev-utilities jq -r ".raw.\"global-ddev-dir\" | select (.!=null) // \"$HOME/.ddev\"" 2>/dev/null)"
cp $HOME/.bashrc $DDEV_DIR/homeadditions/.bashrc
```

### Bash Aliases

If you like the traditional `ll` Bash alias for `ls -lhA`, add a `.bash_aliases` file to either the global or project `homeadditions`:

```bash
DDEV_DIR="$(ddev version -j | docker run -i --rm ddev/ddev-utilities jq -r ".raw.\"global-ddev-dir\" | select (.!=null) // \"$HOME/.ddev\"" 2>/dev/null)"
echo 'alias ll="ls -lhA"' > $DDEV_DIR/homeadditions/.bash_aliases
```

### Altering the In-Container `$PATH`

Sometimes it’s easiest to put the command you need into the existing `$PATH` using a symbolic link rather than changing the in-container `$PATH`. For example, the project `bin` directory is already included the `$PATH`. So if you have a command you want to run that’s not already in the `$PATH`, you can add a symlink.

Examples:

- On Craft CMS, the `craft` script is often in the project root, which is not in the `$PATH`. But if you `mkdir bin && ln -s craft bin/craft` you should be able to run `ddev exec craft`. (Note however that `ddev craft` takes care of this for you.)
- On projects where the `vendor` directory is not in the project root (Acquia projects, for example, have `composer.json` and `vendor` in the `docroot` directory), you can `mkdir bin && ln -s docroot/vendor/bin/drush bin/drush` to put `drush` in your `$PATH`. (With projects like this, make sure to set `composer_root: docroot` so that `ddev composer` works properly.)

You can also modify the `PATH` environment variable by adding a script to `<project>/.ddev/homeadditions/.bashrc.d/` or (global) `$HOME/.ddev/homeadditions/.bashrc.d/` (see [global configuration directory](../usage/architecture.md#global-files)). For example, if your project vendor directory is not in the expected place (`/var/www/html/vendor/bin`) you can add a `<project>/.ddev/homeadditions/.bashrc.d/path.sh`:

```bash
export PATH=$PATH:/var/www/html/somewhereelse/vendor/bin
```

## Persisting Changes Across Restarts

The `web` container is created again from its image after every [`ddev restart`](../usage/commands.md#restart), [`ddev stop`](../usage/commands.md#stop), or [`ddev poweroff`](../usage/commands.md#poweroff). Project files and databases are kept, but other changes in the container are lost, including everything in the home directory. `homeadditions` only copies files into the container, never back out.

By default, only the [`ddev-global-cache` volume](../usage/architecture.md#the-ddev-global-cache-volume) survives outside the project. This is where DDEV keeps caches and shell history.

| Change made in the container | How to keep it |
| --- | --- |
| `composer global require` | [Keep `~/.composer` in `ddev-global-cache`](#keeping-home-directories-in-ddev-global-cache), or install the tool with a [custom Dockerfile](customizing-images.md#examples) |
| `composer self-update` | [`composer_version`](../configuration/config.md#composer_version) |
| `n install <version>` | [`nodejs_version`](../configuration/config.md#nodejs_version) |
| `npm install -g` | `RUN npm install -g` in a [custom Dockerfile](customizing-images.md#examples) |
| `sudo apt-get install` | [`webimage_extra_packages`](../configuration/config.md#webimage_extra_packages) |
| Edited dotfiles, like `~/.bashrc` | [`homeadditions`](#using-homeadditions-to-customize-in-container-home-directory) |
| Files a tool writes, like `~/.config/gh` | [Keep the directory in `ddev-global-cache`](#keeping-home-directories-in-ddev-global-cache) |

### Persisting Subdirectories of home directory in `ddev-global-cache`

To persist a subdirectory from your `ddev-webserver` home directory, move it to `ddev-global-cache` and leave a symlink in its place. This script does that each time the `web` container starts:

```bash
# .ddev/web-entrypoint.d/persist.sh
# Keep these home subdirectories in ddev-global-cache so they survive restarts.
# Files from the image and homeadditions replace the stored copies at each start.
(
  for dir in .composer; do
    target="/mnt/ddev-global-cache/persist/${HOSTNAME}/${dir}"
    mkdir -p "${target}" "$(dirname ~/"${dir}")"
    if [ -d ~/"${dir}" ] && [ ! -L ~/"${dir}" ]; then
      cp -a ~/"${dir}"/. "${target}"/
      rm -rf ~/"${dir}"
    fi
    ln -sfn "${target}" ~/"${dir}"
  done
)
```

- Add directories to the `for` line, relative to the home directory, for example `for dir in .composer .config/gh; do`. They must be owned by your user, not `root`, otherwise the script can't move them and the `web` container doesn't start.
- Each project gets its own copy in `/mnt/ddev-global-cache/persist/<project>-web`. You can rename `persist` to anything DDEV doesn't [already use](../usage/architecture.md#the-ddev-global-cache-volume), but keep `${HOSTNAME}` directly under it, because `ddev delete` doesn't look deeper.
- At each start, files from the image and from `homeadditions` replace their stored copies. Files that exist only in the stored copy, like the packages you installed, are kept.

For `.composer`, also add Composer's global `bin` directory to `$PATH` with a `homeadditions` script:

```bash
# .ddev/homeadditions/.bashrc.d/composer-global.sh
export PATH="$PATH:$HOME/.composer/vendor/bin"
```

!!!warning "Data, not programs"
    Because files from the image replace their stored copies, this doesn't keep updates a tool makes to itself, see [Tools That Update Themselves](creating-add-ons.md#tools-that-update-themselves). Keep directories, not single files: many tools save a file by replacing it, which removes the symlink.

[`ddev delete`](../usage/commands.md#delete) removes the stored directories. To remove them without deleting the project:

```bash
ddev exec 'rm -rf /mnt/ddev-global-cache/persist/${HOSTNAME}'
ddev restart
```

Use single quotes, so `${HOSTNAME}` expands inside the container.

### Keeping a Directory in a Docker Volume

Instead of the script, you can mount a Docker volume over the directory with a `.ddev/docker-compose.*.yaml` file. Each directory needs its own volume:

```yaml
# .ddev/docker-compose.composer-home.yaml
services:
  web:
    volumes:
      - composer-home:/home/${DDEV_USER}/.composer
volumes:
  composer-home:
```

On first use, Docker fills the empty volume with what the image has in that directory. After that, the volume hides the image's copy, so changes to that directory from DDEV upgrades or add-ons don't show up. `homeadditions` files are still copied in at each start. `ddev delete` removes the volume.

## Changing `ddev ssh` Shell

You can define a default shell for [`ddev ssh`](../usage/commands.md#ssh) using the `x-ddev` extension field in your `.ddev/docker-compose.*.yaml` configuration.

Use the `x-ddev.ssh-shell` key and make sure that shell (such as `zsh` or `bash`) is included in the container image so `ddev ssh` work correctly. The selected shell also appears in the [`ddev describe`](../usage/commands.md#describe) output (if it's not the default one).

Changing the default shell to `zsh` in the `web` and `db` containers:

```yaml
# .ddev/config.yaml
webimage_extra_packages: [zsh]
dbimage_extra_packages: [zsh]
```

```yaml
# .ddev/docker-compose.ssh-shell.yaml
services:
  web:
    x-ddev:
      ssh-shell: zsh
  db:
    x-ddev:
      ssh-shell: zsh
```

To change the shell for a custom service, add the `x-ddev.ssh-shell` field to that service's configuration and ensure the desired shell is [installed in the image](./customizing-images.md).

## Changing the Container User

Some third-party images expect work to happen as a specific non-root user, so
the image's default user isn't right for that service. Set
`x-ddev.container-user` on that service so [`ddev ssh`](../usage/commands.md#ssh),
[`ddev exec`](../usage/commands.md#exec), and [custom commands](./custom-commands.md)
all run as that user without requiring an explicit `-u`. An explicit `-u` still
wins.

```yaml
# .ddev/docker-compose.devilbox-php.yaml
services:
  devilbox-php:
    container_name: ddev-${DDEV_SITENAME}-devilbox-php
    image: devilbox/php-fpm-5.3
    x-ddev:
      ssh-shell: bash
      container-user: www-data
```

With this in place, `ddev ssh -s devilbox-php` logs in as `www-data`, and the configured
user also appears in [`ddev describe`](../usage/commands.md#describe) output.

!!!tip
    See the [`x-ddev` Extension](../extend/custom-docker-services.md#x-ddev-extension) for all supported keys, including `describe-*` and `omit-ddev-labels`.

## Using `NO_COLOR` Inside Containers

To set the `NO_COLOR` variable in all containers across all projects, define the `NO_COLOR` environment variable in your shell configuration file (e.g., `$HOME/.bashrc` or `$HOME/.zshrc`), outside of DDEV, for example:

```bash
export NO_COLOR=1
```

`NO_COLOR=1` can also be implicitly set using [`simple_formatting`](../configuration/config.md#simple_formatting) option.

## Using `PAGER` Inside Containers

To set the `PAGER` variable in the `web` and `db` containers across all projects, define the `DDEV_PAGER` environment variable in your shell configuration file (e.g., `$HOME/.bashrc` or `$HOME/.zshrc`), outside of DDEV, for example:

```bash
export DDEV_PAGER="less -SFXR"
```

## In-container `ssh` or `rsync` failures

If you use `ddev auth ssh` and use `ssh` or `rsync` inside the container and see a message like this:

```bash
$ ddev exec ssh <hostname>
@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@
@    WARNING: REMOTE HOST IDENTIFICATION HAS CHANGED!     @
@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@
IT IS POSSIBLE THAT SOMEONE IS DOING SOMETHING NASTY!
```

It means that the host you are connecting to has actually changed its identification. If you know why that is, and accept the situation, you can clean up the situation with this command:

```bash
ddev exec ssh-keygen -f '/home/.ssh-agent/known_hosts' -R '<hostname>'
```

Use the hostname that gave you trouble.

## Too Many Authentication Failures

If `ssh` or `rsync` inside the container fails with:

```text
Received disconnect from <host> port 22:2: Too many authentication failures
```

the *server* you're connecting to has a `MaxAuthTries` limit and gave up after your agent offered more keys than it allows, not because any key was wrong. This is more likely with [`ssh_agent_upstream`](../usage/cli.md#using-an-existing-ssh-agent), which relays every key your agent holds, or with a large `~/.ssh` full of key files.

Limit which key gets offered for that host instead of pruning keys everywhere:

- With key files, use `ddev auth ssh -d <dir>` with a directory that has only the keys you need.
- With an agent, including `ssh_agent_upstream`, add a `Host` block to your [homeadditions `~/.ssh/config`](#ssh-configuration) with `IdentitiesOnly yes` and `IdentityFile` pointing at the key's `.pub` file — safe to keep in the container, since it's public. The agent then offers only the matching private key for that host:

```text
Host github.com
    IdentitiesOnly yes
    IdentityFile ~/.ssh/id_ed25519.pub
```
