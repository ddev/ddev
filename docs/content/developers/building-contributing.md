# Building, Testing, and Contributing

## Testing Latest Commits on HEAD

There are several ways to use DDEV’s latest-committed HEAD version:

* **Download** the latest master branch artifacts from [nightly.link](https://nightly.link/ddev/ddev/workflows/master-build/master). Each of these is built by the CI system, signed, and notarized. Get the one you need and place it in your `$PATH`.
* **Homebrew install HEAD**: On macOS and Linux, run `brew unlink ddev && brew install drud/ddev/ddev --HEAD --fetch-HEAD` to get the latest DDEV commit, even if it’s unreleased. Since you’re building this on your own computer, it’s not signed or notarized, and you’ll get a notification that instrumentation doesn’t work, which is fine. If you’re using Linux/WSL2, you’ll likely need to install build-essential by running the following command: `sudo apt install -y build-essential`.
* **Build manually**: If you have normal build tools like `make` and `go` installed, you can check out the code and run `make`.
* **Gitpod** You can use the latest build by visiting DDEV on [Gitpod](https://gitpod.io/#https://github.com/ddev/ddev).

## Testing a PR

Each [PR build](https://github.com/ddev/ddev/actions/workflows/pr-build.yml) creates GitHub artifacts you can use for testing, so you can download the one you need from the PR page, install it locally, and test using that build.

Download and unzip the appropriate binary and place it in your `$PATH`.

If you’re using Homebrew, start by unlinking your current binary:

```
brew unlink ddev
```

Next, unzip the binary you downloaded, make it executable, and move it to your bin folder:

```
unzip ddev.zip
chmod +x ddev && sudo mv ddev /usr/local/bin/ddev
```

Verify the replacement worked by running `ddev -v`. The output should be something like `ddev version v1.19.1-42-g5334d3c1`, instead of the regular `ddev version v1.19.1`.

!!!tip "macOS and Unsigned Binaries"
    macOS doesn’t like these downloaded binaries, so you’ll need to bypass the automatic quarantine to use them:

    ```
    xattr -r -d com.apple.quarantine /usr/local/bin/ddev
    ```

    (The binaries on the master branch and the final release binaries _are_ signed.)

You do not typically have to install anything else other than the downloaded binary; when you run it it will access any Docker images that it needs.

After you’re done, you can delete your downloaded binary and re-link the original Homebrew one:

```
sudo rm /usr/local/bin/ddev
brew link --force ddev
```

## Open in Gitpod

[Gitpod](https://www.gitpod.io) provides a quick, preconfigured DDEV experience in the browser for testing a PR easily without the need to set up an environment. In any PR you can use the URL `https://gitpod.io/#https://github.com/ddev/ddev/pulls/<YOUR-PR>` to open that PR and build it in Gitpod.

To open and work on DDEV you can use the button below.
[![Open in Gitpod](https://gitpod.io/button/open-in-gitpod.svg)](https://gitpod.io/#https://github.com/ddev/ddev)

If you want to run a web project, you can check it out into `/workspace/<yourproject>` and use it as usual. The things you’re familiar with work normally, except that `ddev-router` does not run.

A Gitpod dummy project for is provided by default in `/workspace/d9simple`. If you’re testing your own project, you will need to delete it to free up reserved host ports by running `ddev delete -Oy d9simple`. Then you can run [`ddev start`](../users/usage/commands.md#start) to work with your own.

## Making Changes to DDEV Images

If you need to make a change to one of the DDEV images, it will need to be built with a specific tag that’s updated in `pkg/versionconstants/versionconstants.go`.

For example, make a change to `containers/ddev-webserver/Dockerfile`, then build it:

```bash
cd containers/ddev-webserver
make VERSION=20210424_fix_dockerfile
```

Then edit `pkg/versionconstants/versionconstants.go` to set `var WebTag = "20210424_fix_dockerfile"` and

```bash
cd /workspace/ddev
make
```

`ddev version` should show you that you are using the correct webtag, and [`ddev start`](../users/usage/commands.md#start) will show it.

It’s easiest to do this using Gitpod (see above) because Gitpod already has `docker buildx` all set up for you and the built DDEV binary is in the `$PATH`.

## Pull Requests and PR Preparation

When preparing your pull request, please use a branch name like `2022MMDD_<your_username>_short_description` (like `20230901_rfay_short_description`) so it’s easy to identify you as the author.

## Docker Image Changes

If you make changes to a Docker image (like `ddev-webserver`), it won’t have any effect unless you:

* Push an image with a specific tag by navigating to the image directory (like `containers/ddev-webserver`), and running `make push DOCKER_REPO=youruser/yourimage VERSION=<branchname>`.
* Multi-arch images require you to have a Buildx builder, so `docker buildx create --name ddev-builder-multi --use`.
* You can’t push until you `docker login`.
* Push a container to hub.docker.com. Push with the tag that matches your branch. Pushing to `<yourorg>/ddev-webserver` repo is easy to accomplish with `make push DOCKER_ORG=<yourorg> VERSION=<branchname>` **in the container directory**. You might have to use other techniques to push to another repo.
* Update `pkg/versionconstants/versionconstants.go` with the `WebImg` and `WebTag` that relate to the Docker image you pushed.

### Local Builds and Pushes

To use `buildx` successfully you have to have the [`buildx` Docker plugin](https://docs.docker.com/buildx/working-with-buildx/), which is in many environments by default.

To build multi-platform images you must `docker buildx create --use` as a one-time initialization.

* If you want to work locally with a quick build for your architecture, you can:
    * `make VERSION=<version>`
    * for `ddev-dbserver`: `make mariadb_10.3 VERSION=<version>` etc.

* To push manually:

```markdown
cd containers/ddev-webserver
make push VERSION=<tag> 
```

If you’re pushing to a repo other than the one wired into the Makefile (like `drud/ddev-webserver`):

```
cd containers/ddev-webserver
make push VERSION=<tag> DOCKER_REPO=your/dockerrepo
```

### Pushes Using GitHub Actions

To manually push using GitHub Actions,

#### For Most Images

* Visit [Actions → Push tagged image](https://github.com/ddev/ddev/actions/workflows/push-tagged-image.yml)
* Click “Run workflow” in the blue band near the top.
* Choose the branch, usually `master` and then the image to be pushed, `ddev-webserver`, `ddev-dbserver`, etc. Also you can use `all` to build and push all of them. Include a tag for the pushed image and GitHub will do all the work.

#### For `ddev-dbserver`

* Visit [Actions → Push tagged db image](https://github.com/ddev/ddev/actions/workflows/push-tagged-dbimage.yml)
* Click “Run workflow” in the blue band near the top.
* Choose the branch, usually `master`. Include a tag for the pushed image and GitHub will do all the work.

## Building

* You'll want both your fork/branch and the upstream as remotes in git, so that tags can be determined. For example, the upstream git remote can be `https://github.com/ddev/ddev` and your fork's remote can be `git@github.com:<yourgithubuser>/ddev`. Without the upstream, git may not know about tags that it needs for tests to work.
* To run tests, you'll want `~/tmp` to be allowed in docker. This is not normally an issue as the home directory is available by default in most docker providers.

Build the project with `make` and your resulting executable will end up in `.gotmp/bin/linux_amd64/ddev` or `.gotmp/bin/linux_arm64/ddev` (for Linux) or `.gotmp/bin/windows_amd64/ddev.exe` (for Windows) or `.gotmp/bin/darwin_amd64/ddev` or `.gotmp/bin/darwin_arm64/ddev` (for macOS).

Build/test/check static analysis with:

```
make # Builds on current os/architecture
make linux_amd64
make linux_arm64
make darwin_amd64
make darwin_arm64
make windows_amd64
make test
make clean
make staticrequired
```

## Testing

Normal test invocation is `make test`. Run a single test with an invocation like `go test -v -run TestDevAddSites ./pkg/...` or `make testpkg TESTARGS="-run TestDevAddSites"`. The easiest way to run tests is from inside the excellent golang IDE [GoLand](https://www.jetbrains.com/go/). Click the arrowhead to the left of the test name.

To see which DDEV commands the tests are executing, set the environment variable `DDEV_DEBUG=true`.

Use `GOTEST_SHORT=true` to run just one CMS in each test, or `GOTEST_SHORT=<integer>` to run exactly one project type from the list of project types in the [TestSites array](https://github.com/ddev/ddev/blob/a4ab2827d8b6e706b2420700045d889a3a69f3f2/pkg/ddevapp/ddevapp_test.go#L43). For example, `GOTEST_SHORT=5 make testpkg TESTARGS="-run TestDdevFullSiteSetup"` will run only `TestDdevFullSiteSetup` against TYPO3.

To run a test (in the `cmd` package) against a individually-compiled DDEV binary, set the `DDEV_BINARY_FULLPATH` environment variable, for example `DDEV_BINARY_FULLPATH=$PWD/.gotmp/bin/linux_amd64/ddev make testcmd`.

The easiest way to run tests is using GoLand (or VS Code) with their built-in test runners and debuggers. You can step through a specific test; you can stop at the point before the failure and experiment with the site that the test has set up.

## Automated Testing

Anybody can view the CircleCI automated tests, and they usually show up any problems that are not OS-specific. Just click through on the testing section of the PR to see them.

The Buildkite automated tests require special access, which we typically grant to any PR contributor that asks for it.

## Docker Image Development

The Docker images that DDEV uses are included in the `containers/` directory:

* `containers/ddev-php-base` the base build for `ddev-webserver`.
* `containers/ddev-webserver` provides the web servers for per-project `web` containers.
* `containers/ddev-dbserver` provides the `db` container for per-project databases.
* `containers/ddev-router` provides the central router image.
* `containers/ddev-ssh-agent` provides a single in-Docker-network SSH agent so projects can use your SSH keys.

When changes are made to an image, they have to be temporarily pushed to a tag—ideally with the same as the branch name of the PR—and the tag updated in `pkg/versionconstants/versionconstants.go`. Please ask if you need a container pushed to support a pull request.

## Pull Request Pro Tips

* **[Fork](https://docs.github.com/en/get-started/quickstart/contributing-to-projects) the repository** and clone it locally. Connect your local to the original ‘upstream’ repository by adding it as a remote, and pull upstream changes often so you stay up to date and reduce the likelihood of conflicts when you submit your pull request. See more detailed instructions [here](https://help.github.com/articles/syncing-a-fork).
* **Create a [branch](https://docs.github.com/en/get-started/quickstart/github-flow)** for your edits.
* **Be clear** about the problem and how someone can recreate it, or why your feature will help. Be equally clear about the steps you took to make your changes.
* **It’s best to test**. Run your changes against any existing tests and create new tests when needed. Whether tests exist or not, make sure your changes don’t break the existing project.

## Open Pull Requests

Once you’ve opened a pull request, a discussion will start around your proposed changes. Other contributors and users may chime in, but ultimately the decision is made by the maintainer(s). You may be asked to make some changes to your pull request. If so, add more commits to your branch and push them. They’ll automatically go into the existing pull request.

If your pull request is merged, great! If not, no sweat; it may not be what the project maintainer had in mind, or they were already working on it. This happens, so our recommendation is to take any feedback you’ve received and go forth and pull request again. Or create your own open source project.

## Coding Style

Unless explicitly stated, we follow all coding guidelines from the Go community. While some of these standards may seem arbitrary, they somehow seem to result in a solid, consistent codebase.

It is possible that the code base does not currently comply with these guidelines. We are not looking for a massive PR that fixes this since that goes against the spirit of the guidelines. All new contributions should make a best effort to clean up and make the code base better than they left it. Obviously, apply your best judgment. Remember, the goal here is to make the code base easier for humans to navigate and understand. Always keep that in mind when nudging others to comply.

Use `make staticrequired` to ensure that your code can pass the required static analysis tests.

The rules:

1. All code should be formatted with `gofmt -s`.
2. All code should pass the default levels of [`golint`](https://github.com/golang/lint).
3. All code should follow the guidelines covered in [Effective Go](http://golang.org/doc/effective_go.html) and [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments).
4. Comment the code. Tell us the why, the history and the context.
5. Document *all* declarations and methods, even private ones. Declare expectations, caveats and anything else that may be important. If a type gets exported, having the comments already there will ensure it’s ready.
6. Variable name length should be proportional to its context and no longer. `noCommaALongVariableNameLikeThisIsNotMoreClearWhenASimpleCommentWouldDo`. In practice, short methods will have short variable names and globals will have longer names.
7. No underscores in package names. If you need a compound name, step back, and re-examine why you need a compound name. If you still think you need a compound name, lose the underscore.
8. All tests should run with `go test` and outside tooling should not be required. No, we don’t need another unit testing framework. Assertion packages are acceptable if they provide *real* incremental value.
9. Even though we call these “rules” above, they are actually just guidelines. Since you’ve read all the rules, you now know that.

If you are having trouble getting into the mood of idiomatic Go, we recommend reading through [Effective Go](https://golang.org/doc/effective_go.html). The [Go Blog](https://blog.golang.org) is also a great resource. Drinking the kool-aid is a lot easier than going thirsty.
