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

<a name="troubleshooting"></a>
## Testing and Troubleshooting Your Docker Installation

Docker needs to be able to a few things for ddev to work:

* Mount the project code directory from the host into the container; the project code directory is usually somewhere in a subdirectory of your home directory. 
* Access TCP ports on the host to serve HTTP and HTTPS. These are ports 80 and 443 by default, but they can be changed on a per-project basis.
* Mount ~/.ddev for SSL cert cache and import-db. 

So we can use a single docker command to make sure that docker is set up to do what we want:

In your project directory run `docker run -t -p 80:80 -v "$PWD:/tmp/projdir" -v "$HOME:/tmp/homedir" busybox sh -c "echo ---- Project Directory && ls /tmp/projdir && echo ---- Home Directory && ls /tmp/homedir"` - you should see the contents of your home directory displayed. (On Windows, make sure you do this using git-bash or Docker Quickstart Terminal.)

If that fails (if you get an error, or you don't see the contents of your project directory and your home directory) you'll need to troubleshoot:

* "port is already allocated": See [troubleshooting](troubleshooting.md).
* `invalid mount config for type "bind": bind mount source path does not exist: <some path>` means the filesystem isn't successfully shared.
* "The path ... is not shared and is not known to Docker": Visit docker's preferences/settings->File sharing and share the appropriate path or drive.
* "Error response from daemon: Get https://registry-1.docker.io/v2/" - Docker may not be running (restart it) or you may not have any access to the internet.
* "403 authentication required" when trying to `ddev start`: Try `docker logout` and do it again. Docker authentication is *not* required for any normal ddev action.
 
If you are on Docker for Windows or Docker for Mac and you are seeing shared directories not show up in the web container (nothing there when you `ddev ssh`) then:

* Unshare and then reshare the drive
* Consider resetting Docker to factory defaults. This often helps in this situation because Docker goes through the whole authentication process again.

