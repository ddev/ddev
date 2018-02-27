<h1>Building, Testing, and Contributing</h1>

## Building

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

To see which ddev commands the tests are executing, set the environment variable to DRUD_DEBUG=true.

## Contributing
Interested in contributing to ddev? We would love your suggestions, contributions, and help! Please review our [Guidelines for Contributing](https://github.com/drud/ddev/blob/master/CONTRIBUTING.md), then [create an issue](https://github.com/drud/ddev/issues/new) or open a pull request!
