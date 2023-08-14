# Docker Installation

You’ll need a Docker provider on your system before you can [install DDEV](ddev-installation.md).

=== "macOS"

    ## macOS

    Install either [Colima](#colima) or [Docker Desktop](#docker-desktop-for-mac).

    ### Colima

    We recommend [Colima](https://github.com/abiosoft/colima), a project that bundles a container management tool called [Lima](https://github.com/lima-vm/lima) with a Docker (Linux) backend.

    !!!tip "Wait ... Colima?"
        Yes! See *Why do you recommend Colima over Docker Desktop on macOS?* in [the FAQ](../usage/faq.md).

    1. Run `docker help` to make sure you’ve got the Docker client installed. If you get an error, install it with [Homebrew](https://brew.sh) by running `brew install docker`.
    2. Install Colima with `brew install colima` or one of the other [installation options](https://github.com/abiosoft/colima/blob/main/docs/INSTALL.md).
    3. Start Colima with 4 CPUs, 6GB memory, 100GB storage, and Cloudflare DNS, adjusting as needed:
    ```
    colima start --cpu 4 --memory 6 --disk 100 --vm-type=qemu --mount-type=sshfs --dns=1.1.1.1
    ```
    (On macOS versions before Ventura, omit the `--vm-type=qemu` flag as it doesn't work on older OS versions.)
    
    After the initial run above, you can use `colima start` or use `colima start -e` to edit the configuration file. Run `colima status` at any time to check Colima’s status.

    When your computer restarts, you’ll need to `colima start` again. This will eventually be automated in later versions of Colima.

    !!!tip "Colima disk allocation"
        In Colima versions starting with 0.5.4 you can increase—but not decrease—the disk allocation by editing `~/.colima/default/colima.yaml` to change the `disk` setting to a higher value. For example, `disk: 200` will increase allocation to 200 gigabytes. Then `colima restart` will result in the new disk allocation.

    !!!warning "Docker contexts let the Docker client point at the right Docker server"
        Colima activates its own Docker context to prevent conflicts with Docker Desktop. If you run `docker context ls`, you’ll see a list of available contexts where the currently-active one is indicated with a `*`—which will be `colima` after you’ve started it. You can change to the default (Docker Desktop) with `docker context use default` or change back with `docker context use colima`. This means you can run Docker Desktop and Colima at the same time, but be mindful of which context you’re pointing at!

    !!!warning "Colima can only work in your home directory unless you do further configuration"
        By default, Colima only mounts your home directory, so it’s easiest to use it in a subdirectory there. See the `~/.colima/default/colima.yaml` for more information, or notes in [colima.yaml](https://github.com/abiosoft/colima/blob/fc948f8f055600986f87e29e3e632daf56ac8774/embedded/defaults/colima.yaml#L130-L143).


    #### Migrating Projects from Docker Desktop to Colima

    1. Move your project databases from Docker Desktop to Colima using the technique in [How can I migrate from one Docker provider to another?](../usage/faq.md#how-can-i-migrate-from-one-docker-provider-to-another).

    2. Docker Desktop may have left a bad `~/.docker/config.json`". If you have trouble running `ddev start` with a project you’ve migrated, remove the `credsStore` line in `~/.docker/config.json` 

    ### Docker Desktop for Mac

    Docker Desktop for Mac can be installed via Homebrew (`brew install homebrew/cask/docker`) or can be downloaded from [docker.com](https://www.docker.com/products/docker-desktop). It has long been supported by DDEV and has extensive automated testing.

    We do not recommend the `VirtioFS` option with Docker Desktop for Mac. While it’s performant, it can cause mysterious problems that are not present with [Mutagen](performance.md#mutagen)—which offers comparable performance when enabled.

=== "Linux"

    ## Linux

    !!!warning "Avoid Docker Desktop for Linux"
        Current releases of Docker Desktop for Linux are not usable with DDEV for a number of reasons, and also exhibit some of the problems Docker Desktop has on other platforms. Please use the normal `docker-ce` installation described here.

    Docker installation on Linux depends on what flavor you’re using. It’s best to use your native package repository (`apt`, `yum`, etc.):

    * [Ubuntu](https://docs.docker.com/install/linux/docker-ce/ubuntu/)
    * [CentOS](https://docs.docker.com/install/linux/docker-ce/centos/)
    * [Debian](https://docs.docker.com/install/linux/docker-ce/debian/)
    * [Fedora](https://docs.docker.com/install/linux/docker-ce/fedora/)
    * [binaries](https://docs.docker.com/install/linux/docker-ce/binaries/)


    Linux installation **absolutely** requires adding your Linux user to the `docker` group, and configuring the Docker daemon to start at boot. See [Post-installation steps for Linux](https://docs.docker.com/engine/install/linux-postinstall/).

    !!!warning "Don’t `sudo` with `docker` or `ddev`"
        Don’t use `sudo` with the `docker` command. If you find yourself needing it, you haven’t finished the installation. You also shouldn’t use `sudo` with `ddev` unless it’s specifically for the [`ddev hostname`](../usage/commands.md#hostname) command.

    On systems without `systemd` or its equivalent—mostly if you’re installing inside WSL2—you’ll need to manually start Docker with `service docker start` or the equivalent in your distro. You can add this to your shell profile.

=== "Windows"

    ## Windows

    If you’re working inside WSL2, which we recommend, you can [install Docker Engine (docker-ce) inside of it](#docker-ce-inside-windows-wsl2). Otherwise, you can [install Docker Desktop](#docker-desktop-for-windows), which works with both traditional Windows and WSL2.

    ### Docker CE Inside Windows WSL2

    Many have moved away from using Docker Desktop in favor of the Docker-provided open-source `docker-ce` package inside WSL2.

    The instructions for [DDEV Installation in WSL2](ddev-installation.md#windows-wsl2) include Docker CE setup and a script that does almost all the work. Please use those.

    ### Docker Desktop for Windows

    Docker Desktop for Windows can be downloaded via [Chocolatey](https://chocolatey.org/install) with `choco install docker-desktop` or it can be downloaded from [docker.com](https://www.docker.com/products/docker-desktop). It has extensive automated testing with DDEV, and works with DDEV both on traditional Windows and in WSL2.

    See [WSL2 DDEV Installation](ddev-installation.md#windows-wsl2) for help installing DDEV with Docker Desktop on WSL2.

=== "Gitpod"

    ## Gitpod

    With [Gitpod](https://www.gitpod.io) you don’t have to install anything at all. Docker is all set up for you.

=== "Codespaces"

    ## GitHub Codespaces

    You can set up [GitHub Codespaces](https://github.com/features/codespaces) following the instructions in the [DDEV Installation](ddev-installation.md#github-codespaces) section.

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

* For a “port is already allocated” error, see the [Troubleshooting](../usage/troubleshooting.md#web-server-ports-already-occupied) page.
* “invalid mount config for type "bind": bind mount source path does not exist: [some path]” means the filesystem isn’t successfully shared into the Docker container.
* If you’re seeing “The path (...) is not shared and is not known to Docker”, find *File sharing* in your Docker settings make sure the appropriate path or drive is included.
* “Error response from daemon: Get registry-1.docker.io/v2/” may mean Docker isn’t running or you don’t have internet access. Try starting or restarting Docker, and confirm you have a working internet connection.
* If you’re seeing “403 authentication required” trying to [`ddev start`](../usage/commands.md#start), run `docker logout` and try again. Docker authentication is *not* required for any normal DDEV action.
