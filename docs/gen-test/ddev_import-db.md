---
layout: manual
permalink: /:path/:basename
---

{% raw %}## ddev import-db

```
ddev import-db [project] [flags]
```

Import a sql file into the project.
The database dump file can be provided as a SQL dump in a .sql, .sql.gz, sql.bz2, sql.xz, .mysql, .mysql.gz, .zip, .tgz, or .tar.gz
format. For the zip and tar formats, the path to a .sql file within the archive
can be provided if it is not located at the top level of the archive. An optional target database
can also be provided; the default is the default database named "db".
Also note the related "ddev mysql" command

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
	<dd>If provided asset is an archive, provide the path to extract within the archive.</dd>

	<dt><code>--no-drop</code></dt>
	<dd>Set if you do NOT want to drop the db before importing</dd>

	<dt><code>-p</code>, <code>--progress</code></dt>
	<dd>Display a progress bar during import</dd>

	<dt><code>-f</code>, <code>--src &lt;string&gt;</code></dt>
	<dd>Provide the path to a sql dump in .sql or tar/tar.gz/tgz/zip format</dd>

	<dt><code>-d</code>, <code>--target-db &lt;string&gt;</code></dt>
	<dd>If provided, target-db is alternate database to import into</dd>
</dl>


### Options inherited from parent commands


<dl class="flags">
	<dt><code>-j</code>, <code>--json-output</code></dt>
	<dd>If true, user-oriented output will be in JSON format.</dd>
</dl>


{% endraw %}
### Examples

{% highlight bash %}{% raw %}
ddev import-db
ddev import-db --src=.tarballs/junk.sql
ddev import-db --src=.tarballs/junk.sql.gz
ddev import-db --target-db=newdb --src=.tarballs/db.sql.gz
ddev import-db --src=.tarballs/db.sql.bz2
ddev import-db --src=.tarballs/db.sql.xz
ddev import-db <db.sql
ddev import-db someproject <db.sql
gzip -dc db.sql.gz | ddev import-db{% endraw %}{% endhighlight %}

### See also

* [ddev](./ddev)
