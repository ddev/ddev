<h1>Building, Testing, and Contributing</h1>

## Building

In the past, ddev would be checked out in the $GOPATH, but as of go 1.11, this is no longer appropriate. You should check out your fork *outside* the $GOPATH. 

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


## Testing
Normal test invocation is just `make test`. Run a single test with an invocation like `go test -v -run TestDevAddSites ./pkg/...` or `make testpkg TESTARGS="-run TestDevAddSites"`.

To see which ddev commands the tests are executing, set the environment variable DRUD_DEBUG=true.

Use GOTEST_SHORT=true to run just one CMS in each test.

## Docker container development

The docker containers that ddev uses are included in the containers/ directory:

* containers/ddev-webserver: Provides the web servers (the "web" container).
* containers/ddev-dbserver: Provides the "db" container.
* containers/ddev-bgsync: Fast web directory syncing
* containers/phpmyadmin: Provides the phpmyadmin (dba) container
* containers/ddev-router: The router image

When changes are made to a container, they have to be temporarily pushed to a tag that is preferably the same as the branch name of the PR, and the tag updated in pkg/version/version.go. Just ask if you need a container pushed to support a PR.

## Contributing
Interested in contributing to ddev? We would love your suggestions, contributions, and help! Please review our [Guidelines for Contributing](https://github.com/drud/ddev/blob/master/CONTRIBUTING.md), then [create an issue](https://github.com/drud/ddev/issues/new) or open a pull request!
