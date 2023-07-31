# Shell Completion & Autocomplete

Most people like to have shell completion on the command line. In other words, when you're typing a command, you can hit `<TAB>` and the shell will show you what the options are. For example, if you type `ddev <TAB>`, you'll see all the possible commands. `ddev debug <TAB>` will show you the options for the command. And `ddev list -<TAB>` will show you all the flags available for [`ddev list`](../usage/commands.md#list).

Shells like Bash and zsh need help to do this though, they have to know what the options are. DDEV provides the necessary hint scripts, and if you use Homebrew, they get installed automatically. But if you use oh-my-zsh, for example, you may have to manually install the hint script.

=== "macOS Bash + Homebrew"

    ## macOS Bash + Homebrew

    The easiest way to use Bash completion on macOS is install it with Homebrew. `brew install bash-completion`. When you install it though, it will warn you with something like this, which **may vary on your system**.

    ```
    Add the following line to your ~/.bash_profile:
    [[ -r "$(brew --prefix)/etc/profile.d/bash_completion.sh" ]] && . "$(brew --prefix)/etc/profile.d/bash_completion.sh"
    ```

    !!!note "Bash profile"
        You must add the include to your `.bash_profile` or `.profile` or nothing will work. Use `source ~/.bash_profile` or `source ~/.profile` to make it take effect immediately.

    * Link completions with `brew completions link`.

    When you install DDEV via Homebrew, each new release will automatically get a refreshed completions script.

=== "Bash/Zsh/Fish on Linux"

    ## Bash/Zsh/Fish on Linux

    On Debian and Yum based systems, using `apt install ddev` you should find that `bash`, `zsh`, and `fish` completions are automatically installed.

    Manual installation is easy though, the completion script is exactly the same, it’s just that you have to download and install it yourself. Each system may have a slightly different technique, and you’ll need to figure it out. On Debian/Ubuntu, manually install like this:

    1. Download the completion files and extract them with
        ```bash
        VERSION=v1.21.1
        curl -sSLf https://github.com/ddev/ddev/releases/download/${VERSION}/ddev_shell_completion_scripts.${VERSION}.tar.gz
        tar -zxf ddev_shell_completion_scripts.${VERSION}.tar.gz
        ```
    2. Run `sudo mkdir -p /usr/share/bash-completion/completions && sudo cp ddev_bash_completion.sh /usr/share/bash-completion/completions/ddev`. This deploys the `ddev_bash_completion.sh` script where it needs to be. Again, every Linux distro has a different technique, and you may have to figure yours out.

    If you installed DDEV using `apt install` then the `ddev_bash_completion.sh` file is already available in `/usr/bin/ddev_bash_completion.sh`. Starting with DDEV v1.21.2 this will be automatically installed into `/usr/share/bash-completion/completions`.

=== "Oh-My-Zsh"

    ## Oh-My-Zsh

    If you installed zsh with Homebrew, DDEV’s completions will be automatically installed when you `brew install ddev/ddev/ddev`.

    Otherwise, Oh-My-Zsh may be set up very differently in different places, so as a power `zsh` user you’ll need to put `ddev_bash_completion.sh` (see tar archive download above) where it belongs. `echo $fpath` will show you the places that it’s most likely to belong. An obvious choice is `~/.oh-my-zsh/completions`; if that exists, so you can run `mkdir -p ~/.oh-my-zsh/completions && cp ddev_zsh_completion.sh ~/.oh-my-zsh/completions/_ddev`, then `autoload -Uz compinit && compinit`.

=== "Fish"

    ## Fish

    The `fish` shell’s completions are also supported and are automatically installed into `/usr/local/share/fish/vendor_completions.d/` when you install ddev via Homebrew. If you have installed `fish` without Homebrew, you can extract the fish completions from the `ddev_shell_completion_scripts` tarball that is included with each release.

=== "Git Bash"

    ## Git Bash

    Completions in Git Bash are sourced from at least `~/bash_completion.d` so you can use `mkdir -p ~/bash_completion.d && tar -C ~/.bash_completion.d -zxf /z/Downloads/ddev_shell_completion_scripts.v1.15.0-rc3.tar.gz ddev_bash_completion.sh && mv ~/bash_completion.d/ddev_bash_completion.sh ~/bash_completion.d/ddev.bash` to extract the Bash completions and put them where they belong.

=== "PowerShell"

    ## PowerShell

    PowerShell completions are also provided in the `ddev_shell_completions tarball` included with each release. You can run the `ddev_powershell_completion.ps1` script manually or install it so it will be run whenever PS is opened using the technique at [Run PowerShell Script When You Open PowerShell](https://superuser.com/questions/886951/run-powershell-script-when-you-open-powershell).

## tar Archive of Completion Scripts for Manual Deployment

Although most people will use techniques like Homebrew for installation, a tar archive of the shell completion scripts is available in each release, called `ddev_shell_completion_scripts.<version>.tar.gz`. If you need to manually install, you can download and untar the scripts, then copy them as needed to where they have to go. For example, `sudo cp ddev_bash_completion.sh /etc/bash_completion.d/ddev`.

Note that scripts for the `fish` shell and Windows PowerShell are also provided, but no instructions are given here for deploying them.
