---
search:
  boost: .2
---
# Quickstart Maintenance

Project Type Quickstarts are pretty easy to add as just docs in [quickstarts](../users/quickstart.md),

- We want to keep them consistent.
- To make them maintainable, we need automated tests.

## Creating a Quickstart

The quickstart can be based on one of the existing [quickstarts](../users/quickstart.md).

The general intention is that it should be a *teaching* tool. It will not cover all the complexities of the project type being discussed. Try for the most straightforward, repeatable, testable approach so that a naive person with no experience on the project type can walk through it.

In general:
1. Add a link to the upstream installation or "Getting Started" web page, so people can know where the instructions are coming from.
1. Use `mkdir my-<projecttype>-site && cd my-<projecttype>-site` as the opener.
2. Composer-based recipes are preferable, unless the project does not use or prefer composer.
3. If your project type does not yet appear in the DDEV documentation, its name may need to be added to the [.spellcheckwordlist.txt](https://github.com/ddev/ddev/blob/main/.spellcheckwordlist.txt) so it can pass the spellcheck test.
4. If your project type includes folders that accept public files (such as images), for example, `public/media`, make sure to add them to the [config](../users/configuration/config.md#upload_dirs) command:

    ```bash
    ddev config ... --upload-dirs=public/media
    ```

Testing:

1. Each new quickstart needs to have automated tests.
2. You can base your test on an example like the [Backdrop test](https://github.com/ddev/ddev/blob/main/docs/tests/backdrop.bats) and adapt to cover the steps in your quickstart.
3. You can run `bats` locally.
    - See [`bats-core` documentation](https://bats-core.readthedocs.io/en/stable/).
    - See [`bats assert` and `bats-support` libraries documentation](https://github.com/ztombol/bats-docs).
    - If you install `bats` libraries manually (without package managers), make sure to set the `BATS_LIB_PATH` environment variable to the appropriate path. For example:

        ```bash
        export BATS_LIB_PATH=/path/to/bats
        ```

    - To run the docs tests, `cd docs && bats tests` or `bats tests/backdrop.bats` for example.
