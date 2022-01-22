# Working with docs

You can see what the docs will look like after you've edited them by running `make mkdocs-serve`. It will launch a webserver on port 8000 so you can see what the docs will look like when they land on readthedocs.io.

It's easiest to [install mkdocs locally](https://www.mkdocs.org/user-guide/installation/) with `pip3 install mkdocs` and `pip3 install -r requirements.txt`, but if mkdocs isn't installed, `make mkdocs-serve` will run a docker command to serve them on `http://localhost:8000`.
