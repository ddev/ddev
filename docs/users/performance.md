<h1>Performance</h1>

Every developer wants both quick starts of the environment and quick response to web page requests. DDEV-Local is always focused on improving this. However, both Docker Desktop for Windows and Docker Desktop for Mac have significant performance problems with mounted filesystems (like the mounted project where code can be edited either inside the container or on the host). There are currently two ways to work around this Docker performance issue.

## Using NFS to Mount the Project into the Container

NFS (Network File System) is a classic, mature Unix technique to mount a filesystem from one device to another. It's provided as an experimental technique for improving webserving performance. DDEV-Local supports this technique, but it **does requires a small amount of configuration on your host computer.**

__Before starting with NFS mounting, please make sure your project runs successfully without NFS mounting (without `nfs_mount_enabled: true` in your project's `.ddev/config.yaml` file.)__

To enable NFS mounting, use `nfs_mount_enabled: true` in your .ddev/config.yaml, or `ddev config --nfs-mount-enabled=true`. This won't work until you have done the one-time configuration for your system described below.

Note that you can use the NFS setup described here (and the scripts provided) or you can set up NFS any way that works for you. For example, if you're already using NFS with vagrant on macOS,and you already have a number of exports, the default export here (/Users) won't work, because you'll have mount overlaps. Or on Windows, you may want to use an NFS server other than Winnfsd, for example the [Allegrao NFS Server](https://nfsforwindows.com). The setups provided below and the scripts provided below are just intended to get you started if you don't already use NFS.

### macOS NFS Setup

__macOS Mojave (and later) warning:__ You'll need to give your terminal "Full disk access" before you (or the script provided) can edit /etc/exports. If you're using iterm2, here are [full instructions for iterm2](https://gitlab.com/gnachman/iterm2/wikis/fulldiskaccess). The basic idea is that in the Mac preferences -> Security and Privacy -> Privacy you need to give "Full Disk Access" permissions to your terminal app. Note that the "Full Disk Access" privilege is only needed when the /etc/exports file is being edited by you, normally a one-time event.

Download, inspect, and run the macos_ddev_nfs_setup.sh script  from [macos_ddev_nfs_setup.sh](https://raw.githubusercontent.com/drud/ddev/master/scripts/macos_ddev_nfs_setup.sh)). This stops running ddev projects, adds your home directory to the /etc/exports config file that nfsd uses, and enabled nfsd to run on your computer. This is one-time setup. Note that this shares the /Users directory via NFS to any client, so it's critical to consider security issues and verify that your firewall is enabled and configured. If your DDEV-Local projects are set up outside /Users, you'll need to edit /etc/exports for the correct values.

Note: If you're on macOS Catalina and above, and your projects are in a subdirectory of the ~/Documents directory, you must grant "Full Disk Access" privilege to /sbin/nfsd in the Privacy settings in the System Control Panel. 

### Windows NFS Setup

The executable components required for Windows NFS (winnfsd and nssm) are packaged with the ddev_windows_installer in each release, so if you've used the windows installer, they're available already. To enable winnfsd as a service, please download, inspect and run the script "windows_ddev_nfs_setup.sh" installed by the installer (or download from [windows_ddev_nfs_setup.sh](https://raw.githubusercontent.com/drud/ddev/master/scripts/windows_ddev_nfs_setup.sh)) in a git-bash session on windows. If your DDEV-Local projects are set up outside your home directory, you'll need to edit the ~/.ddev/nfs_exports.sh created by the script and then restart the service with `sudo nssm restart nfsd`.

**Firewall issues**: On Windows 10 you will likely run afoul of the Windows Defender Firewall, and it will be necessary to allow winnfsd to bypass it. If you're getting a timeout with no information after `ddev start`, try going to "Windows Defender Firewall" -> "Allow an app or feature through Windows Defender Firewall", "Change Settings", "Allow another app". Then choose C:\Program Files\ddev\winnfsd.exe, assuming that's where you installed winnfsd.

Also see the debugging section below, and the special WIndows debugging section.

### Debian/Ubuntu Linux NFS Setup

The nfsmount_enabled feature does not really add performance on Linux systems because Docker on Linux is already quite fast. The primary reason for using it on a Linux systme would be just to keep consistent with other team members working on other host OSs.

Note that for all Linux systems, you can and should install and configure the NFS daemon and configure /etc/exports as you see fit and share the directories that you choose to share. The Debian/Ubuntu Linux script is just one way of accomplishing it. 

Download, inspect, and run the [debian_ubuntu_linux_ddev_nfs_setup.sh](https://raw.githubusercontent.com/drud/ddev/master/scripts/debian_ubuntu_linux_ddev_nfs_setup.sh)). This stops running ddev projects, adds your home directory to the /etc/exports config file that nfs uses, and installs nfs-kernel-server  on your computer. This is one-time setup. 

Note that the script sets up a very restrictive /etc/exports that is based on the primary IP address of the Linux system at the time the script is run. You may want to edit it to make it less restrictive, or make it very open and use your firewall to control access.

### Debugging `ddev start` failures with `nfs_mount_enabled: true`

There are a number of reasons that the NFS mount can fail on `ddev start`:

* NFS Server not running
* Trying to start more than one NFS server. (This is typically only an issue on Windows)
* NFS exports overlap. This is typically an issue if you've had another NFS setup (like vagrant). You'll need to reconfigure your exports paths so they don't overlap.
* Path of project not shared in `/etc/exports` (or `~/.ddev/nfs_exports.txt` on Windows)
* Primary IP address not properly listed in /etc/exports (Linux)

Tools to debug and solve permission problems:

* Try `ddev debug nfsmount` to see if basic NFS mounting is working. If that works, it's likely that everything else will.
* When debugging, please do `ddev restart` in between each change. Otherwise, you can have stale mounts inside the container and you'll miss any benefit you may find in the debugging process.
* Inspect the /etc/exports (or `~/.ddev/nfs_exports.txt` on Windows).
* Restart the server (`sudo nfsd restart` on macOS, `sudo nssm restart nfsd` on Windows, `sudo systemctl restart nfs-kernel-server` on Debian/Ubuntu, other commaonds for other Unices).
* `showmount -e` on macOS or Linux will show the shared mounts.
* On Linux, the primary IP address needs to be in /etc/exports. Temporarily set the share in /etc/exports to `/home *`, which shares /home with any client, and `sudo systemctl restart nfs-kernel-server`. Then start a ddev project doing an nfs mount, and `showmount -a` and you'll find out what the primary IP address in use is. You can add that address to /etc/exports.

### macOS Catalina Upgrades

If you're upgrading an existing NFS/ddev setup and you've upgraded to macOS Catalina, the share path format in /etc/exports has been changed. If you previously had a line in /etc/exports like `/Users/rfay -alldirs -mapall=501:20 localhost` it will have to be changed to something like `/System/Volumes/Data/Users/rfay/workspace -alldirs -mapall=501:20 localhost` (Add "/System/Volumes/Data" to the front of the shared path.) You can also just run the NFS setup script [macos_ddev_nfs_setup.sh](https://raw.githubusercontent.com/drud/ddev/master/scripts/macos_ddev_nfs_setup.sh) again and it will handle this, but it won't remove any obsolete or broken lines.

So Catalina upgrade step-by-step:

* Edit /etc/exports or run the NFS setup script [macos_ddev_nfs_setup.sh](https://raw.githubusercontent.com/drud/ddev/master/scripts/macos_ddev_nfs_setup.sh) again. If you previously had a line in /etc/exports like `/Users/rfay -alldirs -mapall=501:20 localhost` it will have to be changed to something like `/System/Volumes/Data/Users/rfay -alldirs -mapall=501:20 localhost` (Add "/System/Volumes/Data" to the front of the shared path.)
* `sudo nfsd restart`
* Use `ddev debug nfsmount` in a project directory to make sure it gives successful output like
    ```
    $ ddev debug nfsmount
    Successfully accessed NFS mount of /Users/rfay/workspace/d8composer
    TARGET    SOURCE                                                FSTYPE OPTIONS
    /nfsmount :/System/Volumes/Data/Users/rfay/workspace/d8composer nfs    rw,relatime,vers=3,rsize=65536,wsize=65536,namlen=255,hard,nolock,proto=tcp,timeo=600,retrans=2,sec=sys,mountaddr=192.168.65.2,mountvers=3,mountproto=tcp,local_lock=all,addr=192.168.65.2
    /nfsmount/.ddev
    ```

Remember to use `ddev debug nfsmount` to verify

### macOS-specific NFS debugging

* Use `showmount -e` to find out what is exported via NFS. If you don't see a parent of your project directory in there, then NFS can't work.
* If nothing is showing, use `nfsd checkexports` and read carefully for errors
* Use `ps -ef | grep nfsd` to make sure nfsd is running
* Restart nfsd with `sudo nfsd restart`
* Add the following to your /etc/nfsd.conf:
  ```
  nfs.server.mount.require_resv_port = 0
  nfs.server.verbose = 3
  ```
* Run Console.app and put "nfs" in the search box at the top. `sudo nfsd restart` and read the messages carefully. Attempt to `ddev debug nfsmount` the problematic project directory.

### Windows-specific NFS debugging

* You can only have one NFS daemon running, so if another application has installed one, you'll want to use that NFS daemon and reconfigure it to allow NFS mounts of your projects. 

1. Stop the running winnfsd service: `sudo nssm stop nfsd`
2. Run winnfsd manually in the foreground: `winnfsd "C:\\"`. If it returns to the shell prompt immediately there's likely another nfsd service running. 
3. In another window, in a ddev project directory, `ddev debug nfsmount` to see if it can mount successfully. (The project need not be started.). `ddev debug nfsmount` is successful, then everything is probably going to work.
4. After verifying that ~/.ddev/nfs_exports.txt has a line that includes your project directories, `sudo nssm start nfsd` and `nssm status nfsd`. The status command should show SERVICE_RUNNING.
5. These [nssm](https://nssm.cc/) commands may be useful: `nssm help`, `sudo nssm start nfsd`, `sudo nssm stop nfsd`, `nssm status nfsd`, `sudo nssm edit nfsd` (pops up a window that may be hidden), and `sudo nssm remove nfsd` (also pops up a window, doesn't work predictably if you haven't already stopped the service). 
6. nssm logs failures and what it's doing to the system event log. Run "Event Viewer" and filter events as in the image below: ![Windows Event Viewer](images/windows_event_viewer.png).
7. Please make sure you have excluded winnfsd from the Windows Defender Firewall, as described in the installation instructions above.
8. On Windows 10 Pro you can "Turn Windows features on or off" and enable "Services for NFS"-> "Client for NFS". The `showmount -e` command will then show available exports on the current machine. This can help find out if a conflicting server is running or exactly what the problem with exports may be.

<a name="webcache"></a>
## Using webcache_enabled to Cache the Project Directory (deprecated, macOS only)

A separate webcache container is also provided as a separate (deprecated) performance technique; this works only on macOS. It does not rely on any host configuration, but in some cases when large changes are made in the filesystem it can stop syncing and be unstable.

Although webcache is deprecated, some people still use it for running tests in the web container, where consistency is not important and performance is highly valued.

We no longer run automated tests against the webcache feature.

To enable, edit .ddev/config.yaml to set `webcache_enabled:true` and `ddev start` to get a caching container going so that actual webserving happens on a much faster filesystem. This was experimental and was not reliable enough for most uses (the consistency with the host filesystem would sometimes be lost). It takes longer to do a ddev start because your entire project has to be pushed into the container, but after that hitting a page is way, way more satisfying. Note that .git directories are not copied into the webcache, git won't work inside the web container. It just seemed too risky to combine 2-way file synchronization with your precious git repository, so do git operations on the host. Note that if you have a lot of files or big files in your repo, they have to be pushed into the container, and that can take time. For example, cleaning up the .ddev/db_snapshots directory rather than waiting for the docker cp is a good idea.
