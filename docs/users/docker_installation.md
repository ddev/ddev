<h1>Docker Installation</h1>

## macOS Installation: Docker-ce For Mac

Most MacOS versions and computers will run Docker for Mac. Homebrew users can `brew cask install docker` or you can download from [download.docker.com](https://download.docker.com/mac/stable/Docker.dmg). 

## Windows Installation: Docker-ce For Windows

Docker For Windows is the preferred docker environment for Windows 10 Pro and Windows 10 Enterprise. 

[Download Docker-ce for Windows](https://download.docker.com/win/stable/Docker%20for%20Windows%20Installer.exe)

[Chocolatey](https://chocolatey.org/install) users: `choco install docker-for-windows`

__Please note that you *must* share your local drives in the "settings" after installation or ddev will not be able to mount your project.__


## Windows Installation: Docker Toolbox

Docker Toolbox is only recommended for systems that absolutely won't run Docker-ce for Windows (Windows 10 Home, etc.)

[Download and install docker toolbox](https://download.docker.com/win/stable/DockerToolbox.exe). 

[Chocolatey](https://chocolatey.org/install) users: `choco install docker-toolbox`


## Linux Installation: Docker-ce

__Please don't forget that Linux installation absolutely requires post-install steps (below).__

__docker-compose must be installed separately, as it is not bundled with docker-ce on Linux, see below.__

docker-ce installation on Linux depends on what flavor you're using. In all cases using the Ubuntu/Deb/yum repository is the preferred technique.

* [Ubuntu](https://docs.docker.com/install/linux/docker-ce/ubuntu/)
* [CentOS](https://docs.docker.com/install/linux/docker-ce/centos/)
* [Debian](https://docs.docker.com/install/linux/docker-ce/debian/)
* [Fedora](https://docs.docker.com/install/linux/docker-ce/fedora/)
* [binaries](https://docs.docker.com/install/linux/docker-ce/binaries/)


After installing docker-ce you *must* install docker-compose separately. [Follow download instructions](https://docs.docker.com/compose/install/#install-compose) (select "linux" tab). This really is just downloading docker-compose binary from https://github.com/docker/compose/releases and installing it in /usr/local/bin with executable permissions.

### Linux Post-installation steps (required)

See [Docker's post-installation steps](https://docs.docker.com/install/linux/linux-postinstall/#manage-docker-as-a-non-root-user). You need to add your linux user to the "docker" group. and normally set up docker to start on boot.


## Testing Your Docker Installation

macOS or Linux: Run `docker-compose version && docker run -t -p 80:80 -v "$PWD:/tmp/homedir" busybox:latest ls /tmp/homedir` - you should see the contents of your home directory displayed.

Windows in cmd window: Run `docker run -t -p 80:80 -v "%USERPROFILE%:/tmp/homedir" busybox ls /tmp/homedir` - you should see the contents of your home directory displayed.

Windows in git-bash window: run ` docker run -t -p 80:80 -v "$USERPROFILE:/tmp/homedir" busybox ls //tmp/homedir` - you should see the contents of your home directory displayed.

If any of these steps fails you'll need to troubleshoot. 

* "port is already allocated": See [troubleshooting](troubleshooting.md).
* "The path ... is not shared and is not known to Docker": Visit docker's preferences/settings->File sharing and share the appropriate path or drive.
* "Error response from daemon: Get https://registry-1.docker.io/v2/" - Docker may not be running (restart it) or you may not have any access to the internet.
* "403 authentication required": Try `docker logout` and do it again. 