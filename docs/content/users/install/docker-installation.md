# Docker Installation

=== "macOS"

    ## macOS

    The two easy docker providers for macOS are Colima and Docker Desktop for Mac. You only need one of them.

    ### Colima

    [Colima](https://github.com/abiosoft/colima) the preferred docker provider for macOS. Colima is an open-source project that bundles the container management tool [lima](https://github.com/lima-vm/lima) with a docker (linux) back-end. This is similar to what Docker Desktop actually does, but Colima and Lima are entirely open-source and just focused on running containers. They work on both `amd64` and `arm64` (M1) macs. Colima does not require installation of Docker Desktop, or does it require paying a license fee to Docker, Inc., and it seems to be the most stable alternative at this time.
    
    Reasons to use Colima include:

    * Preferring to use open-source software (Docker Desktop, unlike Docker, is proprietary software).
    * Working for an organization that due to its size requires a paid Docker plan to use Docker Desktop, and wanting to avoid that cost and business relationship.
    * Preferring a CLI-focused approach to Docker Desktop's GUI focus.
    * Stability

    !!!tip "Install the docker client if you need it"
        If you don't have the `docker` client installed, you'll need to install it. (If `docker help` returns an error, you don't have it.) Use `brew install docker` to install it.

    1. Install colima with `brew install colima` using homebrew or see the other [installation options](https://github.com/abiosoft/colima/blob/main/docs/INSTALL.md).
    2. Configure your system to use mutagen, which is nearly essential for Colima. `ddev config global --mutagen-enabled`.
    3. Start colima: `colima start --cpu 4 --memory 4 --dns=1.1.1.1` will set up a colima instance with 4 CPUs and 4GB of memory allocated and using DNS server 1.1.1.1 (Cloudflare). Your needs may vary. After the first start you can just use `colima start`. Use `colima start -e` to edit the configuration file. (Configuring the DNS server is critical if you're using Pantheon or other tenants of `storage.googleapis.com`.)
    4. `colima status` will show colima's status.
    6. After a computer restart you'll need to `colima start` again. This will eventually be automated in later versions of colima.

    !!!warning "Docker contexts let the docker client point at the right docker server"
        Colima activates its own docker context in order to not conflict with Docker Desktop, so if you `docker context ls` you'll see a list of available contexts with currently active context indicated with an "\*" (which will be "colima" after you've started colima). You can change to the default (Docker Desktop) with `docker context use default` or change back with `docker context use colima`. This means you can actually run Docker Desktop and Colima at the same time... but be careful which context you're pointing at.

    DDEV has extensive automated test coverage for colima on macOS, but of course Colima is young, so please share your results and open issues or [contact us](../support.md) for help.

    #### Moving projects from Docker Desktop to Colima

    To move project databases from Docker Desktop to Colima:

    1. Make sure all your projects are listed in `ddev list`
    2. In Docker Desktop, `ddev snapshot --all`
    3. After starting Colima, start each project as needed and `ddev snapshot restore --latest`
    
    ### Docker Desktop for Mac

    Docker Desktop for Mac can be installed via Homebrew (`brew install  homebrew/cask/docker`) or can be downloaded from [docker.com](https://www.docker.com/products/docker-desktop). It has long been supported by DDEV and has extensive automated testing.

=== "Windows"

    ## Windows

    On Windows, you can install Docker Desktop, which works with both traditional Windows and WSL2, or if you're working inside WSL2 (recommended) you can just install docker engine (docker-ce) inside WSL2.

    ### Docker Desktop for Windows

    Docker Desktop for Windows can be downloaded via [Chocolatey](https://chocolatey.org/install) with `choco install docker-desktop` or it can be downloaded from [docker.com](https://www.docker.com/products/docker-desktop). It has extensive automated testing with DDEV, and works with DDEV both on traditional Windows and in WSL2.

    ### Windows WSL2: Docker-ce installed inside WSL2

    Although the traditional approach on Windows/WSL2 has been to use Docker Desktop, a number of people have moved away from using Docker Desktop and just installing the Docker-provided open-source `docker-ce` package inside WSL2. This uses entirely open-source software and does not require a license fee to Docker, Inc.

    Most of the installation is the same as on Linux, but it can be summarized as:

    * If you don't already have WSL2, install it with `wsl --install`, which will likely require a reboot.
    * `wsl --set-default-version 2`
    * Install a distro. Ubuntu 20.04 is recommended, `wsl -s Ubuntu-20.04`.
    * Install `docker-ce` in WSL2 using the normal Linux instructions, for Debian/Ubuntu follow these instructions inside the WSL2 distro:

    ```bash
    sudo apt-get remove docker docker-engine docker.io containerd runc
    sudo apt-get update && sudo apt-get install ca-certificates curl gnupg lsb-release
    curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo gpg --dearmor -o /usr/share/keyrings/docker-archive-keyring.gpg
    echo "deb [arch=$(dpkg --print-architecture) signed-by=/usr/share/keyrings/docker-archive-keyring.gpg] https://download.docker.com/linux/ubuntu $(lsb_release -cs) stable" | sudo tee /etc/apt/sources.list.d/docker.list > /dev/null
    sudo apt-get update && sudo apt-get install docker-ce docker-ce-cli containerd.io
    sudo groupadd docker && sudo usermod -aG docker $USER
    ```

    * You have to start docker-ce yourself on login, or use a script to do it. To have it start on entry to git-bash, a startup line to your (windows-side) `~/.bashrc` with:

    ```bash
    echo "wsl.exe -u root service docker status > /dev/null || wsl.exe -u root service docker start > /dev/null" >> ~/.bashrc
    ```

    You can then `source ~/.bashrc` to start immediately, or it should start the next time you open git-bash.

    * [Install mkcert](https://github.com/FiloSottile/mkcert#windows) on the Windows side; this may be easiest with [Chocolatey](https://chocolatey.org/install): In an administrative PowerShell, `Set-ExecutionPolicy Bypass -Scope Process -Force; [System.Net.ServicePointManager]::SecurityProtocol = [System.Net.ServicePointManager]::SecurityProtocol -bor 3072; iex ((New-Object System.Net.WebClient).DownloadString('https://chocolatey.org/install.ps1'))`
    * In an administrative PowerShell: `choco install -y mkcert`
    * In an administrative PowerShell, run `mkcert -install` and answer the prompt allowing the installation of the Certificate Authority.
    * In an administrative PowerShell, run the command `setx CAROOT "$(mkcert -CAROOT)"; If ($Env:WSLENV -notlike "*CAROOT/up:*") { setx WSLENV "CAROOT/up:$Env:WSLENV" }`. This will set WSL2 to use the Certificate Authority installed on the Windows side.
    * Double-check in Ubuntu (or your distro): `echo $CAROOT` should show something like `/mnt/c/Users/<you>/AppData/Local/mkcert`
    * Inside your WSL2 distro, `mkcert -install`.

=== "Linux"

    ## Linux

    !!!warning "Don't forget the Docker-ce post-install steps"
        Please don't forget that Linux installation absolutely requires post-install steps (below).
    
    !!!warning "Don't use `sudo` with the docker command"
        Please don't use `sudo` with docker. If you're needing it, you haven't finished the installation. Don't use `sudo` with ddev, except the rare case where you need the `ddev hostname` command.

    !!!warning "Docker Desktop for Linux is not yet mature enough to use"
        The release of Docker Desktop for Linux in 2022 was welcomed by many, but the system does not yet seem stable enough for predictable use, and has some of the problems of Docker Desktop on other platforms. We recommend that you stay with the traditional docker-ce installation described here.

    Docker installation on Linux depends on what flavor you're using. Where possible the Ubuntu/Deb/yum repository is the preferred technique.

    * [Ubuntu](https://docs.docker.com/install/linux/docker-ce/ubuntu/)
    * [CentOS](https://docs.docker.com/install/linux/docker-ce/centos/)
    * [Debian](https://docs.docker.com/install/linux/docker-ce/debian/)
    * [Fedora](https://docs.docker.com/install/linux/docker-ce/fedora/)
    * [binaries](https://docs.docker.com/install/linux/docker-ce/binaries/)

    !!!note "One-time post-installation step"
        Required post-installation steps: See [Docker's post-installation steps](https://docs.docker.com/engine/install/linux-postinstall/). You need to add your linux user to the "docker" group and configure the docker daemon to start on boot.
        ```bash
        sudo groupadd docker
        sudo usermod -aG docker $USER
        ```

    On systems that do not include systemd or equivalent (mostly if installing inside WSL2) you'll need to manually start docker with `service docker start` or the equivalent in your distro. You can add this into your `~/.profile` or equivalent.

=== "Gitpod.io"

    ## Gitpod.io

    With gitpod.io you don't have to install anything at all. Docker is all set up for you. 

<a name="troubleshooting"></a>

## Testing and Troubleshooting Your Docker Installation

Docker needs to be able to do a few things for ddev to work:

* Mount the project code directory from the host into the container; the project code directory is usually somewhere in a subdirectory of your home directory.
* Access TCP ports on the host to serve HTTP and HTTPS. These are ports 80 and 443 by default, but they can be changed on a per-project basis.

We can use a single docker command to make sure that docker is set up to do what we want:

In your *project directory* run `docker run --rm -t -p 80:80 -p 443:443 -v "//$PWD:/tmp/projdir" busybox sh -c "echo ---- Project Directory && ls /tmp/projdir"` - you should see the files in your project directory displayed. (On Windows, make sure you run this using git-bash.)

If that fails (if you get an error, or you don't see the contents of your project directory and your home directory) you'll need to troubleshoot:

* "port is already allocated": See [troubleshooting](../basics/troubleshooting.md).
* `invalid mount config for type "bind": bind mount source path does not exist: <some path>` means the filesystem isn't successfully shared into the docker container.
* "The path ... is not shared and is not known to Docker": Visit docker's preferences/settings->File sharing and share the appropriate path or drive.
* `Error response from daemon: Get https://registry-1.docker.io/v2/` - Docker may not be running (restart it) or you may not have any access to the internet.
* "403 authentication required" when trying to `ddev start`: Try `docker logout` and do it again. Docker authentication is *not* required for any normal ddev action.
