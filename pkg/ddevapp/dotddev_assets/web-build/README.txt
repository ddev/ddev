#ddev-generated
Files in this directory will be used to customize the webimage, you can add:

* .ddev/web-build/Dockerfile
* .ddev/web-build/Dockerfile.*

Additionally, you can use `pre.` variants that are inserted before everything else:

* .ddev/web-build/pre.Dockerfile
* .ddev/web-build/pre.Dockerfile.*

Examine the resultant generated Dockerfile (which you will never edit directly), at `.ddev/.webimageBuild/Dockerfile`. You can force a rebuild with `ddev debug rebuild -s web`.

You can use the `.ddev/web-build` directory as the Docker “context” directory as well. So for example, if a file named `file.txt` exists in `.ddev/web-build`, you can use `ADD file.txt /` in the Dockerfile.

See https://ddev.readthedocs.io/en/stable/users/extend/customizing-images/ for advanced examples.
