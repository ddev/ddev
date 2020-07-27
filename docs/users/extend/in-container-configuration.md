## In-Container home directory configuration

Custom shell config (bash or your preferred shell), your usual git configuration, a composer auth.json and more can be achieved within your containers.  Place all your dotfiles in your `~/.ddev/homeadditions` directory and DDev will use these in all your projects' Docker containers.

On `ddev start`, ddev attempts to create a user inside the web and db containers with the same name and use id as the one you have on the host.

DDev looks for `homeadditions` directory either in `~/.ddev/homeadditions` (the global .ddev directory) or the `.ddev/homeadditions` directory of a particular project, and will copy their contents recursively into the in-container home directory during `ddev start`. (Note that project homeadditions contents override the global homeadditions.)

If you want to use a shell other than bash, add it as a package to [customize the Docker image](https://ddev.readthedocs.io/en/stable/users/extend/customizing-images/).

Usage examples:

* If you make git commits inside the container, you may want to copy your ~/.gitconfig into ~/.ddev/homeadditions or the project's .ddev/homeadditions so that use of git inside the container will use your regular username and email, etc.
* If you use private password-protected composer repositories with satis, for example, and use a global auth.json, you might want to `cp ~/.composer/auth.json into .ddev/homeadditions/.composer/auth.json`, but be careful that you exclude it from checking using a .gitignore or equivalent.
* Some people have specific configuration needs for their .ssh/config. If you provide your own .ssh/config though, please make sure it includes these lines:

    ```
    UserKnownHostsFile=/home/.ssh-agent/known_hosts
    StrictHostKeyChecking=no
    ```

* If you have a favorite .bashrc, copy it in to either the global or project homeadditions.

* If you like the traditional `ll` bash alias for `ls -l`, add a .ddev/homeadditions/.bash_aliases with these contents:

    ```
    alias ll="ls -lhA"
    ```

Caveats:

* Absolute symlinks inside a homeadditions directory won't work because they can't be resolved inside the container.
