#ddev-generated
Scripts in this directory will be executed inside the solr
container (if it exists, of course). This is just an example,
but any named service can have a directory with commands.

Note that /mnt/ddev_config must be mounted into the 3rd-party service
with a stanza like this in the docker-compose.solr.yaml:

    volumes:
      - type: "bind"
        source: "."
        target: "/mnt/ddev_config"


See https://ddev.readthedocs.io/en/stable/users/extend/custom-commands/#environment-variables-provided for a list of environment variables that can be used in the scripts.
