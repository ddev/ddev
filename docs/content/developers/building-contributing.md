---
search:
  boost: .5
---
# Building, Testing, and Contributing

## Testing Latest Commits on HEAD

There are several ways to use DDEV’s latest-committed HEAD version:

* **Download** the latest master branch artifacts from [nightly.link](https://nightly.link/ddev/ddev/workflows/master-build/master). Each of these is built by the CI system, signed, and notarized. Get the one you need and place it in your `$PATH`.
* **Homebrew install HEAD**: On macOS and Linux, run `brew unlink ddev && brew install ddev/ddev/ddev --HEAD --fetch-HEAD` to get the latest DDEV commit, even if it’s unreleased.
* **Install via script**: You can download and run the [install_ddev_head.sh](https://raw.githubusercontent.com/ddev/ddev/refs/heads/master/scripts/install_ddev_head.sh)  script, or run it automatically:

    ```bash
    # Download and run the install script
    curl -fsSL https://raw.githubusercontent.com/ddev/ddev/refs/heads/master/scripts/install_ddev_head.sh | bash
    ```

* **Build manually**: If you have normal build tools like `make` and `go` installed, you can check out the code and run `make`.
* **Gitpod** You can use the latest build by visiting DDEV on [Gitpod](https://gitpod.io/#https://github.com/ddev/ddev).

## Testing a PR

Each [PR build](https://github.com/ddev/ddev/actions/workflows/pr-build.yml) creates GitHub artifacts you can use for testing, so you can download the one you need from the PR page, install it locally, and test using that build.

Download and unzip the appropriate binary and place it in your `$PATH`.

## Rollback to a previous version

You can also [downgrade to an older version of DDEV](../users/usage/faq.md#how-can-i-install-a-specific-version-of-ddev).

### Homebrew with macOS or Linux

If you’re using Homebrew, start by unlinking your current binary:

```
brew unlink ddev
```

Next, unzip the binary you downloaded, make it executable, and move it to your bin folder:

```
unzip ddev.zip
chmod +x ddev && sudo mv ddev /usr/local/bin/ddev
```

Verify the replacement worked by running `ddev -v`. The output should be something like `ddev version v1.22.5-alpha1-70-g0852fc2df`, instead of the regular `ddev version v1.22.5`.

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

### Installing a Downloaded Binary in the `$PATH`

Normally, you can put any executable in your path, and it takes precedence, so you don't need to remove or disable an already installed DDEV instance, which we will use here. This example uses `~/bin`. `echo $PATH` and `which ddev` are valuable commands for debugging. Since not every distro has `~/bin` in `$PATH`, you can create the folder and add it to your path in `~/.bashrc` with these commands:

```
mkdir -p ~/bin
export PATH="~/bin:$PATH"
```

Next, unzip the ZIP file you downloaded, make it executable, and move it to a folder in your path. Check with `echo $PATH`:

```
unzip ddev.zip
chmod +x ddev && mv ddev ~/bin
```

Now, close and reopen your terminal, and verify the replacement worked by running `ddev version`. The output should be something like `DDEV version v1.22.3-39-gfbb878843`, instead of the regular `DDEV version v1.22.3`.

You need to run `ddev poweroff` and `ddev start` to download the Docker images that it needs.

After you’re done testing, you can delete your downloaded executable, restart your terminal, and again use the standard DDEV:

```
rm ~/bin/ddev
```

## Open in Gitpod

[Gitpod](https://www.gitpod.io) provides a quick, preconfigured DDEV experience in the browser for testing a PR easily without the need to set up an environment. For any PR you can use the URL `https://gitpod.io/#https://github.com/ddev/ddev/pull/<YOUR-PR>` to open that PR and build it in Gitpod.

It also allows you to work on the DDEV master branch and test modifiying DDEV's source code.

To get started use the button below:

[![Open in Gitpod](https://gitpod.io/button/open-in-gitpod.svg)](https://gitpod.io/#https://github.com/ddev/ddev)

For a simple test, edit `r/cmd/ddev/cmd/start.go` and change the line

```go
    output.UserOut.Printf("Starting %s...", project.GetName())
```

to

```go
    output.UserOut.Printf("Let's gooooo ... %s...", project.GetName())
```

Compile and install your new modified DDEV version:

```bash
cd /workspace/ddev/
make
```

The command `ddev -v` now will output something like `ddev version v1.23.1-20-g70fc4cd7b-dirty`. The version will stay the same for all compilations until you make a commit.

A Gitpod dummy project for is provided by default in `/workspace/d10simple` to test your changes:

```bash
cd /workspace/d10simple/
ddev start
```

If you want to create a new project or use your own project, you will need to delete the dummy project to free up reserved host ports by running `ddev delete -Oy d10simple`.

Afterwards you can run [`ddev config`](../users/usage/commands.md#config) as usual:

```bash
cd /workspace/
mkdir my-new-project/
cd my-new-project/
ddev config
```

If you want to use an existing web project, also check it out into `/workspace/<yourproject>` and use it as usual.

The things you’re familiar with work normally, except that `ddev-router` does not run.

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

## Docker Image Changes

If you make changes to a Docker image (like `ddev-webserver`), it won’t have any effect unless you:

* Push an image with a specific tag by navigating to the image directory (like `containers/ddev-webserver`), and running `make push DOCKER_REPO=youruser/yourimage VERSION=<branchname>`.
* Multi-arch images require you to have a Buildx builder, so `docker buildx use multi-arch-builder || docker buildx create --name multi-arch-builder --use`.
* You can’t push until you `docker login`.
* Push a container to hub.docker.com. Push with the tag that matches your branch. Push to `<yourorg>/ddev-webserver` repository with `make push DOCKER_ORG=<yourorg> VERSION=<branchname>` **in the container directory**. You might have to use other techniques to push to another repository.
* Update `pkg/versionconstants/versionconstants.go` with the `WebImg` and `WebTag` that relate to the Docker image you pushed.

### Local Builds and Pushes

To use `buildx` successfully you have to have the [`buildx` Docker plugin](https://docs.docker.com/buildx/working-with-buildx/), which is in many environments by default.

To build multi-platform images you must `docker buildx use multi-arch-builder || docker buildx create --name multi-arch-builder --use` as a one-time initialization.

* If you want to work locally with a quick build for your architecture, you can:
    * `make VERSION=<version>`
    * for `ddev-dbserver`: `make mariadb_10.3 VERSION=<version>` etc.

* To push manually:

```markdown
cd containers/ddev-webserver
make push VERSION=<tag>
```

If you’re pushing to a repository other than the one wired into the Makefile (like `ddev/ddev-webserver`):

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

## Instrumentation

The instrumentation implementation is generated using the [Ampli Codegen](https://www.docs.developers.amplitude.com/data/sdks/go/ampli/).

To synchronize the implementation with the latest changes at Amplitude, the CLI
tool has to be installed locally:

```bash
npm install -g @amplitude/ampli
```

Make changes to the event definition using the GUI at <https://data.amplitude.com/ddev/DDEV>:

* create a new branch
* create or change events and properties
* save changes to the new branch
* update the implementation with `ampli checkout <branch name>`
* make changes to the code

Once finished, save the changes to publish a new version of the definitions.

Afterwards the changes can be imported running the following command in the
project root:

```bash
ampli pull
```

Once the changes are ready to be merged, merge the changes made in the new
branch to the main branch in the Amplitude backend and switch back to the
main branch:

```bash
ampli checkout main
```

Make sure the API keys are not included to the sources; they are linked during
compilation using a GitHub secret.

### Environments

There are two environments defined, `DDEV - Production` and `DDEV - Development`.
Master builds will deliver the data to production, PR builds to development.

When working on Amplitude, please always make sure the correct environment is
selected or you won't see any data. Selection is possible on most pages.

### User and event data

The first step is always to identify the device, this includes data like OS,
architecture, DDEV version, Docker, etc., details are visible in the
*User Properties*. The devices are called *Users* in the Amplitude backend. So
every user represents an unique device on which DDEV is installed.

The second step is to collect data about the command which was called by the
user and is delivered by a dedicated `Command` event.

The `Project` event finally collects data about the loaded project(s) which
includes important configuration details like PHP version, database, etc.

### Debugging

Information about data debugging can be found at <https://www.docs.developers.amplitude.com/data/debugger/>.
*Ingestion debugger* or via *User lookup* are the most useful options for DDEV.

Don't forget to select the matching environment while debugging.

### Examining data on Amplitude.com

First, local `ddev` binaries have to be built with `AmplitudeAPIKey` set. Visit `https://app.amplitude.com/data/ddev/DDEV/sources/production` and select either "Production" or "Development", then click the "Go SDK" line to get the API key. Then set `export AmplitudeAPIKey=<key>` and build the binaries with `make`.

Then run `ddev` commands as usual, and the data will be sent to Amplitude.

* You can examine data on the local side with `export DDEV_VERBOSE=true` but it's awkward. However, the actual data is always marked with `AMPLITUDE:` and the `EventType` will be `Command`, `Project`, or `$identify` (User data). For example, DDEV_VERBOSE=true ddev start 2>&1 | grep AMPLITUDE`
* To see the data show up on Amplitude, you'll need to `ddev debug instrumentation flush`.
* To make it easier to find your data, use the "Development" key and set your `instrumentation_user` to a familiar value in `~/.ddev/global_config.yaml`. For example, `instrumentation_user: rfay` would make it so you can find the user `rfay`.
* To inspect data, visit "User Lookup", (`https://app.amplitude.com/analytics/ddev/activity`) and choose the correct source in the upper left ("DDEV Production" or "DDEV Development"). Then use "Search users" in the upper right to find the user you are studying. If you've used an `instrumentation_user` it will be searchable as "User".  (Advanced->where: "User" = "rfay". for example). You'll then have a page devoted to the events of that user.

## Building

* You'll want both your fork/branch and the upstream as remotes in Git, so that tags can be determined. For example, the upstream Git remote can be `https://github.com/ddev/ddev` and your fork's remote can be `git@github.com:<yourgithubuser>/ddev`. Without the upstream, Git may not know about tags that it needs for tests to work.
* To run tests, you'll want `~/tmp` to be allowed in Docker. This is not normally an issue as the home directory is available by default in most Docker providers.

Build the project with `make` and your resulting executable will end up in `.gotmp/bin/linux_amd64/ddev` or `.gotmp/bin/linux_arm64/ddev` (for Linux) or `.gotmp/bin/windows_amd64/ddev.exe` or `.gotmp/bin/windows_arm64/ddev.exe` (for Windows) or `.gotmp/bin/darwin_amd64/ddev` or `.gotmp/bin/darwin_arm64/ddev` (for macOS).

You can add additional `go build` args with `make BUILDARGS=<something>`, for example, `make BUILDARGS=-race`.

Build/test/check static analysis with:

```
make # Builds on current os/architecture
make BUILDARGS=-race
make linux_amd64
make linux_arm64
make darwin_amd64
make darwin_arm64
make windows_amd64
make windows_arm64
make test
make clean
make staticrequired
```

## Testing

Normal test invocation is `make test`. Run a single test with an invocation like `go test -v -run TestDevAddSites ./pkg/...` or `make test TESTARGS="-run TestDevAddSites"`. The easiest way to run tests is from inside the excellent golang IDE [GoLand](https://www.jetbrains.com/go/). Click the arrowhead to the left of the test name. This is also easy to do in Visual Studio Code.

To test with race detection, `make test TESTARGS="-race"` for example.

To see which DDEV commands the tests are executing, set the environment variable `DDEV_DEBUG=true`.

Use `GOTEST_SHORT=true` to run one CMS in each test, or `GOTEST_SHORT=<integer>` to run exactly one project type from the list of project types in the [TestSites array](https://github.com/ddev/ddev/blob/master/pkg/ddevapp/ddevapp_test.go#L43). For example, `GOTEST_SHORT=5 make test TESTARGS="-run TestDdevFullSiteSetup"` will run only `TestDdevFullSiteSetup` against TYPO3.

To run a test (in the `cmd` package) against a individually-compiled DDEV binary, set the `DDEV_BINARY_FULLPATH` environment variable, for example `DDEV_BINARY_FULLPATH=$PWD/.gotmp/bin/linux_amd64/ddev make testcmd`.

The easiest way to run tests is using GoLand (or VS Code) with their built-in test runners and debuggers. You can step through a specific test; you can stop at the point before the failure and experiment with the site that the test has set up.

## Automated Testing

Anybody can view the CircleCI automated tests, and they usually show up any problems that are not OS-specific. Click through on the testing section of the PR to see them.

The Buildkite automated tests require special access, which we typically grant to any PR contributor that asks for it.

## Docker Image Development

The Docker images that DDEV uses are included in the `containers/` directory:

* `containers/ddev-gitpod-base` is the image used in GitPod by [ddev-gitpod-launcher](https://github.com/ddev/ddev-gitpod-launcher)
* `containers/ddev-php-base` the base build for `ddev-webserver`.
* `containers/ddev-webserver` provides the web servers for per-project `web` containers.
* `containers/ddev-dbserver` provides the `db` container for per-project databases.
* `containers/ddev-nginx-proxy-router` is the (deprecated) the nginx-proxy router image.
* `containers/ddev-ssh-agent` provides a single in-Docker-network SSH agent so projects can use your SSH keys.
* `containers/ddev-traefik-router` is the current Traefik-based router image.

When changes are made to an image, they have to be temporarily pushed to a tag—ideally with the same as the branch name of the PR—and the tag updated in `pkg/versionconstants/versionconstants.go`. Please ask if you need a container pushed to support a pull request.

## Pull Requests

To contribute your fixes or improvements to DDEV, make a pull request on GitHub. If you're undertaking a large change, create an issue first so it can be discussed before you invest a lot of time. When you're ready, create a pull request, and a discussion will start around your proposed changes. Other contributors and users may chime in, but ultimately the decision is made by the maintainer(s). You may be asked to make some changes to your pull request. If so, add more commits to your branch and push them. They’ll automatically go into the existing pull request.

If your pull request is merged, great! If not, no sweat; it may not be what the project maintainer had in mind, or they were already working on it. This happens, so our recommendation is to take any feedback you’ve received and go forth and pull request again. Or create your own open source project.

### Preparing a pull request

* **[Fork](https://docs.github.com/en/get-started/quickstart/contributing-to-projects) the repository** and clone it locally. Connect your local to the original ‘upstream’ repository by adding it as a remote, and pull upstream changes often so you stay up to date and reduce the likelihood of conflicts when you submit your pull request. See more detailed instructions [here](https://help.github.com/articles/syncing-a-fork).
* **Create a [branch](https://docs.github.com/en/get-started/quickstart/github-flow)** for your edits. See below for DDEV's conventions for branch names.
* **Be clear** about the problem and how someone can recreate it, or why your feature will help. Be equally clear about the steps you took to make your changes.
* **It’s best to test**. Run your changes against any existing tests and create new tests when needed. Whether tests exist or not, make sure your changes don’t break the existing project.

### Feature branch name

When preparing your pull request, please use a branch name like `YYYYMMDD_<your_username>_short_description` (like `20230901_rfay_short_description`) so it’s easy to identify you as the author.

### Pull Request Title Guidelines

We have very precise rules over how our PR titles (and thus master-branch commits) are to be formatted. This leads to **more readable messages** that are easy to follow when looking through the **project history**. But also, we use the master-branch Git commit messages to **generate the changelog** for the releases.

The pull request title must follow this convention which is based on the [Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0/) specification:

`<type>[optional !]: <description>[, fixes #<issue>]`

#### Examples

* `build: bump mutagen to 0.17.2`
* `ci: enforce commit message convention, fixes #5037`
* `docs: change code refs of Mac M1 to Apple Silicon`
* `feat: allow multiple upload dirs, fixes #4190, fixes #4796`
* `fix: create upload_dir if it doesn't exist in ddev composer create, fixes #5031`
* `refactor: add new Amplitude Property DDEV-Environment`
* `test: optimize caching of downloaded assets`

#### Type

Must be one of the following:

* **build**: Changes that affect the build or external dependencies
* **ci**: Changes to our CI configuration files and scripts
* **docs**: Documentation only changes
* **feat**: A new feature
* **fix**: A bugfix
* **refactor**: A code change that neither fixes a bug nor adds a feature
* **test**: Adding missing tests or correcting existing tests

Regarding SemVer, all types above except `feat` increase the patch version, `feat` increases the minor version.

#### Scope

No scope must be used.

#### Breaking Changes

Breaking changes must have a `!` appended after type/scope.

Regarding SemVer, breaking changes increase the major version.

#### Subject / Description

The subject contains a succinct description of the change:

* use the imperative, present tense: "change" not "changed" nor "changes"
* don't capitalize the first letter
* no dot (.) at the end

If an issue exists for the change, `, fixes #<issue number>` must be appended to the subject.

#### Revert

If the commit reverts a previous commit, it should begin with `revert:`, followed by the header of the reverted commit. In the body it should say: `This reverts commit <hash>.`, where the hash is the SHA of the commit being reverted.

## Coding Style

Unless explicitly stated, we follow all coding guidelines from the Go community. While some of these standards may seem arbitrary, they somehow seem to result in a solid, consistent codebase.

It is possible that the codebase does not currently comply with these guidelines. We are not looking for a massive PR that fixes this since that goes against the spirit of the guidelines. All new contributions should make a best effort to clean up and make the codebase better than they left it. Obviously, apply your best judgment. Remember, the goal here is to make the codebase easier for humans to navigate and understand. Always keep that in mind when nudging others to comply.

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
9. Even though we call these “rules” above, they are guidelines. Since you’ve read all the rules, you now know that.

If you are having trouble getting into the mood of idiomatic Go, we recommend reading through [Effective Go](https://golang.org/doc/effective_go.html). The [Go Blog](https://blog.golang.org) is also a great resource. Drinking the kool-aid is a lot easier than going thirsty.

## Contributor Live Training

We’re actively trying to increase the DDEV community of contributors and maintainers. To do that, we regularly do contributor training, and we’d love to have you come. The trainings are recorded for everybody’s benefit. The recordings and upcoming session dates can be found here: [DDEV Contributor Live Training](https://ddev.com/blog/contributor-training/).
