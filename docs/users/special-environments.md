## Special Environments

### Mac M1 Apple Silicon and Docker Desktop Technical Preview

The Apple Mac M1 with its arm64 architecture has arrived to great acclaim. People love it.

As of early 2021, to use Docker on the Mac M1 you must install the [Docker Desktop Technical Preview for Mac M1](https://docs.docker.com/docker-for-mac/apple-m1/). You may encounter issues with it, and if so you'll want to get into the [Docker Desktop for Mac Issue Queue](https://github.com/docker/for-mac/issues).

There are a few limitations of DDEV-Local on the Mac M1.

* Because Oracle is not (yet?) publishing packages for MySQL on arm64, there are no MySQL images in library/mysql. This means that DDEV-Local does not allow you to choose mysql as a database type.
* Because MariaDB does not publish arm64 packages or images for versions that are out of support, only MariaDB 10.1 through 10.5 are supported by DDEV-Local.

NFS mounting is supported and works well on the Mac M1, and the [NFS setup](https://ddev.readthedocs.io/en/latest/users/performance/#macos-nfs-setup) remains the same.
