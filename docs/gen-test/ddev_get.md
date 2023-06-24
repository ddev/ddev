---
layout: manual
permalink: /:path/:basename
---

{% raw %}## ddev get

```
ddev get <addonOrURL> [project] [flags]
```

Get/Download a 3rd party add-on (service, provider, etc.). This can be a github repo, in which case the latest release will be used, or it can be a link to a .tar.gz in the correct format (like a particular release's .tar.gz) or it can be a local directory. Use 'ddev get --list' or 'ddev get --list --all' to see a list of available add-ons. Without --all it shows only official ddev add-ons. To list installed add-ons, 'ddev get --installed', to remove an add-on 'ddev get --remove <add-on>'.

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
	<dt><code>--all</code></dt>
	<dd>List unofficial add-ons for &#39;ddev get&#39; in addition to the official ones</dd>

	<dt><code>--installed</code></dt>
	<dd>Show installed ddev-get add-ons</dd>

	<dt><code>--list</code></dt>
	<dd>List available add-ons for &#39;ddev get&#39;</dd>

	<dt><code>--remove &lt;string&gt;</code></dt>
	<dd>Remove a ddev-get add-on</dd>

	<dt><code>-v</code>, <code>--verbose</code></dt>
	<dd>Extended/verbose output for ddev get</dd>

	<dt><code>--version &lt;string&gt;</code></dt>
	<dd>Specify a particular version of add-on to install</dd>
</dl>


### Options inherited from parent commands


<dl class="flags">
	<dt><code>-j</code>, <code>--json-output</code></dt>
	<dd>If true, user-oriented output will be in JSON format.</dd>
</dl>


{% endraw %}
### Examples

{% highlight bash %}{% raw %}
ddev get ddev/ddev-redis
ddev get ddev/ddev-redis --version v1.0.4
ddev get https://github.com/ddev/ddev-drupal9-solr/archive/refs/tags/v0.0.5.tar.gz
ddev get /path/to/package
ddev get /path/to/tarball.tar.gz
ddev get --list
ddev get --list --all
ddev get --installed
ddev get --remove someaddonname,
ddev get --remove someowner/ddev-someaddonname,
ddev get --remove ddev-someaddonname
{% endraw %}{% endhighlight %}

### See also

* [ddev](./ddev)
