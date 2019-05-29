<h1>Uninstalling DDEV-Local</h1>

A DDEV-Local installation consists of:

* The binary itself (self-contained)
* The .ddev folder in a project
* The ~/.ddev folder where various global items are stored.
* The docker images and containers created. 
* Any entries in /etc/hosts

To uninstall a project:

`ddev stop --remove-data --stop-ssh-agent` and `rm -r .ddev`

To uninstall the global .ddev: `rm -r ~/.ddev`

To remove all /etc/hosts entries owned by ddev: `ddev hostname --remove-inactive`

If you installed docker just for ddev and want to uninstall it with all containers and images, just uninstall it for your version of Docker. [google link](https://www.google.com/search?q=uninstall+docker&oq=uninstall+docker&aqs=chrome.0.0j69i60j0l2j35i39j0.1970j0j4&sourceid=chrome&ie=UTF-8). 

Otherwise:
* To remove all ddev docker containers that might still exist: `docker rm $(docker ps -a | awk '/ddev/ { print $1 }')`
* To remove all ddev docker images that might exist: `docker rmi $(docker images | awk '/ddev/ {print $3}')`
* To remove any docker volumes: `docker volume rm $(docker volume ls | awk '/ddev|-mariadb/ { print $2 }') `

To remove the ddev binary:
* On macOS with homebrew or Linux with Linuxbrew, `brew uninstall ddev`
* For linux or other simple installs, just remove the binary, for example `sudo rm /usr/local/bin/ddev`
* On Windows (if you used the ddev Windows installer) use the uninstall on the start menu.
