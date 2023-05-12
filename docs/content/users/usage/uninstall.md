# Uninstalling DDEV

A DDEV installation consists of:

* The self-contained `ddev` binary.
* Each project’s `.ddev` directory.
* The global `~/.ddev` directory where various global items are stored.
* The global `~/.ddev_mutagen_data_directory` directory where Mutagen sync data may be stored.
* The associated Docker images and containers DDEV created.
* Any entries in `/etc/hosts`.

Please use [`ddev snapshot`](commands.md#snapshot) or [`ddev export-db`](commands.md#export-db) to make backups of your databases before deleting projects or uninstalling.

You can use [`ddev clean`](commands.md#clean) to uninstall the vast majority of things DDEV has touched. For example, `ddev clean <project>` or `ddev clean --all`.

To uninstall one project, run [`ddev delete <project>`](commands.md#delete). This removes any hostnames in `/etc/hosts` and removes your database. If you don’t want it to make a database backup/snapshot on the way down, include the `--omit-snapshot` option: `ddev delete --omit-snapshot <project>`.

To remove all DDEV-owned `/etc/hosts` entries: [`ddev hostname --remove-inactive`](commands.md#hostname).

To remove the global `.ddev` directory: `rm -r ~/.ddev`.

To remove the global `.ddev_mutagen_data_directory` directory: `rm -r ~/.ddev_mutagen_data_directory`.

If you installed Docker only for DDEV and want to uninstall it with all containers and images, uninstall it for your version of Docker.

Otherwise:

* Remove Docker images from before the current DDEV release with [`ddev delete images`](commands.md#delete-images).
* Remove all DDEV Docker containers that might still exist: `docker rm $(docker ps -a | awk '/ddev/ { print $1 }')`.
* Remove all DDEV Docker images that might exist: `docker rmi $(docker images | awk '/ddev/ {print $3}')`.
* Remove all Docker images of any type (does no harm; they’ll be re-downloaded): `docker rmi -f $(docker images -q)`.
* Remove any Docker volumes: `docker volume rm $(docker volume ls | awk '/ddev|-mariadb/ { print $2 }')`.

To remove the `ddev` binary:

* On macOS or Linux with Homebrew, `brew uninstall ddev`.
* For Linux or other simple installs, remove the binary. Example: `sudo rm /usr/local/bin/ddev`. For Linux installed via apt, `sudo apt remove ddev`.
* On Windows, if you used the DDEV Windows installer, use the uninstall on the Start Menu or in the “Add or Remove Programs” section of Windows Settings.
