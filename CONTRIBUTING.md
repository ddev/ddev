# Contributing to DDEV

Thank you for helping. Start with the
[organization-wide contributing guide](https://github.com/ddev/.github/blob/main/CONTRIBUTING.md),
which covers reporting issues, pull request expectations, and **the rules for
contributions written with AI tools**. This file adds what is specific to
`ddev/ddev`.

## Ways to Help

Code is only one way to contribute:

- **Answer questions** in the [DDEV Discord](https://ddev.com/s/discord).
- **Test pull requests and unreleased builds** and report what you find; see
  [below](#testing-pull-requests-and-head).
- **Improve the documentation.** Every page on
  [docs.ddev.com](https://docs.ddev.com) has an edit link, and the
  [writing style guide](https://docs.ddev.com/en/stable/developers/writing-style-guide/)
  explains the conventions.
- **Write or maintain an add-on.** See
  [Creating Add-ons](https://docs.ddev.com/en/stable/users/extend/creating-add-ons/)
  and the [add-on registry](https://addons.ddev.com).
- **Fix a bug or add a feature**, as described below.
- **[Sponsor DDEV](https://ddev.com/sponsor).** DDEV is funded by its
  community, and sponsorship pays for maintenance.

## Testing Pull Requests and HEAD

Every pull request and every commit to `main` produces downloadable builds.
Trying them before release is one of the most useful things you can do:

```bash
ddev utility download-ddev --pr <PR_NUMBER>   # A pull request's build
ddev utility download-ddev --head             # The latest commit on main
```

The command prints how to use the downloaded binary in your current shell. See
[Testing a PR](https://docs.ddev.com/en/stable/developers/building-contributing/#testing-a-pr)
and
[Testing Latest Commits on HEAD](https://docs.ddev.com/en/stable/developers/building-contributing/#testing-latest-commits-on-head)
for other ways to install them, including Homebrew.

## Making a Code Change

[Building, Testing, and Contributing](https://docs.ddev.com/en/stable/developers/building-contributing/)
has the details. In short:

1. **Discuss larger changes first**, in an issue or on Discord.
2. **Fork and branch.** Name the branch
   `YYYYMMDD_<your_username>_<short_description>`, for example
   `20261012_rfay_fix_networking`.
3. **Build with `make`**, which puts the binary in
   `.gotmp/bin/<os>_<arch>/ddev`. Put that directory first in your `PATH`
   while testing, or an installed DDEV runs instead.
4. **Run the tests that cover your change**, for example
   `make testpkg TESTARGS="-run TestName"`. The full suite is slow, and CI
   runs it on every pull request. Integration tests need Docker.
5. **Run `make staticrequired`** before you push. It runs the linters that CI
   requires.
6. **Open the pull request** using the template, with a
   [Conventional Commits title](https://docs.ddev.com/en/stable/developers/building-contributing/#pull-request-title-guidelines)
   such as `fix: handle spaces in project paths, fixes #1234`.

Changes to the documentation follow the same process; see
[Testing Documentation](https://docs.ddev.com/en/stable/developers/testing-docs/)
to preview them locally.

If you use an AI coding agent, point it at [AGENTS.md](AGENTS.md), which
describes this repository's conventions for agents. The
[AI rules](https://github.com/ddev/.github/blob/main/CONTRIBUTING.md#using-ai-tools)
still apply to you: you are responsible for what the agent produces.

## Learning More

- [Contributor Training](https://ddev.com/blog/contributor-training/):
  recordings of live sessions for contributors, and dates of upcoming ones.
- [Developer documentation](https://docs.ddev.com/en/stable/developers/):
  building, release management, and maintainer guides.
- [DDEV Blog](https://ddev.com/blog/): release notes and in-depth articles.
- [DDEV Discord](https://ddev.com/s/discord): the place to ask contributors and
  maintainers for help with a change.
