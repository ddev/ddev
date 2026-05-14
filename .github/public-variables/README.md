# Public Variables

GitHub repository variables are not available to pull requests from forks. This mechanism
allows maintainers to set CI variables (e.g. to temporarily skip flaky tests) that apply
to all CI runs, including fork PRs.

Edit the files on the [`public-variables` branch](https://github.com/ddev/ddev/tree/public-variables/.github/public-variables)
to set CI variables.

See also: <https://github.com/orgs/community/discussions/44322#discussioncomment-11801819>.

## Files

Each file in this directory (except `README.md`) is exported as a CI environment variable named after the file.

Current variables:

- `DDEV_EMBARGO_TESTS` - pipe-separated patterns to skip tests.
  - **Go tests:** pass the full test function name(s); forwarded verbatim to `go test -skip`, so it's a regex alternation. E.g. `TestLagoonPull|TestAcquiaPull`.
  - **Bats tests:** each pattern is matched as a case-sensitive substring against the bats filename (without `.bats`) or the `@test` description. E.g. `sveltekit` skips all tests in `sveltekit.bats`; `Symfony Composer` skips only the Composer-flavored test in `symfony.bats`. Go and bats patterns can be combined: `TestLagoonPull|sveltekit`.
  - `workflow_dispatch` runs skip loading the `public-variables` branch entirely, so maintainers can verify fixes without removing them from the embargo list first.
- `DDEV_EMBARGO_PHP_VERSIONS` - comma-separated PHP versions to skip in `TestPHPConfig`, e.g. `7.0,7.1`

## Adding a new variable

1. Open a PR to `upstream/main` that:
   - Adds the variable to the **Current variables** list above with a description and example
     (so future maintainers can discover available variables by browsing `main`)
   - Adds an empty file `.github/public-variables/<VAR_NAME>`
     (the empty file on `main` is documentation only; the actual value lives on the `public-variables` branch)
2. Push the variable value to the `public-variables` branch
   (see **How to update** below)

No workflow changes are needed - any file in this directory is picked up automatically.

## How to update

1. Edit the file(s) directly on the `public-variables` branch - no PR required
2. Include `[skip ci]` in the commit message to avoid triggering test workflows
3. To clear a variable, empty the file

## How it works

Used in `.buildkite/test.sh`, `.github/workflows/test-reusable.yml`,
`.github/workflows/test-wsl2-reusable.yml`, and `.github/workflows/quickstart.yml`.

Each CI run does `git fetch --depth=1 --no-tags https://github.com/ddev/ddev public-variables:refs/public-variables-tmp`,
reads all files via `git ls-tree` + `git show`, then deletes the temporary ref.

The load step is skipped for `workflow_dispatch` (manually triggered) runs so maintainers can verify
a previously-embargoed test is fixed without first removing it from the embargo list.

## Branch protection

The `public-variables` branch has a GitHub Ruleset with **Restrict updates** and **Restrict deletions** enabled,
with the "Organization admin" role in the bypass list.
