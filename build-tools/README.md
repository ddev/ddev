# Build tools for standard makefile

**These build tools live at https://github.com/drud/build-tools**. If you are viewing this README in any other repository, it's important to know that modifications should **never** be made directly, and instead should be made to the base repository and pulled in via the subtree merge instructions below.

These tools add standard components (sub-makefiles and build scripts) as well as example starters for the Makefile and circle.yml.

## Add build-tools for the first time to a Makefile

Download the [build_updates.sh](https://github.com/drud/build-tools/blob/master/build_updates.sh) script and run it in the directory where the build-tools should be added.

Alternately, just download the latest release from https://github.com/drud/build-tools/releases/latest, extract it, rename it to build-tools, and commit the result.


## Update build-tools directory from this repository using subtree merge

Run the build_update.sh script in the build-tools directory:

```
cd build-tools
./build_update.sh
```

## Set up a Makefile to begin with

* Copy the Makefile.example to "Makefile" in the root of your project
* Edit the sub-Makefiles included
* Update the variables at the top of the Makefile

## Additional chores when installing:

* Add the items from gitignore_example to your .gitignore
* Update the project README.md to explain how to build - the target reminders in the paragraph below may be helpful.

## Basic targets and capabilities

Using this base will allow you to build with standard targets like build, test, container, push, clean:

```
make 
make linux
make darwin
make gofmt 
make govet
make vendorcheck
make golint
make static (gofmt, govet, golint, vendorcheck)
make test
make container
make push
make VERSION=0.3.0 container
make VERSION=0.3.0 push
make clean
```

## Golang compiler component

golang projects and static analysis functions like gofmt are built in a container from drud/golang-build-container (from https://github.com/drud/golang-build-container). They pick up the "latest" tag by default, so that should be updated when we choose to move to a newer golang version.
