#ddev-generated
Files in this directory will be used to customize the dbimage of every project on this machine, you can add:

* $HOME/.ddev/db-build/Dockerfile
* $HOME/.ddev/db-build/Dockerfile.*

Additionally, you can use `pre.` variants that are inserted before what DDEV adds:

* $HOME/.ddev/db-build/pre.Dockerfile
* $HOME/.ddev/db-build/pre.Dockerfile.*

Finally, you can also use `prepend.` variants that are inserted on top of the Dockerfile allowing for Multi-stage builds and other more complex use cases:

* $HOME/.ddev/db-build/prepend.Dockerfile
* $HOME/.ddev/db-build/prepend.Dockerfile.*

See https://docs.docker.com/build/building/multi-stage/

These files are inserted before the project's own `.ddev/db-build` files, which are applied last.

Examine the resultant generated Dockerfile (which you will never edit directly), at `.ddev/.dbimageBuild/Dockerfile` in each project. You can force a rebuild with `ddev utility rebuild -s db`.

You can use this directory as the Docker “context” directory as well. So for example, if a file named `file.txt` exists in `$HOME/.ddev/db-build`, you can use `COPY file.txt /` in the Dockerfile. A project file of the same name wins.

This directory isn't always `$HOME/.ddev`, see https://docs.ddev.com/en/stable/users/usage/architecture/#global-files

See https://docs.ddev.com/en/stable/users/extend/customizing-images/#global-dockerfiles for advanced examples.
