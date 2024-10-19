# Upgrading DDEV

Installing and upgrading DDEV are nearly the same thing, because you're upgrading the `ddev` binary that talks with Docker. You can update this file like other software on your system, whether it’s with a package manager or traditional installer.

!!!tip "`ddev --version` shows an old version"
    If you have installed or upgraded DDEV to the latest version, but when you check the actual version with `ddev --version`, it shows an older version, please refer to [Why do I have an old DDEV?](../usage/faq.md#why-do-i-have-an-old-ddev)

=== "macOS"

    ## macOS

    ### Homebrew

    ```bash
    # Upgrade DDEV to the latest version
    brew upgrade ddev/ddev/ddev
    ```

    ### Install Script

    ```bash
    # Download and run the script to replace the DDEV binary
    curl -fsSL https://ddev.com/install.sh | bash
    ```

    ### Verify New Version

    Use `ddev --version` to find out the version of the `ddev` binary in your `$PATH`. If `ddev --version` still shows an older version than you installed or upgraded to, use `which -a ddev` to find out where another version of the `ddev` binary must be installed. See the ["Why Do I Have An Old DDEV" FAQ](../usage/faq.md#why-do-i-have-an-old-ddev).

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
    sudo apt-get update && sudo apt-get upgrade
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

    ### Verify New Version

    Use `ddev --version` to find out the version of the `ddev` binary in your `$PATH`. If `ddev --version` still shows an older version than you installed or upgraded to, use `which -a ddev` to find out where another version of the `ddev` binary must be installed. See the ["Why Do I Have An Old DDEV" FAQ](../usage/faq.md#why-do-i-have-an-old-ddev).

=== "Windows"

    ## Windows

    ### WSL2 + Docker

    If you’re using WSL2, the upgrade process is the same regardless of how you installed DDEV.

    Open the WSL2 terminal, for example “Ubuntu” from the Windows start menu, and run the following:

    ```bash
    # Upgrade the DDEV package
    sudo apt-get update && sudo apt-get upgrade -y
    ```

    ### Verify New Version

    Use `ddev --version` to find out the version of the `ddev` binary in your `$PATH`. If `ddev --version` still shows an older version than you installed or upgraded to, use `which -a ddev` to find out where another version of the `ddev` binary must be installed. See the ["Why Do I Have An Old DDEV" FAQ](../usage/faq.md#why-do-i-have-an-old-ddev).

    ### Traditional Windows

    #### Chocolatey (only on Intel machines)

    ```bash
    # Turn off DDEV and upgrade it
    ddev poweroff && choco upgrade ddev
    ```

    #### Installer

    Download and run the Windows installer (for your architecture, most often AMD64) for the latest [DDEV release](https://github.com/ddev/ddev/releases) (`ddev_windows_<architecture>_installer.<version>.exe`).

=== "Gitpod"

    ## Gitpod

    ```bash
    # Update package information and all packages including DDEV
    sudo apt-get update && sudo apt-get upgrade -y
    ```

=== "Codespaces"

    ## GitHub Codespaces

    ```bash
    # Update package information and all packages including DDEV
    sudo apt-get update && sudo apt-get upgrade -y
    ```

=== "Manual"

    ## Manual

    Upgrade using the exact same [manual install](./ddev-installation.md#manual) process:

    * Download and extract the latest [DDEV release](https://github.com/ddev/ddev/releases) for your architecture.
    * Move `ddev` to `/usr/local/bin` with `mv ddev /usr/local/bin/` (may require `sudo`), or another directory in your `$PATH` as preferred.
    * Run `ddev --version` to confirm you’re running the expected version.
