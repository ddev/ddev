## Docker Installation

### macOS Installation: Docker Desktop for Mac

Docker Desktop for Mac can be installed via Homebrew (`brew install  homebrew/cask/docker`) or can be downloaded from [desktop.docker.com](https://www.docker.com/products/docker-desktop).

### macOS Installation: Colima

[Colima](https://github.com/abiosoft/colima) is an open-source project that bundles the container management tool [lima](https://github.com/lima-vm/lima) with a docker (linux) back-end. This is similar to what Docker Desktop actually does, but Colima and Lima are entirely open-source and just focused on running containers. They work on both `amd64` and `arm64` (M1) macs. Colima does not require installation of Docker Desktop, or does it require paying a license fee to Docker, Inc.

Reasons to use Colima include:

* Preferring to use open-source software (Docker Desktop, unlike Docker, is proprietary software).
* Working for an organization that due to its size requires a paid Docker plan to use Docker Desktop, and wanting to avoid that cost and business relationship.
* Preferring a CLI-focused approach to Docker Desktop's GUI focus.

* Install colima with `brew install colima` using homebrew or see the other [installation options](https://github.com/abiosoft/colima/blob/main/docs/INSTALL.md).
* If you don't have Docker Desktop installed, you'll need the docker client, `brew install docker`.
* Start colima: `colima start --cpu 4 --memory 4` will set up a colima instance with 4 CPUs and 4GB of memory allocated. Your needs may vary. After the first start you can just use `colima start`.
* `colima status` will show colima's status.
* After a computer restart you'll need to `colima start` again.
* Colima activates its own docker context in order to not conflict with Docker Desktop, so if you `docker context ls` you'll see a list of available contexts with currently active context indicated with an "\*" (which will be "colima" after you've started colima). You can change to the default (Docker Desktop) with `docker context use default` or change back with `docker context use colima`.
* For webserver performance and predictability mutagen is recommended, `ddev config global --mutagen-enabled`. See [Performance docs](performance.md#using-mutagen). Since the file mounting technique on lima/colima is immature (sshfs) you may want to just use no-bind-mounts, `ddev config global --no-bind-mounts` (which also implies mutagen).

DDEV has extensive automated test coverage for colima on macOS, but of course colima is new and this integration is new, so please share your results and open issues or [contact us](../index.md#support-and-user-contributed-documentation) for help.

### Windows Installation: Docker Desktop for Windows

Docker Desktop for Windows can be downloaded via [Chocolatey](https://chocolatey.org/install) with `choco install docker-desktop` or it can be downloaded from [download.docker.com](https://www.docker.com/products/docker-desktop).

### Windows Installation: WSL2 with Docker (Linux) Installed Inside

Although the traditional approach on Windows/WSL2 has been to use Docker Desktop, a number of people have moved away from using Docker Desktop and just installing the Docker-provided open-source `docker-ce` package inside WSL2. This uses entirely open-source software and does not require a license fee to Docker, Inc.

Most of the installation is the same as [on Linux](#linux-installation-docker), but it can be summarized as:

* If you don't already have WSL2, install it with `wsl --install`, which will likely require a reboot.
* `wsl --set-default-version 2`
* Install a distro. Ubuntu 20.04 is recommended, `wsl -s Ubuntu-20.04`.
* Install `docker-ce` in WSL2 using the normal Linux instructions, for Debian/Ubuntu follow these instructions inside the WSL2 distro:

```bash
sudo apt-get remove docker docker-engine docker.io containerd runc
sudo apt-get update && sudo apt-get install ca-certificates curl gnupg lsb-release
curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo gpg --dearmor -o /usr/share/keyrings/docker-archive-keyring.gpg
echo "deb [arch=$(dpkg --print-architecture) signed-by=/usr/share/keyrings/docker-archive-keyring.gpg] https://download.docker.com/linux/ubuntu $(lsb_release -cs) stable" | sudo tee /etc/apt/sources.list.d/docker.list > /dev/null
sudo apt-get update && sudo apt-get install docker-ce docker-ce-cli containerd.io
sudo groupadd docker && sudo usermod -aG docker $USER

```

* You have to start docker-ce yourself on login, or use a script to do it. To have it start on entry to git-bash, a startup line to your (windows-side) `~/.bashrc` with:

  ```bash
  echo "wsl.exe -u root service docker status > /dev/null || wsl.exe -u root service docker start > /dev/null" >> ~/.bashrc
  ```

  You can then `source ~/.bashrc` to start immediately, or it should start the next time you open git-bash.
* [Install mkcert](https://github.com/FiloSottile/mkcert#windows) on the Windows side; this may be easiest with [Chocolatey](https://chocolatey.org/install): In an administrative PowerShell, `Set-ExecutionPolicy Bypass -Scope Process -Force; [System.Net.ServicePointManager]::SecurityProtocol = [System.Net.ServicePointManager]::SecurityProtocol -bor 3072; iex ((New-Object System.Net.WebClient).DownloadString('https://chocolatey.org/install.ps1'))`
* In an administrative PowerShell: `choco install -y mkcert`
* In an administrative PowerShell, run `mkcert -install` and answer the prompt allowing the installation of the Certificate Authority.
* In an administrative PowerShell, run the command `setx CAROOT "$(mkcert -CAROOT)"; If ($Env:WSLENV -notlike "*CAROOT/up:*") { setx WSLENV "CAROOT/up:$Env:WSLENV" }`. This will set WSL2 to use the Certificate Authority installed on the Windows side.
* Double-check in Ubuntu (or your distro): `echo $CAROOT` should show something like `/mnt/c/Users/<you>/AppData/Local/mkcert`
* Inside your WSL2 distro, `mkcert -install`..

### Linux Installation: Docker

* **Please don't forget that Linux installation absolutely requires post-install steps (below).**

* **Please don't use `sudo` with docker. If you're needing it, you haven't finished the installation. Don't use `sudo` with ddev, except the rare case where you need the `ddev hostname` command.**

Docker installation on Linux depends on what flavor you're using. Where possible the Ubuntu/Deb/yum repository is the preferred technique.

* [Ubuntu](https://docs.docker.com/install/linux/docker-ce/ubuntu/)
* [CentOS](https://docs.docker.com/install/linux/docker-ce/centos/)
* [Debian](https://docs.docker.com/install/linux/docker-ce/debian/)
* [Fedora](https://docs.docker.com/install/linux/docker-ce/fedora/)
* [binaries](https://docs.docker.com/install/linux/docker-ce/binaries/)

#### Linux Post-installation steps (required)

See [Docker's post-installation steps](https://docs.docker.com/install/linux/linux-postinstall/). You need to add your linux user to the "docker" group and configure the docker daemon to start on boot.

On systems that do not include systemd or equivalent (mostly if installing inside WSL2) you'll need to manually start docker with `service docker start` or the equivalent in your distro. You can add this into your .profile or equivalent.

<a name="troubleshooting"></a>

### Testing and Troubleshooting Your Docker Installation

Docker needs to be able to a few things for ddev to work:

* Mount the project code directory from the host into the container; the project code directory is usually somewhere in a subdirectory of your home directory.
* Access TCP ports on the host to serve HTTP and HTTPS. These are ports 80 and 443 by default, but they can be changed on a per-project basis.

We can use a single docker command to make sure that docker is set up to do what we want:

**On Windows this command should be run in git-bash.** In your *project directory* run `docker run --rm -t -p 80:80 -v "/$PWD:/tmp/projdir" busybox sh -c "echo ---- Project Directory && ls /tmp/projdir"` - you should see the contents of your project directory displayed. (On Windows, make sure you run this using git-bash)

If that fails (if you get an error, or you don't see the contents of your project directory and your home directory) you'll need to troubleshoot:

* "port is already allocated": See [troubleshooting](troubleshooting.md).
* `invalid mount config for type "bind": bind mount source path does not exist: <some path>` means the filesystem isn't successfully shared into the docker container.
* "The path ... is not shared and is not known to Docker": Visit docker's preferences/settings->File sharing and share the appropriate path or drive.
* `Error response from daemon: Get https://registry-1.docker.io/v2/` - Docker may not be running (restart it) or you may not have any access to the internet.
* "403 authentication required" when trying to `ddev start`: Try `docker logout` and do it again. Docker authentication is *not* required for any normal ddev action.

If you are on Docker Desktop for Windows or Docker Desktop for Mac and you are seeing shared directories not show up in the web container (nothing there when you `ddev ssh`) then:

* Unshare and then reshare the drive (you may have to re-enter your credentials)
* Consider resetting Docker to factory defaults. This often helps in this situation because Docker goes through the whole authentication process again.

### Experimental Configurations

#### Remote Docker Instances

You can use remote docker instances, whether on the internet or inside your network or running in a virtual machine.

* On the remote machine, the docker port must be exposed if it is not exposed already. See [instructions](https://gist.github.com/styblope/dc55e0ad2a9848f2cc3307d4819d819f) for how to do this on a systemd-based remote server. **Be aware that this has serious security implications and must not be done without taking those into consideration.**. In fact, dockerd will complain

> Binding to IP address without --tlsverify is insecure and gives root access on this machine to everyone who has access to your network.  host="tcp://0.0.0.0:2375".
> Binding to an IP address, even on localhost, can also give access to scripts run in a browser. Be safe out there!  host="tcp://0.0.0.0:2375"
> Binding to an IP address without --tlsverify is deprecated. Startup is intentionally being slowed down to show this message  host="tcp://0.0.0.0:2375"
> Please consider generating tls certificates with client validation to prevent exposing unauthenticated root access to your network  host="tcp://0.0.0.0:2375"
> You can override this by explicitly specifying '--tls=false' or '--tlsverify=false'  host="tcp://0.0.0.0:2375"
> Support for listening on TCP without authentication or explicit intent to run without authentication will be removed in the next release  host="tcp://0.0.0.0:2375"
  "

* If you do not have the docker client installed another way (like from Docker Desktop) then install it with `brew install docker` to get just the client.
* Create a docker context that points to the remote docker instance. For example, if the remote hostname is `debian-11` then `docker context create debian-11 --docker host=tcp://debian-11:2375 && docker use debian-11`. Alternately, you can use the `DOCKER_HOST` environment variable, for example `export DOCKER_HOST=tcp://debian-11:2375`.
* Make sure you can access the remote machine using `docker ps`.
* Bind-mounts cannot work on a remote docker setup, so you must use `ddev config global --no-bind-mounts`. This will cause ddev to push needed information to and from the remote docker instance when needed. This also automatically turns on mutagen caching.
* You may want to use a FQDN other than `*.ddev.site` because the ddev site will *not* be at `127.0.0.1`. For example, `ddev config --fqdns=debian-11` and then use `https://debian-11` to access the site.
* If the docker host is reachable on the internet, you can actually enable real https for it using Let's Encrypt as described in [Casual Webhosting](alternate-uses.md#casual-project-webhosting-on-the-internet-including-lets-encrypt). Just make sure that port 2375 is not available on the internet.

#### Rancher Desktop on macOS

[Rancher Desktop](https://rancherdesktop.io/) is another Docker Desktop alternative that is quickly maturing for macOS. You can install it for many target platforms from the [release page](https://github.com/rancher-sandbox/rancher-desktop/releases).

Rancher desktop integration currently has no automated testing for DDEV integration.

* By default, Rancher desktop will provide a version of the docker client if you do not have one on your machine.

* Rancher changes over the "default" context in docker, so you'll want to turn off Docker Desktop if you're using it.
* Rancher Desktop does not provide bind mounts, so use `ddev config global --no-bind-mounts` which also turns on mutagen.
* Use a non-`ddev.site` name, `ddev config --additional-fqdns=rancher` for example, because the resolution of `*.ddev.site` seems to make it not work.
* Rancher Desktop does not seem to currently work with `mkcert` and `https`, so turn those off with `mkcert -uninstall && rm -r "$(mkcert -CAROOT)"`. This does no harm and can be undone with just `mkcert -install` again.
