---
search:
  boost: .2
---
# Contributing a Quickstart Guide

A quickstart guide is a teaching tool that helps new users get started with a specific project type in DDEV. This document explains how to create and contribute a new quickstart guide.

Project Type quickstarts are pretty easy to add as just docs in [quickstarts](../users/quickstart.md),

- We want to keep them consistent.
- To make them maintainable, we need automated tests.

## Before You Begin

- Review [existing quickstarts](https://github.com/ddev/ddev/blob/main/docs/content/users/quickstart.md) in the DDEV documentation for reference.
- Review [existing automated tests](https://github.com/ddev/ddev/tree/main/docs/tests) to see how the tests work.

## Creating a Quickstart

The quickstart can be based on one of the existing [quickstarts](../users/quickstart.md).

The general intention is that it should be a *teaching* tool. It will not cover all the complexities of the project type being discussed. Try for the most straightforward, repeatable, testable approach so that a naive person with no experience on the project type can walk through it. People with more experience with the project type should be able to adapt the provided instructions to meet their needs.

In general:

1. Add a link to the upstream installation or "Getting Started" web page, so people can know where the instructions are coming from.
2. Use `mkdir my-<projecttype>-site && cd my-<projecttype>-site` as the opener. (There are places like Magento 2 where the project name must be used later in the recipe, in those cases, use an environment variable, like `PROJECT_NAME=my-<projecttype>-site`.)
3. Composer-based recipes are preferable, unless the project does not use or prefer composer.
4. If your project type does not yet appear in the DDEV documentation, your PR should add the name to the [.spellcheckwordlist.txt](https://github.com/ddev/ddev/blob/main/.spellcheckwordlist.txt) so it can pass the spell check test.
5. If your project installation requires providing an administrative username and/or password, make sure to indicate clearly in the instructions what it is.
6. If your project type includes folders that accept public files (such as images), for example, `public/media`, make sure to add them to the [config](../users/configuration/config.md#upload_dirs) command:

    ```bash
    ddev config ... --upload-dirs=public/media
    ```

## Automated Tests

1. Each new quickstart needs to have automated tests.
2. You can base your test on an example like the [Backdrop test](https://github.com/ddev/ddev/blob/main/docs/tests/backdrop.bats) and adapt to cover the steps in your quickstart.
3. You can run `bats` locally.
    - See [`bats-core` documentation](https://bats-core.readthedocs.io/en/stable/).
    - See [`bats-assert`, `bats-file`, and `bats-support` libraries documentation](https://github.com/bats-core/homebrew-bats-core).
    - If you install `bats` libraries manually (without package managers), make sure to set the `BATS_LIB_PATH` environment variable to the appropriate path. For example:

        ```bash
        export BATS_LIB_PATH=/path/to/bats
        ```

    - To run the docs tests, `cd docs && bats tests` or `bats tests/backdrop.bats` for example.

## Final Note

THANK YOU FOR CONTRIBUTING! ❤️
