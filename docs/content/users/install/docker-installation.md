# Docker Installation

You’ll need a Docker provider on your system before you can [install DDEV](ddev-installation.md).

=== "macOS"

    ## macOS

    Install one of the supported Docker providers:

    * [OrbStack](#orbstack): Recommended, easiest to install, most performant, commercial, not open-source.
    * [Lima](#lima): Free, open-source.
    * [Docker Desktop](#docker-desktop-for-mac): Familiar, popular, not open-source, may require license, may be unstable.
    * [Rancher Desktop](#rancher-desktop): Free, open-source, simple installation, slower startup.
    * [Colima](#colima): Free, open-source. Depends on separate Lima installation (managed by Homebrew).

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
    limactl create --name=default --vm-type=vz --mount-type=virtiofs --mount-writable --memory=6 --cpus=4 --disk=100 template:docker
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

    Rancher Desktop is an easy-to-install free and open-source Docker provider. Install from [Rancher Desktop](https://rancherdesktop.io/). It has automated testing with DDEV. When installing, choose only the Docker option and turn off Kubernetes.


    ### Colima

    [Colima](https://github.com/abiosoft/colima) is a free and open-source project which bundles Lima.

    1. Install Colima with `brew install colima`, which also installs Lima and other dependencies.
    2. If you don't have the `docker` client (if `docker help` fails) then install it with `brew install docker`.
    3. Start Colima with 4 CPUs, 6GB memory, 100GB storage, and Cloudflare DNS, adjusting as needed:

        ```bash
        colima start --cpu 4 --memory 6 --disk 100 --vm-type=vz --mount-type=virtiofs --dns=1.1.1.1
        ```

    After the initial run above, you can use `colima start` or use `colima start -e` to edit the configuration file. Run `colima status` at any time to check Colima’s status.

    !!!warning "Docker contexts let the Docker client point at the right Docker server"
        Colima activates its own Docker context to prevent conflicts with Docker Desktop. If you run `docker context ls`, you’ll see a list of available contexts where the currently-active one is indicated with a `*`—which will be `colima` after you’ve started it. You can change to the default (Docker Desktop) with `docker context use default` or change back with `docker context use colima`. This means you can run Docker Desktop and Colima at the same time, but be mindful of which context you’re pointing at!

    !!!warning "Colima can only work in your home directory unless you do further configuration"
        By default, Colima only works with DDEV projects in your home directory. You need to have your projects somewhere in your home directory for DDEV to work unless you do additional configuration. See the `~/.colima/default/colima.yaml` for more information, or notes in [colima.yaml](https://github.com/abiosoft/colima/blob/main/embedded/defaults/colima.yaml#L160-L173).

    ### Docker Buildx

    DDEV requires the [Docker buildx plugin](https://github.com/docker/buildx). Most Docker providers (OrbStack, Docker Desktop, Rancher Desktop, Colima, Lima) already include it. If `docker buildx version` doesn't work, install it with:

    ```bash
    brew install docker-buildx
    ```

    #### Migrating Projects Between Docker Providers

    * OrbStack has built-in migration of images and volumes from Docker Desktop.
    * Move projects between other Docker providers using [How can I migrate from one Docker provider to another?](../usage/faq.md#how-can-i-migrate-from-one-docker-provider-to-another).

=== "Linux"

    ## Linux

    * [Docker](#docker-for-linux): Recommended, free, open-source, best performance and stability.
    * [Docker rootless](#docker-rootless): Experimental support, free, open-source, more complex setup, has limitations.
    * [Podman](#podman-for-linux): Experimental support, free, open-source, more complex setup, has limitations.
    * [Docker Desktop](#docker-desktop-for-linux): Not open-source, may require license, may be unstable, doesn't have explicit support from DDEV and doesn't have automated test coverage.

    ### Docker for Linux

    The best way to install Docker on Linux is to use your native package management tool (`apt`, `dnf`, etc.) with the official Docker repository. While many Linux distributions provide Docker packages in their own repositories, these are often outdated and may not include the latest features required for stability in a development environment like DDEV. To ensure you're using a supported version, install Docker directly from the official Docker repository. 

    Follow these distribution-specific instructions to set up Docker Engine from the official Docker repository:

    * [Ubuntu](https://docs.docker.com/install/linux/docker-ce/ubuntu/)
    * [CentOS](https://docs.docker.com/install/linux/docker-ce/centos/)
    * [Debian](https://docs.docker.com/install/linux/docker-ce/debian/)
    * [Fedora](https://docs.docker.com/install/linux/docker-ce/fedora/)
    * [binaries](https://docs.docker.com/install/linux/docker-ce/binaries/)

    Linux installation **absolutely** requires adding your Linux user to the `docker` group, and configuring the Docker daemon to start at boot. See [Post-installation steps for Linux](https://docs.docker.com/engine/install/linux-postinstall/).

    !!!warning "Don’t `sudo` with `docker` or `ddev`"
        Don’t use `sudo` with the `docker` command. If you find yourself needing it, you haven’t finished the installation. You also shouldn’t use `sudo` with `ddev` unless it’s specifically for the [`ddev hostname`](../usage/commands.md#hostname) command.

    On systems without `systemd` or its equivalent—mostly if you’re installing inside WSL2—you’ll need to manually start Docker with `service docker start` or the equivalent in your distro. You can add this to your shell profile.

    ### Docker Buildx

    DDEV requires the [Docker buildx plugin](https://github.com/docker/buildx). Docker CE installed from the official Docker repository includes it. If `docker buildx version` doesn't work, install the `docker-buildx-plugin` package from the Docker repository (e.g. `sudo apt-get install docker-buildx-plugin` on Debian/Ubuntu) or follow any of the installation instructions from [Docker buildx](https://github.com/docker/buildx#installing).

    ### Docker Rootless

    DDEV has experimental support for Docker rootless mode on Linux. See [Podman and Docker Rootless in DDEV](https://ddev.com/blog/podman-and-docker-rootless/) for details.

    ### Podman for Linux

    DDEV has experimental support for Podman on Linux. See [Podman and Docker Rootless in DDEV](https://ddev.com/blog/podman-and-docker-rootless/) for details.

    ### Docker Desktop for Linux

    Docker Desktop for Linux can be downloaded from [docker.com](https://www.docker.com/products/docker-desktop). Casual manual testing seems to work, but DDEV does not have explicit support for it and does not have automated testing.

=== "Windows"

    ## Windows

    For initial installation of DDEV on Windows, you can use one of the following Docker providers:

    * Docker CE inside WSL2 - The most popular, performant, and best-supported way to run DDEV on Windows. No additional software is required; Docker CE will be installed by the DDEV installer. This approach does not work for traditional Windows (non-WSL2) installations.
    * Docker Desktop for Windows - A popular choice that works with both traditional Windows and WSL2. It has extensive automated testing with DDEV, but has some performance and reliability problems. It is not open-source and may require a license for many users. This approach works for both traditional Windows and WSL2 installations.
    * Rancher Desktop for Windows - A free and open-source Docker provider that has been manually tested with DDEV on traditional Windows, but does not have automated testing. This approach works for both traditional Windows and WSL2 installations.

    ### Using the Windows Installer

    The easiest way to install DDEV on Windows is to use the Windows installer, which can handle different installation scenarios:

    1. **Download the Windows installer**. Make sure to download the correct installer for your system architecture:

        <div style="display: grid; grid-template-columns: repeat(auto-fit, minmax(200px, 1fr)); gap: 1rem; text-align: center;">
        [Download for AMD64 :material-microsoft-windows:<br>(Intel/AMD) :material-download:](https://ddev.com/download/ddev_windows_amd64_installer.exe "For computers with Intel or AMD processors (most common)"){ .md-button .md-button--primary }
        [Download for ARM64 :material-microsoft-windows:<br>(Snapdragon) :material-download:](https://ddev.com/download/ddev_windows_arm64_installer.exe "For computers with ARM processors like Qualcomm Snapdragon"){ .md-button .md-button--primary }
        </div>

        - **AMD64 (x86-64)**: Most traditional Windows PCs (Intel/AMD processors)
        - **ARM64**: Windows on ARM devices like Microsoft Surface Pro X, Surface Pro 9 (5G), or other ARM-based Windows devices

        Or browse all versions on the [DDEV releases page](https://github.com/ddev/ddev/releases).

        !!!tip "Check your system architecture"
            Not sure which architecture you have? Open PowerShell and run: `$env:PROCESSOR_ARCHITECTURE`. It will show `AMD64` or `ARM64`. Alternatively, in WSL2/Ubuntu run `uname -m` which shows `x86_64` for AMD64 or `aarch64` for ARM64.

    2. **Run the installer** and choose your installation type:
       - **Docker CE inside WSL2** (Recommended): The installer will automatically install Docker CE in your WSL2 environment. This is the fastest and most reliable option.
       - **Docker Desktop**: Choose this if you already have Docker Desktop installed or prefer to use it.
       - **Rancher Desktop**: Choose this if you already have Rancher Desktop installed.
       - **Traditional Windows**: Choose this for non-WSL2 installations (requires Docker Desktop or Rancher Desktop).

    The installer will automatically configure DDEV for your chosen Docker provider and WSL2 environment.

    ### Manual Installation Options

    If you prefer to install manually or need more control over the installation process, you can use the following approaches:

    #### Docker CE inside WSL2

    No additional software is required to run DDEV on Windows with Docker CE inside WSL2. The DDEV installer will install Docker CE for you. This is the most popular, performant, and best-supported way to run DDEV on Windows.

    To install manually:
    1. Install or update your Ubuntu-based WSL2 distro (Ubuntu, Ubuntu-20.04, Ubuntu-22.04, etc.)
    2. Install DDEV inside your WSL2 environment using the Linux installation instructions
    3. DDEV will automatically install Docker CE for you on first run

    #### Docker Desktop for Windows

    1. Download and install Docker Desktop from [docker.com](https://www.docker.com/products/docker-desktop).
    2. During installation, ensure "Use WSL 2 instead of Hyper-V" is selected.
    3. After installation, open Docker Desktop settings and navigate to **Resources → WSL Integration**.
    4. Enable integration with your Ubuntu-based WSL2 distro (e.g., Ubuntu, Ubuntu-24.04, Ubuntu-22.04).
    5. Apply the changes and restart Docker Desktop if prompted.
    6. Verify that `docker ps` works in git-bash, PowerShell, or WSL2, wherever you're using it.

    #### Rancher Desktop for Windows

    1. Download and install [Rancher Desktop](https://rancherdesktop.io/).
    2. During installation, choose **Docker** as the container runtime and disable Kubernetes.
    3. After installation, open Rancher Desktop and go to **WSL Integration** in the settings.
    4. Enable integration with your Ubuntu-based WSL2 distro (e.g., Ubuntu, Ubuntu-24.04, Ubuntu-22.04).
    5. Apply the changes and restart Rancher Desktop if needed.
    6. Verify that `docker ps` works in git-bash, PowerShell, or WSL2, wherever you're using it.

    ### Docker Buildx

    DDEV requires the [Docker buildx plugin](https://github.com/docker/buildx). Docker CE inside WSL2, Docker Desktop, and Rancher Desktop all include it. If `docker buildx version` doesn't work, install the `docker-buildx-plugin` package inside WSL2 (e.g. `sudo apt-get install docker-buildx-plugin` on Ubuntu).

=== "Codespaces"

    ## GitHub Codespaces

    You can set up [GitHub Codespaces](https://github.com/features/codespaces) following the instructions in the [DDEV Installation](ddev-installation.md#github-codespaces) section.

<a name="troubleshooting"></a>

## Troubleshooting Docker

### Common Connection Errors

> `Cannot connect to the Docker daemon at unix:///var/run/docker.sock. Is the docker daemon running?`

A message like this can mean that your Docker provider hasn't been started (or hasn't been installed). It can also mean that the wrong Docker context is selected, which happens sometimes when people are switching to a new Docker provider. `docker context ls` will show you the available contexts and which one is in use.

But this error often indicates that either Docker is not installed or the Docker daemon is not running:

- **macOS/Traditional Windows:** Start your Docker provider (e.g., OrbStack, Docker Desktop, Rancher Desktop) from your applications menu
- **Linux/WSL2:** Run `sudo systemctl enable --now docker` to start and enable it to start automatically on boot

> `error during connect: Get "http://host:2375/v1.51/version": dial tcp: lookup host on 127.0.0.53:53: server misbehaving`

or

> `unable to resolve docker endpoint: context "docker-desktop": context not found`

These errors indicate Docker context issues:

- List available contexts: `docker context ls`
- Switch to default (or different) context: `docker context use default`
- Validate the current context: `docker ps`

!!!warning "Environment variables override Docker context"
    If you have set `DOCKER_HOST` and/or `DOCKER_CONTEXT` environment variables, they will override the `docker context` settings. This can lead to connection issues if the host is unreachable or the specified context is incorrect. Check your shell profile (`~/.bashrc`, `~/.zshrc`) for these variables.

!!!tip "Creating a Docker context"
    You can create a new Docker context using the `docker context create` command. See [Remote Docker Environments](../topics/remote-docker.md) for examples.

### Testing Docker Setup

For DDEV to work properly, Docker needs to:

- Mount your project code directory (typically under your home folder) from host to container
- Access TCP ports on the host for HTTP/HTTPS (default ports 80 and 443, configurable per project)

Run this command in your *project directory* to verify Docker configuration (use Git Bash if you're on Windows!):

```bash
docker run --rm -t -p 80:80 -p 443:443 -v "//$PWD:/tmp/projdir" ddev/ddev-utilities sh -c "echo ---- Project Directory && ls /tmp/projdir"
```

**Expected result:** A list of files in the current directory.

Common test command issues:

> `port is already allocated`

Another service is using ports 80 or 443. See the [Web Server Ports Troubleshooting](../usage/troubleshooting.md#web-server-ports-already-occupied) section.

> `invalid mount config for type "bind": bind mount source path does not exist`

The filesystem path isn't properly shared with Docker:

- **Docker Desktop:** Go to Settings → Resources → File Sharing and add your project directory or drive
- **Linux:** Ensure proper permissions on the project directory

> `The path (...) is not shared and is not known to Docker`

**Docker Desktop:** Add the path in Settings → Resources → File Sharing and restart Docker Desktop after making changes.

> `Error response from daemon: Get registry-1.docker.io/v2/`

Docker daemon isn't running, or no internet connection:

- Start/restart Docker and verify internet connectivity
- Check if corporate firewall blocks Docker Hub access

> `403 authentication required` during `ddev start`

Stale Docker Hub authentication interfering with public image pulls:

- **Solution:** Run `docker logout` and try again
- **Note:** Docker authentication is *not* required for normal DDEV operations
