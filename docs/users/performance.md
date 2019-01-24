<h1>Performance</h1>

Every developer wants both quick starts of the environment and quick response to web page requests. DDEV-Local is always focused on improving this. However, both Docker for Windows and Docker for Mac have significant performance problems with mounted filesystems (like the mounted project where code can be edited either inside the container or on the host). There are currently two ways to work around this Docker performance issue.

## Using NFS to Mount the Project into the Container

NFS (Network File System) is a classic, mature Unix technique to mount a filesystem from one device to another. It's provided as an experimental technique for improving webserving performance. DDEV-Local supports this technique, but it **does requires a small amount of configuration on your host computer.**

To enable NFS mounting, use `nfs_mount_enabled: true` in your .ddev/config.yaml, or `ddev config --nfs-mount-enabled=true`. This won't work until you have done the one-time configuration for your system described below.

### macOS NFS Setup

Download, inspect, and run the macos_ddev_nfs_setup.sh script provided by homebrew or install_ddev.sh (or download from [macos_ddev_nfs_setup.sh](https://raw.githubusercontent.com/drud/ddev/master/scripts/macos_ddev_nfs_setup.sh)). This stops running ddev projects, adds the /Users directory to the /etc/exports config file that nfsd uses, and enabled nfsd to run on your computer. This is one-time setup. Note that this shares the /Users directory via NFS to any client, so it's critical to consider security issues and verify that your firewall is enabled and configured. If your DDEV-Local projects are set up outside /Users, you'll need to edit /etc/exports for the correct values.

### Windows NFS Setup

The executable components required for Windows NFS (winnfsd and nssm) are packaged with the ddev_windows_installer in each release, so if you've used the windows installer, they're available already. To enable winnfsd as a service, please download, inspect and run the script "windows_ddev_nfs_setup.sh" installed by the installer (or download from [windows_ddev_nfs_setup.sh](https://raw.githubusercontent.com/drud/ddev/master/scripts/windows_ddev_nfs_setup.sh)) in a git-bash session on windows. This is one-time setup. Note that this shares the C:\Users directory via NFS to any client, so it's critical to consider security issues and verify that your firewall is enabled and configured. If your DDEV-Local projects are set up outside C:\Users, you'll need to edit the configuration used here.

### Debian/Ubuntu NFS Setup

Note that for all Linux systems, you can and should install and configure the NFS daemon and configure /etc/exports as you see fit and share the directories that you choose to share. The debian/ubuntu script is just one way of accomplishing it. 

Download, inspect, and run the [debian_ubuntu_ddev_nfs_setup.sh](https://raw.githubusercontent.com/drud/ddev/master/scripts/debian_ubuntu_ddev_nfs_setup.sh)). This stops running ddev projects, adds the /home directory to the /etc/exports config file that nfs uses, and installs nfs-kernel-server  on your computer. This is one-time setup. Note that this shares the /home directory via NFS to all non-routeable ("public") IP addresses in your network, so it's critical to consider security issues and verify that your firewall is enabled and configured. If your DDEV-Local projects are set up outside /home, you'll need to edit /etc/exports for the correct values and restart nfs-kernel-server.

## Using webcache_enabled to Cache the Project Directory

A separate webcache container is also provided as a separate experimental performance technique. It does not rely on any host configuration, but in some cases when large changes are made in the filesystem it can stop syncing and be unstable.

To enable, edit .ddev/config.yaml to set `webcache_enabled:true` and `ddev start` to get a caching container going so that actual webserving happens on a much faster filesystem. This is experimental and has some risks, we want to know your experience. It takes longer to do a ddev start because your entire project has to be pushed into the container, but after that hitting a page is way, way more satisfying. Note that .git directories are not copied into the webcache, git won't work inside the web container. It just seemed too risky to combine 2-way file synchronization with your precious git repository, so do git operations on the host. Note that if you have a lot of files or big files in your repo, they have to be pushed into the container, and that can take time. For example, cleaning up the .ddev/db_snapshots directory rather than waiting for the docker cp is a good idea.
