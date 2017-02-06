# Build tools for standard makefile

**These build tools live at https://github.com/drud/build-tools**. If you are viewing this README in any other repository, it's important to know that modifications should **never** be made directly, and instead should be made to the base repository and pulled in via the subtree merge instructions below.

These tools add standard components (sub-makefiles and build scripts) as well as example starters for the Makefile and circle.yml.

## Add build-tools to a Makefile

```
git remote add -f build-tools git@github.com:drud/build-tools.git
git merge -s ours --allow-unrelated-histories build-tools/master
git read-tree --prefix=build-tools -u build-tools/master
git commit -m "Added build-tools for standard makefile as subtree"
```

## Update build-tools directory from this repository using subtree merge

```
# If there is not a build-tools remote, add it
git remote add -f build-tools git@github.com:drud/build-tools.git
# Fetch/merge current build-tools (pull doesn't work if set to branch.autosetuprebase=always)
git fetch build-tools
git merge -s subtree build-tools/master
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
make test
make container
make push
make VERSION=0.3.0 container
make VERSION=0.3.0 push
make clean
```
