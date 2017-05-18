<h1>Adding a container to the build</h1>

Adding a container to the build/nightly build requires a few steps:

1. Add container name/tag to version.XxxImage and version.XXXTag in pkg/version/version.go (and use them as needed of course.)
2. In Makefile, add XXXImage and XXXTag to VERSION_VARIABLES.
3. In Makefile, set the default values of XXXImage and XXXTag (currently below VERSION_VARIABLES)
4. Add the container as a git submodule: `git submodule add git@gihub.com:drud/somecontainer.git containers/somecontainer`
5. Add the container to nightly_build.mak's CONTAINER_DIRS
6. Add any variables that should be overridden in the container's build in nightly_build.mak (especially XXXTag) to the nightly_build stanza .circleci/config.yml
7. Add those same override variables to the comment in nightly_build.mak that explains how to run it.
