#ddev-generated
Files in this directory will be used to customize the dbimage, you can add:

* .ddev/db-build/Dockerfile
* .ddev/db-build/Dockerfile.*

Additionally, you can use `pre.` variants that are inserted before what DDEV adds:

* .ddev/db-build/pre.Dockerfile
* .ddev/db-build/pre.Dockerfile.*

Finally, you can also use `prepend.` variants that are inserted on top of the Dockerfile allowing for Multi-stage builds and other more complex use cases:

* .ddev/db-build/prepend.Dockerfile
* .ddev/db-build/prepend.Dockerfile.*

See https://docs.docker.com/build/building/multi-stage/

Examine the resultant generated Dockerfile (which you will never edit directly), at `.ddev/.dbimageBuild/Dockerfile`. You can force a rebuild with `ddev debug rebuild -s db`.

You can use the `.ddev/db-build` directory as the Docker “context” directory as well. So for example, if a file named `file.txt` exists in `.ddev/db-build`, you can use `ADD file.txt /` in the Dockerfile.

See https://ddev.readthedocs.io/en/stable/users/extend/customizing-images/ for advanced examples.
