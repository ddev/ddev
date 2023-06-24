---
layout: manual
permalink: /:path/:basename
---

{% raw %}## ddev stop

```
ddev stop [projectname ...] [flags]
```

Stop and remove the containers of a project. You can run 'ddev stop'
from a project directory to stop/remove that project, or you can stop/remove projects in
any directory by running 'ddev stop projectname [projectname ...]' or 'ddev stop -a'.

By default, stop is a non-destructive operation and will leave database
contents intact. It never touches your code or files directories.

To remove database contents and global listing, 
use "ddev delete" or "ddev stop --remove-data".

To snapshot the database on stop, use "ddev stop --snapshot"; A snapshot is automatically created on
"ddev stop --remove-data" unless you use "ddev stop --remove-data --omit-snapshot".


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
	<dd>Stop and remove all running or container-stopped projects and remove from global projects list</dd>

	<dt><code>-O</code>, <code>--omit-snapshot</code></dt>
	<dd>Omit/skip database snapshot</dd>

	<dt><code>-R</code>, <code>--remove-data</code></dt>
	<dd>Remove stored project data (MySQL, logs, etc.)</dd>

	<dt><code>-s</code>, <code>--select</code></dt>
	<dd>Interactively select a project to stop</dd>

	<dt><code>-S</code>, <code>--snapshot</code></dt>
	<dd>Create database snapshot</dd>

	<dt><code>--stop-ssh-agent</code></dt>
	<dd>Stop the ddev-ssh-agent container</dd>

	<dt><code>-U</code>, <code>--unlist</code></dt>
	<dd>Remove the project from global project list, it won&#39;t show in ddev list until started again</dd>
</dl>


### Options inherited from parent commands


<dl class="flags">
	<dt><code>-j</code>, <code>--json-output</code></dt>
	<dd>If true, user-oriented output will be in JSON format.</dd>
</dl>


{% endraw %}
### Examples

{% highlight bash %}{% raw %}
ddev stop
ddev stop proj1 proj2 proj3
ddev stop --all
ddev stop --all --stop-ssh-agent
ddev stop --remove-data{% endraw %}{% endhighlight %}

### See also

* [ddev](./ddev)
