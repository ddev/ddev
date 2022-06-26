# Working with docs

You can see what the docs will look like after you've edited them by running `make mkdocs-serve`. It will launch a webserver on port 8000 so you can see what the docs will look like when they land on readthedocs.io.

It's easiest to [install mkdocs locally](https://www.mkdocs.org/user-guide/installation/) with `pip3 install mkdocs` and `pip3 install -r requirements.txt`, but if mkdocs isn't installed, `make mkdocs-serve` will run a docker command to serve them on `http://localhost:8000`.

If you don't have `make` on your system, you can easily install it, but you can also just run the docker command that `make mkdocs-serve` runs. `docker run -it -p 8000:8000 -v "$${PWD}:/docs" -e "ADD_MODULES=mkdocs-material mdx_truly_sane_lists mkdocs-git-revision-date-localized-plugin" -e "LIVE_RELOAD_SUPPORT=true"  -e "FAST_MODE=true" -e "DOCS_DIRECTORY=./docs" polinux/mkdocs;`
