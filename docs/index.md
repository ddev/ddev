# Intro to DDEV-Local

[DDEV](https://github.com/drud/ddev) is an open source tool that makes it dead simple to get local PHP development environments up and running within minutes. It's powerful and flexible as a result of its per-project environment configurations, which can be extended, version controlled, and shared. In short, DDEV aims to allow development teams to use Docker in their workflow without the complexities of bespoke configuration.

## System Requirements

* Colima (macOS), `docker-ce` (Linux/WSL2) or [Docker Desktop](https://www.docker.com/products/docker-desktop) or a related docker back-end are required. Installing or upgrading docker-compose is not required as DDEV uses its own private docker-compose version. See [Docker Installation](users/docker_installation.md).
* OS Support
    * macOS Catalina and higher (macOS 10.15 and higher); it should run anywhere docker runs (Current Docker Desktop has deprecated macOS 10.14 and below, but Docker Desktop versions prior to  can still work with DDEV-Local on High Sierra. You can look through the [Docker Desktop for Mac Release Notes](https://docs.docker.com/desktop/mac/release-notes/) for older versions. In addition, Colima supports older versions.)
    * Linux: Most Linux distributions which can run Docker-ce are fine. This includes at least Ubuntu 18.04+ (20.04 is recommended), Debian Jessie+, Fedora 25+. Make sure to follow the docker-ce [post-install steps](https://docs.docker.com/install/linux/linux-postinstall/#manage-docker-as-a-non-root-user)
    * Windows 10/11 (all editions) with WSL2 (version [1903.1049, 1909.1049](https://devblogs.microsoft.com/commandline/wsl-2-support-is-coming-to-windows-10-versions-1903-and-1909/), 2004 or later)
    * (Non-WSL2) Windows 10/11 Home, Pro, or Enterprise with [Docker Desktop for Windows](https://www.docker.com/products/docker-desktop)
* Architecture Support
    * AMD64 is supported on Windows 10/11 (with either traditional Windows or WSL2), macOS, and Linux.
    * ARM64 machines are currently supported on Linux and in WSL2 in Windows ARM64 computers.
    * Apple Silicon M1 (ARM64) is supported since DDEV v1.17.

## Using DDEV alongside other development environments

DDEV by default uses ports 80 and 443 on your system when projects are running. If you are using another local development environment you can either stop the other environment or configure DDEV to use different ports. See [troubleshooting](users/troubleshooting.md#unable-listen) for more detailed problem-solving.

## Installation

### Docker Desktop/CE or Colima Prerequisite

Docker or an alternative is required before anything will work with DDEV. This is pretty easy on most environments; see the [docker_installation](users/docker_installation.md) page to help sort out the details, especially on Windows and Linux. It is not required to install docker-compose because DDEV uses its own private version.

### macOS (Homebrew)

For macOS (both amd64 and arm64) users, we recommend installing and upgrading via [Homebrew](https://brew.sh/): `brew install drud/ddev/ddev`.

As a one-time initialization, run `mkcert -install`.

Later, to upgrade to a newer version of DDEV-Local, run `brew upgrade ddev`.

To install DDEV prereleases, subscribe to the "edge" channel with `brew install drud/ddev-edge/ddev` and to install the latest unreleased DDEV version, `brew unlink ddev && brew install drud/ddev/ddev --HEAD`.

### Linux (Arch-based systems such as `EndeavourOS` or `Manjaro`)

We maintain a package on [Arch Linux (`AUR`)](https://aur.archlinux.org/packages/ddev-bin/).

**NOTE: Package installation on Arch-based systems is preferable to the install script below.**

As a one-time initialization, run `mkcert -install`, which may require your sudo password. See below for additional information.

### Linux, macOS and Windows WSL2 (install script)

**NOTE: macOS users that have installed via Homebrew or Arch Linux users that have installed via the package manager above do not need the install script.**

Linux, macOS and Windows WSL2 (see below) users can use this line of code to your terminal to download, verify, and install (or upgrade) ddev using the [install_ddev.sh script](https://github.com/drud/ddev/blob/master/scripts/install_ddev.sh). Note that this works with both amd64 and arm64 architectures, including Surface Pro X with WSL2 and 64-bit Raspberry Pi OS. It also works with macOS Apple Silicon M1 machines.

```
curl -LO https://raw.githubusercontent.com/drud/ddev/master/scripts/install_ddev.sh && bash install_ddev.sh
```

The installation script can also take a version argument in order to install a specific version or a prerelease version. For example,

```
curl -LO https://raw.githubusercontent.com/drud/ddev/master/scripts/install_ddev.sh && bash install_ddev.sh v1.19.0-alpha5
```

To upgrade DDEV to the latest stable version, just run the script again.

### Windows (WSL2)

**This is the recommended installation method for all Windows users**.

**All Windows 10/11 editions (including Windows 10 Home) support WSL2**. If you're already familiar with DDEV on Windows, you might have been using NFS for better filesystem performance. **You won't need NFS anymore once you switch to WSL2**, since it provides awesome filesystem performance out of the box.

The WSL2 install process involves:

* Installing Chocolatey package manager (optional).
* One time initialization of mkcert.
* Installing WSL2 and installing a distro like Ubuntu.
* Installing or upgrading to the latest Docker Desktop for Windows with WSL2 enabled.
* Installing DDEV inside your distro.

We'll walk through these in more detail. You may prefer other techniques of installation or may not need some steps, but this is the full recipe:

1. If you have previously installed Docker Toolbox, please completely [uninstall Docker Toolbox](https://docs.docker.com/toolbox/toolbox_install_windows/#how-to-uninstall-toolbox).
2. **Chocolatey:** We recommend using [Chocolatey](https://chocolatey.org/install) for installing required Windows apps like mkcert. In an administrative PowerShell, `Set-ExecutionPolicy Bypass -Scope Process -Force; [System.Net.ServicePointManager]::SecurityProtocol = [System.Net.ServicePointManager]::SecurityProtocol -bor 3072; iex ((New-Object System.Net.WebClient).DownloadString('https://chocolatey.org/install.ps1'))`
3. In an administrative PowerShell: `choco install -y mkcert`
4. In an administrative PowerShell, run `mkcert -install` and answer the prompt allowing the installation of the Certificate Authority.
5. In an administrative PowerShell, run the command `setx CAROOT "$(mkcert -CAROOT)"; If ($Env:WSLENV -notlike "*CAROOT/up:*") { setx WSLENV "CAROOT/up:$Env:WSLENV" }`. This will set WSL2 to use the Certificate Authority installed on the Windows side.
6. In administrative PowerShell, run the command `wsl --install`. This will install WSL2 and Ubuntu for you. Reboot when this is done.
7. **Docker Desktop for Windows:** If you already have the latest Docker Desktop, configure it in the General Settings to use the WSL2-based engine. Otherwise install the latest Docker Desktop for Windows and select the WSL2-based engine (not legacy Hyper-V) when installing. Install via Chocolatey with `choco install docker-desktop` or it can be downloaded from [desktop.docker.com](https://desktop.docker.com/win/main/amd64/Docker%20Desktop%20Installer.exe).  Start Docker. It may prompt you to log out and log in again, or reboot.
8. Go to Docker Desktop settings > Resources > WSL integration > enable integration for your distro (now `docker` commands will be available from within your WSL2 distro).
9. Double-check in PowerShell: `wsl -l -v` should show three distros, and your Ubuntu should be the default. All three should be WSL version 2.
10. Double-check in Ubuntu (or your distro): `echo $CAROOT` should show something like `/mnt/c/Users/<you>/AppData/Local/mkcert`
11. Check that docker is working inside Ubuntu (or your distro): `docker ps`
12. Optional: If you prefer to use the *traditional Windows* ddev instead of working inside WSL2, install it with `choco install -y ddev`. The Windows ddev works fine with the WSL2-based Docker engine. However, the WSL2 ddev setup is vastly preferable and at least 10 times as fast. Support for the traditional Windows approach will eventually be dropped.
13. Open the WSL2 terminal, for example `Ubuntu` from the Windows start menu.
14. Install Homebrew: `/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"` (See [brew.sh](https://brew.sh/).)
15. Add brew to your path as prompted:, `echo 'eval $(/home/linuxbrew/.linuxbrew/bin/brew shellenv)' >> ~/.profile && source ~/.profile`
16. `brew install gcc && brew install drud/ddev/ddev`
17. `sudo apt-get update && sudo apt-get install -y xdg-utils` to install the `xdg-utils` package that allows `ddev launch` to work.

That's it! You have now installed DDEV on WSL2. If you're using WSL2 for DDEV (recommended), remember to run all `ddev` commands inside the WSL2 distro.
Follow the instructions in `Linux, macOS and Windows WSL2 (install script)` above.
**Make sure you put your projects in the Linux filesystem (e.g. /home/<your_username>), not in the Windows filesystem (`/mnt/c`), because you'll get vastly superior performance on the Linux filesystem.**

Note that mutagen-enabled or nfs-mount-enabled (and running NFS) are not required on WSL2 because it's so fast even without these options.

### Windows (traditional/legacy)

DDEV does work fine on the Windows side, although it's quite a bit slower than WSL2 by default, but good results have been reported by users who enabled mutagen, `ddev config global --mutagen-enabled`.

* If you use [chocolatey](https://chocolatey.org/) (recommended), then you can just `choco install ddev git` from an administrative shell. Upgrades are just `ddev poweroff && choco upgrade ddev`.
* A windows installer is provided in each [ddev release](https://github.com/drud/ddev/releases) (`ddev_windows_installer.<version>.exe`). Run that and it will do the full installation for you.  Open a new git-bash or PowerShell or cmd window and start using ddev.
* Most people interact with ddev on Windows using git-bash, part of the [Windows git suite](https://git-scm.com/download/win). Although ddev does work with cmd and PowerShell, it's more at home in bash. You can install it with chocolatey using `choco install -y git`.
* For performance, many users enable mutagen, `ddev config global --mutagen-enabled` (global) or `ddev config --mutagen-enabled` just for one project.

### Linux and macOS (manual)

You can also easily perform the installation or upgrade manually if preferred. DDEV is just a single executable, no special installation is actually required, so for all operating systems, the installation is just copying DDEV into place where it's in the system path.

* `ddev poweroff` if upgrading
* Download and extract the latest [ddev release](https://github.com/drud/ddev/releases) for your architecture.
* Move ddev to /usr/local/bin: `mv ddev /usr/local/bin/` (may require sudo), or another directory in your `$PATH` as preferred.
* Run `ddev` to test your installation. You should see DDEV's command usage output.
* As a one-time initialization, run `mkcert -install`, which may require your sudo password. Linux users may have to take additional actions as discussed below in [Linux `mkcert -install` additional instructions](#linux-mkcert--install-additional-instructions). If you don't have mkcert installed, you can install it from <https://github.com/FiloSottile/mkcert/releases>. Download the version for the correct architecture and `sudo mv <downloaded_file> /usr/local/bin/mkcert && sudo chmod +x /usr/local/bin/mkcert`.

### Linux `mkcert -install` additional instructions

The `mkcert -install` step on Linux may provide you with additional instructions.

On variants of Linux you may be prompted for additional package installation to get `certutil` installed, but you can follow the instructions given by mkcert:

  > $ mkcert -install
  > Created a new local CA at "/home/username/.local/share/mkcert" 
  > Installing to the system store is not yet supported on this Linux  but Firefox and/or Chrome/Chromium will still work.
  > You can also manually install the root certificate at `/home/username/.local/share/mkcert/rootCA.pem`.
  > Warning: `certutil` is not available, so the CA can't be automatically installed in Firefox and/or Chrome/Chromium! ⚠️
  > Install `certutil` with `apt install libnss3-tools` or `yum install nss-tools` and re-run `mkcert -install` 

  (Note the prompt `Installing to the system store is not yet supported on this Linux`, which can be a simple result of not having /usr/sbin in the path so that `/usr/sbin/update-ca-certificates` can be found.)

### Windows/Firefox additional instructions

The `mkcert -install` step on Windows does not work for the Firefox browser.
You need to add the created root certificate authority to the security
configuration by your self:

* Run `mkcert -install` (you can use the shortcut from the start menu for that)
* Run `mkcert -CAROOT` to see the local folder used for the newly created root
  certificate authority
* Open Firefox Preferences (about:preferences#privacy)
* Enter `certificates` into the search box on the top
* Click  `View Certificates...`
* Select the tab `Authorities`
* Click to `Import...`
* Go to the folder where your root certificate authority was stored
* Select the file `rootCA.pem`
* Click to `Open`

You should now see your CA under `mkcert development CA`.

### Uninstallation

For instructions to uninstall DDEV-Local see [Uninstallation](users/uninstall.md).

<a name="support"></a>

## Support and User-Contributed Documentation

We love to hear from our users and help them be successful with DDEV. Support options include:

* Lots of built-in help: `ddev help` and `ddev help <command>`. You'll find examples and explanations.
* [DDEV Documentation](users/faq.md)
* [DDEV Stack Overflow](https://stackoverflow.com/questions/tagged/ddev) for support and frequently asked questions. We respond quite quickly here and the results provide quite a library of user-curated solutions.
* [DDEV issue queue](https://github.com/drud/ddev/issues) for bugs and feature requests
* Interactive community support on [Discord](https://discord.gg/hCZFfAMc5k) for everybody, plus sub-channels for CMS-specific questions and answers.
* [ddev-contrib](https://github.com/drud/ddev-contrib) repo provides a number of vetted user-contributed recipes for extending and using DDEV. Your contributions are welcome.
* [awesome-ddev](https://github.com/drud/awesome-ddev) repo has loads of external resources, blog posts, recipes, screencasts, and the like. Your contributions are welcome.
* [Twitter with tag #ddev](https://twitter.com/search?q=%23ddev&src=typd&f=live) will get to us, but it's not as good for interactive support, but we'll answer anywhere.
