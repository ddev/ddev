
export HISTFILE=/mnt/ddev-global-cache/bashhistory/${HOSTNAME}/bash_history
export HISTSIZE=10000
export HISTFILESIZE=100000

# mysql history to be shared between web and db
mkdir -p /mnt/ddev-global-cache/mysqlhistory/${DDEV_PROJECT}
export MYSQL_HISTFILE=/mnt/ddev-global-cache/mysqlhistory/${DDEV_PROJECT}/mysql_history
