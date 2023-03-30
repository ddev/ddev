# Upgrading DDEV

Installing and upgrading DDEV are nearly the same thing, because you're upgrading the `ddev` binary that talks with Docker. You can update this file like other software on your system, whether it’s with a package manager or traditional installer.

=== "macOS"

    ## macOS

    ### Homebrew

    ```bash
    # Upgrade DDEV to the latest version
    brew upgrade ddev
    ```

    ### Install Script

    ```bash
    # Download and run the script to replace the DDEV binary
    curl -fsSL https://ddev.com/install.sh | bash
    ```

    ??? "Need a specific version?"
        Use the `-s` argument to specify a specific stable or prerelease version:

        ```bash
        # Download and run the script to update to DDEV v1.21.4
        curl -fsSL https://ddev.com/install.sh | bash -s v1.21.4
        ```

=== "Linux"

    ## Linux

    ### Debian/Ubuntu

    ```bash
    # Update package information and all packages including DDEV
    sudo apt update && sudo apt upgrade
    ```

    ### Fedora, Red Hat, etc.

    ```bash
    # Upgrade the DDEV package
    sudo dnf upgrade ddev
    ```

    ### Arch Linux

    ```bash
    # Upgrade the DDEV package
    yay -Syu ddev-bin
    ```

=== "Windows"

    ## Windows

    ### WSL2 + Docker Install Script

    If you used the WSL2 install script with [Docker CE inside](./ddev-installation.md#wsl2-docker-ce-inside-install-script) or [Docker Desktop](./ddev-installation.md#wsl2-docker-desktop-install-script), the upgrade process is the same: open an administrative PowerShell (5) and run [this PowerShell script](https://raw.githubusercontent.com/ddev/ddev/master/scripts/install_ddev_wsl2_docker_inside.ps1):

    ```powershell
    Set-ExecutionPolicy Bypass -Scope Process -Force; [System.Net.ServicePointManager]::SecurityProtocol = [System.Net.ServicePointManager]::SecurityProtocol -bor 3072;
    iex ((New-Object System.Net.WebClient).DownloadString('https://raw.githubusercontent.com/ddev/ddev/master/scripts/install_ddev_wsl2_docker_inside.ps1'))
    ```

    ### WSL2/Docker Desktop Manual Installation

    Open the WSL2 terminal, for example `Ubuntu` from the Windows start menu, and run the following:
    
    ```bash
    # Upgrade the DDEV package
    apt upgrade ddev
    ```

    ### Traditional Windows

    #### Chocolatey

    ```bash
    # Turn off DDEV and upgrade it
    ddev poweroff && choco upgrade ddev
    ```

    #### Installer

    Download and run the Windows installer for the latest [DDEV release](https://github.com/ddev/ddev/releases) (`ddev_windows_installer.<version>.exe`).

=== "Gitpod"

    ## Gitpod

    ```bash
    # Update package information and all packages including DDEV
    sudo apt update && sudo apt upgrade
    ```

=== "Codespaces"

    ## GitHub Codespaces

    ```bash
    # Update package information and all packages including DDEV
    sudo apt update && sudo apt upgrade
    ```

=== "Manual"

    ## Manual

    Upgrade using the exact same [manual install](./ddev-installation.md#manual) process:

    * Download and extract the latest [DDEV release](https://github.com/ddev/ddev/releases) for your architecture.
    * Move `ddev` to `/usr/local/bin` with `mv ddev /usr/local/bin/` (may require `sudo`), or another directory in your `$PATH` as preferred.
    * Run `ddev --version` to confirm you’re running the expected version.
