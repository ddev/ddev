## Customizing Docker Images

It's common to have a requirement for the web or db images which is not bundled in them by default. There are two easy ways to extend these docker images:

* `webimage_extra_packages` and `dbimage_extra_packages` in .ddev/config.yaml
* An add-on Dockerfile in your project's `.ddev/web-build` or `.ddev/db-build`

### Adding extra Debian packages with webimage_extra_packages and dbimage_extra_packages

You can add extra Debian packages if that's all that is needed with lines like this in `.ddev/config.yaml`:

```
webimage_extra_packages: [php-yaml, php7.3-ldap]
dbimage_extra_packages: [telnet, netcat]

```

Then the additional packages will be built into the containers during `ddev start`

### Adding extra Dockerfiles for webimage and dbimage

For more complex requirements, you can add .ddev/web-build/Dockerfile or .ddev/db-build/Dockerfile.

Examples of possible Dockerfiles are given in `.ddev/web-build/Dockerfile.example` and `.ddev/db-build/Dockerfile.example` (These examples are created in your project when you `ddev config` the project.)

You can use the .ddev/*-build/ directory as the Docker "context" directory as well. So for example if a file named README.txt exists in .ddev/web-build, you can use `ADD README.txt /` in the Dockerfile.

An example web image `.ddev/web-build/Dockerfile` might be:

```
ARG BASE_IMAGE
FROM $BASE_IMAGE
RUN npm install --global gulp-cli
ADD README.txt /
```

Note that if a Dockerfile is provided, any config.yaml `webimage_extra_packages` or `dbimage_extra_packages` will be ignored. If you need to add packages as well as other custom configuration, add them to your Dockerfile with a line like

```
RUN apt-get update && DEBIAN_FRONTEND=noninteractive apt-get install -y -o Dpkg::Options::="--force-confold" --no-install-recommends --no-install-suggests php-yaml php7.3-ldap
```
