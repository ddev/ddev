# In-Container Home Directory and Shell Configuration

Custom shell configuration (Bash or your preferred shell), your usual Git configuration, a Composer `auth.json` and more can be achieved within your containers.

## Using `homeadditions` to Customize In-Container Home Directory

!!!tip "Finding Your Global DDEV Directory"
    The examples below use `$HOME/.ddev/homeadditions` which is the default location. If you have `$XDG_CONFIG_HOME` set, your global directory will be at `$XDG_CONFIG_HOME/ddev` instead. To find your actual location, run:

    ```bash
    ddev version | grep global-ddev-dir
    ```

    Then use that path with `/homeadditions` appended. See [global configuration directory](../usage/architecture.md#global-files) for details.

Place all your dotfiles in your global `$HOME/.ddev/homeadditions` or your project's `.ddev/homeadditions` directory and DDEV will use these in your project's `web` containers.

!!!tip "Ignore `.ddev/.homeadditions`!"
    A hidden/transient `.ddev/.homeadditions`—emphasis on the leading `.`—is used for processing global `homeadditions` and should be ignored.

On [`ddev start`](../usage/commands.md#start), DDEV attempts to create a user inside the `web` and `db` containers with the same name and user ID as the one you have on the host machine.

DDEV looks for the `homeadditions` directory both in the global `$HOME/.ddev/homeadditions` directory and the project-level `.ddev/homeadditions` directory, and will copy their contents recursively into the in-container home directory during `ddev start`. Project `homeadditions` contents override the global `homeadditions`.

Usage examples:

### Git Configuration

If you use Git inside the container, you may want to symlink your `$HOME/.gitconfig` into `$HOME/.ddev/homeadditions` or the project's `.ddev/homeadditions` so that in-container `git` commands use whatever username and email you've configured on your host machine.

```bash
ln -s $HOME/.gitconfig $HOME/.ddev/homeadditions/.gitconfig
```

### SSH Configuration

If you use SSH inside the container and want to use your `.ssh/config`, you can symlink it into the homeadditions directory. Some people will be able to symlink their entire `.ssh` directory.

```bash
mkdir -p $HOME/.ddev/homeadditions/.ssh
ln -s $HOME/.ssh/config $HOME/.ddev/homeadditions/.ssh/config
```

Or symlink the entire directory:

```bash
ln -s $HOME/.ssh $HOME/.ddev/homeadditions/.ssh
```

If you provide your own `.ssh/config` though, please make sure it includes these lines:

```text
UserKnownHostsFile=/home/.ssh-agent/known_hosts
StrictHostKeyChecking=accept-new
```

### Custom Scripts and Executables

If you need to add a script or other executable component into the project (or global configuration), you can put it in the project or global `.ddev/homeadditions/bin` directory and `$HOME/bin/<script>` will be created inside the container. This is useful for adding a script to one project or every project, or for overriding standard scripts, as `$HOME/bin` is first in the `$PATH` in the `web` container.

For example, to add a custom script:

```bash
# Create the bin directory
mkdir -p $HOME/.ddev/homeadditions/bin
# Add your script
echo '#!/usr/bin/env bash' > $HOME/.ddev/homeadditions/bin/myscript
echo 'echo "Hello from custom script"' >> $HOME/.ddev/homeadditions/bin/myscript
chmod +x $HOME/.ddev/homeadditions/bin/myscript
```

### Composer Authentication

If you use private, password-protected Composer repositories with [Satis](https://composer.github.io/satis/), for example, and use a global `auth.json`, you can symlink it into `homeadditions`. Be careful to exclude it from getting checked in by using a `.gitignore` or equivalent.

```bash
mkdir -p $HOME/.ddev/homeadditions/.composer
ln -s $HOME/.composer/auth.json $HOME/.ddev/homeadditions/.composer/auth.json
```

### Startup Scripts

You can add small scripts to the `.bashrc.d` directory, and they will be executed on [`ddev ssh`](../usage/commands.md#ssh).

For example, create a script that shows which container you're in:

```bash
# Create the .bashrc.d directory
mkdir -p $HOME/.ddev/homeadditions/.bashrc.d

# Add a script that runs on ddev ssh
echo 'echo "I am in the $(hostname) container"' > $HOME/.ddev/homeadditions/.bashrc.d/whereami
```

After `ddev restart`, when you `ddev ssh` this script will be executed.

### Custom Bashrc

If you have a favorite `.bashrc`, copy it into either the global or project `homeadditions`:

```bash
cp $HOME/.bashrc $HOME/.ddev/homeadditions/.bashrc
```

### Bash Aliases

If you like the traditional `ll` Bash alias for `ls -lhA`, add a `.bash_aliases` file to either the global or project `homeadditions`:

```bash
echo 'alias ll="ls -lhA"' > $HOME/.ddev/homeadditions/.bash_aliases
```

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

!!!tip
    See related `x-ddev.describe-*` configuration for [Customizing `ddev describe` Output](../extend/custom-docker-services.md#customizing-ddev-describe-output).

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
