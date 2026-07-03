# Docker Installation

You’ll need a Docker provider on your system before you can [install DDEV](ddev-installation.md).

=== "macOS"

    ## macOS

    Choose a container runtime:

    * **Docker** (recommended) - The best-tested option. Install OrbStack, Lima, Docker Desktop, Rancher Desktop, or Colima.
    * **Podman rootless** - Rootless by default; a good fit where Docker is forbidden. Can't use the default ports 80/443, so DDEV must be configured to use unprivileged ports.

    === "Docker"

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

        ### Migrating Projects Between Docker Providers

        * OrbStack has built-in migration of images and volumes from Docker Desktop.
        * Move projects between other Docker providers using [How can I migrate from one Docker provider to another?](../usage/faq.md#how-can-i-migrate-from-one-docker-provider-to-another).

    === "Podman rootless"

        Podman on macOS runs containers in a lightweight VM managed by `podman machine`. You can also use [Podman Desktop](https://podman-desktop.io/) for a GUI.

        !!!warning "Podman on macOS can't use ports 80/443"
            Podman on macOS cannot bind to privileged ports (80/443), so you must configure DDEV to use unprivileged ports. After installing DDEV, run:

            ```bash
            ddev config global --router-http-port=8080 --router-https-port=8443
            ```

            Your projects will then be reachable at `https://yourproject.ddev.site:8443` instead of the standard `https://yourproject.ddev.site`.

        ### Install Podman

        Install Podman using Homebrew:

        ```bash
        brew install podman
        ```

        Or install [Podman Desktop](https://podman-desktop.io/docs/installation/macos-install) if you prefer a GUI. For more information, see the [official Podman installation guide for macOS](https://podman.io/docs/installation#macos) and the [Podman tutorials](https://github.com/containers/podman/tree/main/docs/tutorials#readme).

        ### Install the Docker CLI

        Podman provides a Docker-compatible API, so you can use the Docker CLI as a front-end for Podman. This lets you use familiar `docker` commands, switch between runtimes with Docker contexts, and keep compatibility with tools that expect the `docker` command.

        ```bash
        brew install docker
        ```

        ### Initialize and start the Podman machine

        ```bash
        # check `podman machine init -h` for more options
        podman machine init --provider applehv
        podman machine start
        ```

        ### Point the Docker CLI at Podman

        ```bash
        SOCKET=$(podman machine inspect | jq -r '.[0].ConnectionInfo.PodmanSocket.Path')
        docker context create podman-rootless \
            --description "Podman (rootless)" \
            --docker "host=unix://${SOCKET}"
        docker context use podman-rootless
        docker ps
        ```

        ### Reclaim disk space

        !!!warning "The Podman machine doesn't reclaim its own disk space"
            The `podman machine` VM grows over time and does not shrink automatically, even after you remove images and containers. Periodically reclaim the freed space by trimming the VM's filesystem:

            ```bash
            podman machine start
            podman machine ssh 'sudo fstrim -av'
            ```

=== "Linux"

    ## Linux

    Choose a container runtime:

    * **Docker** (recommended) - The best-tested option, with the best performance and stability. Install Docker CE or Docker Desktop.
    * **Docker rootless** - Runs the Docker daemon without root, which reduces the attack surface. Full Docker compatibility.
    * **Podman rootless** - Rootless by default; a good fit where Docker is forbidden. More setup and slower than Docker.

    === "Docker"

        Install one of the supported Docker providers:

        * [Docker CE](#docker-for-linux): Recommended, free, open-source, best performance and stability. Install from the official Docker repository.
        * [Docker Desktop for Linux](#docker-desktop-for-linux): Not explicitly supported by DDEV and has no automated testing.

        ### Docker for Linux

        The best way to install Docker on Linux is to use your native package management tool (`apt`, `dnf`, etc.) with the official Docker repository. While many Linux distributions provide Docker packages in their own repositories, these are often outdated and may not include the latest features required for stability in a development environment like DDEV. To ensure you’re using a supported version, install Docker directly from the official Docker repository.

        #### Debian/Ubuntu

        ```bash
        # Ensure sudo credentials are cached for copy/paste of this block
        sudo true

        # Remove any old/conflicting Docker packages
        sudo apt-get remove -y docker.io docker-doc docker-compose podman-docker containerd runc 2>/dev/null || true

        # Install prerequisites
        sudo apt-get update && sudo apt-get install -y ca-certificates curl
        sudo install -m 0755 -d /etc/apt/keyrings

        # Add Docker’s GPG key
        sudo curl -fsSL https://download.docker.com/linux/$(. /etc/os-release && echo "$ID")/gpg -o /etc/apt/keyrings/docker.asc
        sudo chmod a+r /etc/apt/keyrings/docker.asc

        # Remove old repository files if present
        sudo rm -f /etc/apt/keyrings/docker.gpg /etc/apt/sources.list.d/docker.list

        # Add Docker repository in deb822 format
        printf "Types: deb\nURIs: https://download.docker.com/linux/%s\nSuites: %s\nComponents: stable\nSigned-By: /etc/apt/keyrings/docker.asc\n" \
          "$(. /etc/os-release && echo "$ID")" \
          "$(. /etc/os-release && echo "${UBUNTU_CODENAME:-$VERSION_CODENAME}")" \
          | sudo tee /etc/apt/sources.list.d/docker.sources >/dev/null

        # Install Docker CE
        sudo apt-get update && sudo apt-get install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin

        # Add your user to the docker group (create group if it doesn’t exist)
        sudo groupadd -f docker && sudo usermod -aG docker ${SUDO_USER:-$USER}

        # Start and enable Docker
        sudo systemctl enable --now docker
        ```

        Run `sg docker` to open a new subshell with the `docker` group active, then verify with `docker run hello-world`. (A full log out and back in applies the change to your whole session.)

        ??? tip "Prefer to run as a script?"
            Create a script file, then run it:

            ```bash
            cat > /tmp/install-docker.sh << 'SCRIPT'
            #!/usr/bin/env bash
            set -euo pipefail
            sudo true
            sudo apt-get remove -y docker.io docker-doc docker-compose podman-docker containerd runc 2>/dev/null || true
            sudo apt-get update && sudo apt-get install -y ca-certificates curl
            sudo install -m 0755 -d /etc/apt/keyrings
            sudo curl -fsSL "https://download.docker.com/linux/$(. /etc/os-release && echo "$ID")/gpg" -o /etc/apt/keyrings/docker.asc
            sudo chmod a+r /etc/apt/keyrings/docker.asc
            sudo rm -f /etc/apt/keyrings/docker.gpg /etc/apt/sources.list.d/docker.list
            printf "Types: deb\nURIs: https://download.docker.com/linux/%s\nSuites: %s\nComponents: stable\nSigned-By: /etc/apt/keyrings/docker.asc\n" \
              "$(. /etc/os-release && echo "$ID")" \
              "$(. /etc/os-release && echo "${UBUNTU_CODENAME:-$VERSION_CODENAME}")" \
              | sudo tee /etc/apt/sources.list.d/docker.sources >/dev/null
            sudo apt-get update && sudo apt-get install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
            sudo groupadd -f docker && sudo usermod -aG docker ${SUDO_USER:-$USER}
            sudo systemctl enable --now docker
            SCRIPT
            ```

            Review the script, then run it: `bash /tmp/install-docker.sh`

        See the full [Docker Engine installation docs for Ubuntu](https://docs.docker.com/engine/install/ubuntu/) or [Debian](https://docs.docker.com/engine/install/debian/) for more details.

        #### Other Linux Distributions

        Follow these distribution-specific instructions to set up Docker Engine from the official Docker repository:

        * [CentOS](https://docs.docker.com/install/linux/docker-ce/centos/)
        * [Fedora](https://docs.docker.com/install/linux/docker-ce/fedora/)
        * [binaries](https://docs.docker.com/install/linux/docker-ce/binaries/)

        After installing Docker, add your user to the `docker` group and enable the Docker daemon to start at boot:

        ```bash
        sudo groupadd -f docker && sudo usermod -aG docker ${SUDO_USER:-$USER}
        sudo systemctl enable --now docker
        ```

        Run `sg docker` to open a new subshell with the `docker` group active, or log out and back in to apply the change to your whole session. See [Post-installation steps for Linux](https://docs.docker.com/engine/install/linux-postinstall/) for more details.

        !!!warning "Don’t `sudo` with `docker` or `ddev`"
            Don’t use `sudo` with the `docker` command. If you find yourself needing it, you haven’t finished the installation. You also shouldn’t use `sudo` with `ddev` unless it’s specifically for the [`ddev hostname`](../usage/commands.md#hostname) command.

        On systems without `systemd` or its equivalent—mostly if you’re installing inside WSL2—you’ll need to manually start Docker with `service docker start` or the equivalent in your distro. You can add this to your shell profile.

        ### Docker Desktop for Linux

        As an alternative to Docker CE, Docker Desktop for Linux can be downloaded from [docker.com](https://www.docker.com/products/docker-desktop). Casual manual testing seems to work, but DDEV does not have explicit support for it and does not have automated testing.

    === "Docker rootless"

        [Docker rootless mode](https://docs.docker.com/engine/security/rootless/) runs the Docker daemon and containers as a non-root user, which reduces the attack surface (no root daemon, container processes can't touch root-owned files). DDEV supports Docker rootless on Linux and WSL2, with full Docker compatibility.

        !!!tip "`no_bind_mounts` is no longer required"
            Earlier DDEV versions (v1.25.0-v1.25.2) required [`no_bind_mounts`](../configuration/config.md#no_bind_mounts) with Docker rootless. That is no longer necessary—Docker rootless now works with bind mounts.

            With bind mounts under Docker rootless, you appear as `root` inside the web container (for example, in [`ddev ssh`](../usage/commands.md#ssh)). This is expected: rootless maps container `root` to your regular host user, so it has no special privileges on the host. If you prefer not to work as `root` in the container, run `ddev config global --no-bind-mounts=true`. This enables [`no_bind_mounts`](../configuration/config.md#no_bind_mounts), which relies on [Mutagen](performance.md#mutagen) to sync files rather than mounting them directly.

        ### Install Docker rootless

        Follow the official [Docker rootless installation guide](https://docs.docker.com/engine/security/rootless/).

        ### Configure the system

        ```bash
        # Allow privileged port access if needed
        if [ -f /proc/sys/net/ipv4/ip_unprivileged_port_start ]; then
          if [ "1024" = "$(cat /proc/sys/net/ipv4/ip_unprivileged_port_start)" ]; then
            echo 'net.ipv4.ip_unprivileged_port_start=0' | sudo tee -a /etc/sysctl.d/60-rootless.conf
            sudo sysctl --system
          fi
        fi
        # Allow loopback connections (needed for working Xdebug)
        # See https://github.com/moby/moby/issues/47684#issuecomment-2166149845
        mkdir -p ~/.config/systemd/user/docker.service.d
        cat << 'EOF' > ~/.config/systemd/user/docker.service.d/override.conf
        [Service]
        Environment="DOCKERD_ROOTLESS_ROOTLESSKIT_DISABLE_HOST_LOOPBACK=false"
        EOF
        ```

        ### Enable the Docker socket

        ```bash
        systemctl --user enable --now docker.socket

        # You should see `/run/user/1000/docker.sock` (the number may vary):
        ls $XDG_RUNTIME_DIR/docker.sock
        ```

        ### Point the Docker CLI at the rootless socket

        ```bash
        # View existing contexts
        docker context ls

        # Create the rootless context if it doesn't exist
        docker context inspect rootless >/dev/null 2>&1 || \
            docker context create rootless \
                --description "Rootless runtime socket" \
                --docker host="unix://$XDG_RUNTIME_DIR/docker.sock"

        # Switch to the context
        docker context use rootless

        # Verify it works
        docker ps
        ```

    === "Podman rootless"

        [Podman](https://podman.io/) is rootless by default, which makes it a good fit for environments that forbid Docker or require rootless operation. Podman is slower than Docker and has more setup, but Podman rootless on Linux is solid.

        !!!warning "Podman versions"
            Some distributions ship outdated Podman. Ubuntu 24.04, for example, has Podman 4.9.3. DDEV works best with Podman 5.0 or newer; with Podman 4.x you can proceed by ignoring the warning on `ddev start`.

        ### Install Podman

        Install Podman with your distribution's package manager (see the [official Podman installation guide for Linux](https://podman.io/docs/installation#installing-on-linux)):

        ```bash
        # Ubuntu/Debian
        sudo apt-get update && sudo apt-get install podman
        # Fedora
        sudo dnf install --refresh podman
        ```

        You can also install [Podman Desktop](https://podman-desktop.io/docs/installation/linux-install) if you prefer a GUI. For more information, see the [Podman tutorials](https://github.com/containers/podman/tree/main/docs/tutorials#readme).

        ### Install the Docker CLI

        Podman provides a Docker-compatible API, so you can use the Docker CLI as a front-end for Podman. This lets you use familiar `docker` commands, switch between runtimes with Docker contexts, and keep compatibility with tools that expect the `docker` command.

        1. [Set up Docker's repository](https://docs.docker.com/engine/install/).
        2. Install only the CLI (you don't need `docker-ce`, the Docker engine):

            ```bash
            # Ubuntu/Debian
            sudo apt-get update && sudo apt-get install docker-ce-cli
            # Fedora
            sudo dnf install --refresh docker-ce-cli
            ```

        ### Configure Podman rootless

        This is the recommended configuration for most users.

        Prepare the system by configuring `subuid`/`subgid` ranges and enabling `userns` options (see the [Arch Linux Wiki](https://wiki.archlinux.org/title/Podman#Rootless_Podman) for details):

        ```bash
        # Add subuid and subgid ranges if they don't exist for the current user
        grep "^$(id -un):\|^$(id -u):" /etc/subuid >/dev/null 2>&1 || sudo usermod --add-subuids 100000-165535 $(whoami)
        grep "^$(id -un):\|^$(id -u):" /etc/subgid >/dev/null 2>&1 || sudo usermod --add-subgids 100000-165535 $(whoami)
        # Propagate changes to subuid and subgid
        podman system migrate
        # Debian requires setting unprivileged_userns_clone
        if [ -f /proc/sys/kernel/unprivileged_userns_clone ]; then
          if [ "1" != "$(cat /proc/sys/kernel/unprivileged_userns_clone)" ]; then
            echo 'kernel.unprivileged_userns_clone=1' | sudo tee -a /etc/sysctl.d/60-rootless.conf
            sudo sysctl --system
          fi
        fi
        # Fedora requires setting max_user_namespaces
        if [ -f /proc/sys/user/max_user_namespaces ]; then
          if [ "0" = "$(cat /proc/sys/user/max_user_namespaces)" ]; then
            echo 'user.max_user_namespaces=28633' | sudo tee -a /etc/sysctl.d/60-rootless.conf
            sudo sysctl --system
          fi
        fi
        # Allow privileged port access if needed
        if [ -f /proc/sys/net/ipv4/ip_unprivileged_port_start ]; then
          if [ "1024" = "$(cat /proc/sys/net/ipv4/ip_unprivileged_port_start)" ]; then
            echo 'net.ipv4.ip_unprivileged_port_start=0' | sudo tee -a /etc/sysctl.d/60-rootless.conf
            sudo sysctl --system
          fi
        fi
        ```

        Enable the Podman socket and verify it's running (see the [Podman socket activation documentation](https://github.com/containers/podman/blob/main/docs/tutorials/socket_activation.md)):

        ```bash
        systemctl --user enable --now podman.socket

        # You should see `/run/user/1000/podman/podman.sock` (the number may vary):
        ls $XDG_RUNTIME_DIR/podman/podman.sock

        # You can also check the socket path with:
        podman info --format '{{.Host.RemoteSocket.Path}}'
        ```

        Configure the Docker CLI to use Podman (see the [Podman rootless tutorial](https://github.com/containers/podman/blob/main/docs/tutorials/rootless_tutorial.md)):

        ```bash
        # View existing contexts
        docker context ls

        # Create Podman rootless context
        docker context create podman-rootless \
            --description "Podman (rootless)" \
            --docker host="unix://$XDG_RUNTIME_DIR/podman/podman.sock"

        # Switch to the new context
        docker context use podman-rootless

        # Verify it works
        docker ps
        ```

        ### Podman rootless performance optimization

        Podman rootless is slower than Docker (see [Podman run/build performance issues](https://github.com/containers/podman/issues/13226) and the [Podman performance documentation](https://github.com/containers/podman/blob/main/docs/tutorials/performance.md)). To improve performance, install `fuse-overlayfs` and configure the overlay storage driver:

        ```bash
        # Ubuntu/Debian
        sudo apt-get update && sudo apt-get install fuse-overlayfs
        # Fedora
        sudo dnf install --refresh fuse-overlayfs
        ```

        ```bash
        mkdir -p ~/.config/containers
        cat << 'EOF' > ~/.config/containers/storage.conf
        [storage]
        driver = "overlay"
        [storage.options.overlay]
        mount_program = "/usr/bin/fuse-overlayfs"
        EOF
        ```

        !!!warning "Existing containers require a reset"
            If you already have Podman containers, images, or volumes, you'll need to reset Podman for the storage change to take effect: `podman system reset`. This removes all existing containers, images, and volumes (similar to `docker system prune -a`).

=== "Windows"

    ## Windows

    First [install WSL2](#install-wsl2) below, then [choose a container runtime](#choose-a-container-runtime):

    * **Docker** (recommended) - The best-tested option. Run Docker CE inside WSL2, or use Docker Desktop or Rancher Desktop.
    * **Docker rootless** - Runs the Docker daemon without root, which reduces the attack surface. Runs inside WSL2 using the same setup as Linux.
    * **Podman rootless** - Rootless by default; a good fit where Docker is forbidden. Works via Podman Desktop, but is less mature on Windows than on Linux.

    ### Install WSL2

    In PowerShell, run:

    ```powershell
    # Install WSL2; reboot if prompted, then continue:
    wsl --install --no-distribution

    # Update WSL2 if previously installed:
    wsl --update
    ```

    Create a Debian-based WSL2 distro (skip if you're using Traditional Windows). Ubuntu is recommended:

    ```powershell
    # You'll be asked to set a username and password for the distro:
    wsl --install Ubuntu-26.04 --name DDEV
    ```

    !!!tip "\"DDEV\" is just a suggested name — use any name you like."

    !!!tip "Other Debian-based distros also work"
        The DDEV installer supports Ubuntu and Debian and has been tested on Kali Linux and eLxr. If you prefer one of those, substitute its name, for example `wsl --install Debian --name DDEV`.

    Verify the "DDEV" distro is set as default:

    ```powershell { .no-copy }
    > wsl -l -v
      NAME                   STATE           VERSION
    * DDEV                   Stopped         2
    ```

    ### Choose a container runtime

    === "Docker"

        Install one of the supported Docker providers:

        * [Docker CE inside WSL2](#docker-ce-inside-wsl2): Recommended, most popular, performant, and best-supported. The [DDEV installer](ddev-installation.md#windows) installs it automatically. Requires WSL2.
        * [Docker Desktop for Windows](#docker-desktop-for-windows): Works with both WSL2 and traditional Windows. Extensive automated testing, but some performance and reliability problems. Not open-source; may require a license.
        * [Rancher Desktop for Windows](#rancher-desktop-for-windows): Free, open-source. Manually tested on traditional Windows; no automated testing. Works with both WSL2 and traditional Windows.

        ### Docker CE inside WSL2

        The [DDEV Windows installer](ddev-installation.md#step-2-install-ddev) automatically installs Docker CE in your WSL2 environment — no manual Docker installation is needed. After installing WSL2 above, proceed to [install DDEV](ddev-installation.md#step-2-install-ddev).

        To install Docker CE manually instead, open your WSL2 terminal (Ubuntu) and run:

        ```bash
        # Ensure sudo credentials are cached for copy/paste of this block
        sudo true

        # Remove any old/conflicting Docker packages
        sudo apt-get remove -y docker.io docker-doc docker-compose podman-docker containerd runc 2>/dev/null || true

        # Install prerequisites
        sudo apt-get update && sudo apt-get install -y ca-certificates curl
        sudo install -m 0755 -d /etc/apt/keyrings

        # Add Docker's GPG key
        sudo curl -fsSL https://download.docker.com/linux/ubuntu/gpg -o /etc/apt/keyrings/docker.asc
        sudo chmod a+r /etc/apt/keyrings/docker.asc

        # Remove old repository files if present
        sudo rm -f /etc/apt/keyrings/docker.gpg /etc/apt/sources.list.d/docker.list

        # Add Docker repository in deb822 format
        printf "Types: deb\nURIs: https://download.docker.com/linux/ubuntu\nSuites: %s\nComponents: stable\nSigned-By: /etc/apt/keyrings/docker.asc\n" \
          "$(. /etc/os-release && echo "${UBUNTU_CODENAME:-$VERSION_CODENAME}")" \
          | sudo tee /etc/apt/sources.list.d/docker.sources >/dev/null

        # Install Docker CE
        sudo apt-get update && sudo apt-get install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin

        # Add your user to the docker group (create group if it doesn't exist)
        sudo groupadd -f docker && sudo usermod -aG docker ${SUDO_USER:-$USER}
        ```

        Run `sg docker` to open a new subshell with the `docker` group active, or log out and back in to apply the change to your whole session. On WSL2 systems without `systemd`, you may need to start Docker manually with `sudo service docker start`.

        ??? tip "Prefer to run as a script?"
            Create a script file, then run it:

            ```bash
            cat > /tmp/install-docker.sh << 'SCRIPT'
            #!/usr/bin/env bash
            set -euo pipefail
            sudo true
            sudo apt-get remove -y docker.io docker-doc docker-compose podman-docker containerd runc 2>/dev/null || true
            sudo apt-get update && sudo apt-get install -y ca-certificates curl
            sudo install -m 0755 -d /etc/apt/keyrings
            sudo curl -fsSL https://download.docker.com/linux/ubuntu/gpg -o /etc/apt/keyrings/docker.asc
            sudo chmod a+r /etc/apt/keyrings/docker.asc
            sudo rm -f /etc/apt/keyrings/docker.gpg /etc/apt/sources.list.d/docker.list
            printf "Types: deb\nURIs: https://download.docker.com/linux/ubuntu\nSuites: %s\nComponents: stable\nSigned-By: /etc/apt/keyrings/docker.asc\n" \
              "$(. /etc/os-release && echo "${UBUNTU_CODENAME:-$VERSION_CODENAME}")" \
              | sudo tee /etc/apt/sources.list.d/docker.sources >/dev/null
            sudo apt-get update && sudo apt-get install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
            sudo groupadd -f docker && sudo usermod -aG docker ${SUDO_USER:-$USER}
            SCRIPT
            ```

            Review the script, then run it: `bash /tmp/install-docker.sh`

        ### Docker Desktop for Windows

        1. Download and install Docker Desktop from [docker.com](https://www.docker.com/products/docker-desktop).
        2. During installation, ensure "Use WSL 2 instead of Hyper-V" is selected.
        3. After installation, open Docker Desktop settings and navigate to **Resources → WSL Integration**.
        4. Enable integration with your Debian-based WSL2 distro (e.g., Ubuntu-26.04, Debian).
        5. Apply the changes and restart Docker Desktop if prompted.
        6. Verify that `docker ps` works in git-bash, PowerShell, or WSL2, wherever you're using it.

        ### Rancher Desktop for Windows

        1. Download and install [Rancher Desktop](https://rancherdesktop.io/).
        2. During installation, choose **Docker** as the container runtime and disable Kubernetes.
        3. After installation, open Rancher Desktop and go to **WSL Integration** in the settings.
        4. Enable integration with your Debian-based WSL2 distro (e.g., Ubuntu-26.04, Debian).
        5. Apply the changes and restart Rancher Desktop if needed.
        6. Verify that `docker ps` works in git-bash, PowerShell, or WSL2, wherever you're using it.

    === "Docker rootless"

        Docker rootless runs inside WSL2 using the same setup as Linux. In your WSL2 distro, follow the [Docker rootless instructions for Linux](#linux-docker-rootless).

    === "Podman rootless"

        Windows users can run Podman via [Podman Desktop](https://podman-desktop.io/), which manages the WSL2-based VM for you. Podman on Windows works but is less mature than on Linux.

        ### Install Podman

        Install [Podman Desktop](https://podman-desktop.io/docs/installation/windows-install), which includes Podman. Alternatively, install Podman directly following the [official Podman installation guide for Windows](https://podman.io/docs/installation#windows). For more information, see the [Podman tutorials](https://github.com/containers/podman/tree/main/docs/tutorials#readme).

        ### Configure Podman

        Setup and configuration follow the same patterns as the Linux/WSL2 setup, but with Podman Desktop managing the VM for you. Follow the [Podman rootless instructions for Linux](#linux-podman-rootless).

=== "Codespaces"

    ## GitHub Codespaces

    You can set up [GitHub Codespaces](https://github.com/features/codespaces) following the instructions in the [DDEV Installation](ddev-installation.md#github-codespaces) section.

## Docker Buildx

DDEV requires the [Docker buildx plugin](https://github.com/docker/buildx). GUI-based providers (OrbStack, Docker Desktop, Rancher Desktop) bundle it. If `docker buildx version` doesn't work, install it:

- **macOS (Homebrew):** `brew install docker-buildx`
- **Linux/WSL2 (Debian/Ubuntu):** `sudo apt-get install docker-buildx-plugin` (requires the [official Docker repository](https://docs.docker.com/engine/install/) to be configured)
- **Other:** see [Docker buildx requirement](https://ddev.com/blog/docker-buildx-requirement-v1-25-1/) for installation instructions across platforms.

!!!tip "Can't install the plugin?"
    As a fallback, DDEV can manage its own copy via [`docker_buildx_version`](../configuration/config.md#docker_buildx_version).

## Running Multiple Container Runtimes

You can run Docker and Podman sockets simultaneously and switch between them using Docker contexts.

For example, here's a system with three active Docker contexts:

```bash
$ docker context ls
NAME                DESCRIPTION                               DOCKER ENDPOINT
default             Current DOCKER_HOST based configuration   unix:///var/run/docker.sock
podman-rootless *   Podman (rootless)                         unix:///run/user/1000/podman/podman.sock
rootless            Rootless runtime socket                   unix:///run/user/1000/docker.sock
```

Switch between them with:

```bash
docker context use "<context-name>"
```

## Switching Runtimes with DDEV

DDEV automatically detects your active container runtime. To switch:

1. Stop DDEV projects:

   ```bash
   ddev poweroff
   ```

2. Switch the Docker context or change the `DOCKER_HOST` environment variable.
3. Start your project:

   ```bash
   ddev start
   ```

For background on the trade-offs between these runtimes, see [Podman and Docker Rootless in DDEV](https://ddev.com/blog/podman-and-docker-rootless/).

<a name="troubleshooting"></a>

## Troubleshooting Docker

### Common Connection Errors

> `Cannot connect to the Docker daemon at unix:///var/run/docker.sock. Is the docker daemon running?`

A message like this can mean that your Docker provider hasn't been started (or hasn't been installed). It can also mean that the wrong Docker context is selected, which happens sometimes when people are switching to a new Docker provider. `docker context ls` will show you the available contexts and which one is in use.

But this error often indicates that either Docker is not installed or the Docker daemon is not running:

- **macOS/Traditional Windows:** Start your Docker provider (e.g., OrbStack, Docker Desktop, Rancher Desktop) from your applications menu
- **Linux/WSL2:** Run `sudo systemctl enable --now docker` to start and enable it to start automatically on boot

---

> `permission denied while trying to connect to the docker API at unix:///var/run/docker.sock`

A message like this means that your user is not in the `docker` group:

```bash
sudo groupadd -f docker && sudo usermod -aG docker ${SUDO_USER:-$USER}
# sg opens a new subshell with the `docker` group active;
# log out and back in to apply the change to your full session
sg docker
```

---

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

Run this command in your _project directory_ to verify Docker configuration (use Git Bash if you're on Windows!):

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
- **Note:** Docker authentication is _not_ required for normal DDEV operations
