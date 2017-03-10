#ddev

The purpose of *ddev* is to support developers with a local copy of a site for development purposes. It runs the site in a Docker containers similar to a normal hosting environment.

You can see all "ddev" usages using the help commands, like `ddev -h`, `ddev add -h`, etc.
 
 
 ## Building
 
 Environment variables:
 * DRUD_DEBUG: Will display more extensive information as a site is deployed.
 
 ```
 make 
 make linux
 make darwin
 make test
 make clean
 ```

## Testing

Normal test invocation is just `make test`. Run a single test with an invocation like `go test -v -run TestDevAddSites ./pkg/...`

* DRUD_DEBUG: It helps a lot to set DRUD_DEBUG=true to see what ddev commands are being executed in the tests.
* DDEV_BINARY_FULLPATH should be set to the full pathname of the ddev binary we're attempting to test. That way it won't accidentally use some other version of ddev that happens to be on the filesystem.
* SKIP_COMPOSE_TESTS=true allows skipping tests that require docker-compose. 


## Basic Usage

**Key prerequisites**
* In `ddev add drudio production` the first argument is the repo/site name, and the second is an arbitrary "environment name" (and source for the dev database), which is typically either "production" or "staging".

Examples:

```
ddev add drudio production
Successfully added drudio-production
Your application can be reached at: http://legacy-drudio-production

ddev list 

ddev stop drudio production
ddev start drudio production

ddev rm drudio production


```
