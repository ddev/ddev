---
layout: manual
permalink: /:path/:basename
---

{% raw %}## ddev snapshot

```
ddev snapshot [projectname projectname...] [flags]
```

Uses mariabackup or xtrabackup command to create a database snapshot in the .ddev/db_snapshots folder. These are compatible with server backups using the same tools and can be restored with "ddev snapshot restore".

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
	<dt><code>-a</code>, <code>--all</code></dt>
	<dd>Snapshot all projects. Will start the project if it is stopped or paused</dd>

	<dt><code>-C</code>, <code>--cleanup</code></dt>
	<dd>Cleanup snapshots</dd>

	<dt><code>-l</code>, <code>--list</code></dt>
	<dd>List snapshots</dd>

	<dt><code>-n</code>, <code>--name &lt;string&gt;</code></dt>
	<dd>provide a name for the snapshot</dd>

	<dt><code>-y</code>, <code>--yes</code></dt>
	<dd>Yes - skip confirmation prompt</dd>
</dl>


### Options inherited from parent commands


<dl class="flags">
	<dt><code>-j</code>, <code>--json-output</code></dt>
	<dd>If true, user-oriented output will be in JSON format.</dd>
</dl>


{% endraw %}
### Examples

{% highlight bash %}{% raw %}
ddev snapshot
ddev snapshot --name some_descriptive_name
ddev snapshot --cleanup
ddev snapshot --cleanup -y
ddev snapshot --list
ddev snapshot --all{% endraw %}{% endhighlight %}

### See also

* [ddev](./ddev)
