---
layout: manual
permalink: /:path/:basename
---

{% raw %}## ddev start

```
ddev start [projectname ...] [flags]
```

Start initializes and configures the web server and database containers
to provide a local development environment. You can run 'ddev start' from a
project directory to start that project, or you can start stopped projects in
any directory by running 'ddev start projectname [projectname ...]'

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
	<dd>Start all projects</dd>

	<dt><code>-s</code>, <code>--select</code></dt>
	<dd>Interactively select a project to start</dd>

	<dt><code>-y</code>, <code>--skip-confirmation</code></dt>
	<dd>Skip any confirmation steps</dd>
</dl>


### Options inherited from parent commands


<dl class="flags">
	<dt><code>-j</code>, <code>--json-output</code></dt>
	<dd>If true, user-oriented output will be in JSON format.</dd>
</dl>


{% endraw %}
### Examples

{% highlight bash %}{% raw %}
ddev start
ddev start <project1> <project2>
ddev start --all{% endraw %}{% endhighlight %}

### See also

* [ddev](./ddev)
