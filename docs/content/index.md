---
hide:
  - toc
---

# Get Started with DDEV

[DDEV](https://github.com/drud/ddev) is an open source tool for launching local PHP development environments in minutes.

These environments can be extended, version controlled, and shared, so you can take advantage of a Docker workflow without Docker experience or bespoke configuration. Projects can be changed, powered down, or removed just as easily as they’re started.

## System Requirements

=== "macOS"

    ### macOS
    
    Runs natively on ARM64 (Apple Silicon) and AMD64 machines.

    * RAM: 8GB
    * Storage: 256GB
    * [Colima](https://github.com/abiosoft/colima) (preferred) or [Docker Desktop](https://www.docker.com/products/docker-desktop/) (supported)
    * Docker Desktop requires macOS Catalina (10.15) or higher; Colima runs on older systems

=== "Windows WSL2"
    ### Windows WSL2

    * RAM: 8GB
    * Storage: 256GB
    * [Docker Desktop](https://www.docker.com/products/docker-desktop/) on the Windows side or [Docker CE](https://docs.docker.com/engine/install/ubuntu/) inside WSL2
    * Ubuntu or an Ubuntu-derived distro is recommended, though others may work fine

=== "Traditional Windows"

    ### Traditional Windows

    * Any recent edition of Windows Home or Windows Pro.
    * RAM: 8GB
    * Storage: 256GB
    * [Docker Desktop](https://www.docker.com/products/docker-desktop/) using the WSL2 backend

=== "Linux"

    ### Linux

    Most distros and most versions work fine, on both AMD64 and ARM64 architectures.

    * RAM: 8GB
    * Storage: 256GB

=== "Gitpod"

    ### Gitpod

    With [Gitpod](https://www.gitpod.io) you don’t install anything; you only need a browser and an internet connection.
