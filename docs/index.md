# Intro to DDEV-Local

[ddev](https://github.com/drud/ddev) is an open source tool that makes it dead simple to get local PHP development environments up and running within minutes. It's powerful and flexible as a result of its per-project environment configurations, which can be extended, version controlled, and shared. In short, ddev aims to allow development teams to use Docker in their workflow without the complexities of bespoke configuration.

## System Requirements

* [Docker](https://www.docker.com/products/docker-desktop) version 18.06 or higher. Linux users make sure you upgrade docker-compose and do the [post-install steps](https://docs.docker.com/install/linux/linux-postinstall/#manage-docker-as-a-non-root-user)

* docker-compose 1.21.0 and higher (bundled with Docker in Docker Desktop for Mac and Docker Desktop for Windows)
* OS Support
    * macOS Sierra and higher (macOS 10.12 and higher; it should run anywhere Docker Desktop for Mac runs.
    * Linux: Most Linux distributions which can run Docker-ce are fine. This includes at least Ubuntu 16.04+, Debian Jessie+, Fedora 25+. Make sure to follow the docker-ce [post-install steps](https://docs.docker.com/install/linux/linux-postinstall/#manage-docker-as-a-non-root-user)
    * Windows 10 (all editions) with WSL2 (version 2004 or later)
    * (Non-WSL2) Windows 10 Home, Pro, or Enterprise with [Docker Desktop for Windows](https://docs.docker.com/docker-for-windows/install/)

### Using ddev alongside other development environments

ddev by default uses ports 80 and 443 on your system when projects are running. If you are using another local development environment you can either stop the other environment or configure ddev to use different ports. See [troubleshooting](users/troubleshooting.md#unable-listen) for more detailed problem solving.

## Installation

_When upgrading, please run `ddev poweroff` and check the [release notes](https://github.com/drud/ddev/releases) for actions you might need to take on each project._

### Docker Installation

Docker and docker-compose are required before anything will work with ddev. This is pretty easy on most environments, but see the [docker_installation](users/docker_installation.md) page to help sort out the details, especially on Windows and Linux.

### Homebrew/Linuxbrew - macOS/Linux

For macOS and Linux users, we recommend installing and upgrading via [homebrew](https://brew.sh/) (macOS) or [Homebrew on Linux](https://docs.brew.sh/Homebrew-on-Linux) (Linux):

```
brew tap drud/ddev && brew install ddev
```

If you would like more frequent "edge" releases then use `brew tap drud/ddev-edge` instead.

(Optional) As a one-time initialization, run `mkcert -install`. Linux users may have to take additional actions as discussed below in "Linux `mkcert -install` additional instructions".

Later, to upgrade to a newer version of ddev, run:

```
ddev poweroff && brew upgrade ddev
```

### Installation or Upgrade - Windows (WSL2)

**This is the recommended installation method for all Windows users that are on Windows 10 2004 or higher.** If you don't have this version yet, or if you don't want to use WSL2, please follow the legacy instructions for Windows below.

**All Windows 10 editions (including Windows 10 Home) support WSL2**. Docker Toolbox support for DDEV is deprecated and will be removed, as we'll move testing capacity towards WSL2. If you're already familiar with DDEV on Windows, you might have been using NFS for better filesystem performance. **You won't need NFS anymore once you switch to WSL2**, since it provides awesome filesystem performance out of the box 🚀

Take the steps below to install DDEV into WSL2.

* Install WSL2 by [following Microsoft's docs](https://docs.microsoft.com/en-us/windows/wsl/install-win10). Make sure the WSL version is set to 2: `wsl --set-default-version 2`
* Install a WSL2 distro, e.g. [Ubuntu 20.04](https://www.microsoft.com/store/productId/9N6SVWS3RX71). Install Ubuntu (or any other distro) _before_ installing Docker, so that Ubuntu becomes the default WSL distro.
* Install Docker Desktop for Windows (versions 2.3.0.2 and higher automatically use the WSL2 backend): [download Docker Desktop for Windows from Docker Hub](https://hub.docker.com/editions/community/docker-ce-desktop-windows/).
* Go to Docker Desktop settings > Resources > WSL integration > enable integration for your distro (now `docker` commands will be available from within your WSL2 distro).
* Install [Chocolatey](https://chocolatey.org/install) on Windows.
* Open a git-bash or PowerShell terminal with administrator rights and install mkcert by running `choco install mkcert`
* Run `mkcert -install`. Your certificates are installed in `%localappdata%\mkcert`, we'll need these later on.
* Open the Ubuntu 20.04 terminal from the Windows start menu.
* Run `export CAROOT=/mnt/c/Users/YOUR_WINDOWS_USERNAME/AppData/Local/mkcert`. This will make sure that `mkcert` on Linux uses your Windows certificates when issuing SSL certificates, so you don't get SSL warnings in your browser on Windows.
* Run `echo 'export CAROOT=/mnt/c/Users/YOUR_WINDOWS_USERNAME/AppData/Local/mkcert' >> ~/.profile` to store this in your profile for future sessions.
* Now we'll install DDEV **for Linux** within WSL2. Open your WSL2 distro and follow the instructions for Linux below, **then come back here**.
* After installation, run `mkcert -install` and you'll notice that mkcert will use your Windows CA certificates 🚀:
  > Using the local CA at "/mnt/c/Users/YOUR_WINDOWS_USERNAME/AppData/Local/mkcert"
* That's it! You have now installed DDEV on WSL2 🎉 Remember to run all `ddev` commands in your Ubuntu/WSL2 terminal, **not** in PowerShell/Command Prompt.

**Make sure you put your projects in the Linux filesystem (e.g. /home/LINUX_USERNAME), _not_ in the Windows filesystem (/mnt/c), because you'll get vastly better performance on the Linux filesystem.**

### Installation or Upgrade - Windows (legacy)

* A windows installer is provided in each [ddev release](https://github.com/drud/ddev/releases) (`ddev_windows_installer.<version>.exe`). Run that and it will do the full installation for you.  Open a new terminal or cmd window and start using ddev.
* If you use [chocolatey](https://chocolatey.org/) (highly recommended), then you can just `choco install ddev` from an administrative-privileged shell. Upgrades are just `ddev poweroff && choco upgrade ddev`.
* As a one-time initialization, run `mkcert -install`
* Most people interact with ddev on Windows using git-bash, part of the [Windows git suite](https://git-scm.com/download/win). Although ddev does work with cmd and PowerShell, it's more at home in bash. You can install it with chocolatey using `choco install -y git`.

### Installation/Upgrade Script - Linux and macOS

Linux and macOS end-users can use this line of code to your terminal to download, verify, and install (or upgrade) ddev using our [install script](https://github.com/drud/ddev/blob/master/scripts/install_ddev.sh):

```
curl -L https://raw.githubusercontent.com/drud/ddev/master/scripts/install_ddev.sh | bash
```

* As a one-time initialization, run `mkcert -install`, which may require your sudo password. Linux users may have to take additional actions as discussed below in "Linux `mkcert -install` additional instructions".

Later, to upgrade ddev to the latest version, just run `ddev poweroff` and run the script again.

### Manual Installation or Upgrade - Linux and macOS

You can also easily perform the installation or upgrade manually if preferred. ddev is just a single executable, no special installation is actually required, so for all operating systems, the installation is just copying ddev into place where it's in the system path.

* `ddev poweroff` if upgrading
* Download and extract the latest [ddev release](https://github.com/drud/ddev/releases) for your architecture.
* Move ddev to /usr/local/bin: `mv ddev /usr/local/bin/` (may require sudo), or another directory in your `$PATH` as preferred.
* Run `ddev` to test your installation. You should see ddev's command usage output.
* (Optional) As a one-time initialization, run `mkcert -install`, which may require your sudo password. Linux users may have to take additional actions as discussed below in "Linux `mkcert -install` additional instructions

### Installation via package managers - Linux

The preferred Linux package manager is [Linuxbrew](http://linuxbrew.sh/) : `brew tap drud/ddev && brew install ddev`

We also currently maintain a package on [Arch Linux (AUR)](https://aur.archlinux.org/packages/ddev-bin/)

(Optional) As a one-time initialization, run `mkcert -install`, which may require your sudo password. See below for additional information.

### Linux `mkcert -install` additional instructions

The `mkcert -install` step on Linux may provide you with additional instructions.

On variants of Linux you may be prompted for additional package installation to get certutil installed, but you can follow the instructions given by mkcert:

  > $ mkcert -install
  > Created a new local CA at "/home/username/.local/share/mkcert" 
  > Installing to the system store is not yet supported on this Linux  but Firefox and/or Chrome/Chromium will still work.
  > You can also manually install the root certificate at "/home/username/.local/share/mkcert/rootCA.pem".
  > Warning: "certutil" is not available, so the CA can't be automatically installed in Firefox and/or Chrome/Chromium! ⚠️
  > Install "certutil" with "apt install libnss3-tools" or "yum install nss-tools" and re-run "mkcert -install" 
  
  (Note the prompt `Installing to the system store is not yet supported on this Linux`, which can be a simple result of not having /usr/sbin in the path so that `/usr/sbin/update-ca-certificates` can be found.)

### Windows and Firefox `mkcert -install` additional instructions

The `mkcert -install` step on Windows does not work for the Firefox browser.
You need to add the created root certificate authority to the security
configuration by your self:

* Run `mkcert -install` (you can use the shortcut from the start menu for that)
* Run `mkcert -CAROOT` to see the local folder used for the newly created root
  certificate authority
* Open the Firefox settings
* Enter `certificates` into the search box on the top
* Click to `Show certificates...`
* Select the tab `Certificate authorities`
* Click to `Import...`
* Go to the folder where your root certificate authority was stored
* Select the file `rootCA.pem`
* Click to `Open`

You should now see your CA under `mkcert development CA`.

### Uninstallation

For instructions to uninstall DDEV-Local see [Uninstallation](users/uninstall.md).

<a name="support"></a>

## Support and User-Contributed Documentation

We love to hear from our users and help them be successful with ddev. Support options include:

* [ddev Documentation](users/faq.md)
* [ddev StackOverflow](https://stackoverflow.com/questions/tagged/ddev) for support and frequently asked questions. We respond quite quickly here and the results provide quite a library of user-curated solutions.
* [ddev issue queue](https://github.com/drud/ddev/issues) for bugs and feature requests
* The [gitter drud/ddev channel](https://gitter.im/drud/ddev) (it's easy to log in many different ways)
* The `#ddev` channels in [Drupal Slack](https://www.drupal.org/slack), [TYPO3 Slack](https://my.typo3.org/index.php?id=35) for interactive, immediate community support.
* [ddev-contrib](https://github.com/drud/ddev-contrib) repo provides a number of vetted user-contributed recipes for extending and using ddev. Your contributions are welcome.
* [awesome-ddev](https://github.com/drud/awesome-ddev) repo has loads of external resources, blog posts, recipes, screencasts, and the like. Your contributions are welcome.
* [Twitter with tag #ddev](https://twitter.com/search?q=%23ddev&src=typd&f=live) will get to us, but it's not as good for interactive support, but we'll answer anywhere.
