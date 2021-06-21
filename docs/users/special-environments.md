## Special Environments

### Mac M1 Apple Silicon and Docker Desktop

The Apple Mac M1 with its arm64 architecture has arrived to great acclaim. People love it.

Docker on the Mac M1 requires the Apple Silicon version of Docker Desktop, which you can get at https://www.docker.com/products/docker-desktop "Mac with Apple Chip".

There are a few limitations of DDEV-Local on the Mac M1.

* Because Oracle is not (yet?) publishing packages for MySQL on arm64, there are no MySQL images in library/mysql. This means that DDEV-Local does not allow you to choose mysql as a database type.
* Because MariaDB does not publish arm64 packages or images for versions that are out of support, only MariaDB versions 10.1 and higher are supported by DDEV-Local.

NFS mounting is supported and works well on the Mac M1, and the [NFS setup](https://ddev.readthedocs.io/en/latest/users/performance/#macos-nfs-setup) remains the same.
