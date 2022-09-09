# Get Started with DDEV

[DDEV](https://github.com/drud/ddev) is an open source tool that makes it dead simple to get local PHP development environments up and running within minutes. It's powerful and flexible as a result of its per-project environment configurations, which can be extended, version controlled, and shared. In short, DDEV aims to allow development teams to use Docker in their workflow without the complexities of bespoke configuration.

DDEV works great on macOS, Windows WSL2, traditional Windows, Linux and Gitpod.io. It works perfectly on amd64 and arm64 architectures, meaning it works fine natively on mac M1 systems and on Linux with both amd64 and arm64. It also works great on Gitpod, where you don't have to install anything at all.

## System Requirements

=== "macOS"

    ### macOS
    * DDEV runs natively on arm64 (Mac M1) systems as well as amd64 machines.
    * RAM: 8GB
    * Storage: 256GB
    * Colima or Docker Desktop is required.
    * Docker Desktop requires macOS Catalina (macOS 10.15) or higher.
    * Colima can even run on older systems.
    * DDEV should run anywhere Docker Desktop or Colima runs.

=== "Windows WSL2"

    ### Windows WSL2

    * RAM: 8GB
    * Storage: 256GB
    * Systems that can run Docker Desktop on the Windows side or docker-ce inside WSL2 do fine.
    * The preferred distro is Ubuntu or an Ubuntu-derived distro, but people use lots of different distros.

=== "Traditional Windows"

    ### Traditional Windows

    * Any recent edition of Windows Home, Pro, and several others.
    * RAM: 8GB
    * Storage: 256GB
    * Docker Desktop using the WSL2 back-end

=== "Linux"

    ### Linux

    * Most distros and most versions work fine.
    * RAM: 8GB
    * Storage: 256GB

=== "Gitpod"

    ### Gitpod

    With [Gitpod](https://www.gitpod.io) all you need is a browser and an internet connection. You don't have to install anything at all. You can use any kind of computer that has a browser, or tablet, or whatever you like.
