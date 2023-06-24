---
layout: manual
permalink: /:path/:basename
---

{% raw %}## ddev config

Create or modify a ddev project configuration in the current directory

```
ddev config [provider or 'global'] [flags]
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
	<dt><code>--additional-fqdns &lt;string&gt;</code></dt>
	<dd>A comma-delimited list of FQDNs for the project</dd>

	<dt><code>--additional-hostnames &lt;string&gt;</code></dt>
	<dd>A comma-delimited list of hostnames for the project</dd>

	<dt><code>--auto</code></dt>
	<dd>Automatically run config without prompting.</dd>

	<dt><code>--bind-all-interfaces</code></dt>
	<dd>Bind host ports on all interfaces, not just on localhost network interface</dd>

	<dt><code>--composer-root &lt;string&gt;</code></dt>
	<dd>Overrides the default composer root directory for the web service</dd>

	<dt><code>--composer-root-default</code></dt>
	<dd>Unsets a web service composer root directory override</dd>

	<dt><code>--composer-version &lt;string&gt;</code></dt>
	<dd>Specify override for composer version in web container. This may be &#34;&#34;, &#34;1&#34;, &#34;2&#34;, &#34;2.2&#34;, &#34;stable&#34;, &#34;preview&#34;, &#34;snapshot&#34; or a specific version.</dd>

	<dt><code>--create-docroot</code></dt>
	<dd>Prompts ddev to create the docroot if it doesn&#39;t exist</dd>

	<dt><code>--database &lt;string&gt;</code></dt>
	<dd>Specify the database type:version to use. Defaults to mariadb:10.4</dd>

	<dt><code>--db-image &lt;string&gt;</code></dt>
	<dd>Sets the db container image</dd>

	<dt><code>--db-image-default</code></dt>
	<dd>Sets the default db container image for this ddev version</dd>

	<dt><code>--db-working-dir &lt;string&gt;</code></dt>
	<dd>Overrides the default working directory for the db service</dd>

	<dt><code>--db-working-dir-default</code></dt>
	<dd>Unsets a db service working directory override</dd>

	<dt><code>--dba-image &lt;string&gt;</code></dt>
	<dd>Sets the dba container image</dd>

	<dt><code>--dba-image-default</code></dt>
	<dd>Sets the default dba container image for this ddev version</dd>

	<dt><code>--dba-working-dir &lt;string&gt;</code></dt>
	<dd>Overrides the default working directory for the dba service</dd>

	<dt><code>--dba-working-dir-default</code></dt>
	<dd>Unsets a dba service working directory override</dd>

	<dt><code>--dbimage-extra-packages &lt;string&gt;</code></dt>
	<dd>A comma-delimited list of Debian packages that should be added to db container when the project is started</dd>

	<dt><code>--default-container-timeout &lt;int&gt;</code></dt>
	<dd>default time in seconds that ddev waits for all containers to become ready on start</dd>

	<dt><code>--disable-settings-management</code></dt>
	<dd>Prevent ddev from creating or updating CMS settings files</dd>

	<dt><code>--docroot &lt;string&gt;</code></dt>
	<dd>Provide the relative docroot of the project, like &#39;docroot&#39; or &#39;htdocs&#39; or &#39;web&#39;, defaults to empty, the current directory</dd>

	<dt><code>--fail-on-hook-fail</code></dt>
	<dd>Decide whether &#39;ddev start&#39; should be interrupted by a failing hook</dd>

	<dt><code>--host-db-port &lt;string&gt;</code></dt>
	<dd>The db container&#39;s localhost-bound port</dd>

	<dt><code>--host-dba-port &lt;string&gt;</code></dt>
	<dd>The dba (PHPMyAdmin) container&#39;s localhost-bound port, if exposed via bind-all-interfaces</dd>

	<dt><code>--host-https-port &lt;string&gt;</code></dt>
	<dd>The web container&#39;s localhost-bound https port</dd>

	<dt><code>--host-webserver-port &lt;string&gt;</code></dt>
	<dd>The web container&#39;s localhost-bound port</dd>

	<dt><code>--http-port &lt;string&gt;</code></dt>
	<dd>The router HTTP port for this project</dd>

	<dt><code>--https-port &lt;string&gt;</code></dt>
	<dd>The router HTTPS port for this project</dd>

	<dt><code>--image-defaults</code></dt>
	<dd>Sets the default web, db, and dba container images</dd>

	<dt><code>--mailhog-https-port &lt;string&gt;</code></dt>
	<dd>Router port to be used for mailhog access (https)</dd>

	<dt><code>--mailhog-port &lt;string&gt;</code></dt>
	<dd>Router port to be used for mailhog access</dd>

	<dt><code>--mutagen-enabled</code></dt>
	<dd>enable mutagen asynchronous update of project in web container</dd>

	<dt><code>--nfs-mount-enabled</code></dt>
	<dd>enable NFS mounting of project in container</dd>

	<dt><code>--ngrok-args &lt;string&gt;</code></dt>
	<dd>Provide extra args to ngrok in ddev share</dd>

	<dt><code>--no-project-mount</code></dt>
	<dd>Whether or not to skip mounting project code into the web container</dd>

	<dt><code>--nodejs-version &lt;string&gt;</code></dt>
	<dd>Specify the nodejs version to use if you don&#39;t want the default NodeJS 18</dd>

	<dt><code>--omit-containers &lt;string&gt;</code></dt>
	<dd>A comma-delimited list of container types that should not be started when the project is started</dd>

	<dt><code>--php-version &lt;string&gt;</code></dt>
	<dd>The version of PHP that will be enabled in the web container</dd>

	<dt><code>--phpmyadmin-https-port &lt;string&gt;</code></dt>
	<dd>Router port to be used for PHPMyAdmin (dba) container access (https)</dd>

	<dt><code>--phpmyadmin-port &lt;string&gt;</code></dt>
	<dd>Router port to be used for PHPMyAdmin (dba) container access</dd>

	<dt><code>--project-name &lt;string&gt;</code></dt>
	<dd>Provide the project name of project to configure (normally the same as the last part of directory name)</dd>

	<dt><code>--project-tld &lt;string&gt;</code></dt>
	<dd>set the top-level domain to be used for projects, defaults to ddev.site</dd>

	<dt><code>--project-type &lt;string&gt;</code></dt>
	<dd>Provide the project type (one of backdrop, craftcms, django4, drupal10, drupal6, drupal7, drupal8, drupal9, laravel, magento, magento2, php, python, shopware6, typo3, wordpress). This is autodetected and this flag is necessary only to override the detection.</dd>

	<dt><code>--router-http-port &lt;string&gt;</code></dt>
	<dd>The router HTTP port for this project</dd>

	<dt><code>--router-https-port &lt;string&gt;</code></dt>
	<dd>The router HTTPS port for this project</dd>

	<dt><code>--show-config-location</code></dt>
	<dd>Output the location of the config.yaml file if it exists, or error that it doesn&#39;t exist.</dd>

	<dt><code>--timezone &lt;string&gt;</code></dt>
	<dd>Specify timezone for containers and php, like Europe/London or America/Denver or GMT or UTC</dd>

	<dt><code>--upload-dir &lt;string&gt;</code></dt>
	<dd>Sets the project&#39;s upload directory, the destination directory of the import-files command.</dd>

	<dt><code>--use-dns-when-possible</code></dt>
	<dd>Use DNS for hostname resolution instead of /etc/hosts when possible</dd>

	<dt><code>--web-environment &lt;string&gt;</code></dt>
	<dd>Set the environment variables in the web container: --web-environment=&#34;TYPO3_CONTEXT=Development,SOMEENV=someval&#34;</dd>

	<dt><code>--web-environment-add &lt;string&gt;</code></dt>
	<dd>Append environment variables to the web container: --web-environment=&#34;TYPO3_CONTEXT=Development,SOMEENV=someval&#34;</dd>

	<dt><code>--web-image &lt;string&gt;</code></dt>
	<dd>Sets the web container image</dd>

	<dt><code>--web-image-default</code></dt>
	<dd>Sets the default web container image for this ddev version</dd>

	<dt><code>--web-working-dir &lt;string&gt;</code></dt>
	<dd>Overrides the default working directory for the web service</dd>

	<dt><code>--web-working-dir-default</code></dt>
	<dd>Unsets a web service working directory override</dd>

	<dt><code>--webimage-extra-packages &lt;string&gt;</code></dt>
	<dd>A comma-delimited list of Debian packages that should be added to web container when the project is started</dd>

	<dt><code>--webserver-type &lt;string&gt;</code></dt>
	<dd>Sets the project&#39;s desired webserver type: nginx-fpm/apache-fpm/nginx-gunicorn</dd>

	<dt><code>--working-dir-defaults</code></dt>
	<dd>Unsets all service working directory overrides</dd>

	<dt><code>--xdebug-enabled</code></dt>
	<dd>Whether or not XDebug is enabled in the web container</dd>
</dl>


### Options inherited from parent commands


<dl class="flags">
	<dt><code>-j</code>, <code>--json-output</code></dt>
	<dd>If true, user-oriented output will be in JSON format.</dd>
</dl>


{% endraw %}
### Examples

{% highlight bash %}{% raw %}
"ddev config" or "ddev config --docroot=web  --project-type=drupal8"{% endraw %}{% endhighlight %}

### See also

* [ddev](./ddev)
