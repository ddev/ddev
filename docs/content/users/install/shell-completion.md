# Shell Completion & Autocomplete

Most people like to have shell completion on the command line. In other words, when you're typing a command, you can hit `<TAB>` and the shell will show you what the options are. For example, if you type `ddev <TAB>`, you'll see all the possible commands. `ddev debug <TAB>` will show you the options for the command. And `ddev list -<TAB>` will show you all the flags available for [`ddev list`](../usage/commands.md#list).

Shells like Bash and Zsh need help to do this though, they have to know what the options are. DDEV provides the necessary hint scripts, and if you use Homebrew, they get installed automatically.

## macOS Bash with Homebrew

The easiest way to use Bash completion on macOS is install it with Homebrew ([docs](https://docs.brew.sh/Shell-Completion#configuring-completions-in-bash)). `brew install bash-completion`. When you install it though, it will warn you with something like this, which **may vary on your system**. Add the following line to your `~/.bash_profile` file (or if that doesn't exist, to your `~/.profile`:

```
[[ -r "$(brew --prefix)/etc/profile.d/bash_completion.sh" ]] && . "$(brew --prefix)/etc/profile.d/bash_completion.sh"
```

!!!note "Bash profile"
    You must add the include to your `.bash_profile` or `.profile` or nothing will work. Use `source ~/.bash_profile` or `source ~/.profile` to make it take effect immediately in your current terminal window.

Link completions by running `brew completions link`.

When you install DDEV via Homebrew, each new release will automatically get a refreshed completions script.

## macOS Zsh with Homebrew

To make Homebrew completions available in Zsh the Homebrew-managed path `zsh/site-functions` has to be added to the FPATH variable.

=== "macOS Zsh"

    Add the following block to your `~/.zshrc` file (see [docs](https://docs.brew.sh/Shell-Completion#configuring-completions-in-zsh)):

    ```bash
    if type brew &>/dev/null
    then
      FPATH="$(brew --prefix)/share/zsh/site-functions:${FPATH}"
      autoload -Uz compinit
      compinit
    fi
    ```

    Note that the updating of the FPATH variable has to happen before the Zsh completion index is initialized with `compinit`.

=== "macOS Zsh with Oh My Zsh"

    Oh My Zsh is calling `compinit` for you when `oh-my-zsh.sh` is sourced (see [docs](https://docs.brew.sh/Shell-Completion#configuring-completions-in-zsh)). Instead of adding the block that was necessary for `macOS Zsh` place the following line right before the `oh-my-zsh.sh` file is sourced in your `~/.zshrc` file:

    ```bash
    FPATH="$(brew --prefix)/share/zsh/site-functions:${FPATH}"
    ```

To avoid any potential caching issue remove and rebuild the `.zcompdump` file, which is the index for Zsh completions:

```bash
rm -f ~/.zcompdump; compinit
```

In case you run into any `zsh compinit: insecure directories` warnings, run:

```bash
chmod -R go-w "$(brew --prefix)/share"
```

## macOS Fish with Homebrew

`fish` shell completions are automatically installed at `/usr/local/share/fish/vendor_completions.d/ddev_fish_completion.sh` when you install DDEV via Homebrew.

If you installed `fish` without Homebrew, you can extract the fish completions (`ddev_fish_completion.sh`) from the tar archive of completion scripts included with each release. See [below](#tar-archive-of-completion-scripts-for-manual-deployment).

## Bash/Zsh/Fish on Linux including WSL2

On Debian and Yum based systems, if you installed DDEV using `apt-get install ddev`, the `bash`, `zsh`, and `fish` completions should be automatically installed at `/usr/share/bash-completion/completions/ddev`, `/usr/share/zsh/vendor-completions/_ddev` and `/usr/share/fish/completions/ddev.fish` respectively, and the `bash` completions should be automatically installed at `/usr/share/bash-completion/completions/bash`.

Otherwise, you can download the completion files for manual installation as described [below](#tar-archive-of-completion-scripts-for-manual-deployment). Every Linux distro requires a different manual installation technique. On Debian/Ubuntu, you could deploy the `ddev_bash_completion.sh` script where it needs to be by running `sudo mkdir -p /usr/share/bash-completion/completions && sudo cp ddev_bash_completion.sh /usr/share/bash-completion/completions/ddev`.

## Git Bash

Git Bash completions (`ddev_bash_completion.sh`) are provided in the tar archive of completion scripts included with each release. See [below](#tar-archive-of-completion-scripts-for-manual-deployment).

Completions in Git Bash are sourced from at least the `~/bash_completion.d` directory. You can copy `ddev_bash_completion.sh` to that directory by running `mkdir -p ~/bash_completion.d && cp ddev_bash_completion.sh ~/bash_completion.d/ddev.bash`.

## PowerShell

PowerShell completions (`ddev_powershell_completion.ps1`) are provided in the tar archive of completion scripts included with each release. See [below](#tar-archive-of-completion-scripts-for-manual-deployment).

You can run the `ddev_powershell_completion.ps1` script manually or install it so it will be run whenever PS is opened using the technique described at [Run PowerShell Script When You Open PowerShell](https://superuser.com/questions/886951/run-powershell-script-when-you-open-powershell).

## tar Archive of Completion Scripts for Manual Deployment

Although most people will use techniques like Homebrew for installation, a tar archive of shell completion scripts for various shells is available in each release, called `ddev_shell_completion_scripts.<version>.tar.gz`. If you need to manually install, you can download the files and extract them with the following commands, replacing the VERSION number in the first line with your version:

```bash
VERSION=v1.22.6
curl -sSLf https://github.com/ddev/ddev/releases/download/${VERSION}/ddev_shell_completion_scripts.${VERSION}.tar.gz
tar -zxf ddev_shell_completion_scripts.${VERSION}.tar.gz
```

Alternatively, you could download the tar archive using a browser, from a URL such as the following, replacing the version numbers with your version: [https://github.com/ddev/ddev/releases/download/v1.22.6/ddev_shell_completion_scripts.v1.22.6.tar.gz](https://github.com/ddev/ddev/releases/download/v1.22.6/ddev_shell_completion_scripts.v1.22.6.tar.gz).

After extracting the archive, copy the appropriate completion script where you need it, for example by running `sudo cp ddev_bash_completion.sh /etc/bash_completion.d/ddev`. Detailed instructions for various shells are given above.
