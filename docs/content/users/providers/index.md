# Hosting Provider Integration

DDEV offers hosting provider integration and sample integrations for [Pantheon](https://pantheon.io), [Platform.sh](https://platform.sh) and [Acquia](https://www.acquia.com) hosting, along with other examples.

Hosting provider integration allows connecting with your upstream hosting. `ddev pull <provider>` downloads and `ddev push <provider>` uploads the **database** and the **user-generated files** to an upstream provider. It does *not* push (deploy) or pull your code. Your code should be under version control in for example Git.

DDEV provides ready-to-go integrations for Platform.sh, Acquia, and Lagoon in every project, see the .ddev/providers directory. These can be used as is, or they can be modified as you see fit (but remove the `#ddev-generated` line so DDEV doesn't replace them with the defaults).

In addition, each project includes [example recipes](https://github.com/ddev/ddev/tree/master/pkg/ddevapp/dotddev_assets/providers) for Pantheon, Git, local files, and `rsync` in its `.ddev/providers` directory, which you can use and adapt however you’d like.

DDEV provides the `pull` command with whatever recipes you have configured. For example, `ddev pull platform` is available by default, and `ddev pull pantheon` is available if you have created `.ddev/providers/pantheon.yaml`.

DDEV also provides the `push` command to push database and files to upstream. This is very dangerous to your upstream site and should only be used when appropriate. We don’t even recommended implementing the push stanzas in your YAML file, but it’s there if it suits your workflow.

Each provider recipe is a YAML file that can have whatever name you want. The examples are mostly named after the hosting providers, but they could be named `upstream.yaml` or `live.yaml`, so you could `ddev pull upstream` or `ddev pull live`. If you wanted different upstream environments to pull from, you could name one “prod” and one “dev” and `ddev pull prod` and `ddev pull dev`.

[Recipes](https://github.com/ddev/ddev/blob/master/pkg/ddevapp/dotddev_assets/providers/) are provided for [Acquia](https://github.com/ddev/ddev/blob/master/pkg/ddevapp/dotddev_assets/providers/acquia.yaml), [Local files](https://github.com/ddev/ddev/blob/master/pkg/ddevapp/dotddev_assets/providers/localfile.yaml.example) (like Dropbox, for example), [Pantheon](https://github.com/ddev/ddev/blob/master/pkg/ddevapp/dotddev_assets/providers/pantheon.yaml.example), [Platform.sh](https://github.com/ddev/ddev/blob/master/pkg/ddevapp/dotddev_assets/providers/platform.yaml), and [rsync](https://github.com/ddev/ddev/blob/master/pkg/ddevapp/dotddev_assets/providers/rsync.yaml.example). We know you’ll find improvements to these examples and will have lots to contribute for other hosting providers, and we look forward to your contributions as pull requests here or in [ddev-contrib](https://github.com/ddev/ddev-contrib).

Each provider recipe is a file named `<provider>.yaml` and consists of several mostly-optional stanzas:

* `environment_variables`: Environment variables will be created in the web container for each of these during pull or push operations. They’re used to provide context (project ID, environment name, etc.) for each of the other stanzas. This stanza is not used in more recent hosting integrations, since providing the environment variables in `config.yaml` or via `ddev pull xxx --environment=VARIABLE=value` is preferred.
* `db_pull_command`: A script that determines how DDEV should obtain a database. Its job is to create a gzipped database dump in `/var/www/html/.ddev/.downloads/db.sql.gz`. This is optional; if nothing has to be done to obtain the database dump, this step can be omitted.
* `db_import_command`: (optional) A script that imports the downloaded database. This is for advanced usages like multiple databases. The default behavior only imports a single database into the `db` database. The [localfile example](https://github.com/ddev/ddev/blob/master/pkg/ddevapp/dotddev_assets/providers/localfile.yaml.example) uses this technique.
* `files_pull_command`: A script that determines how DDEV can get user-generated files from upstream. Its job is to copy the files from upstream to `/var/www/html/.ddev/.downloads/files`. If nothing has to be done to obtain the files, this step can run `true`.
* `files_import_command`: (optional) A script that imports the downloaded files. There are a number of situations where it’s messy to push a directory of files around, and one can put it directly where it’s needed. The [localfile example](https://github.com/ddev/ddev/blob/master/pkg/ddevapp/dotddev_assets/providers/localfile.yaml.example) uses this technique.
* `db_push_command`: A script that determines how DDEV should push a database. Its job is to take a gzipped database dump from `/var/www/html/.ddev/.downloads/db.sql.gz` and load it on the hosting provider.
* `files_push_command`: A script that determines how DDEV push user-generated files to upstream. Its job is to copy the files from the project’s user-files directories (`$DDEV_FILES_DIRS`) to the correct places on the upstream provider.

The [environment variables provided to custom commands](../extend/custom-commands.md#environment-variables-provided) are also available for use in these recipes.

There are [hooks](../configuration/hooks.md) available to execute commands before and after each pull or push: `pre-pull`, `post-pull`, `pre-push`, `post-push`. These could be for example a [`ddev snapshot`](../usage/commands.md#snapshot) to backup the database before a pull or a specific task to clear/warm-up caches of your application.

## Example Integrations and Hints

* All of the [supplied integrations](https://github.com/ddev/ddev/tree/master/pkg/ddevapp/dotddev_assets/providers) are examples of what you can do.
* You can name a provider anything you want. For example, an Acquia integration doesn’t have to be named “acquia”, it can be named “upstream”. This is a great technique for [downloading a particular multisite](https://stackoverflow.com/a/68553116/215713).

## Provider Debugging

You can uncomment the `set -x` in each stanza to see more of what’s going on. It really helps. Watch it as you do a `ddev pull <whatever>`.

Although the various commands could be executed on the host or in other containers if configured that way, most commands are executed in the web container. So the best thing to do is to [`ddev ssh`](../usage/commands.md#ssh) and manually execute each command you want to use. When you have it right, use it in the YAML file.
