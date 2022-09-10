# Working on the Docs

This section is designed for people who want to contribute to the DDEV documentation.

We recommend testing documentation changes locally so you can see your changes and confirm they look great.

## Fork / Clone the DDEV Repository

To start making changes you’ll need a local copy of the DDEV documentation, so [fork the DDEV repository](https://github.com/drud/ddev/fork) which includes the documentation.

After forking the repository, you can clone it to your local machine.

## Make Changes

Now that you’ve got a local copy, you can make your changes.

| Action               | Path                                                                    |
|----------------------|-------------------------------------------------------------------------|
| Documentation        | `./docs/content/users/*` <br> `./docs/content/developers/*`             |
| MkDocs configuration | `./mkdocs.yml`                                                          |
| Front end            | `./docs/content/assets/extra.css` <br> `./docs/content/assets/extra.js` |

## Preview Changes

Preview your changes locally by running `make mkdocs-serve`.

This will launch a web server on port 8000 and automatically refresh pages as they’re edited.

!!!tip "No need to install MkDocs locally!"
    It’s easiest to install [install MkDocs locally](https://www.mkdocs.org/user-guide/installation/), but you don’t have to. The `make mkdocs-serve` command will look for and use a local binary, otherwise using `make` to build and serve the documenation. If you don’t have `make` installed on your system, you can directly run the command it would have instead:

    ```
    docker run -it -p 8000:8000 -v "${PWD}:/docs" -e "ADD_MODULES=mkdocs-material mkdocs-redirects mkdocs-minify-plugin mdx_truly_sane_lists mkdocs-git-revision-date-localized-plugin" -e "LIVE_RELOAD_SUPPORT=true" -e "FAST_MODE=true" -e "DOCS_DIRECTORY=./docs" polinux/mkdocs;
    ```

## Check Markdown for Errors

Run `make markdownlint` before you publish changes to quickly check your files for errors or inconsistencies.

!!!warning "`markdownlint-cli` required!"
    The `make markdownlint` command requires you to have `markdownlint-cli` installed, which you can do by executing `npm install -g markdownlint-cli`

## Publish Changes

If all looks good, it’s time to commit your changes and make a pull request back into the official DDEV repository.

When you make a pull request, several tasks and test actions will be run. One of those is a task named `docs/readthedocs.org:ddev`, which builds a version of the docs containing all the changes from your pull request. You can use that to confirm the final result is exactly what you’d expect.
