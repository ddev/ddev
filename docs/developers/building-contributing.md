<h1>Building, Testing, and Contributing</h1>

## Building

 ```
 make
 make linux
 make darwin
 make test
 make clean
 ```

 Note that although this git repository contains submodules (in the containers/ directory) they are not used in a normal build, but rather by the nightly build. You can safely ignore the git submodules and the containers/ directory.

## Testing
Normal test invocation is just `make test`. Run a single test with an invocation like `go test -v -run TestDevAddSites ./pkg/...`

If you set the environment variable DRUD_DEBUG=true you can see what ddev commands are being executed in the tests.

## Contributing
Interested in contributing to ddev? We would love your suggestions, contributions, and help! Please review our [Guidelines for Contributing](https://github.com/drud/ddev/blob/master/CONTRIBUTING.md), then [create an issue](https://github.com/drud/ddev/issues/new) or open a pull request!
