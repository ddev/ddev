---
hide:
  - toc
---

# Get Started with DDEV

[DDEV](https://github.com/ddev/ddev) is an open source tool for launching local web development environments in minutes. It supports PHP, Node.js, and Python (experimental).

These environments can be extended, version controlled, and shared, so you can take advantage of a Docker workflow without Docker experience or bespoke configuration. Projects can be changed, powered down, or removed as easily as they’re started.

## System Requirements

=== "macOS"

    ### macOS

    Runs natively on ARM64 (Apple Silicon) and AMD64 machines.

    * macOS Big Sur (11) or higher, [mostly](./users/usage/faq.md#can-i-run-ddev-on-an-older-mac)
    * RAM: 8GB
    * Storage: 256GB
    * [OrbStack](https://orbstack.dev/) or [Lima](https://github.com/lima-vm/lima) or [Docker Desktop](https://www.docker.com/products/docker-desktop/) or [Rancher Desktop](https://rancherdesktop.io/) or [Colima](https://github.com/abiosoft/colima)

    **Next steps:**

    *You’ll need a Docker provider on your system before you can install DDEV.*
    
    1. Install Docker with [recommended settings](users/install/docker-installation.md#macos).
    2. Install [DDEV for macOS](users/install/ddev-installation.md#macos).
    3. Launch your [first project](users/project.md) and start developing. 🚀

=== "Windows WSL2"

    ### Windows WSL2

    * RAM: 8GB
    * Storage: 256GB
    * [Docker Desktop](https://www.docker.com/products/docker-desktop/) on the Windows side or [Docker CE](https://docs.docker.com/engine/install/ubuntu/) inside WSL2
    * Ubuntu or an Ubuntu-derived distro is recommended, though others may work fine

    **Next steps:**

    *You’ll need a Docker provider on your system before you can install DDEV.*
    
    1. Install Docker with [recommended settings](users/install/docker-installation.md#windows).
    2. Install [DDEV for Windows](users/install/ddev-installation.md#windows).
    3. Launch your [first project](users/project.md) and start developing. 🚀

=== "Traditional Windows"

    ### Traditional Windows

    * Any recent edition of Windows Home or Windows Pro.
    * RAM: 8GB
    * Storage: 256GB
    * [Docker Desktop](https://www.docker.com/products/docker-desktop/) using the WSL2 backend

    **Next steps:**

    *You’ll need a Docker provider on your system before you can install DDEV.*
    
    1. Install Docker with [recommended settings](users/install/docker-installation.md#windows).
    2. Install [DDEV for Windows](users/install/ddev-installation.md#windows).
    3. Launch your [first project](users/project.md) and start developing. 🚀

=== "Linux"

    ### Linux

    Most distros and most versions work fine, on both AMD64 and ARM64 architectures.

    * RAM: 8GB
    * Storage: 256GB

    **Next steps:**

    *You’ll need a Docker provider on your system before you can install DDEV.*
    
    1. Install Docker with [recommended settings](users/install/docker-installation.md#linux).
    2. Install [DDEV for Linux](users/install/ddev-installation.md#linux).
    3. Launch your [first project](users/project.md) and start developing. 🚀

=== "Gitpod & Codespaces"

    ### Gitpod and GitHub Codespaces

    With [Gitpod](https://www.gitpod.io) and [GitHub Codespaces](https://github.com/features/codespaces) you don’t install anything; you only need a browser and an internet connection.

    **Next steps:**

    1. Install DDEV within [Gitpod](users/install/ddev-installation.md#gitpod) or [GitHub Codespaces](users/install/ddev-installation.md#github-codespaces).
    2. Launch your [first project](users/project.md) and start developing. 🚀
