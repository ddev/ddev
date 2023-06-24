---
layout: manual
permalink: /:path/:basename
---

{% raw %}## ddev config global

Change global configuration

```
ddev config global [flags]
```

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
	<dt><code>--auto-restart-containers</code></dt>
	<dd>If true, automatically restart containers after a reboot or docker restart</dd>

	<dt><code>--disable-http2</code></dt>
	<dd>Optionally disable http2 in ddev-router, &#39;ddev global --disable-http2&#39; or `ddev global --disable-http2=false&#39;</dd>

	<dt><code>--fail-on-hook-fail</code></dt>
	<dd>If true, &#39;ddev start&#39; will fail when a hook fails.</dd>

	<dt><code>--instrumentation-opt-in</code></dt>
	<dd>instrumentation-opt-in=true</dd>

	<dt><code>--internet-detection-timeout-ms &lt;int&gt;</code></dt>
	<dd>Increase timeout when checking internet timeout, in milliseconds</dd>

	<dt><code>--letsencrypt-email &lt;string&gt;</code></dt>
	<dd>Email associated with Let&#39;s Encrypt, `ddev global --letsencrypt-email=me@example.com&#39;</dd>

	<dt><code>--mutagen-enabled</code></dt>
	<dd>If true, web container will use mutagen caching/asynchronous updates.</dd>

	<dt><code>--nfs-mount-enabled</code></dt>
	<dd>Enable NFS mounting on all projects globally</dd>

	<dt><code>--no-bind-mounts</code></dt>
	<dd>If true, don&#39;t use bind-mounts - useful for environments like remote docker where bind-mounts are impossible</dd>

	<dt><code>--omit-containers &lt;string&gt;</code></dt>
	<dd>For example, --omit-containers=dba,ddev-ssh-agent</dd>

	<dt><code>--project-tld &lt;string&gt;</code></dt>
	<dd>Override default project tld</dd>

	<dt><code>--required-docker-compose-version &lt;string&gt;</code></dt>
	<dd>Override default docker-compose version</dd>

	<dt><code>--router-bind-all-interfaces</code></dt>
	<dd>router-bind-all-interfaces=true</dd>

	<dt><code>--router-http-port &lt;string&gt;</code></dt>
	<dd>The router HTTP port for this project</dd>

	<dt><code>--router-https-port &lt;string&gt;</code></dt>
	<dd>The router HTTPS port for this project</dd>

	<dt><code>--simple-formatting</code></dt>
	<dd>If true, use simple formatting and no color for tables</dd>

	<dt><code>--table-style &lt;string&gt;</code></dt>
	<dd>Table style for list and describe, see ~/.ddev/global_config.yaml for values</dd>

	<dt><code>--use-docker-compose-from-path</code></dt>
	<dd>If true, use docker-compose from path instead of private ~/.ddev/bin/docker-compose</dd>

	<dt><code>--use-hardened-images</code></dt>
	<dd>If true, use more secure &#39;hardened&#39; images for an actual internet deployment.</dd>

	<dt><code>--use-letsencrypt</code></dt>
	<dd>Enables experimental Let&#39;s Encrypt integration, &#39;ddev global --use-letsencrypt&#39; or `ddev global --use-letsencrypt=false&#39;</dd>

	<dt><code>--use-traefik</code></dt>
	<dd>If true, use traefik for ddev-router</dd>

	<dt><code>--web-environment &lt;string&gt;</code></dt>
	<dd>Set the environment variables in the web container: --web-environment=&#34;TYPO3_CONTEXT=Development,SOMEENV=someval&#34;</dd>

	<dt><code>--web-environment-add &lt;string&gt;</code></dt>
	<dd>Append environment variables to the web container: --web-environment-add=&#34;TYPO3_CONTEXT=Development,SOMEENV=someval&#34;</dd>

	<dt><code>--wsl2-no-windows-hosts-mgt</code></dt>
	<dd>WSL2 only; make DDEV ignore Windows-side hosts file</dd>

	<dt><code>--xdebug-ide-location &lt;string&gt;</code></dt>
	<dd>For less usual IDE locations specify where the IDE is running for Xdebug to reach it</dd>
</dl>


### Options inherited from parent commands


<dl class="flags">
	<dt><code>-j</code>, <code>--json-output</code></dt>
	<dd>If true, user-oriented output will be in JSON format.</dd>
</dl>


{% endraw %}
### Examples

{% highlight bash %}{% raw %}
ddev config global --instrumentation-opt-in=false
ddev config global --omit-containers=dba,ddev-ssh-agent{% endraw %}{% endhighlight %}

### See also

* [ddev config](./ddev_config)
