# In-Container home directory and shell configuration

Custom shell configuration (bash or your preferred shell), your usual git configuration, a composer auth.json and more can be achieved within your containers.  Place all your dotfiles in your global`~/.ddev/homeadditions` or your project's `.ddev/homeadditions` directory and DDEV will use these in your project's web containers.  (Note that there is also a hidden/transient `.ddev/.homeadditions`; this is used for processing global homeadditions and should be ignored.)

On `ddev start`, ddev attempts to create a user inside the web and db containers with the same name and use id as the one you have on the host.

DDEV looks for the `homeadditions` directory either in `~/.ddev/homeadditions` (the global .ddev directory) or the `.ddev/homeadditions` directory of a particular project, and will copy their contents recursively into the in-container home directory during `ddev start`. (Note that project homeadditions contents override the global homeadditions.)

Usage examples:

* If you use git inside the container, you may want to symlink your `~/.gitconfig` into `~/.ddev/homeadditions` or the project's `.ddev/homeadditions` so that use of git inside the container will use your regular username and email, etc. For example, `ln -s ~/.gitconfig ~/.ddev/homeadditions/.gitconfig`.
* If you use ssh inside the container and want to use your `.ssh/config`, consider `mkdir -p ~/.ddev/homeadditions/.ssh && ln -s ~/.ssh/config ~/.ddev/homeadditions/.ssh/config`. Some people will be able to symlink their entire `.ssh` directory, `ln -s ~/.ssh ~/.ddev/homeadditions/ssh`. If you provide your own .ssh/config though, please make sure it includes these lines:

    ```
    UserKnownHostsFile=/home/.ssh-agent/known_hosts
    StrictHostKeyChecking=no
    ```
  
* If you need to add a script or other executable component into the project (or global configuration), you can put it in the project or global `.ddev/homeadditions/bin` directory and `~/bin/<script` will be created inside the container. This is useful for adding a script to a project or to every project, or for overriding standard scripts, as ~/bin is first in the $PATH in the web container.
* If you use private password-protected composer repositories with satis, for example, and use a global auth.json, you might want to `mkdir -p ~/.ddev/homeadditions/.composer && ln -s ~/.composer/auth.json ~/.ddev/homeadditions/.composer/auth.json`, but be careful that you exclude it from getting checked in by using a .gitignore or equivalent.

* You can add small scripts to the `.bashrc.d` directory and they will be executed on `ddev ssh`. For example, add a `~/.ddev/homeadditions/.bashrc.d/whereami` containing `echo "I am in the $(hostname) container"` and (after `ddev restart`) when you `ddev ssh` that will be executed.
* If you have a favorite .bashrc, copy it in to either the global or project homeadditions.

* If you like the traditional `ll` bash alias for `ls -l`, add a .ddev/homeadditions/.bash_aliases with these contents:

    ```
    alias ll="ls -lhA"
    ```
