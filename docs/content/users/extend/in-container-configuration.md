# In-Container Home Directory and Shell Configuration

Custom shell configuration (Bash or your preferred shell), your usual Git configuration, a Composer `auth.json` and more can be achieved within your containers.

Place all your dotfiles in your global`~/.ddev/homeadditions` or your project’s `.ddev/homeadditions` directory and DDEV will use these in your project’s `web` containers.

!!!tip "Ignore `.ddev/.homeadditions`!"
    A hidden/transient `.ddev/.homeadditions`—emphasis on the leading `.`—is used for processing global `homeadditions` and should be ignored.

On [`ddev start`](../usage/commands.md#start), DDEV attempts to create a user inside the `web` and `db` containers with the same name and user ID as the one you have on the host machine.

DDEV looks for the `homeadditions` directory both in the global `~/.ddev/homeadditions` directory and the project-level `.ddev/homeadditions` directory, and will copy their contents recursively into the in-container home directory during `ddev start`. Project `homeadditions` contents override the global `homeadditions`.

Usage examples:

* If you use Git inside the container, you may want to symlink your `~/.gitconfig` into `~/.ddev/homeadditions` or the project’s `.ddev/homeadditions` so that in-container `git` commands use whatever username and email you’ve configured on your host machine. For example, `ln -s ~/.gitconfig ~/.ddev/homeadditions/.gitconfig`.
* If you use SSH inside the container and want to use your `.ssh/config`, consider `mkdir -p ~/.ddev/homeadditions/.ssh && ln -s ~/.ssh/config ~/.ddev/homeadditions/.ssh/config`. Some people will be able to symlink their entire `.ssh` directory, `ln -s ~/.ssh ~/.ddev/homeadditions/.ssh`. If you provide your own `.ssh/config` though, please make sure it includes these lines:

    ```
    UserKnownHostsFile=/home/.ssh-agent/known_hosts
    StrictHostKeyChecking=no
    ```

* If you need to add a script or other executable component into the project (or global configuration), you can put it in the project or global `.ddev/homeadditions/bin` directory and `~/bin/<script` will be created inside the container. This is useful for adding a script to one project or every project, or for overriding standard scripts, as `~/bin` is first in the `$PATH` in the `web` container.
* If you use private, password-protected Composer repositories with [Satis](https://composer.github.io/satis/), for example, and use a global `auth.json`, you might want to `mkdir -p ~/.ddev/homeadditions/.composer && ln -s ~/.composer/auth.json ~/.ddev/homeadditions/.composer/auth.json`, but be careful that you exclude it from getting checked in by using a `.gitignore` or equivalent.
* You can add small scripts to the `.bashrc.d` directory and they will be executed on [`ddev ssh`](../usage/commands.md#ssh). For example, add a `~/.ddev/homeadditions/.bashrc.d/whereami` containing `echo "I am in the $(hostname) container"` and (after `ddev restart`) when you `ddev ssh` that will be executed.
* If you have a favorite `.bashrc`, copy it into either the global or project `homeadditions`.
* If you like the traditional `ll` Bash alias for `ls -l`, add a `.ddev/homeadditions/.bash_aliases` with these contents:

    ```
    alias ll="ls -lhA"
    ```
