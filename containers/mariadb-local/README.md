# Mariadb-local for ddev

This docker image builds a mariadb container for ddev.

It builds/copies a simple starter database (an empty database named "db") and starts up the mariadb server.

# Updating the default starter mariadb databases

In the future there may be a need to add another database or rename a database, etc.

The create_base_db.sh script is there for that. You can run it from the
root of this repository like this and it will update the db starter file:


```
docker run -it -v "$PWD/files/var/tmp:/mysqlbase" --rm --entrypoint=/create_base_db.sh drud/mariadb-local:<your_version>
```

Of course the assumption is that you might have to change the name of the output
file or make other changes in the process.

But then rebuild the container with whatever other changes you're working on.
