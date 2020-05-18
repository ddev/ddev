# Building, Testing, and Contributing

## Pull Requests and PR Preparation

When preparing your pull request, please use a branch name like "2020_<your_username>_short_description" so that it's easy to track to you.

## Docker Image changes

If you make changes to a docker image (like ddev-webserver), it won't have any effect unless you

* Push a container to hub.docker.com. Push with the tag that matches your branch.
* Update pkg/version/version.go with the WebImg and WebTag that relate to the docker image you pushed.

## Building

Build the project with `make` and your resulting executable will end up in .gotmp/bin/ddev (for Linux) or .gotmp/bin/windows_amd64/ddev.exe (for Windows) or .gotmp/bin/darwin/ddev (for macOS).

Build/test/check static analysis with

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

Normal test invocation is just `make test`. Run a single test with an invocation like `go test -v -run TestDevAddSites ./pkg/...` or `make testpkg TESTARGS="-run TestDevAddSites"`. The easiest way to run tests is from inside the excellent golang IDE Goland. Just click the arrowhead to the left of the test name.

To see which ddev commands the tests are executing, set the environment variable DDEV_DEBUG=true.

Use GOTEST_SHORT=true to run just one CMS in each test, or GOTEST_SHORT=<integer> to run exactly one project type from the list of project types in the [TestSites array](https://github.com/drud/ddev/blob/a4ab2827d8b6e706b2420700045d889a3a69f3f2/pkg/ddevapp/ddevapp_test.go#L43). For example, GOTEST_SHORT=5 will run many tests only against TYPO3.

## Docker container development

The docker containers that ddev uses are included in the containers/ directory:

* containers/ddev-webserver: Provides the web servers (the "web" container).
* containers/ddev-dbserver: Provides the "db" container.
* containers/phpmyadmin: Provides the phpmyadmin (dba) container
* containers/ddev-router: The router image

When changes are made to a container, they have to be temporarily pushed to a tag that is preferably the same as the branch name of the PR, and the tag updated in pkg/version/version.go. Just ask if you need a container pushed to support a PR.

## Contributing

Interested in contributing to ddev? We would love your suggestions, contributions, and help! Please review our [Guidelines for Contributing](https://github.com/drud/ddev/blob/master/CONTRIBUTING.md), then [create an issue](https://github.com/drud/ddev/issues/new) or open a pull request!
