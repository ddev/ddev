---
layout: manual
permalink: /:path/:basename
---

{% raw %}## ddev export-db

```
ddev export-db [project] [flags]
```

Dump a database to a file or to stdout

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
	<dt><code>--bzip2</code></dt>
	<dd>Use bzip2 compression</dd>

	<dt><code>-f</code>, <code>--file &lt;string&gt;</code></dt>
	<dd>Provide the path to output the dump</dd>

	<dt><code>-z</code>, <code>--gzip</code></dt>
	<dd>Use gzip compression</dd>

	<dt><code>-d</code>, <code>--target-db &lt;string&gt;</code></dt>
	<dd>If provided, target-db is alternate database to export</dd>

	<dt><code>--xz</code></dt>
	<dd>Use xz compression</dd>
</dl>


### Options inherited from parent commands


<dl class="flags">
	<dt><code>-j</code>, <code>--json-output</code></dt>
	<dd>If true, user-oriented output will be in JSON format.</dd>
</dl>


{% endraw %}
### Examples

{% highlight bash %}{% raw %}
ddev export-db --file=/tmp/db.sql.gz
ddev export-db -f /tmp/db.sql.gz
ddev export-db --gzip=false --file /tmp/db.sql
ddev export-db > /tmp/db.sql.gz
ddev export-db --gzip=false > /tmp/db.sql
ddev export-db myproject --gzip=false --file=/tmp/myproject.sql
ddev export-db someproject --gzip=false --file=/tmp/someproject.sql {% endraw %}{% endhighlight %}

### See also

* [ddev](./ddev)
