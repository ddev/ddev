<h1>Building, Testing, and Contributing</h1>

## Building

As with all golang projects, you need to have your $GOPATH set up and ddev code must in the $GOPATH. This is an inflexibility of golang. We recommend that you set `GOPATH=~/go` and clone ddev into `~/go/src/github.com/drud/ddev`.

 ```
 make
 make linux
 make darwin
 make windows
 make test
 make clean
 ```

 Note that although this git repository contains submodules (in the containers/ directory) they are not used in a normal build, but rather by the nightly build. You can safely ignore the git submodules and the containers/ directory.


## Testing
Normal test invocation is just `make test`. Run a single test with an invocation like `go test -v -run TestDevAddSites ./pkg/...`

If you set the environment variable DRUD_DEBUG=true you can see what ddev commands are being executed in the tests.

## Docker container development

ddev depends on docker containers that are regularly updated and whose source code is in other repositories.

* [drud/nginx-php-fpm-local webimage](https://hub.docker.com/r/drud/nginx-php-fpm-local/) - https://github.com/drud/docker.nginx-php-fpm-local
* [drud/mariadb-local dbimage](https://hub.docker.com/r/drud/mariadb-local) - https://github.com/drud/mariadb-local
* [drud/phpmyadmin dbaimage](https://hub.docker.com/r/drud/phpmyadmin) - https://github.com/drud/docker.phpmyadmin
* [drud/ddev-router routerimage](https://hub.docker.com/r/drud/ddev-router) - https://github.com/drud/docker.ddev-router

## Contributing
Interested in contributing to ddev? We would love your suggestions, contributions, and help! Please review our [Guidelines for Contributing](https://github.com/drud/ddev/blob/master/CONTRIBUTING.md), then [create an issue](https://github.com/drud/ddev/issues/new) or open a pull request!
