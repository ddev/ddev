# Public Variables

GitHub repository variables are not available to pull requests from forks. This mechanism
allows maintainers to set CI variables (e.g. to temporarily skip flaky tests) that apply
to all CI runs, including fork PRs.

Edit the files on the [`public-variables` branch](https://github.com/ddev/ddev/tree/public-variables/.github/public-variables)
to set CI variables.

See also: <https://github.com/orgs/community/discussions/44322#discussioncomment-11801819>.

## Files

Each file in this directory (except `README.md`) is exported as a CI environment variable named after the file.

In every file, put one value per line. A `#` starts a comment that runs to the end of the line, so use it to say why a value is there. Comments, empty lines, spaces at either end of a line, and line endings (LF or CRLF) are ignored:

```text
# Waiting on an upstream release, see https://github.com/example/project/issues/1
TestLagoonPull
TestAcquiaPull  # Flaky since the provider changed its API
```

Current variables:

- `DDEV_EMBARGO_TESTS` - tests to skip.
    - **Go tests:** the full test function name, passed to `go test -skip`, e.g. `TestLagoonPull`.
    - **Bats tests:** a case-sensitive substring of the bats filename (without `.bats`) or the `@test` description. E.g. `sveltekit` skips all tests in `sveltekit.bats`; `Symfony Composer` skips only the Composer test in `symfony.bats`.
    - Go and bats values can be mixed in one file.
- `DDEV_EMBARGO_PHP_VERSIONS` - PHP versions to skip in `TestPHPConfig`, e.g. `7.0`.

## Adding a new variable

1. Open a PR to `upstream/main` that:
   - Adds the variable to the **Current variables** list above with a description and example
     (so future maintainers can discover available variables by browsing `main`)
   - Adds an empty file `.github/public-variables/<VAR_NAME>`
     (the empty file on `main` is documentation only; the actual value lives on the `public-variables` branch)
2. Push the variable value to the `public-variables` branch
   (see **How to update** below)

No workflow changes are needed - any file in this directory is picked up automatically.
CI removes comments and joins the lines of each file with `|`, so the code that reads a new variable must split
the value on `|` and ignore empty values and spaces. See `IsPHPVersionEmbargoed` in
`pkg/util/utils.go` for an example.

## How to update

1. Edit the file(s) directly on the `public-variables` branch - no PR required
2. Include `[skip ci]` in the commit message to avoid triggering test workflows
3. To clear a value, remove its line. To clear a variable, empty the file

## How it works

`.ci-scripts/load-public-variables.sh` fetches the `public-variables` branch into a temporary ref,
reads each file, removes comments and empty lines, joins the remaining lines with `|`, and prints `NAME=value`.
It then deletes the temporary ref.

`.buildkite/test.sh` and `.buildkite/installer-test.sh` export its output.
`.github/workflows/test-reusable.yml`, `.github/workflows/test-wsl2-reusable.yml`,
and `.github/workflows/quickstart.yml` append it to `$GITHUB_ENV`.

The load step is skipped for `workflow_dispatch` (manually triggered) runs so maintainers can verify
a previously-embargoed test is fixed without first removing it from the embargo list.

## Branch protection

The `public-variables` branch has a GitHub Ruleset with **Restrict updates** and **Restrict deletions** enabled,
with the "Organization admin" role in the bypass list.
