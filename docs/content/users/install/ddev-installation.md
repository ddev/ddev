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

    ### Install Script

    The [install script](https://github.com/ddev/ddev/blob/main/scripts/install_ddev.sh) is another option. It downloads, verifies, and sets up the `ddev` executable:

    ```bash
    # Download and run the install script
    curl -fsSL https://ddev.com/install.sh | bash
    ```

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

    (Some versions of Firefox (Developer Edition, Flatpak) may need some [extra work](https://github.com/FiloSottile/mkcert/issues/370#issuecomment-1280377305) with `mkcert`, see also [this issue](https://github.com/ddev/ddev/issues/5415).)

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

    Signed yum repository support will be added in the future.

    ### Arch Linux

    We maintain the [ddev-bin](https://aur.archlinux.org/packages/ddev-bin/) package in AUR for Arch-based systems including Arch Linux, EndeavourOS and Manjaro. Install with `yay` or your AUR tool of choice.

    ```bash
    # Install DDEV
    yay -S ddev-bin

    # One-time initialization of mkcert
    mkcert -install
    ```

    ### Homebrew (AMD64 only)

    ```bash
    # Install DDEV using Homebrew
    brew install ddev/ddev/ddev

    # One-time initialization of mkcert
    mkcert -install
    ```

    <!-- we’re using HTML here to customize the #install-script-linux anchor -->
    <h3 id="install-script-linux">Install Script<a class="headerlink" href="#install-script-linux" title="Permanent link">¶</a></h3>

    The [install script](https://github.com/ddev/ddev/blob/main/scripts/install_ddev.sh) is another option. It downloads, verifies, and sets up the `ddev` executable:

    ```bash
    # Download and run the install script
    curl -fsSL https://ddev.com/install.sh | bash
    ```

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

    You can install DDEV on Windows three ways:

    1. [Using WSL2 with Docker inside](#wsl2-docker-ce-inside-install-script) **Recommended, best performance, most reliable**
    2. [Using WSL2 with Docker Desktop](#wsl2-docker-desktop-install-script) **May require license, less reliable**
    3. [Installing directly on traditional Windows](#traditional-windows) with an installer **Legacy, slower performance**

    **We strongly recommend using WSL2 for your Windows DDEV development environment.** While its Linux experience may be new for some Windows users, it’s worth the performance benefit and common experience of working with Ubuntu and Bash.

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
    * Custom hostnames are managed via the Windows hosts file, not within WSL2.
      DDEV attempts to manage custom hostnames via the Windows-side hosts file—usually at `C:\Windows\system32\drivers\etc\hosts`—and it can only do this if it’s installed on the Windows side. (DDEV inside WSL2 uses `ddev.exe` on the Windows side as a proxy to update the Windows hosts file.) If `ddev.exe --version` shows the same version as `ddev --version` you’re all set up. Otherwise, install DDEV on Windows using `choco upgrade -y ddev` or by downloading and running the Windows installer. (The WSL2 scripts below install DDEV on the Windows side, taking care of that for you.) If you frequently run into Windows UAC Escalation, you can calm it down by running `gsudo.exe cache on` and `gsudo.exe config CacheMode auto`, see [gsudo docs](https://github.com/gerardog/gsudo#credentials-cache).
    * WSL2 is not the same as Docker Desktop’s WSL2 engine.
      Using WSL2 to install and run DDEV is not the same as using Docker Desktop’s WSL2 engine, which itself runs in WSL2, but can serve applications running in both traditional Windows and inside WSL2.

    The WSL2 install process involves:

    * Installing Chocolatey package manager (optional).
    * One time initialization of mkcert.
    * Installing WSL2 and installing a distro like Ubuntu.
    * Optionally installing Docker Desktop for Windows and enabling WSL2 integration with the distro (if you're using the Docker Desktop approach).
    * Installing DDEV inside your distro; this is normally done by running one of the two scripts below, but can be done manually step-by-step as well.
    * Verify that your distro uses WSL version 2 with `wsl.exe -l -v`.

    ### WSL2 + Docker CE Inside Install Script

    This technique is our favorite, as it uses the most reliable WSL2 Docker provider (`docker-ce`), which is also free and open-source.

    This script prepares your default WSL2 Ubuntu distro and doesn’t require Docker Desktop, and you can run the script multiple times without breaking anything.

    In all cases:

    1. Install WSL2 with an Ubuntu distro.

        * Install WSL:
            ```
            wsl --install
            ```

        * Reboot if required. (Usually required.)

        * Verify that you have an Ubuntu distro set as default by running `wsl.exe -l -v`.
          If you have WSL2 but not an Ubuntu distro, install one by running `wsl.exe --install Ubuntu`. If this doesn’t work, see [manual installation](https://docs.microsoft.com/en-us/windows/wsl/install-manual) and [troubleshooting](https://docs.microsoft.com/en-us/windows/wsl/troubleshooting#installation-issues).

        * Verify that your Ubuntu default distro is WSL v2 using `wsl -l -v`.

    2. In an administrative PowerShell run [this PowerShell script](https://raw.githubusercontent.com/ddev/ddev/main/scripts/install_ddev_wsl2_docker_inside.ps1) by executing:

        ```powershell
        Set-ExecutionPolicy Bypass -Scope Process -Force; [System.Net.ServicePointManager]::SecurityProtocol = [System.Net.ServicePointManager]::SecurityProtocol -bor 3072;
        iex ((New-Object System.Net.WebClient).DownloadString('https://raw.githubusercontent.com/ddev/ddev/main/scripts/install_ddev_wsl2_docker_inside.ps1'))
        ```
    3. In *Windows Update Settings* → *Advanced Options* enable *Receive updates for other Microsoft products*. You may want to occasionally run `wsl.exe --update` as well.

    Now you can use the "Ubuntu" terminal app or Windows Terminal to access your Ubuntu distro, which has DDEV and Docker working inside it.

    ### WSL2 + Docker Desktop Install Script

    WSL2 with Docker Desktop is a less-favored choice because Docker Desktop may be lightly supported, and has many features not required for use with DDEV that do not add particular value. It is also not free software (although smaller organizations can use it free of charge) and it is not open-source.

    The script here prepares your default WSL2 Ubuntu distro for use with Docker Desktop, and you can run the script multiple times without breaking anything.

    In all cases:

    4. Install WSL2 with an Ubuntu distro. On a system without WSL2, run:
        ```powershell
        wsl --install
        ```

        * Verify that you have an Ubuntu distro set as the default default with `wsl -l -v`.

        * If you have WSL2 but not an Ubuntu distro, install one with `wsl --install Ubuntu`.
          If that doesn't work for you, see [manual installation](https://docs.microsoft.com/en-us/windows/wsl/install-manual) and [troubleshooting](https://docs.microsoft.com/en-us/windows/wsl/troubleshooting#installation-issues).

        If you prefer to use another Ubuntu distro, install it and set it as default. For example, `wsl --set-default Ubuntu-24.04`.

    5. In *Windows Update Settings* → *Advanced Options* enable *Receive updates for other Microsoft products*. You may want to occasionally run `wsl.exe --update` as well.

    6. Install Docker Desktop. If you already have Chocolatey, run `choco install -y docker-desktop`. Otherwise [download Docker Desktop from Docker](https://www.docker.com/products/docker-desktop/).
    7. Start Docker Desktop. You should now be able to run `docker ps` in PowerShell or Git Bash.
    8. In *Docker Desktop* → *Settings* → *Resources* → *WSL2 Integration*, verify that Docker Desktop is integrated with your distro.
    9. In an administrative PowerShell run [this PowerShell script](https://raw.githubusercontent.com/ddev/ddev/main/scripts/install_ddev_wsl2_docker_desktop.ps1) by executing:

        ```powershell
        Set-ExecutionPolicy Bypass -Scope Process -Force; [System.Net.ServicePointManager]::SecurityProtocol = [System.Net.ServicePointManager]::SecurityProtocol -bor 3072;
        iex ((New-Object System.Net.WebClient).DownloadString('https://raw.githubusercontent.com/ddev/ddev/main/scripts/install_ddev_wsl2_docker_desktop.ps1'))
        ```

    Now you can use the "Ubuntu" terminal app or Windows Terminal to access your Ubuntu distro, which has DDEV and Docker Desktop integrated with it.

    ### WSL2/Docker Desktop Manual Installation

    You can manually step through the process the install script attempts to automate:

    1. Install [Chocolatey](https://chocolatey.org/install):
        ```powershell
        Set-ExecutionPolicy Bypass -Scope Process -Force; [System.Net.ServicePointManager]::SecurityProtocol = [System.Net.ServicePointManager]::SecurityProtocol -bor 3072;
        iex ((New-Object System.Net.WebClient).DownloadString('https://chocolatey.org/install.ps1'))`
        ```
    2. In an administrative PowerShell, run `choco install -y ddev mkcert`.
    3. In an administrative PowerShell, run `mkcert -install` and follow the prompt to install the Certificate Authority.
    4. In an administrative PowerShell, run `$env:CAROOT="$(mkcert -CAROOT)"; setx CAROOT $env:CAROOT; If ($Env:WSLENV -notlike "*CAROOT/up:*") { $env:WSLENV="CAROOT/up:$env:WSLENV"; setx WSLENV $Env:WSLENV }`. This will set WSL2 to use the Certificate Authority installed on the Windows side. In some cases it takes a reboot to work correctly.
    5. In administrative PowerShell, run `wsl --install`. This will install WSL2 and Ubuntu for you. Reboot when this is done.
    6. **Docker Desktop for Windows:** If you already have the latest Docker Desktop, configure it in the General Settings to use the WSL2-based engine. Otherwise install the latest Docker Desktop for Windows and select the WSL2-based engine (not legacy Hyper-V) when installing. Install with Chocolatey by running `choco install docker-desktop`, or download the installer from [desktop.docker.com](https://desktop.docker.com/win/main/amd64/Docker%20Desktop%20Installer.exe).  Start Docker. It may prompt you to log out and log in again, or reboot.
    7. Go to Docker Desktop’s *Settings* → *Resources* → *WSL integration* → *enable integration for your distro*. Now `docker` commands will be available from within your WSL2 distro.
    8. Double-check in PowerShell: `wsl -l -v` should show three distros, and your Ubuntu should be the default. All three should be WSL version 2.
    9. Double-check in Ubuntu (or your distro): `echo $CAROOT` should show something like `/mnt/c/Users/<you>/AppData/Local/mkcert`
    10. Check that Docker is working inside Ubuntu (or your distro) by running `docker ps`.
    11. Open the WSL2 terminal, for example `Ubuntu` from the Windows start menu.
    12. Install DDEV:

        ```bash
        sudo apt-get update && sudo apt-get install -y curl
        sudo install -m 0755 -d /etc/apt/keyrings
        curl -fsSL https://pkg.ddev.com/apt/gpg.key | gpg --dearmor | sudo tee /etc/apt/keyrings/ddev.gpg > /dev/null
        echo "deb [signed-by=/etc/apt/keyrings/ddev.gpg] https://pkg.ddev.com/apt/ * *" | sudo tee /etc/apt/sources.list.d/ddev.list >/dev/null
        sudo apt-get update && sudo apt-get install -y ddev
        ```

    13. In WSL2, run `mkcert -install`.

    You have now installed DDEV on WSL2. If you’re using WSL2 for DDEV, remember to run all `ddev` commands inside the WSL2 distro.

    !!!note "Path to certificates"
        If you get the prompt `Installing to the system store is not yet supported on this Linux`, you may need to add `/usr/sbin` to the `$PATH` so that `/usr/sbin/update-ca-certificates` can be found.

    ### Traditional Windows

    If you must use traditional Windows, then Docker Desktop is your only choice of a Docker provider. DDEV is supported in this configuration but it's not as performant as the WSL2 options.

    * We recommend using [Chocolatey](https://chocolatey.org/). Once installed, you can run `choco install ddev docker-desktop git` from an administrative shell. You can upgrade by running `ddev poweroff && choco upgrade ddev`.
    * Each [DDEV release](https://github.com/ddev/ddev/releases) includes Windows installers for AMD64 and ARM64 Windows (`ddev_windows_<architecture>_installer.<version>.exe`). After running that, you can open a new Git Bash, PowerShell, or cmd.exe window and start using DDEV.

    Most people interact with DDEV on traditional Windows using Git Bash, part of the [Windows Git suite](https://git-scm.com/download/win). Although DDEV does work with cmd.exe and PowerShell, it's more at home in Bash. You can install Git Bash with Chocolatey by running `choco install -y git`.

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

    Choose any of the following methods to launch your project with [Gitpod](https://www.gitpod.io):

    1. Add a [.gitpod.yml](https://www.gitpod.io/docs/references/gitpod-yml) to your project that installs DDEV and starts your project. The easy way to do this is with the excellent [ddev-gitpod-setup](https://github.com/tyler36/ddev-gitpod-setup) add-on. `ddev get tyler36/ddev-gitpod-setup` and add it to your git repository, push it, and then open it. You can even do all of this right on gitpod by opening your repository, doing the `ddev get` there and then pushing it back and restarting.
    2. [Open any repository](https://www.gitpod.io/docs/getting-started) using Gitpod and run the following:
        ```bash
        # Add DDEV’s GPG key to your keyring
        curl -fsSL https://pkg.ddev.com/apt/gpg.key | gpg --dearmor | sudo tee /etc/apt/keyrings/ddev.gpg > /dev/null

        # Add DDEV releases to your package repository
        echo "deb [signed-by=/etc/apt/keyrings/ddev.gpg] https://pkg.ddev.com/apt/ * *" | sudo tee /etc/apt/sources.list.d/ddev.list >/dev/null


        # Update package information and install DDEV
        sudo apt-get update && sudo apt-get install -y ddev
        ```

        * You can install your web app there, or import a database.
        * You may want to implement one of the `ddev pull` provider integrations to pull from a hosting provider or an upstream source.
    3. Use the [ddev-gitpod-launcher](https://ddev.github.io/ddev-gitpod-launcher/) form to launch a repository.
        You’ll provide a source repository and click a button to open a newly-established environment. You can specify a companion artifacts repository and automatically load `db.sql.gz` and `files.tgz` from it. (More details in the [repository’s README](https://github.com/ddev/ddev-gitpod-launcher/blob/main/README.md).)
    4. Save the following link to your bookmark bar: <a href="javascript: if %28 %2Fbitbucket%2F.test %28 window.location.host %29 %20 %29 %20%7B%20paths%3Dwindow.location.pathname.split %28 %22%2F%22 %29 %3B%20repo%3D%5Bwindow.location.origin%2C%20paths%5B1%5D%2C%20paths%5B2%5D%5D.join %28 %22%2F%22 %29 %20%7D%3B%20if %28 %2Fgithub.com%7Cgitlab.com%2F.test %28 window.location.host %29  %29 %20%7Brepo%20%3D%20window.location.href%7D%3B%20if%20 %28 repo %29 %20%7Bwindow.location.href%20%3D%20%22https%3A%2F%2Fgitpod.io%2F%23DDEV_REPO%3D%22%20%2B%20encodeURIComponent %28 repo %29 %20%2B%20%22%2CDDEV_ARTIFACTS%3D%22%20%2B%20encodeURIComponent %28 repo %29 %20%2B%20%22-artifacts%2Fhttps%3A%2F%2Fgithub.com%2Fddev%2Fddev-gitpod-launcher%2F%22%7D%3B">Open in ddev-gitpod</a>.
        It’s easiest to drag the link into your bookmarks. When you’re on a Git repository, click the bookmark to open it with DDEV in Gitpod. It does the same thing as the second option, but it works on non-Chrome browsers and you can use native browser keyboard shortcuts.

    It can be complicated to get private databases and files into Gitpod, so in addition to the launchers, the [`git` provider example](https://github.com/ddev/ddev/blob/main/pkg/ddevapp/dotddev_assets/providers/git.yaml.example) demonstrates pulling a database and files without complex setup or permissions. This was created explicitly for Gitpod integration, because in Gitpod you typically already have access to private Git repositories, which are a fine place to put a starter database and files. Although [ddev-gitpod-launcher](https://ddev.github.io/ddev-gitpod-launcher/) and the web extension provide the capability, you may want to integrate a Git provider—or one of the [other providers](https://github.com/ddev/ddev/tree/main/pkg/ddevapp/dotddev_assets/providers)—for each project.

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
