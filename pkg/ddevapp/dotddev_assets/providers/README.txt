Providers README
================

#ddev-generated

## Introduction to Hosting Provider Integration

DDEV's hosting provider integration lets you integrate with any upstream source of database dumps and files (such as your production or staging server) and provides examples of configuration for Acquia, Platform.sh, Pantheon, rsync, etc.

The best part of this is you can change them and adapt them in any way you need to, they're all short scripted recipes. There are several example recipes created in the .ddev/providers directory of every project or see them in the code at https://github.com/ddev/ddev/tree/master/pkg/ddevapp/dotddev_assets/providers.

ddev provides the `pull` command with whatever recipes you have configured. For example, `ddev pull acquia` if you have created `.ddev/providers/acquia.yaml`.

ddev also provides the `push` command to push database and files to upstream. This is very dangerous to your upstream site and should only be used with extreme caution. It's recommended not even to implement the push stanzas in your yaml file, but if it fits your workflow, use it well.

Each provider recipe is a yaml file that can be named any way you want to name it. The examples are mostly named after the hosting providers, but they could be named "upstream.yaml" or "live.yaml", so you could `ddev pull upstream` or `ddev pull live`. If you wanted different upstream environments to pull from, you could name one "prod" and one "dev" and `ddev pull prod` and `ddev pull dev`.

Several example recipes are at https://github.com/ddev/ddev/tree/master/pkg/ddevapp/dotddev_assets/providers and in this directory.

Each provider recipe is a file named `<provider>.yaml` and consists of several mostly-optional stanzas:

* `environment_variables`: Environment variables will be created in the web container for each of these during pull or push operations. They're used to provide context (project id, environment name, etc.) for each of the other stanzas.
* `db_pull_command`: A script that determines how ddev should pull a database. It's job is to create a gzipped database dump in /var/www/html/.ddev/.downloads/db.sql.gz.
* `files_pull_command`: A script that determines how ddev can get user-generated files from upstream. Its job is to copy the files from upstream to  /var/www/html/.ddev/.downloads/files.
* `db_push_command`: A script that determines how ddev should push a database. It's job is to take a  gzipped database dump from /var/www/html/.ddev/.downloads/db.sql.gz and load it on the hosting provider.
* `files_pull_command`: A script that determines how ddev push user-generated files to upstream. Its job is to copy the files from the project's user-files directories ($DDEV_FILES_DIRS) to the correct places on the upstream provider.

The environment variables provided to custom commands (see https://ddev.readthedocs.io/en/stable/users/extend/custom-commands/#environment-variables-provided) are also available for use in these recipes.

### Provider Debugging

You can uncomment the `set -x` in each stanza to see more of what's going on. It really helps.

Although the various commands could be executed on the host or in other containers if configured that way, most commands are executed in the web container. So the best thing to do is to `ddev ssh` and manually execute each command you want to use. When you have it right, use it in the yaml file.
