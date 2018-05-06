# Build tools for standard makefile

**These build tools live at https://github.com/drud/build-tools**. If you are viewing this README in any other repository, it's important to know that modifications should **never** be made directly, and instead should be made to the base repository and pulled in via the instructions below.

These tools add standard components (sub-makefiles and build scripts) as well as example starters for the Makefile and .circleci/config.yml.

## Add build-tools for the first time to a Makefile

Download the [build_updates.sh](https://raw.githubusercontent.com/drud/build-tools/master/build_update.sh) script and run it in the directory where the build-tools should be added.

## Update build-tools directory from this repository using subtree merge

Download the [latest build_updates.sh](https://raw.githubusercontent.com/drud/build-tools/master/build_update.sh) and run it in the build-tools directory:

```
cd build-tools
wget -O build_updates.sh https://raw.githubusercontent.com/drud/build-tools/master/build_update.sh
chmod +x build_update.sh
./build_update.sh
```

## Set up a Makefile to begin with

* Copy the Makefile.example to "Makefile" in the root of your project
* Edit the sub-Makefiles included
* Update the variables at the top of the Makefile

## Additional chores when installing:

* Add the items from gitignore_example to the .gitignore in each directory that has a Makefile
* Update the project README.md to explain how to build - the target reminders in the paragraph below may be helpful.

## Basic targets and capabilities

Using this base will allow you to build with standard targets like build, test, container, push, clean:

```
make
make linux
make darwin
make gofmt
make govet
make govendor
make golint
make codecoroner
make static (gofmt, govet, golint, govendor)
make test
make container
make push
make VERSION=0.3.0 container
make VERSION=0.3.0 push
make clean
```

On Windows, using the tools described below, use the command:

```
"C:\Program Files\git\bin\bash" -c "make"
"C:\Program Files\git\bin\bash" -c "make test"
"C:\Program Files\git\bin\bash" -c "make test TESTARGS='-run TestSomething'"
"C:\Program Files\git\bin\bash" -c "make gofmt"
```

If you're using Powershell instead of cmd, just prepend an `&` on the command, as in:

```
&"C:\Program Files\git\bin\bash" -c "make"
```

(Note that if you're working with the code, you can just run git bash and do make (and anything else you want) from inside it.)

## Installed requirements

You'll need:
* docker-ce (will work with move versions and platforms)
* gnu make
* golang

On windows the building is somewhat more difficult due to the build being bash/linux/make-oriented, but support is provided. You need:
* [chocolatey](https://chocolatey.org/install) installed
* make for Windows 3.81 (Recommended package [choco install make](https://chocolatey.org/packages/make) on chocolatey.org)
* git for windows (Recommended package [choco install git](https://chocolatey.org/packages/git.install))
* docker for windows.

(You can certainly install the base gnu make package, and the traditional git for windows package should work fine. Chocolatey installs are recommended here because there are many, many ways to get mixes of unix-style components that absolutely don't work. Microsoft's lovely bash-for-windows is a great tool, but it's an actual Ubuntu environment so isn't a good place for testing Windows builds.)

## Golang compiler component

golang projects and static analysis functions like gofmt are built in a container from drud/golang-build-container (from https://github.com/drud/golang-build-container). The version of the container is specified in build-tools.
