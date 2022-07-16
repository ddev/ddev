# Working on the docs

This section is designed for people who want to contribute to the DDEV documentation.
When you are going to make any changes to the documentation it is recommended that you test them locally to see what your changes look like.

## Fork / clone the DDEV repository

To start making changes you need to get a local copy of the DDEV documentation, we recommend you to [fork the DDEV repository](https://github.com/drud/ddev/fork) as that's where the documentation lives.
After forking the repository you can clone it to your local machine.

## Make your changes

Now that you've got a local copy you can make your changes;

| Action                        | Path                                                                    |
|-------------------------------|-------------------------------------------------------------------------|
| Changing Documentation        | `./docs/content/users/*` <br> `./docs/content/developers/*`             |
| Changing MkDocs Configuration | `./mkdocs.yml`                                                          |
| Changing the front-end        | `./docs/content/assets/extra.css` <br> `./docs/content/assets/extra.js` |

## Preview your changes

You should see how your changes look before making a pull request, you can do so by running `make mkdocs-serve`.
It will launch a webserver on port 8000 so you can see what the docs will look like when they land on readthedocs.io.
> **Please note:** <br>
> While it's easiest to [install mkdocs locally](https://www.mkdocs.org/user-guide/installation/) it is not required, `make mkdocs-serve` will look for MkDocs but when it is not found it will run a docker command to serve the documentation on `http://localhost:8000`.<br><br>
> If you don't have `make` on your system, you can easily install it, but alternatively you can also just run the docker command that `make mkdocs-serve` runs:
>
> ```
> docker run -it -p 8000:8000 -v "${PWD}:/docs" -e "ADD_MODULES=mkdocs-material mkdocs-redirects mkdocs-minify-plugin mdx_truly_sane_lists mkdocs-git-revision-date-localized-plugin" -e "LIVE_RELOAD_SUPPORT=true" -e "FAST_MODE=true" -e "DOCS_DIRECTORY=./docs" polinux/mkdocs;
> ```

## Check changed markdown files for potential errors

Before you publish your changes it is recommended to use markdownlint to check your files for any errors or inconsistencies.
You can do so by running `make markdownlint`.
> **Please note:** <br>
> The command `make markdownlint` requires you to have markdownlint-cli installed, which you can do by executing `npm install -g markdownlint-cli`

## Publish your changes

If all looks good it's time to commit your changes and make a pull request back into the official DDEV repository.
> **Please note:** <br>
> When you make a pull requests several tests/tasks will be ran.
> One task named 'docs/readthedocs.org:ddev' will build a version of the docs containing all the changes from your pull request which you can use to check the final result.
