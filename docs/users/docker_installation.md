<h1>Docker Installation</h1>

## macOS Installation: Docker Desktop for Mac

Most MacOS versions and computers will run Docker Desktop for Mac. Homebrew users can `brew cask install docker` or you can download from [download.docker.com](https://download.docker.com/mac/stable/Docker.dmg). 

## Windows Installation: Docker Desktop for Windows

Docker Desktop for Windows is the preferred docker environment for Windows 10 Pro and Windows 10 Enterprise. 

[Download Docker-ce for Windows](https://download.docker.com/win/stable/Docker%20for%20Windows%20Installer.exe)

[Chocolatey](https://chocolatey.org/install) users: `choco install docker-for-windows`

__Please note that you *must* share your local drives in the "settings" after installation or ddev will not be able to mount your project.__


## Windows Installation: Docker Toolbox

Docker Toolbox is only recommended for systems that absolutely won't run Docker Desktop for Windows (Windows 10 Home, etc.)

[Download and install docker toolbox](https://download.docker.com/win/stable/DockerToolbox.exe). 

[Chocolatey](https://chocolatey.org/install) users: `choco install -y docker-toolbox`

Special considerations for Docker Toolbox:

* Your project directory must be inside your home directory, as only the home directory is shared with Docker by default. Docker Toolbox (via Virtualbox) can share other paths, see [link](https://stackoverflow.com/a/35498478/215713).
* Please increase the memory allocated from the default 1GB to at least 2GB and increase the disk size to at least 50GB.

    1. `docker-machine rm default`
    2. `docker-machine create -d virtualbox --virtualbox-cpu-count=2 --virtualbox-memory=2048 --virtualbox-disk-size=50000 default`
    3. `docker-machine stop`
    4. Then exit Docker Quickstart Terminal and restart it to restart Docker Toolbox.


## Linux Installation: Docker-ce

* __Please don't forget that Linux installation absolutely requires post-install steps (below).__

* __docker-compose must be installed or upgraded separately, as it is not bundled with docker-ce on Linux, see below.__

* __Please never use sudo to run `ddev start`. If you do this it will set wrong permissions on files, and it means that you didn't follow the post-install instructions below to add your user to the docker group.__

docker-ce installation on Linux depends on what flavor you're using. In all cases using the Ubuntu/Deb/yum repository is the preferred technique.

* [Ubuntu](https://docs.docker.com/install/linux/docker-ce/ubuntu/)
* [CentOS](https://docs.docker.com/install/linux/docker-ce/centos/)
* [Debian](https://docs.docker.com/install/linux/docker-ce/debian/)
* [Fedora](https://docs.docker.com/install/linux/docker-ce/fedora/)
* [binaries](https://docs.docker.com/install/linux/docker-ce/binaries/)


After installing docker-ce you *must* install docker-compose separately. If using Linuxbrew you can `brew install docker-compose`, otherwise [Follow download instructions](https://docs.docker.com/compose/install/#install-compose) (select "linux" tab). This really is just downloading docker-compose binary from https://github.com/docker/compose/releases and installing it in /usr/local/bin with executable permissions.

### Linux Post-installation steps (required)

See [Docker's post-installation steps](https://docs.docker.com/install/linux/linux-postinstall/). You need to add your linux user to the "docker" group. and normally set up docker to start on boot.  __Please do not ever use sudo to run `ddev start`, it will break things.__

<a name="troubleshooting"></a>
## Testing and Troubleshooting Your Docker Installation

Docker needs to be able to a few things for ddev to work:

* Mount the project code directory from the host into the container; the project code directory is usually somewhere in a subdirectory of your home directory. 
* Mount ~/.ddev for SSL cert cache and import-db. 
* Access TCP ports on the host to serve HTTP and HTTPS. These are ports 80 and 443 by default, but they can be changed on a per-project basis.

So we can use a single docker command to make sure that docker is set up to do what we want:

**On Windows this command should be run in git-bash (or Docker Quickstart Terminal with Docker Toolbox).** In your project directory run `docker run --rm -t -p 80:80 -v "/$PWD:/tmp/projdir" -v "/$HOME:/tmp/homedir" busybox sh -c "echo ---- Project Directory && ls //tmp/projdir && echo ---- Home Directory && ls //tmp/homedir"` - you should see the contents of your home directory displayed. (On Windows, make sure you do this using git-bash or Docker Quickstart Terminal.)

If that fails (if you get an error, or you don't see the contents of your project directory and your home directory) you'll need to troubleshoot:

* "port is already allocated": See [troubleshooting](troubleshooting.md).
* `invalid mount config for type "bind": bind mount source path does not exist: <some path>` means the filesystem isn't successfully shared into the docker container.
* "The path ... is not shared and is not known to Docker": Visit docker's preferences/settings->File sharing and share the appropriate path or drive.
* "Error response from daemon: Get https://registry-1.docker.io/v2/" - Docker may not be running (restart it) or you may not have any access to the internet.
* "403 authentication required" when trying to `ddev start`: Try `docker logout` and do it again. Docker authentication is *not* required for any normal ddev action.
 
If you are on Docker Desktop for Windows or Docker Desktop for Mac and you are seeing shared directories not show up in the web container (nothing there when you `ddev ssh`) then:

* Unshare and then reshare the drive (you may have to re-enter your credentials)
* Consider resetting Docker to factory defaults. This often helps in this situation because Docker goes through the whole authentication process again.

If you are on Linux, the most common problem is having an old docker-compose, since the docker-compose that installs by default is incompatible with ddev. You'll find out about this right away because ddev will tell you on `ddev start` or most other ddev commands.

If you are on Docker Toolbox on Windows, the most common problem is trying to put the project directory outside the home directory.
