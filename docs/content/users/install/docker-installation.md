# Docker Installation

You’ll need a Docker provider on your system before you can [install DDEV](ddev-installation.md).

=== "macOS"

    ## macOS

    Install either [Colima](#colima) or [Docker Desktop](#docker-desktop-for-mac).

    ### Colima

    We recommend [Colima](https://github.com/abiosoft/colima), a project that bundles a container management tool called [Lima](https://github.com/lima-vm/lima) with a Docker (Linux) backend.

    !!!tip "Wait ... Colima?"
        Yes! See *Why do you recommend Colima over Docker Desktop on macOS?* in [the FAQ](../basics/faq.md).
    
    1. Run `docker help` to make sure you’ve got the Docker client installed. If you get an error, install it with [Homebrew](https://brew.sh) by running `brew install docker`.
    2. Install Colima with `brew install colima` or one of the other [installation options](https://github.com/abiosoft/colima/blob/main/docs/INSTALL.md).
    3. Start Colima with 4 CPUs, 6GB memory, 100GB storage, and Cloudflare DNS, adjusting as needed:  
    ```
    colima start --cpu 4 --memory 6 --disk 100 --dns=1.1.1.1
    ```
    4. After [installing DDEV](ddev-installation.md), configure your system to use Mutagen—essential for DDEV with Colima—with `ddev config global --mutagen-enabled`.
    
    After the initial run above, you can use `colima start` or use `colima start -e` to edit the configuration file. Run `colima status` at any time to check Colima’s status.
    
    When your computer restarts, you’ll need to `colima start` again. This will eventually be automated in later versions of Colima.
    
    !!!tip "Colima disk allocation"
        We recommend allocating lots of storage for Colima because there’s no way to increase the size later. You can reduce usage with `ddev clean`, and kill off disk images with `docker rm -f $(docker ps -aq) && docker rmi -f $(docker images -q)`. If you have to rebuild your Colima instance, use the technique described below for migrating from Docker Desktop.

    !!!warning "Docker contexts let the Docker client point at the right Docker server"
        Colima activates its own Docker context to prevent conflicts with Docker Desktop. If you run `docker context ls`, you’ll see a list of available contexts where the currently-active one is indicated with a `*`—which will be `colima` after you’ve started it. You can change to the default (Docker Desktop) with `docker context use default` or change back with `docker context use colima`. This means you can run Docker Desktop and Colima at the same time, but be mindful of which context you’re pointing at!

    !!!warning "Colima can only work in your home directory unless you do further configuration"
        By default, Colima only mounts your home directory, so it’s easiest to use it in a subdirectory there. See the `~/.colima/default/colima.yaml` for more information, or notes in [colima.yaml](https://github.com/abiosoft/colima/blob/fc948f8f055600986f87e29e3e632daf56ac8774/embedded/defaults/colima.yaml#L130-L143).


    #### Migrating Projects from Docker Desktop to Colima

    Move your project databases from Docker Desktop to Colima:

    1. Make sure all your projects are listed in `ddev list`.
    2. In Docker Desktop, `ddev snapshot --all`.
    3. After starting Colima, start each project and `ddev snapshot restore --latest`.
    
    ### Docker Desktop for Mac

    Docker Desktop for Mac can be installed via Homebrew (`brew install homebrew/cask/docker`) or can be downloaded from [docker.com](https://www.docker.com/products/docker-desktop). It has long been supported by DDEV and has extensive automated testing.

=== "Windows"

    ## Windows

    If you’re working inside WSL2, which we recommend, you can [install Docker Engine (docker-ce) inside of it](#docker-ce-inside-windows-wsl2). Otherwise, you can [install Docker Desktop](#docker-desktop-for-windows), which works with both traditional Windows and WSL2.

    ## Docker CE Inside Windows WSL2

    Many have moved away from using Docker Desktop in favor of the Docker-provided open-source `docker-ce` package inside WSL2.

    Most of the installation is the same as on Linux:

    * If you already have Docker Desktop installed, make sure to disable its integration with your WSL2 distro. In *Resources* → *WSL Integration*, disable integration with the default distro and with your particular distro.
    * If you don’t already have WSL2, install it with `wsl --install`. This will likely require a reboot.
    * Run `wsl --set-default-version 2`.
    * Install a distro. We recommend Ubuntu 20.04: `wsl --install Ubuntu-20.04`.
    * Install `docker-ce` in WSL2 using the normal Linux instructions. For Debian/Ubuntu, run the following inside the WSL2 distro:
        ```bash
        sudo apt-get remove docker docker-engine docker.io containerd runc
        sudo apt-get update && sudo apt-get install ca-certificates curl gnupg lsb-release
        curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo gpg --dearmor -o /usr/share/keyrings/docker-archive-keyring.gpg
        echo "deb [arch=$(dpkg --print-architecture) signed-by=/usr/share/keyrings/docker-archive-keyring.gpg] https://download.docker.com/    linux/ubuntu $(lsb_release -cs) stable" | sudo tee /etc/apt/sources.list.d/docker.list > /dev/null
        sudo apt-get update && sudo apt-get install docker-ce docker-ce-cli containerd.io
        sudo groupadd docker && sudo usermod -aG docker $USER
        ```
    * You have to start `docker-ce` yourself on login, or use a script to automate it. To have it start on entry to Git Bash, add a startup line to your (Windows-side) `~/.bashrc` with:
        ```bash
        echo "wsl.exe -u root service docker status > /dev/null || wsl.exe -u root service docker start > /dev/null" >> ~/.bashrc
        ```

        `source ~/.bashrc` to start immediately, or it should start with your next Git Bash session.

    * [Install mkcert](https://github.com/FiloSottile/mkcert#windows) on the Windows side, which may be easiest with [Chocolatey](https://chocolatey.org/install): 
        * In an administrative PowerShell: 
            ```
            Set-ExecutionPolicy Bypass -Scope Process -Force; [System.Net.ServicePointManager]::SecurityProtocol = [System.Net.        ServicePointManager]::SecurityProtocol -bor 3072; iex ((New-Object System.Net.WebClient).DownloadString('https://chocolatey.org/        install.ps1'))
            ```
        * In an administrative PowerShell: `choco install -y mkcert`.
        * In an administrative PowerShell, run `mkcert -install` and follow prompts to install the Certificate Authority.
        * In an administrative PowerShell, run `setx CAROOT "$(mkcert -CAROOT)"; If ($Env:WSLENV -notlike "*CAROOT/up:*") { setx WSLENV "CAROOT/up:$Env:WSLENV" }`. This will set WSL2 to use the Certificate Authority installed on the Windows side.
        * Double-check in Ubuntu (or your distro): `echo $CAROOT` should show something like `/mnt/c/Users/<you>/AppData/Local/mkcert`.
        * Inside your WSL2 distro, `mkcert -install`.

    ### Docker Desktop for Windows

    Docker Desktop for Windows can be downloaded via [Chocolatey](https://chocolatey.org/install) with `choco install docker-desktop` or it can be downloaded from [docker.com](https://www.docker.com/products/docker-desktop). It has extensive automated testing with DDEV, and works with DDEV both on traditional Windows and in WSL2.

=== "Linux"

    ## Linux

    !!!warning "Avoid Docker Desktop for Linux"
        The 2022 release of Docker Desktop for Linux doesn’t seem stable enough for regular use, and exhibits some problems Docker Desktop has on other platforms. We recommend staying with the traditional `docker-ce` installation described here.

    Docker installation on Linux depends on what flavor you’re using. It’s best to use your native package repository (`apt`, `yum`, etc.):

    * [Ubuntu](https://docs.docker.com/install/linux/docker-ce/ubuntu/)
    * [CentOS](https://docs.docker.com/install/linux/docker-ce/centos/)
    * [Debian](https://docs.docker.com/install/linux/docker-ce/debian/)
    * [Fedora](https://docs.docker.com/install/linux/docker-ce/fedora/)
    * [binaries](https://docs.docker.com/install/linux/docker-ce/binaries/)


    Linux installation **absolutely** requires adding your Linux user to the `docker` group, and configuring the Docker daemon to start at boot. See [Post-installation steps for Linux](https://docs.docker.com/engine/install/linux-postinstall/).

    !!!warning "Don’t `sudo` with `docker` or `ddev`"
        Don’t use `sudo` with the `docker` command. If you find yourself needing it, you haven’t finished the installation. You also shouldn’t use `sudo` with `ddev` unless it’s specifically for the `ddev hostname` command.

    On systems without `systemd` or its equivalent—mostly if you’re installing inside WSL2—you’ll need to manually start Docker with `service docker start` or the equivalent in your distro. You can add this to your shell profile.

=== "Gitpod"

    ## Gitpod

    With [Gitpod](https://www.gitpod.io) you don’t have to install anything at all. Docker is all set up for you. 

<a name="troubleshooting"></a>

## Testing and Troubleshooting Your Docker Installation

Docker needs to be able to do a few things for DDEV to work:

* Mount the project code directory, typically a subdirectory of your home folder, from the host into the container.
* Access TCP ports on the host to serve HTTP and HTTPS. These are ports 80 and 443 by default, but they can be changed on a per-project basis.

We can use a single Docker command to make sure Docker is set up to do what we want:

In your *project directory* run the following (using Git Bash if you’re on Windows!):

```
docker run --rm -t -p 80:80 -p 443:443 -v "//$PWD:/tmp/projdir" busybox sh -c "echo ---- Project Directory && ls /tmp/projdir"
```

The result should be a list of the files in your project directory.

If you get an error or don’t see the contents of your project directory, you’ll need to troubleshoot further:

* For a “port is already allocated” error, see the [Troubleshooting](../basics/troubleshooting.md#web-server-ports-already-occupied) page.
* “invalid mount config for type "bind": bind mount source path does not exist: [some path]” means the filesystem isn’t successfully shared into the Docker container.
* If you’re seeing “The path (...) is not shared and is not known to Docker”, find *File sharing* in your Docker settings make sure the appropriate path or drive is included.
* “Error response from daemon: Get registry-1.docker.io/v2/” may mean Docker isn’t running or you don’t have internet access. Try starting or restarting Docker, and confirm you have a working internet connection.
* If you’re seeing “403 authentication required” trying to `ddev start`, run `docker logout` and try again. Docker authentication is *not* required for any normal DDEV action.
