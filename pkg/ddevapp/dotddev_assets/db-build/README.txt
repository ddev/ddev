#ddev-generated
Files in this directory will be used to customize the dbimage, you can add:

* .ddev/db-build/Dockerfile
* .ddev/db-build/Dockerfile.*

Additionally, you can use `pre.` variants that are inserted before everything else:

* .ddev/db-build/pre.Dockerfile
* .ddev/db-build/pre.Dockerfile.*

Examine the resultant generated Dockerfile (which you will never edit directly), at `.ddev/.dbimageBuild/Dockerfile`. You can force a rebuild with `ddev debug rebuild -s db`.

You can use the `.ddev/db-build` directory as the Docker “context” directory as well. So for example, if a file named `file.txt` exists in `.ddev/db-build`, you can use `ADD file.txt /` in the Dockerfile.

See https://ddev.readthedocs.io/en/stable/users/extend/customizing-images/ for advanced examples.
