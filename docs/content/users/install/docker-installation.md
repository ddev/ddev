# Docker Installation

You’ll need a Docker provider on your system before you can [install DDEV](ddev-installation.md).

=== "macOS"

    ## macOS

    Install one of the supported Docker providers:

    * [OrbStack](#orbstack): Recommended, easiest to install, most performant, commercial, not open-source.
    * [Lima](#lima): Free, open-source.
    * [Docker Desktop](#docker-desktop-for-mac): Familiar, popular, not open-source, may require license, may be unstable.
    * [Rancher Desktop](#rancher-desktop): Free, open-source, simple installation, slower startup.
    * [Colima](#colima): Free, open-source, no longer recommended.

    ### OrbStack

    [OrbStack](https://orbstack.dev) is a newer Docker provider that is very popular with DDEV users because it’s fast, lightweight, and easy to install. It’s a good choice for most users. It is *not* open-source, and it is not free for professional use.
    
    1. Install OrbStack with `brew install orbstack` or [download it directly](https://orbstack.dev/download).
    2. Run the OrbStack app (from Applications) to finish setup, choosing "Docker" as the option. Answer any prompts to allow OrbStack access.

    ### Lima

    [Lima](https://github.com/lima-vm/lima) is a free and open-source project supported by the [Cloud Native Computing Foundation](https://cncf.io/).

    1. Install Lima with `brew install lima`.
    2. If you don't have the `docker` client (if `docker help` fails) then install it with `brew install docker`.
    3. Create a 100GB VM in Lima with 4 CPUs, 6GB memory, and Cloudflare DNS. Adjust to your own needs:
    ```
    limactl create --name=default --vm-type=vz --mount-type=virtiofs --mount-writable --memory=6 --cpus=4 --disk=100 template://docker
    docker context create lima-default --docker "host=unix://$HOME/.lima/default/sock/docker.sock"
    docker context use lima-default
    ```
    After the initial run above, you can use `limactl start`.  Run `limactl list` to see configured setup.

    When your computer restarts, you’ll need to `limactl start` again.

    !!!warning "Docker contexts let the Docker client point at the right Docker server"
        The Docker provider you're using is selected with `docker context`. You can see the available contexts with `docker context ls` and the currently selected one with `docker context show`. With the setup above you'll want `docker context use lima-default`.

    !!!warning "Lima only mounts filesystems in your home directory unless you do further configuration"
        By default, Lima only works with DDEV projects in your home directory. You must have your projects somewhere in your home directory for DDEV to work unless you do additional configuration. If your project is not in your home directory, you must add additional mounts, as described in [mounts example](https://github.com/lima-vm/lima/blob/e9423da6b7c60083aaa455a0c6ecb5c729edfe1f/examples/docker.yaml#L25-L28).

    ### Docker Desktop for Mac

    Docker Desktop for Mac can be downloaded from [docker.com](https://www.docker.com/products/docker-desktop). It has long been supported by DDEV and has extensive automated testing. It is not open-source, may require a license for many users, and sometimes has stability problems.

    !!!warning "Ports unavailable?"
        If you get messages like `Ports are not available... exposing port failed... is vmnetd running?` it means you need to check the "Allow privileged port mapping (requires password)" checkbox in the "Advanced" section of the Docker Desktop configuration. You may have to stop and restart Docker Desktop, and you may have to turn it off, restart Docker Desktop, turn it on again, restart Docker Desktop. (More extensive problem resolution is in [Docker Desktop issue](https://github.com/docker/for-mac/issues/6677).)

    ### Rancher Desktop

    Rancher Desktop is an easy-to-install free and open-source Docker provider. Install from [Rancher Desktop.io](https://rancherdesktop.io/). It has automated testing with DDEV. When installing, choose only the Docker option and turn off Kubernetes.


    ### Colima

    [Colima](https://github.com/abiosoft/colima) is a free and open-source project which bundles Lima. It is no longer a top recommendation because it has been unstable. You can migrate to other providers using [How can I migrate from one Docker provider to another?](../usage/faq.md#how-can-i-migrate-from-one-docker-provider-to-another).

    1. Install Colima with `brew install colima`.
    2. If you don't have the `docker` client (if `docker help` fails) then install it with `brew install docker`.
    3. Start Colima with 4 CPUs, 6GB memory, 100GB storage, and Cloudflare DNS, adjusting as needed:
    ```
    colima start --cpu 4 --memory 6 --disk 100 --vm-type=vz --mount-type=virtiofs --dns=1.1.1.1
    ```
    (On macOS versions before Ventura, use `colima start --cpu 4 --memory 6 --disk 100 --mount-type=sshfs --dns=1.1.1.1` as the `--vm-type` option are not supported for older macOS versions.)

    After the initial run above, you can use `colima start` or use `colima start -e` to edit the configuration file. Run `colima status` at any time to check Colima’s status.

    !!!warning "Docker contexts let the Docker client point at the right Docker server"
        Colima activates its own Docker context to prevent conflicts with Docker Desktop. If you run `docker context ls`, you’ll see a list of available contexts where the currently-active one is indicated with a `*`—which will be `colima` after you’ve started it. You can change to the default (Docker Desktop) with `docker context use default` or change back with `docker context use colima`. This means you can run Docker Desktop and Colima at the same time, but be mindful of which context you’re pointing at!

    !!!warning "Colima can only work in your home directory unless you do further configuration"
        By default, Colima only works with DDEV projects in your home directory. You need to have your projects somewhere in your home directory for DDEV to work unless you do additional configuration. See the `~/.colima/default/colima.yaml` for more information, or notes in [colima.yaml](https://github.com/abiosoft/colima/blob/main/embedded/defaults/colima.yaml#L160-L173).

    #### Migrating Projects Between Docker Providers

    * OrbStack has built-in migration of images and volumes from Docker Desktop.
    * Move projects between other Docker providers using [How can I migrate from one Docker provider to another?](../usage/faq.md#how-can-i-migrate-from-one-docker-provider-to-another).

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

    The instructions for [DDEV Installation in WSL2](ddev-installation.md#wsl2-docker-ce-inside-install-script) include Docker CE setup and a script that does almost all the work. Please use those.

    ### Docker Desktop for Windows

    Docker Desktop for Windows can be downloaded via [Chocolatey](https://chocolatey.org/install) with `choco install docker-desktop` or it can be downloaded from [docker.com](https://www.docker.com/products/docker-desktop). It has extensive automated testing with DDEV, and works with DDEV both on traditional Windows and in WSL2.

    See [WSL2 DDEV Installation](ddev-installation.md#wsl2-docker-desktop-install-script) for help installing DDEV with Docker Desktop on WSL2.

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
