## Using DDEV in Gitpod.io

DDEV is fully supported in Gitpod.io, and there are many ways to use it.

1. Just [open any repository](https://www.gitpod.io/docs/getting-started) using gitpod and `brew install drud/ddev/ddev` and use ddev as you would normally use it.
   * To use `ddev launch` you'll need to `sudo apt-get update && sudo apt-get install -y xdg-utils`.
   * You can just install your web app there, or import a database.
   * You may want to implement one of the `ddev pull` provider integrations to pull from a hosting provider or an upstream source.
2. Use [ddev-gitpod-launcher](https://drud.github.io/ddev-gitpod-launcher/) form to launch a repository. See the actual instructions on the [repository](https://github.com/drud/ddev-gitpod-launcher). You just click the button and it opens a fully-set-up environment. If a companion artifacts repository with the suffix `-artifacts` is available, then the `db.sql.gz` and `files.tgz` from it will be automatically loaded.
3. Use the [DDEV Gitpod Launcher Chrome Extension](https://chrome.google.com/webstore/detail/ddev-gitpod-launcher/fhceifmhhglkcahegniblknjgdggmkbn). This does the same thing as the `ddev-gitpod-launcher` form, but is installed as an easy-to-use Chrome extension.
4. Save the following link, <a href="javascript: window.location.href %3D window.location.href.includes(%27https://github.com/%27) %3F %27https://gitpod.io/%23DDEV_REPO%3D%27 %2B encodeURIComponent(window.location.href) %2B %27,DDEV_ARTIFACTS%3D/https://github.com/drud/ddev-gitpod-launcher/%27 : window.location.href"
        >Github -> ddev-gitpod</a>, to your bookmark bar; the "drag-and-drop" method is easiest. When you are on a Github repository, click the new bookmark to open DDEV Gitpod. Its essentially the same as previous options, however, it works on non-chrome browsers and native browser keyboard shortcuts can be used.

It can be complicated to get private databases and files into Gitpod, so in addition to the launchers, DDEV v1.18.2 introduces a [`git` provider example](https://github.com/drud/ddev/blob/master/pkg/ddevapp/dotddev_assets/providers/git.yaml.example), so you can pull database and files without complex setup or permissions. This was created explicitly for Gitpod integration, because in Gitpod you typically already have access to private git repositories, which are a fine place to put a starter database and files. Although [ddev-gitpod-launcher](https://drud.github.io/ddev-gitpod-launcher/) and the web extension provide the capability, you may want to integrate a git provider for each project (or, of course, one of the [other providers](https://github.com/drud/ddev/tree/master/pkg/ddevapp/dotddev_assets/providers)).
