# Building, Testing, and Contributing

## Testing a PR

Each build of a PR has artifacts created in github, so you can click the details of the "Build DDEV Executables" test, and you can use the pulldown menu to access the ddev executable you need.

![Build ddev executables test](images/build_ddev_executables.png)

![Github artifacts pulldown](images/github_artifacts.png)

After you download and unzip the appropriate binary, you can place it in your $PATH. The easiest way to do this if you're using homebrew is `brew unlink ddev` and then `unzip ddev.zip && chmod +x ddev && mv ddev /usr/local/bin/ddev`. After you're done, you can remove the downloaded binary and `brew link ddev`.

(On macOS Big Sur these downloaded binaries are not signed, so you will want to `xattr -r -d com.apple.quarantine /path/to/ddev` in order to use them. The final binaries in any release are signed, of course.)

You do not typically have to install anything else other than the downloaded binary; when you run it it will access any docker images that it needs.

## Open in Gitpod

Gitpod.io provides a quick preconfigured ddev experience in the browser, so you can test your PR (or someone else's) easily and without setting up an environment. In any PR you can use the URL `https://gitpod.io/#https://github.com/drud/ddev/pulls/<YOURPR>` to open that PR and build it in Gitpod.

To just open and work on ddev you can use the button below.
[![Open in Gitpod](https://gitpod.io/button/open-in-gitpod.svg)](https://gitpod.io/#https://github.com/drud/ddev)

If you want to run a project in there, the easy way to set it up in `/workspace/<yourproject>` based on [ddev-gitpod](https://github.com/shaal/ddev-gitpod) is to `ddev config` and then `cd .ddev && cp ~/bin/gitpod-setup-ddev.sh . && ./gitpod-setup-ddev.sh`. This sets up ports for the project that are compatible with gitpod, etc.

A dummy project for gitpod is also provided already set up in /workspace/simpleproj.

## Making changes to ddev images

If you need to make a change to one of the ddev images, you can make the change but then it has to be built with a specific tag, and the tag has to be updated in pkg/version/version.go. So for example, make a change to containers/ddev-webserver/Dockerfile, then built it:

```bash
cd containers/ddev-webserver
make VERSION=20210424_fix_dockerfile
```

Then edit pkg/version/version.go to set `var WebTag = "20210424_fix_dockerfile"` and

```bash
cd /workspace/ddev
make
```

`ddev version` should show you that you are using the correct webtag, and `ddev start` will show it.

It's easiest to do this using gitpod (see above) because gitpod already has `docker buildx` all set up for you and the built ddev is in the $PATH.

## Pull Requests and PR Preparation

When preparing your pull request, please use a branch name like "2020_<your_username>_short_description" so that it's easy to track to you.

If you're doing a docs-only PR that does not require full testing, please add "[skip ci][ci skip]" to your commit messages; it saves a lot of testing resources.

## Docker Image changes

If you make changes to a docker image (like ddev-webserver), it won't have any effect unless you:

* You can build an image with a specific tag by going to the image directory (like containers/ddev-webserver) by just doing `make container VERSION=<branchname>` in the containers/ddev-webserver directory.
* Push a container to hub.docker.com. Push with the tag that matches your branch. Pushing to drud/ddev-webserver repo is easy to accomplish with `make push VERSION=<branchname>` **in the container directory**. You might have to use other techniques to push to another repo (`docker push`)
* Update pkg/version/version.go with the WebImg and WebTag that relate to the docker image you pushed.

## Building

Build the project with `make` and your resulting executable will end up in .gotmp/bin/ddev (for Linux) or .gotmp/bin/windows_amd64/ddev.exe (for Windows) or .gotmp/bin/darwin/ddev (for macOS).

Build/test/check static analysis with

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

Normal test invocation is just `make test`. Run a single test with an invocation like `go test -v -run TestDevAddSites ./pkg/...` or `make testpkg TESTARGS="-run TestDevAddSites"`. The easiest way to run tests is from inside the excellent golang IDE Goland. Just click the arrowhead to the left of the test name.

To see which DDEV commands the tests are executing, set the environment variable DDEV_DEBUG=true.

Use GOTEST_SHORT=true to run just one CMS in each test, or GOTEST_SHORT=<integer> to run exactly one project type from the list of project types in the [TestSites array](https://github.com/drud/ddev/blob/a4ab2827d8b6e706b2420700045d889a3a69f3f2/pkg/ddevapp/ddevapp_test.go#L43). For example, GOTEST_SHORT=5 will run many tests only against TYPO3.

## Automated testing

Anybody can view the CircleCI automated tests, and they usually show up any problems that are not OS-specific. Just click through on the testing section of the PR to see them.

The Buildkite automated tests require special access, which we typically grant to any PR contributor that asks for it.

## Docker image development

The Docker images that DDEV uses are included in the containers/ directory:

* containers/ddev-webserver: Provides the web servers (the "web" container).
* containers/ddev-dbserver: Provides the "db" container.
* containers/ddev-router: The router image
* containers/ddev-ssh-agent

When changes are made to an image, they have to be temporarily pushed to a tag that is preferably the same as the branch name of the PR, and the tag updated in pkg/version/version.go. Just ask if you need a container pushed to support a PR.

## Pull Request Pro Tips

* **[Fork](http://guides.github.com/activities/forking/) the repository** and clone it locally. Connect your local to the original ‘upstream’ repository by adding it as a remote. - Pull in changes from ‘upstream’ often so that you stay up to date so that when you submit your pull request, merge conflicts - will be less likely. See more detailed instructions [here](https://help.github.com/articles/syncing-a-fork).
* **Create a [branch](http://guides.github.com/introduction/flow/)** for your edits.
* **Be clear** about what problem is occurring and how someone can recreate that problem or why your feature will help. Then be equally as clear about the steps you took to make your changes.
* **It’s best to test**. Run your changes against any existing tests if they exist and create new ones when needed. Whether tests exist or not, make sure your changes don’t break the existing project.

## Open Pull Requests

Once you’ve opened a pull request, a discussion will start around your proposed changes. Other contributors and users may chime in, but ultimately the decision is made by the maintainer(s). You may be asked to make some changes to your pull request. If so, add more commits to your branch and push them – they’ll automatically go into the existing pull request.

If your pull request is merged – great! If it is not, no sweat, it may not be what the project maintainer had in mind, or they were already working on it. This happens, so our recommendation is to take any feedback you’ve received and go forth and pull request again – or create your own open source project.

Adapted from [GitHub Guides](https://guides.github.com/activities/contributing-to-open-source/)

## Coding Style

Unless explicitly stated, we follow all coding guidelines from the Go community. While some of these standards may seem arbitrary, they somehow seem to result in a solid, consistent codebase.

It is possible that the code base does not currently comply with these guidelines. We are not looking for a massive PR that fixes this since that goes against the spirit of the guidelines. All new contributions should make a best effort to clean up and make the code base better than they left it. Obviously, apply your best judgment. Remember, the goal here is to make the code base easier for humans to navigate and understand. Always keep that in mind when nudging others to comply.

Just use `make staticrequired` to ensure that your code can pass the required static analysis tests.

The rules:

1. All code should be formatted with `gofmt -s`.
2. All code should pass the default levels of [`golint`](https://github.com/golang/lint).
3. All code should follow the guidelines covered in [Effective Go](http://golang.org/doc/effective_go.html) and [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments).
4. Comment the code. Tell us the why, the history and the context.
5. Document _all_ declarations and methods, even private ones. Declare expectations, caveats and anything else that may be important. If a type gets exported, having the comments already there will ensure it's ready.
6. Variable name length should be proportional to its context and no longer. `noCommaALongVariableNameLikeThisIsNotMoreClearWhenASimpleCommentWouldDo`. In practice, short methods will have short variable names and globals will have longer names.
7. No underscores in package names. If you need a compound name, step back, and re-examine why you need a compound name. If you still think you need a compound name, lose the underscore.
8. All tests should run with `go test` and outside tooling should not be required. No, we don't need another unit testing framework. Assertion packages are acceptable if they provide _real_ incremental value.
9. Even though we call these "rules" above, they are actually just guidelines. Since you've read all the rules, you now know that.

If you are having trouble getting into the mood of idiomatic Go, we recommend reading through [Effective Go](https://golang.org/doc/effective_go.html). The [Go Blog](https://blog.golang.org) is also a great resource. Drinking the kool-aid is a lot easier than going thirsty.
