<h1> In-Container home directory configuration</h1>

There are a number of reasons that your web container home directory may need custom configuration. You may want your normal git configuration in there, or a composer auth.json, etc.

On `ddev start`, ddev attempts to create a user inside the web and db containers with the same name and use id as the one you have on the host. 

If your project has a .ddev/homeadditions directory, its contents will be copied recursively into the in-container home directory during `ddev start`. 

Usage examples:

* If you make git commits inside the container, you may want to copy your ~/.gitconfig into .ddev/homeadditions so that use of git inside the container will use your regular username and email, etc.
* If you use private password-protected composer repositories with satis, for example, and use a global auth.json, you might want to `cp ~/.composer/auth.json into .ddev/homeadditions/.composer/auth.json`, but be careful that you exclude it from checking using a .gitignore or equivalent.
* Some people have specific configuration needs for their .ssh/config. If you provide your own .ddev/homeadditions/.ssh/config though, please make sure it includes these lines:
    ```
    UserKnownHostsFile=/home/.ssh-agent/known_hosts
    StrictHostKeyChecking=no
    ```
* If you have a favorite .bashrc, copy it in.

* If you like the traditional `ll` bash alias for `ls -l`, add a .ddev/homeadditions/.bash_aliases with these contents:
    ```
    alias ll="ls -lhA" 
    ```

Caveats:

* Absolute symlinks inside .ddev/homeadditions won't work because they can't be resolved inside the container.
