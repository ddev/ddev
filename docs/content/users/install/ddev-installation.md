# DDEV Installation

Once you’ve [installed a Docker provider](docker-installation.md), you’re ready to install DDEV!

Installing and upgrading DDEV are nearly the same thing, because you're upgrading the `ddev` binary that talks with Docker. You can update this file like other software on your system, whether it’s with a package manager or traditional installer.

=== "macOS"

    ## macOS

    ### Homebrew

    [Homebrew](https://brew.sh/) is the easiest way to install and upgrade DDEV:

    ```bash
    brew install drud/ddev/ddev
    ```

    ```bash
    brew upgrade ddev
    ```

    As a one-time initialization, run

    ```bash
    mkcert -install
    ```

    ### Install Script

    !!!tip
        The install script works on macOS, Linux, and Windows WSL2.


    Run the [install script](https://github.com/drud/ddev/blob/master/scripts/install_ddev.sh) to install or update DDEV. It downloads, verifies, and sets up the `ddev` binary:

    ```
    curl -fsSL https://raw.githubusercontent.com/drud/ddev/master/scripts/install_ddev.sh | bash
    ```

    You can include a `-s <version>` argument to install a specific release or a prerelease version:

    ```
    curl -fsSL https://raw.githubusercontent.com/drud/ddev/master/scripts/install_ddev.sh | bash -s v1.19.5
    ```

=== "Linux"

    ## Linux

    ### Debian/Ubuntu

    DDEV’s Debian and RPM packages work with `apt` and `yum` repositories and most variants that use them, including Windows WSL2:

    ```bash
    curl -fsSL https://apt.fury.io/drud/gpg.key | gpg --dearmor | sudo tee /etc/apt/trusted.gpg.d/ddev.gpg > /dev/null
    echo "deb [signed-by=/etc/apt/trusted.gpg.d/ddev.gpg] https://apt.fury.io/drud/ * *" | sudo tee /etc/apt/sources.list.d/ddev.list
    sudo apt update && sudo apt install -y ddev
    ```

    Update with your usual commands:

    ```bash
    sudo apt update && sudo apt upgrade
    ```
    
    !!!tip "Removing Previous Install Methods"
        If you previously used DDEV’s [install script](#install-script), you can remove that version:

        ```
        sudo rm -f /usr/local/bin/ddev /usr/local/bin/mkcert /usr/local/bin/*ddev_nfs_setup.sh
        ```
    
        If you previously [installed DDEV with Homebrew](#homebrew), you can run `brew unlink ddev` to get rid of the Homebrew version.

    ### Fedora, Red Hat, etc.

    ```bash
    echo '[ddev]
    name=DDEV Repo
    baseurl=https://yum.fury.io/drud/
    enabled=1
    gpgcheck=0' | sudo tee -a /etc/yum.repos.d/ddev.repo

    sudo dnf install --refresh ddev
    ```

    In the future you can update as usual using `sudo dnf upgrade ddev`. (Signed repository support will be added in the near future.)

    ### Arch Linux

    For Arch-based systems including `Arch Linux`, `EndeavourOS` and `Manjaro` we maintain the [ddev-bin](https://aur.archlinux.org/packages/ddev-bin/) package in AUR. To install use `yay -S ddev` or whatever other AUR tool you use; to upgrade `yay -Syu ddev`.

    As a one-time initialization, run `mkcert -install`.

    ### Alternate Linux Install Methods

    You can also use two macOS install methods to install or update DDEV on Linux: [Homebrew](#homebrew) (only on AMD64 computers) and the standalone [install script](#install-script).

=== "Windows WSL2"

    ## Windows WSL2

    Windows WSL2 is a fantastic way to run DDEV and your web components. It’s Linux, which means a different experience for many Windows users. It’s Ubuntu Linux by default as described here, so it’s worth taking a little time to explore how Ubuntu and Bash work, including standard system commands and installation and upgrade procedures.

    **WSL2 is the recommended installation method for all Windows users**.

    **Using WSL2 to install and run DDEV is not the same as using Docker Desktop's WSL2 engine, which itself runs in WSL2, but can serve applications running in both traditional WIndows and inside WSL2.**

    **All Windows 10/11 editions (including Windows 10 Home) support WSL2**. If you’re already familiar with DDEV on Windows, you might have been using NFS for better filesystem performance. **You won't need NFS anymore once you switch to WSL2**, since it provides awesome filesystem performance out of the box.

    The WSL2 install process involves:

    * Installing Chocolatey package manager (optional).
    * One time initialization of mkcert.
    * Installing WSL2 and installing a distro like Ubuntu.
    * Installing Docker Desktop for Windows and enabling WSL2 integration with the distro (optional, do this if you're using the Docker Desktop approach).
    * Installing DDEV inside your distro.

    ### WSL2 + Docker CE Inside Install Script

    This scripted installation prepares your default WSL2 Ubuntu distro and has no dependency on Docker Desktop.

    The provided PowerShell script can do most of the work for you, or you can handle these things manually. 
    In all cases:
    
    1. Install WSL2 with an Ubuntu distro. On a system without WSL2, run:
        ```powershell
        wsl --install
        ```

        Verify that you have an Ubuntu distro set to default by running `wsl -l -v`.

        If you already have WSL2 but don't have an Ubuntu distro, install one by running `wsl --install Ubuntu`. 

        If that doesn't work for you, see the [manual installation](https://docs.microsoft.com/en-us/windows/wsl/install-manual) and linked [troubleshooting](https://docs.microsoft.com/en-us/windows/wsl/troubleshooting#installation-issues).
        
        If you prefer to use another Ubuntu distro, install it and set it as default. For example, `wsl --set-default Ubuntu-22.04`.

    2. In an administrative PowerShell run [this PowerShell script](https://raw.githubusercontent.com/drud/ddev/master/scripts/install_ddev_wsl2_docker_inside.ps1) by executing: 

        ```powershell
        Set-ExecutionPolicy Bypass -Scope Process -Force; [System.Net.ServicePointManager]::SecurityProtocol = [System.Net.ServicePointManager]::SecurityProtocol -bor 3072; 
        iex ((New-Object System.Net.WebClient).DownloadString('https://raw.githubusercontent.com/drud/ddev/master/scripts/install_ddev_wsl2_docker_inside.ps1'))
        ```
    Now you can use the "Ubuntu" terminal app or Windows Terminal to access your Ubuntu distro, which has DDEV and Docker working inside it.

    ### WSL2 + Docker Desktop Install Script

    This scripted installation prepares your default WSL2 Ubuntu distro for use with Docker Desktop.

    You can do these things manually, or you can do most of it with the provided PowerShell script. 
    In all cases:
    
    1. Install WSL2 with an Ubuntu distro. On a system without WSL2, just run:
        ```powershell
        wsl --install
        ```

        Verify that you have an Ubuntu distro set as the default default with `wsl -l -v`.

        If you already have WSL2 but don't have an Ubuntu distro, install one with `wsl --install Ubuntu`. 

        If that doesn't work for you, see the [manual installation](https://docs.microsoft.com/en-us/windows/wsl/install-manual) and linked [troubleshooting](https://docs.microsoft.com/en-us/windows/wsl/troubleshooting#installation-issues).
        
        If you prefer to use another Ubuntu distro, just install it and set it as default. For example, `wsl --set-default Ubuntu-22.04`.

    2. Install Docker Desktop. If you already have chocolatey, `choco install -y docker-desktop` or [download Docker Desktop from Docker](https://www.docker.com/products/docker-desktop/).
    3. Start Docker Desktop. You should now be able to do `docker ps` in PowerShell or Git Bash.
    4. In `Docker Desktop -> Settings -> Resources -> WSL2 Integration` verify that Docker Desktop is integrated with your distro.
    5. In an administrative PowerShell run [this PowerShell script](https://raw.githubusercontent.com/drud/ddev/master/scripts/install_ddev_docker_desktop_wsl2.ps1) by executing:

        ```powershell
        Set-ExecutionPolicy Bypass -Scope Process -Force; [System.Net.ServicePointManager]::SecurityProtocol = [System.Net.ServicePointManager]::SecurityProtocol -bor 3072; 
        iex ((New-Object System.Net.WebClient).DownloadString('https://raw.githubusercontent.com/drud/ddev/master/scripts/install_ddev_docker_desktop_wsl2.ps1'))
        ```

    Now you can use the "Ubuntu" terminal app or Windows Terminal to access your Ubuntu distro, which has DDEV and Docker Desktop integrated with it.

    ### WSL2/Docker Desktop Manual Installation

    You can do all of the steps manually of course:

    1. Install [Chocolatey](https://chocolatey.org/install):
        ```powershell
        Set-ExecutionPolicy Bypass -Scope Process -Force; [System.Net.ServicePointManager]::SecurityProtocol = [System.Net.ServicePointManager]::SecurityProtocol -bor 3072; 
        iex ((New-Object System.Net.WebClient).DownloadString('https://chocolatey.org/install.ps1'))`
        ```
    2. In an administrative PowerShell: `choco install -y mkcert`
    3. In an administrative PowerShell, run `mkcert -install` and answer the prompt allowing the installation of the Certificate Authority.
    4. In an administrative PowerShell, run the command `$env:CAROOT="$(mkcert -CAROOT)"; setx CAROOT $env:CAROOT; If ($Env:WSLENV -notlike "*CAROOT/up:*") { $env:WSLENV="CAROOT/up:$env:WSLENV"; setx WSLENV $Env:WSLENV }`. This will set WSL2 to use the Certificate Authority installed on the Windows side. In some cases it takes a reboot to work correctly.
    5. In administrative PowerShell, run the command `wsl --install`. This will install WSL2 and Ubuntu for you. Reboot when this is done.
    6. **Docker Desktop for Windows:** If you already have the latest Docker Desktop, configure it in the General Settings to use the WSL2-based engine. Otherwise install the latest Docker Desktop for Windows and select the WSL2-based engine (not legacy Hyper-V) when installing. Install via Chocolatey with `choco install docker-desktop` or it can be downloaded from [desktop.docker.com](https://desktop.docker.com/win/main/amd64/Docker%20Desktop%20Installer.exe).  Start Docker. It may prompt you to log out and log in again, or reboot.
    7. Go to Docker Desktop’s *Settings* → *Resources* → *WSL integration* → *enable integration for your distro*. Now `docker` commands will be available from within your WSL2 distro.
    8. Double-check in PowerShell: `wsl -l -v` should show three distros, and your Ubuntu should be the default. All three should be WSL version 2.
    9. Double-check in Ubuntu (or your distro): `echo $CAROOT` should show something like `/mnt/c/Users/<you>/AppData/Local/mkcert`
    10. Check that Docker is working inside Ubuntu (or your distro): `docker ps`
    11. Optional: If you prefer to use the *traditional Windows* DDEV instead of working inside WSL2, install it with `choco install -y ddev`. The Windows DDEV works fine with the WSL2-based Docker engine. However, the WSL2 DDEV setup is vastly preferable and at least 10 times as fast. Support for the traditional Windows approach will eventually be dropped.
    12. Open the WSL2 terminal, for example `Ubuntu` from the Windows start menu.
    13. Install DDEV using

        ```bash
        curl https://apt.fury.io/drud/gpg.key | sudo apt-key add -
        echo "deb https://apt.fury.io/drud/ * *" | sudo tee -a /etc/apt/sources.list.d/ddev.list
        sudo apt update && sudo apt install -y ddev
        ```

    14. In WSL2 run `mkcert -install`.

    That’s it! You have now installed DDEV on WSL2. If you’re using WSL2 for DDEV (recommended), remember to run all `ddev` commands inside the WSL2 distro.

    To upgrade DDEV in WSL2 Ubuntu, use `apt upgrade ddev` as described in the [Linux installation section](#apt-packages-for-Debian-based-systems).

    !!!warning "Projects go in `/home`, not on the Windows filesystem"
        Make sure you put your projects in the Linux filesystem (e.g. `/home/<your_username>`), **not** in the Windows filesystem (`/mnt/c`), because you’ll get vastly superior performance on the Linux filesystem. You will be very unhappy if you put your project in `/mnt/c`.

    !!!note "Path to certificates"
        Note the prompt `Installing to the system store is not yet supported on this Linux`, which can be a simple result of not having `/usr/sbin` in the path so that `/usr/sbin/update-ca-certificates` can be found.)

=== "Traditional Windows"

    ## Traditional Windows

    DDEV works fine on the Windows side, but it’s slower than WSL2 by default. Enable either [Mutagen](performance/#system-requirements) or [NFS](performance/#nfs) for the best performance.

    * If you use [Chocolatey](https://chocolatey.org/) (recommended), you can run `choco install ddev git` from an administrative shell. Upgrades are just `ddev poweroff && choco upgrade ddev`.
    * A Windows installer is provided in each [DDEV release](https://github.com/drud/ddev/releases) (`ddev_windows_installer.<version>.exe`). Run that and it will do the full installation for you. Open a new Git Bash or PowerShell or cmd window and start using DDEV.
    * Most people interact with DDEV on Windows using Git Bash, part of the [Windows Git suite](https://git-scm.com/download/win). Although DDEV does work with cmd and PowerShell, it's more at home in Bash. You can install it with Chocolatey using `choco install -y git`.
    * For performance, many users enable Mutagen, `ddev config global --mutagen-enabled` (global) or `ddev config --mutagen-enabled` just for one project.

    !!!note "Windows Firefox Trusted CA"

        The `mkcert -install` step on Windows isn’t enough for Firefox.
        You need to add the created root certificate authority to the security configuration yourself:

        * Run `mkcert -install` (you can use the shortcut from the Start Menu for that)
        * Run `mkcert -CAROOT` to see the local folder used for the newly-created root certificate authority
        * Open Firefox Preferences (`about:preferences#privacy`)
        * Enter “certificates” into the search box on the top
        * Click *View Certificates...*
        * Select *Authorities* tab
        * Click to *Import...*
        * Navigate to the folder where your root certificate authority was stored
        * Select the `rootCA.pem` file
        * Click to *Open*

        You should now see your CA under `mkcert development CA`.

=== "Gitpod"

    ## Gitpod

    DDEV is fully supported in [Gitpod](https://www.gitpod.io), where you don’t have to install anything at all.

    Choose any of the following methods to launch your project:

    1. [Open any repository](https://www.gitpod.io/docs/getting-started) using Gitpod, run `brew install drud/ddev/ddev`, and use DDEV!
        * You can install your web app there, or import a database.
        * You may want to implement one of the `ddev pull` provider integrations to pull from a hosting provider or an upstream source.
    2. Use the [ddev-gitpod-launcher](https://drud.github.io/ddev-gitpod-launcher/) form to launch a repository.  
        You’ll provide a source repository and click a button to open a newly-established environment. You can specify a companion artifacts repository and automatically load `db.sql.gz` and `files.tgz` from it. (More details in the [repository’s README](https://github.com/drud/ddev-gitpod-launcher/blob/main/README.md).)
    3. Save the following link to your bookmark bar: <a href="javascript: if %28 %2Fbitbucket%2F.test %28 window.location.host %29 %20 %29 %20%7B%20paths%3Dwindow.location.pathname.split %28 %22%2F%22 %29 %3B%20repo%3D%5Bwindow.location.origin%2C%20paths%5B1%5D%2C%20paths%5B2%5D%5D.join %28 %22%2F%22 %29 %20%7D%3B%20if %28 %2Fgithub.com%7Cgitlab.com%2F.test %28 window.location.host %29  %29 %20%7Brepo%20%3D%20window.location.href%7D%3B%20if%20 %28 repo %29 %20%7Bwindow.location.href%20%3D%20%22https%3A%2F%2Fgitpod.io%2F%23DDEV_REPO%3D%22%20%2B%20encodeURIComponent %28 repo %29 %20%2B%20%22%2CDDEV_ARTIFACTS%3D%22%20%2B%20encodeURIComponent %28 repo %29 %20%2B%20%22-artifacts%2Fhttps%3A%2F%2Fgithub.com%2Fdrud%2Fddev-gitpod-launcher%2F%22%7D%3B">Open in ddev-gitpod</a>.  
        It’s easiest to drag the link into your bookmarks. When you’re on a Git repository, click the bookmark to open it with DDEV in Gitpod. It does the same thing as the second option, but it works on non-Chrome browsers and you can use native browser keyboard shortcuts.

    It can be complicated to get private databases and files into Gitpod, so in addition to the launchers, the [`git` provider example](https://github.com/drud/ddev/blob/master/pkg/ddevapp/dotddev_assets/providers/git.yaml.example) demonstrates pulling a database and files without complex setup or permissions. This was created explicitly for Gitpod integration, because in Gitpod you typically already have access to private Git repositories, which are a fine place to put a starter database and files. Although [ddev-gitpod-launcher](https://drud.github.io/ddev-gitpod-launcher/) and the web extension provide the capability, you may want to integrate a Git provider—or one of the [other providers](https://github.com/drud/ddev/tree/master/pkg/ddevapp/dotddev_assets/providers)—for each project.

=== "Manual"

    ## Manual

    DDEV is a single executable, so installation on any OS is a matter of copying the a `ddev` binary for your architecture into the appropriate system path on your machine.

    * Download and extract the latest [DDEV release](https://github.com/drud/ddev/releases) for your architecture.
    * Move `ddev` to `/usr/local/bin` with `mv ddev /usr/local/bin/` (may require `sudo`), or another directory in your `$PATH` as preferred.
    * Run `ddev` to test your installation. You should see DDEV’s command usage output.
    * As a one-time initialization, run `mkcert -install`, which may require your `sudo` password.  
    
        If you don’t have `mkcert` installed, download the [latest release](https://github.com/FiloSottile/mkcert/releases) for your architecture and `sudo mv <downloaded_file> /usr/local/bin/mkcert && sudo chmod +x /usr/local/bin/mkcert`.
