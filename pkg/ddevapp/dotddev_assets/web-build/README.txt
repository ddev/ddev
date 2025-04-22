#ddev-generated
Files in this directory will be used to customize the webimage, you can add:

* .ddev/web-build/Dockerfile
* .ddev/web-build/Dockerfile.*

Additionally, you can use `pre.` variants that are inserted before what DDEV adds:

* .ddev/web-build/pre.Dockerfile
* .ddev/web-build/pre.Dockerfile.*

Finally, you can also use `prepend.` variants that are inserted on top of the Dockerfile allowing for Multi-stage builds and other more complex use cases:

* .ddev/web-build/prepend.Dockerfile
* .ddev/web-build/prepend.Dockerfile.*

See https://docs.docker.com/build/building/multi-stage/

Examine the resultant generated Dockerfile (which you will never edit directly), at `.ddev/.webimageBuild/Dockerfile`. You can force a rebuild with `ddev debug rebuild -s web`.

You can use the `.ddev/web-build` directory as the Docker “context” directory as well. So for example, if a file named `file.txt` exists in `.ddev/web-build`, you can use `ADD file.txt /` in the Dockerfile.

See https://ddev.readthedocs.io/en/stable/users/extend/customizing-images/ for advanced examples.
