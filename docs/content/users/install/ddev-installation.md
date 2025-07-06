# DDEV Installation

Once you’ve [installed a Docker provider](docker-installation.md), you’re ready to install DDEV!

=== "macOS"

    ## macOS

    ### Homebrew

    [Homebrew](https://brew.sh/) is the easiest and most reliable way to install and [upgrade](./ddev-upgrade.md) DDEV:

    ```bash
    # Install DDEV
    brew install ddev/ddev/ddev

    # One-time initialization of mkcert
    mkcert -install
    ```

    For unusual browsers and situations that don't automatically support the `mkcert` certificate authority, [configure your browser](configuring-browsers.md).

    ### Install Script

    The [install script](https://github.com/ddev/ddev/blob/main/scripts/install_ddev.sh) is another option. It downloads, verifies, and sets up the `ddev` executable:

    ```bash
    # Download and run the install script
    curl -fsSL https://ddev.com/install.sh | bash
    ```

    For unusual browsers and situations that don't automatically support the `mkcert` certificate authority, [configure your browser](configuring-browsers.md).

    ??? "Do you still have an old version after installing or upgrading?"
        If `ddev --version` still shows an older version than you installed or upgraded to, use `which -a ddev` to find out where another version of the `ddev` executable must be installed. See the ["Why Do I Have An Old DDEV" FAQ](../usage/faq.md#why-do-i-have-an-old-ddev).
    ??? "Need a specific version?"
        Use the `-s` argument to specify a specific stable or prerelease version:

        ```bash
        # Download and run the script to install DDEV v1.23.5
        curl -fsSL https://ddev.com/install.sh | bash -s v1.23.5
        ```

=== "Linux"

    ## Linux

    ### Debian/Ubuntu

    DDEV’s Debian and RPM packages work with `apt` and `yum` repositories and most variants that use them, including Windows WSL2:

    ```bash
    # Add DDEV’s GPG key to your keyring
    sudo sh -c 'echo ""'
    sudo apt-get update && sudo apt-get install -y curl
    sudo install -m 0755 -d /etc/apt/keyrings
    curl -fsSL https://pkg.ddev.com/apt/gpg.key | gpg --dearmor | sudo tee /etc/apt/keyrings/ddev.gpg > /dev/null
    sudo chmod a+r /etc/apt/keyrings/ddev.gpg

    # Add DDEV releases to your package repository
    sudo sh -c 'echo ""'
    echo "deb [signed-by=/etc/apt/keyrings/ddev.gpg] https://pkg.ddev.com/apt/ * *" | sudo tee /etc/apt/sources.list.d/ddev.list >/dev/null

    # Update package information and install DDEV
    sudo sh -c 'echo ""'
    sudo apt-get update && sudo apt-get install -y ddev

    # One-time initialization of mkcert
    mkcert -install
    ```

    For unusual browsers and situations that don't automatically support the `mkcert` certificate authority, [configure your browser](configuring-browsers.md).

    ??? "Do you still have an old version after installing or upgrading?"
        If `ddev --version` still shows an older version than you installed or upgraded to, use `which -a ddev` to find out where another version of the `ddev` executable must be installed. See the ["Why Do I Have An Old DDEV" FAQ](../usage/faq.md#why-do-i-have-an-old-ddev).

    ??? "Need to remove a previously-installed variant?"
        If you previously used DDEV’s [install script](#install-script), you can remove that version:

        ```
        sudo rm -f /usr/local/bin/ddev /usr/local/bin/mkcert /usr/local/bin/*ddev_nfs_setup.sh
        ```

        If you previously [installed DDEV with Homebrew](#homebrew), you can run `brew unlink ddev` to get rid of the Homebrew version.

    ### Fedora, Red Hat, etc.

    ```bash
    # Add DDEV releases to your package repository
    sudo sh -c 'echo ""'
    echo '[ddev]
    name=ddev
    baseurl=https://pkg.ddev.com/yum/
    gpgcheck=0
    enabled=1' | perl -p -e 's/^ +//' | sudo tee /etc/yum.repos.d/ddev.repo >/dev/null

    # Install DDEV
    sudo sh -c 'echo ""'
    sudo dnf install --refresh ddev

    # One-time initialization of mkcert
    mkcert -install
    ```

    For unusual browsers and situations that don't automatically support the `mkcert` certificate authority, [configure your browser](configuring-browsers.md).

    Signed yum repository support will be added in the future.

    ### Arch Linux

    We maintain the [ddev-bin](https://aur.archlinux.org/packages/ddev-bin/) package in AUR for Arch-based systems including Arch Linux, EndeavourOS and Manjaro. Install with `yay` or your AUR tool of choice.

    ```bash
    # Install DDEV
    yay -S ddev-bin

    # One-time initialization of mkcert
    mkcert -install
    ```

    For unusual browsers and situations that don't automatically support the `mkcert` certificate authority, [configure your browser](configuring-browsers.md).

    ### Homebrew (AMD64 only)

    ```bash
    # Install DDEV using Homebrew
    brew install ddev/ddev/ddev

    # One-time initialization of mkcert
    mkcert -install
    ```

    For unusual browsers and situations that don't automatically support the `mkcert` certificate authority, [configure your browser](configuring-browsers.md).

    <!-- we’re using HTML here to customize the #install-script-linux anchor -->
    <h3 id="install-script-linux">Install Script<a class="headerlink" href="#install-script-linux" title="Permanent link">¶</a></h3>

    The [install script](https://github.com/ddev/ddev/blob/main/scripts/install_ddev.sh) is another option. It downloads, verifies, and sets up the `ddev` executable:

    ```bash
    # Download and run the install script
    curl -fsSL https://ddev.com/install.sh | bash
    ```

    For unusual browsers and situations that don't automatically support the `mkcert` certificate authority, [configure your browser](configuring-browsers.md).

    ??? "Need a specific version?"
        Use the `-s` argument to specify a specific stable or prerelease version:

        ```bash
        # Download and run the script to install DDEV v1.23.5
        curl -fsSL https://ddev.com/install.sh | bash -s v1.23.5
        ```

    ??? "Do you still have an old version after installing or upgrading?"
        If `ddev --version` still shows an older version than you installed or upgraded to, use `which -a ddev` to find out where another version of the `ddev` executable must be installed. See the ["Why Do I Have An Old DDEV" FAQ](../usage/faq.md#why-do-i-have-an-old-ddev).

=== "Windows"

    ## Windows

    You can install DDEV on Windows using the Windows installer or manual installation:

    ### Important Considerations for WSL2 and DDEV

    * You **must** use WSL2, not WSL version 1.  
      Use `wsl.exe -l -v` to see the versions of the distros you are using they should be v2.
    * WSL2 is supported on Windows 10 and 11.  
      All Windows 10/11 editions, including Windows 10 Home support WSL2.
    * WSL2 offers a faster, smoother experience.  
      It’s vastly more performant, and you’re less likely to have obscure Windows problems.

    * Execute DDEV commands inside WSL2.  
      You’ll want to run DDEV commands inside Ubuntu, for example, and never on the Windows side in PowerShell or Git Bash.
    * Projects should live under the home directory of the Linux filesystem.  
      WSL2’s Linux filesystem (e.g. `/home/<your_username>`) is much faster and has proper permissions, so keep your projects there and **not** in the slower Windows filesystem (`/mnt/c`).
    * Custom hostnames (non `ddev.site` hostnames) are managed via the Windows hosts file, not within WSL2.  
      DDEV attempts to manage custom hostnames via the Windows-side hosts file—usually at `C:\Windows\system32\drivers\etc\hosts`.

    The WSL2 install process involves:

    * Installing WSL2 and installing an Ubuntu distro (or some other distro if you're installing manually).
    * Optionally installing Docker Desktop or Rancher Desktop for Windows and enabling WSL2 integration with the distro (if you're using the Docker/Rancher Desktop approach).
    * Running the DDEV installer and choosing the approach you want.

    ### Windows Installer

    The easiest way to install DDEV on Windows is to use the DDEV Windows installer, which can handle different installation scenarios:

    1. **Download the Windows installer** from the [DDEV releases page](https://github.com/ddev/ddev/releases).
    2. **Run the installer** and choose your installation type:

           - **Docker CE inside WSL2** (Recommended): The installer will automatically install Docker CE in your WSL2 environment. This is the fastest and most reliable option.
           - **Docker Desktop/Rancher Desktop**: Choose this if you already have Docker Desktop or Rancher Desktop installed or prefer to use them.
           - **Traditional Windows**: Choose this for non-WSL2 installations (requires Docker Desktop or Rancher Desktop). Note that both Docker Desktop and Rancher Desktop themselves use WSL2, but you will interact with DDEV in Git Bash, PowerShell, or Cmd on the Windows side.

    The installer will automatically configure DDEV for your chosen Docker provider and WSL2 environment.

    !!!tip "Windows Installer Silent Mode"
        The Windows installer supports silent mode for automated installations and testing:

        ```bash
        # WSL2 with Docker CE and specified distro
        ddev_windows_installer.exe /docker-ce /distro=Ubuntu-24.04 /S

        # WSL2 with Docker Desktop and specified distro
        ddev_windows_installer.exe /docker-desktop /distro=Ubuntu-22.04 /S

        # Traditional Windows (requires Docker Desktop/Rancher Desktop)
        ddev_windows_installer.exe /traditional /S

        # Get help with all options
        ddev_windows_installer.exe /help
        ```

        The `/S` flag makes the installation completely silent. Use `/distro=<name>` to specify your WSL2 distribution name (required for WSL2 options).

    ### Manual Installation

    If you prefer to install DDEV manually on WSL2, it's not hard. These techniques can be used to install on a non-Ubuntu distro, and can be adapted for Yum-based or ArchLinux-based distros as well:

    1. In an administrative PowerShell, run `mkcert -install` and follow the prompt to install the Certificate Authority.
    2. In an administrative PowerShell, run `$env:CAROOT="$(mkcert -CAROOT)"; setx CAROOT $env:CAROOT; If ($Env:WSLENV -notlike "*CAROOT/up:*") { $env:WSLENV="CAROOT/up:$env:WSLENV"; setx WSLENV $Env:WSLENV }`. This will set WSL2 to use the Certificate Authority installed on the Windows side. In some cases it takes a reboot to work correctly.
    3. In administrative PowerShell, run `wsl --install <distro>`, for example `wsl --install Debian`. This will install the WSL2 distro for you.
    5. **Docker CE:** Follow the instructions in the [Linux install section here](docker-installation.md#linux) to install Docker CE and DDEV.
    6. **Docker Desktop for Windows:** 
        * Install Docker Desktop for Windows and configure the WSL2-based engine (not legacy Hyper-V) when installing. Download the installer from [docker.com](https://www.docker.com/products/docker-desktop/). Start Docker Desktop. It may prompt you to log out and log in again, or reboot.
        * Go to Docker Desktop’s *Settings* → *Resources* → *WSL integration* → *enable integration for your distro*. Now `docker` commands will be available from within your WSL2 distro.
    7. Double-check your distro: `echo $CAROOT` should show something like `/mnt/c/Users/<you>/AppData/Local/mkcert`
    8. Check that Docker is working inside Ubuntu (or your distro) by running `docker ps` in the distro.
    9. Open the WSL2 terminal, for example `Debian` from the Windows start menu.
    10. Follow the [Linux install instructions](#linux) to install DDEV. You can use the `install_ddev_wsl2_docker_inside.sh` script, which is available in the [DDEV GitHub repository](
    11. In WSL2, run `mkcert -install`.

    12. For unusual browsers and situations that don't automatically support the `mkcert` certificate authority, [configure your browser](configuring-browsers.md).

    You have now installed DDEV on WSL2. If you’re using WSL2 for DDEV, remember to run all `ddev` commands inside the WSL2 distro, and make sure your projects are in the WSL2 filesystem (e.g. `/home/<your_username>/projects`) and **not** in the Windows filesystem (e.g. `/mnt/c/Users/<your_username>/projects`).

    !!!note "Path to certificates"
        If you get the prompt `Installing to the system store is not yet supported on this Linux`, you may need to add `/usr/sbin` to the `$PATH` so that `/usr/sbin/update-ca-certificates` can be found.

    !!!note "Windows Firefox Trusted CA"

        The `mkcert -install` step on Windows isn’t enough for Firefox. You need to [configure your browser](configuring-browsers.md).

=== "Codespaces"

    ## GitHub Codespaces

    You can use DDEV in remote [GitHub Codespaces](https://github.com/features/codespaces) without having to run Docker locally; you only need a browser and an internet connection.

    Start by creating a `.devcontainer/devcontainer.json` file in your GitHub repository:

    ```json
    {
      "image": "mcr.microsoft.com/devcontainers/universal:2",
      "features": {
        "ghcr.io/ddev/ddev/install-ddev:latest": {}
      },
      "postCreateCommand": "echo 'it should all be set up'"
    }
    ```

    Launch your repository in Codespaces:

    <div style="text-align:center;"><img style="max-width:400px;" src="./../../../images/codespaces-launch.png" alt="Screenshot of codespace create dialog in a repository on GitHub"></div>

    <div style="text-align:center;"><img style="max-width:400px;" src="./../../../images/codespaces-setting-up.png" alt="Screenshot of codespace create dialog in a repository on GitHub"></div>

    DDEV is now available within your new codespace instance:

    <div style="text-align:center;"><img src="./../../../images/codespaces-hello-screen.png" alt=""></div>

    Run `ddev config` to [start a new blank project](./../project.md) - or [install a CMS](./../quickstart.md).

    Run `ddev start` if there is already a configured DDEV project in your repository.

    **Troubleshooting**:

    If there are errors after restarting a codespace, use `ddev restart` or `ddev poweroff`.

    You can also use the commands

    - "Codespaces: Rebuild container"
    - "Codespaces: Full rebuild container" (Beware: database will be deleted)

    via the [Visual Studio Code Command Palette](https://docs.github.com/en/enterprise-cloud@latest/codespaces/codespaces-reference/using-the-vs-code-command-palette-in-codespaces):

    - <kbd>⌘</kbd> + <kbd>SHIFT</kbd> + <kbd>P</kbd> on a Mac
    - <kbd>CTRL</kbd> + <kbd>SHIFT</kbd> + <kbd>P</kbd> on Windows/Linux
    - from the Application Menu, click View > Command Palette (Firefox)

    If you need DDEV-specific assistance or have further questions, see [support](./../support.md).

    Your updated `devcontainer.json` file may differ depending on your project, but you should have `install-ddev` in the `features` section.

    !!!note "Normal Linux installation also works"
        You can also install DDEV as if it were on any normal [Linux installation](#linux).

    ### Docker integration

    DDEV in Codespaces relies on [`docker-in-docker`](https://github.com/devcontainers/features), which is installed by default when you use the image `"mcr.microsoft.com/devcontainers/universal:2"`. Please be aware: GitHub Codespaces and its Docker-integration (docker-in-docker) are relatively new. See [devcontainers/features](https://github.com/devcontainers/features) for general support and issues regarding Docker-support.

    ###  DDEV's router is not used

    Since Codespaces handles all the routing, the internal DDEV router will not be used on Codespaces. Therefore config settings like [`web_extra_exposed_ports`](./../configuration/config.md#web_extra_exposed_ports) will have no effect.

    You can expose ports via the `ports` setting, which is usually not recommended if you work locally due to port conflicts. But you can load these additional Docker compose files only when Codespaces is detected. See [Defining Additional Services](./../extend/custom-compose-files.md#docker-composeyaml-examples) for more information.

    ```yaml
    services:
        web:
            ports:
                - "5174:5174"
    ```

    ### Default environment variables

    Codespace instances already provide some [default environment values](https://docs.github.com/en/codespaces/developing-in-codespaces/default-environment-variables-for-your-codespace). You can inherit and inject them in your `.ddev/config.yaml`:

    ```yaml
    web_environment:
        - CODESPACES
        - CODESPACE_NAME
        - GITHUB_CODESPACES_PORT_FORWARDING_DOMAIN
    ```

    ### Advanced usage via devcontainer.json

    A lot more customization is possible via the [`devcontainer.json`-configuration](https://containers.dev/implementors/json_reference/). You can install Visual Studio Code extensions by default or run commands automatically.

    #### postCreateCommand

    The [`postCreateCommand`](https://containers.dev/implementors/json_reference/) lets you run commands automatically when a new codespace is launched. DDEV commands are available here.

    The event is triggered on: fresh creation, rebuilds and full rebuilds. `ddev poweroff` is used in this example to avoid errors on rebuilds since some Docker containers are kept.

    You usually want to use a separate bash script to do this, as docker [might not yet be available when the command starts to run](https://github.com/devcontainers/features/issues/780).

    ```json
    {
        "image": "mcr.microsoft.com/devcontainers/universal:2",
        "features": {
            "ghcr.io/ddev/ddev/install-ddev:latest": {}
        },
        "portsAttributes": {
            "3306": {
                "label": "database"
            },
            "8027": {
                "label": "mailpit"
            },
            "8080": {
                "label": "web http"
            },
            "8443": {
                "label": "web https"
            }
        },
        "postCreateCommand": "bash .devcontainer/setup_project.sh"
    }
    ```

    ```bash
    #!/usr/bin/env bash
    set -ex

    wait_for_docker() {
      while true; do
        docker ps > /dev/null 2>&1 && break
        sleep 1
      done
      echo "Docker is ready."
    }

    wait_for_docker

    # download images beforehand, optional
    ddev debug download-images

    # avoid errors on rebuilds
    ddev poweroff

    # start ddev project automatically
    ddev start -y

    # further automated install / setup steps, e.g.
    ddev composer install
    ```

    To check for errors during the `postCreateCommand` action, use the command

    - "Codespaces: View creation log”

    via the [Visual Studio Code Command Palette](https://docs.github.com/en/enterprise-cloud@latest/codespaces/codespaces-reference/using-the-vs-code-command-palette-in-codespaces):

    - <kbd>⌘</kbd> + <kbd>SHIFT</kbd> + <kbd>P</kbd> on a Mac
    - <kbd>CTRL</kbd> + <kbd>SHIFT</kbd> + <kbd>P</kbd> on Windows/Linux
    - from the Application Menu, click View > Command Palette (Firefox)

    <div style="text-align:center;"><img src="./../../../images/codespaces-creation-log.png" alt=""></div>

=== "Manual"

    ## Manual

    DDEV is a single executable, so installation on any OS is a matter of copying the `ddev` executable for your architecture into the appropriate system path on your machine.

    * Download and extract the latest [DDEV release](https://github.com/ddev/ddev/releases) for your architecture.
    * Move `ddev` to `/usr/local/bin` with `mv ddev /usr/local/bin/` (may require `sudo`), or another directory in your `$PATH` as preferred.
    * Run `ddev` to test your installation. You should see DDEV’s command usage output.
    * As a one-time initialization, run `mkcert -install`, which may require your `sudo` password.

        If you don’t have `mkcert` installed, download the [latest release](https://github.com/FiloSottile/mkcert/releases) for your architecture and `sudo mv <downloaded_file> /usr/local/bin/mkcert && sudo chmod +x /usr/local/bin/mkcert`.

    * For unusual browsers and situations that don't automatically support the `mkcert` certificate authority, [configure your browser](configuring-browsers.md).
