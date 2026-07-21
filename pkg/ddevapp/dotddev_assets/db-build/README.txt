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

Examine the resultant generated Dockerfile (which you will never edit directly), at `.ddev/.dbimageBuild/Dockerfile`. You can force a rebuild with `ddev utility rebuild -s db`.

You can use the `.ddev/db-build` directory as the Docker “context” directory as well. So for example, if a file named `file.txt` exists in `.ddev/db-build`, you can use `ADD file.txt /` in the Dockerfile.

Global variants in `$HOME/.ddev/db-build/` are also supported and apply to all projects on this machine. Global files are inserted *before* project-level files. See https://docs.ddev.com/en/stable/users/usage/architecture/#global-files to find out where this directory actually lives on your system, since it isn't always `$HOME/.ddev`.

See https://docs.ddev.com/en/stable/users/extend/customizing-images/ for advanced examples.
