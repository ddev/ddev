# Building, Testing, and Contributing

## Building

If you have `make` and docker, you can build for your environment with just `make`. Since the Makefile uses docker to build, it's not generally essential to install go on your machine, although it will make things easier.

Build/test/check static analysis with:

 ```
 make
 make linux
 make darwin
 make windows
 make test
 make clean
 make staticrequired
 ```

The binaries are built into .gotmp/bin; although normal command-line `go build` or `go install` will work (and everything works fine with IDEs like Goland or vscode) the official build technique is via `make` which uses a completely consistent golang-build-container so that the build is identical no matter what machine or OS it might be built on.

## Testing

Normal test invocation is just `make test`. Run a single test with an invocation like `go test -v -run TestDevAddSites ./pkg/...` or `make testpkg TESTARGS="-run TestDevAddSites"`.

To see which ddev commands the tests are executing, set the environment variable DDEV_DEBUG=true.

Use GOTEST_SHORT=true to run just one CMS in each test.

## Docker container development

The docker containers that ddev uses are included in the containers/ directory:

* containers/ddev-webserver: Provides the web servers (the "web" container).
* containers/ddev-dbserver: Provides the "db" container.
* containers/phpmyadmin: Provides the phpmyadmin (dba) container
* containers/ddev-router: The router image

When changes are made to a container, they have to be temporarily pushed to a tag that is preferably the same as the branch name of the PR, and the tag updated in pkg/version/version.go. Just ask if you need a container pushed to support a PR.

## Contributing

Interested in contributing to ddev? We would love your suggestions, contributions, and help! Please review our [Guidelines for Contributing](https://github.com/drud/ddev/blob/master/CONTRIBUTING.md), then [create an issue](https://github.com/drud/ddev/issues/new) or open a pull request!
