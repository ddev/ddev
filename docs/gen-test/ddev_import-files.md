---
layout: manual
permalink: /:path/:basename
---

{% raw %}## ddev import-files

```
ddev import-files [flags]
```

Pull the uploaded files directory of an existing project to the default
public upload directory of your project. The files can be provided as a
directory path or an archive in .tar, .tar.gz, .tar.xz, .tar.bz2, .tgz, or .zip format. For the
.zip and tar formats, the path to a directory within the archive can be
provided if it is not located at the top-level of the archive. If the
destination directory exists, it will be replaced with the assets being
imported.

The destination directory can be configured in your project's config.yaml
under the upload_dir key. If no custom upload directory is defined, the app
type's default upload directory will be used.

* [ddev auth](./ddev_auth)
* [ddev blackfire](./ddev_blackfire)
* [ddev clean](./ddev_clean)
* [ddev composer](./ddev_composer)
* [ddev config](./ddev_config)
* [ddev craft](./ddev_craft)
* [ddev debug](./ddev_debug)
* [ddev delete](./ddev_delete)
* [ddev describe](./ddev_describe)
* [ddev exec](./ddev_exec)
* [ddev export-db](./ddev_export-db)
* [ddev get](./ddev_get)
* [ddev heidisql](./ddev_heidisql)
* [ddev help](./ddev_help)
* [ddev hostname](./ddev_hostname)
* [ddev import-db](./ddev_import-db)
* [ddev import-files](./ddev_import-files)
* [ddev launch](./ddev_launch)
* [ddev list](./ddev_list)
* [ddev logs](./ddev_logs)
* [ddev mutagen](./ddev_mutagen)
* [ddev mysql](./ddev_mysql)
* [ddev npm](./ddev_npm)
* [ddev nvm](./ddev_nvm)
* [ddev pause](./ddev_pause)
* [ddev php](./ddev_php)
* [ddev poweroff](./ddev_poweroff)
* [ddev pull](./ddev_pull)
* [ddev push](./ddev_push)
* [ddev restart](./ddev_restart)
* [ddev restore-snapshot](./ddev_restore-snapshot)
* [ddev self-upgrade](./ddev_self-upgrade)
* [ddev service](./ddev_service)
* [ddev share](./ddev_share)
* [ddev showport](./ddev_showport)
* [ddev snapshot](./ddev_snapshot)
* [ddev ssh](./ddev_ssh)
* [ddev start](./ddev_start)
* [ddev stop](./ddev_stop)
* [ddev version](./ddev_version)
* [ddev xdebug](./ddev_xdebug)
* [ddev xhprof](./ddev_xhprof)
* [ddev yarn](./ddev_yarn)


### Options


<dl class="flags">
	<dt><code>--extract-path &lt;string&gt;</code></dt>
	<dd>If provided asset is an archive, optionally provide the path to extract within the archive.</dd>

	<dt><code>--src &lt;string&gt;</code></dt>
	<dd>Provide the path to the source directory or tar/tar.gz/tgz/zip archive of files to import</dd>
</dl>


### Options inherited from parent commands


<dl class="flags">
	<dt><code>-j</code>, <code>--json-output</code></dt>
	<dd>If true, user-oriented output will be in JSON format.</dd>
</dl>


{% endraw %}
### Examples

{% highlight bash %}{% raw %}
ddev import-files --src=/path/to/files.tar.gz
ddev import-files --src=/path/to/dir
ddev import-files --src=/path/to/files.tar.xz
ddev import-files --src=/path/to/files.tar.bz2{% endraw %}{% endhighlight %}

### See also

* [ddev](./ddev)
