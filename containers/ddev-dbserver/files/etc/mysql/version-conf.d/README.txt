Each item in verion-conf.d is an override of the /etc/my.cnf,
which includes these.

The item that is most specific is linked at start time.

For example, for mariadb:10.8, the mariadb_10.8.cnf.txt is the most specific
so it will get linked to mariadb_10.8.cnf and will be included.

For mariadb:10.4, the closest match is mariadb.cnf.txt, so it gets linked to
mariadb.cnf.

Only one file will be linked, not all that might match.
