## macOS Installation: Docker Desktop for Mac

Docker Desktop for Mac can be installed via Homebrew (`brew install  homebrew/cask/docker`) or can be downloaded from [desktop.docker.com](https://desktop.docker.com/mac/stable/Docker.dmg).

## Windows Installation: Docker Desktop for Windows

Docker Desktop for Windows can be downloaded via [Chocolatey](https://chocolatey.org/install) with `choco install docker-desktop` or it can be downloaded from [download.docker.com](https://download.docker.com/win/stable/Docker%20for%20Windows%20Installer.exe).

## Linux Installation: Docker and docker-compose

* __Please don't forget that Linux installation absolutely requires post-install steps (below).__

* __docker-compose must be installed or upgraded separately except on very recent distros, as it is not bundled with in the Docker repositories, see below.__

* __Please don't use `sudo` with docker. If you're needing it, you haven't finished the installation. Don't use `sudo` with ddev, except the rare case where you need the `ddev hostname` command.__

Docker installation on Linux depends on what flavor you're using. Where possible the Ubuntu/Deb/yum repository is the preferred technique.

* Ubuntu 20.04+ has recent enough versions that you can `sudo apt-get update && sudo apt-get install docker.io docker-compose`
* [Ubuntu before 20.04](https://docs.docker.com/install/linux/docker-ce/ubuntu/)
* [CentOS](https://docs.docker.com/install/linux/docker-ce/centos/)
* [Debian](https://docs.docker.com/install/linux/docker-ce/debian/)
* [Fedora](https://docs.docker.com/install/linux/docker-ce/fedora/). Recent versions of Fedora (32, 33+) require a [different approach, installing the original CGroups](https://fedoramagazine.org/docker-and-fedora-32/). In addition, you must [disable SELinux](https://www.cyberciti.biz/faq/disable-selinux-on-centos-7-rhel-7-fedora-linux/).
* [binaries](https://docs.docker.com/install/linux/docker-ce/binaries/)

After installing Docker you *must* install docker-compose separately except on Ubuntu 20.04+: If using Homebrew you can `brew install docker-compose`, otherwise [Follow download instructions](https://docs.docker.com/compose/install/#install-compose) (select "linux" tab). This really is just downloading docker-compose binary from <https://github.com/docker/compose/releases> and installing it in /usr/local/bin with executable permissions. On ARM64 computers you may have to install docker-compose using `pip install docker-compose` or `pip3 install docker-compose`.

### Linux Post-installation steps (required)

See [Docker's post-installation steps](https://docs.docker.com/install/linux/linux-postinstall/). You need to add your linux user to the "docker" group and configure the docker daemon to start on boot.

<a name="troubleshooting"></a>

## Testing and Troubleshooting Your Docker Installation

Docker needs to be able to a few things for ddev to work:

* Mount the project code directory from the host into the container; the project code directory is usually somewhere in a subdirectory of your home directory.
* Access TCP ports on the host to serve HTTP and HTTPS. These are ports 80 and 443 by default, but they can be changed on a per-project basis.

We can use a single docker command to make sure that docker is set up to do what we want:

**On Windows this command should be run in git-bash).** In your *project directory* run `docker run --rm -t -p 80:80 -v "/$PWD:/tmp/projdir" busybox sh -c "echo ---- Project Directory && ls /tmp/projdir"` - you should see the contents of your project directory displayed. (On Windows, make sure you run this using git-bash)

If that fails (if you get an error, or you don't see the contents of your project directory and your home directory) you'll need to troubleshoot:

* "port is already allocated": See [troubleshooting](troubleshooting.md).
* `invalid mount config for type "bind": bind mount source path does not exist: <some path>` means the filesystem isn't successfully shared into the docker container.
* "The path ... is not shared and is not known to Docker": Visit docker's preferences/settings->File sharing and share the appropriate path or drive.
* "Error response from daemon: Get <https://registry-1.docker.io/v2/"> - Docker may not be running (restart it) or you may not have any access to the internet.
* "403 authentication required" when trying to `ddev start`: Try `docker logout` and do it again. Docker authentication is *not* required for any normal ddev action.

If you are on Docker Desktop for Windows or Docker Desktop for Mac and you are seeing shared directories not show up in the web container (nothing there when you `ddev ssh`) then:

* Unshare and then reshare the drive (you may have to re-enter your credentials)
* Consider resetting Docker to factory defaults. This often helps in this situation because Docker goes through the whole authentication process again.

If you are on Linux, the most common problem is having an old docker-compose, since the docker-compose that installs by default is incompatible with ddev. You'll find out about this right away because ddev will tell you on `ddev start` or most other ddev commands.
