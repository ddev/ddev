---
layout: manual
permalink: /:path/:basename
---

{% raw %}## ddev list

```
ddev list [flags]
```

List projects. Shows all projects by default, shows active projects only with --active-only

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
	<dt><code>-A</code>, <code>--active-only</code></dt>
	<dd>If set, only currently active projects will be displayed.</dd>

	<dt><code>--continuous</code></dt>
	<dd>If set, project information will be emitted until the command is stopped.</dd>

	<dt><code>-I</code>, <code>--continuous-sleep-interval &lt;int&gt;</code></dt>
	<dd>Time in seconds between ddev list --continuous output lists.</dd>

	<dt><code>-t</code>, <code>--type &lt;string&gt;</code></dt>
	<dd>Show only projects of this type</dd>

	<dt><code>-W</code>, <code>--wrap-table</code></dt>
	<dd>Display table with wrapped text if required.</dd>
</dl>


### Options inherited from parent commands


<dl class="flags">
	<dt><code>-j</code>, <code>--json-output</code></dt>
	<dd>If true, user-oriented output will be in JSON format.</dd>
</dl>


{% endraw %}
### Examples

{% highlight bash %}{% raw %}
ddev list
ddev list --active-only
ddev list -A
ddev list --type=drupal8
ddev list -t drupal8{% endraw %}{% endhighlight %}

### See also

* [ddev](./ddev)
