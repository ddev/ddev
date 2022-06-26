## Special Environments

### Mac M1 Apple Silicon and Docker Desktop

The Apple Mac M1 with its arm64 architecture has arrived to great acclaim. People love it.

Docker on the Mac M1 requires the Apple Silicon version of Docker Desktop, which you can get at [Docker Desktop Download](https://www.docker.com/products/docker-desktop) - you want "Mac with Apple Chip".

There are a few limitations of DDEV-Local on the Mac M1.

* Only mysql 5.7 and 8.0 are available for arm64/Mac M1 users.
* Because MariaDB does not publish arm64 packages or images for versions that are out of support, only MariaDB versions 10.1 and higher are supported by DDEV.

Both [Mutagen](performance.md#using-mutagen) and [NFS mounting](performance.md#macos-nfs-setup) are supported and work great on the Mac M1.
