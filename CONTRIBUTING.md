# Contributing

The general workflow for contributing to this project is outlined in this document.

## Create an Issue

If you find a bug in this project, have trouble following the documentation, or have a question about the project, create an issue! There’s nothing to it and whatever issue you’re having, you’re likely not the only one, so others will find your issue helpful, too. For more information on how issues work, check out GitHub's [Issues guide](http://guides.github.com/features/issues). Or there are lots of [other places for immediate support](https://ddev.readthedocs.io/en/stable/#support-and-user-contributed-documentation).

### Issues Pro Tips

- **Check existing issues** for your issue. Duplicating an issue is slower for both parties, so search through open and closed issues to see if what you’re running into has been addressed already.
- **Be clear** about what your problem is: What was the expected outcome? What happened instead? Detail how someone else can recreate the problem.
- **Link to examples** recreate or display the problem with screenshots, screencasts, or code examples using [The Go Playground](https://play.golang.org). The better you can demonstrate the problem, the more attention your issue is likely to get.
- **Include system details** like what the browser, library or operating system you’re using and its version.
- **Paste error output** or logs in your issue or in a [Gist](http://gist.github.com/). If pasting them in the issue, wrap it in three backticks: ` ``` ` so that it renders nicely.

## Stack Overflow Questions and Documentation

There are a number of situations where a particular approach to a ddev solution can be stated more easily in [Stack Overflow](https://stackoverflow.com/tags/ddev) (use the "ddev" tag). We respond there quickly, but if you know the answer already, create the question there and then click the checkbox at the bottom "Answer your own question". Stack Overflow is often the best place to incubate documentation that affects just a few people or that just needs time to get responses. And it's highly searchable on the web.

## Pull Request

If you’re able to patch the bug or add the feature yourself – fantastic, make a pull request with the code! Once you’ve submitted a pull request the maintainer(s) can compare your branch to the existing one and decide whether or not to incorporate (pull in) your changes.

Refer to [Building, Testing, and Contributing](docs/developers/building-contributing.md) for help with how to build and test the project.

### Pull Request Pro Tips

- **[Fork](http://guides.github.com/activities/forking/) the repository** and clone it locally. Connect your local to the original ‘upstream’ repository by adding it as a remote. - Pull in changes from ‘upstream’ often so that you stay up to date so that when you submit your pull request, merge conflicts - will be less likely. See more detailed instructions [here](https://help.github.com/articles/syncing-a-fork).
- **Create a [branch](http://guides.github.com/introduction/flow/)** for your edits.
- **Be clear** about what problem is occurring and how someone can recreate that problem or why your feature will help. Then be equally as clear about the steps you took to make your changes.
- **It’s best to test**. Run your changes against any existing tests if they exist and create new ones when needed. Whether tests exist or not, make sure your changes don’t break the existing project.

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

## Using Gitpod.io to test PRs

[Gitpod.io](https://www.gitpod.io/) is integrated with DDEV-Local so you don't have to set up a development environment to work on bugs or features. You can just spin up a gitpod instance and go. There's a link on every PR build in the checks to let you do it.
