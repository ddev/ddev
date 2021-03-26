## Shell Completion

Most people like to have shell completion on the command line. In other words, when you're typing a command, you can hit `<TAB>` and the shell will show you what the options are. For example, if you type `ddev <TAB>`, you'll see all the possible commands. `ddev debug <TAB>` will show you the options for the command. And `ddev list -<TAB>` will show you all the flags available for `ddev list`.

Shells like bash and zsh need help to do this though, they have to know what the options are. DDEV-Local provides the necessary hint scripts, and if you use homebrew, they get installed automatically. But if you use oh-my-zsh, for example, you may have to manually install the hint script.

### tar Archive of Completion Scripts for Manual Deployment

Although most people will use techniques like homebrew for installation, a tar archive of the shell completion scripts is available in each release, called "ddev_shell_completion_scripts.\<version\>.tar.gz". If you need to manually install, you can download and untar the scripts, then copy them as needed to where they have to go. For example, `sudo cp ddev_bash_completion.sh /etc/bash_completion.d/ddev`.

Note that scripts for the fish shell and Windows PowerShell are also provided, but no instructions are given here for deploying them.

## Bash Completion with Homebrew

**Bash Completion with Homebrew**: The easiest way to use bash completion on either macOS or Linux is to install with homebrew. `brew install bash-completion`. When you install it though, it will warn you with something like this, which may vary on your system.

```bash
Add the following line to your ~/.bash_profile:
     [[ -r "/usr/local/etc/profile.d/bash_completion.sh" ]] && . "/usr/local/etc/profile.d/bash_completion.sh"
```

* Link completions with `brew completions link`.
* *You need to add the suggested line to your ~/.bash_profile or ~/.profile to get it to work*, and then in the current shell you need to `source ~/.bash_profile` or `source ~/.profile` to make it take effect. (You can also just open a new shell window.)

Then, if you're installing ddev from homebrew, each new release will automatically get a refreshed completions script.

**Bash Completion without Homebrew**: The completion script is exactly the same, it's just that you have to install it yourself. Each system may have a slightly different technique, and you'll need to figure it out. On Debian/Ubuntu, you would use [these instructions](http://crsouza.com/2008/07/28/enabling-bash-autocompletion-on-debian/) to enable bash-completion, and then `sudo mkdir -p /etc/bash_completion.d && sudo cp ddev_bash_completion.sh /etc/bash_completion.d`. This deploys the ddev_bash_completion.sh script where it needs to be. Again, every Linux distro has a different technique, and you may have to figure yours out.

### Zsh Completion

**Zsh Completion with Homebrew**: This works exactly the same as bash completion. `brew install zsh-completions`. You'll get instructions something like this:

```bash
  if type brew &>/dev/null; then
    FPATH=$(brew --prefix)/share/zsh-completions:$FPATH

    autoload -Uz compinit
    compinit
  fi

You may also need to force rebuild `zcompdump`:

  rm -f ~/.zcompdump; compinit

Additionally, if you receive "zsh compinit: insecure directories" warnings when attempting
to load these completions, you may need to run this:

  chmod go-w '/usr/local/share'
```

So follow those instructions and your zsh should be set up.

### Oh-My-Zsh Completion

If you installed zsh with homebrew, ddev's completions will be automatically installed when you `brew install drud/ddev/ddev`.

Otherwise, Oh-My-Zsh may be set up very differently in different places, so as a power zsh user you'll need to put ddev_bash_completion.sh (see tar archive download above) where it belongs. `echo $fpath` will show you the places that it's most likely to belong. An obvious choice is ~/.oh-my-zsh/completions if that exists, so you can `mkdir -p ~/.oh-my-zsh/completions && cp ddev_zsh_completion.sh ~/.oh-my-zsh/completions/_ddev` and then `autoload -Uz compinit && compinit`.

### Fish Completion

The fish shell's completions are also supported and are automatically installed into /usr/local/share/fish/vendor_completions.d when you install ddev via Homebrew.  If you have installed fish without homebrew, you can extract the fish completions from the ddev_shell_completion_scripts tarball that is included with each release.

### Git-bash Completion

Completions in git-bash are sourced from at least ~/bash_completion.d so you can use `mkdir -p ~/bash_completion.d && tar -C ~/.bash_completion.d -zxf /z/Downloads/ddev_shell_completion_scripts.v1.15.0-rc3.tar.gz ddev_bash_completion.sh && mv ~/bash_completion.d/ddev_bash_completion.sh ~/bash_completion.d/ddev.bash` to extract the bash completions and put them where they belong.

### PowerShell Completion

PowerShell completions are also provided in the ddev_shell_completions tarball included with each release. You can run the ddev_powershell_completion.ps1 script manually or install it so it will be run whenever PS is opened using the technique at [Run PowerShell Script When You Open PowerShell](https://superuser.com/questions/886951/run-powershell-script-when-you-open-powershell)
