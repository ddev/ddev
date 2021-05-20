## .ddev/config.yaml options

Each project has a (hidden) directory named .ddev, and
the .ddev/config.yaml is the primary configuration for the project.

* You can override the config.yaml with extra files named "config.\*.yaml". For example, many teams use `config.local.yaml` for configuration that is specific to one environment, and that is not intended to be checked into the team's default config.yaml.
* Most configuration options take effect on `ddev start`
* Nearly all configuration options have equivalent `ddev config` flags, so you can use ddev config instead of editing the config.yaml. See `ddev help config` to see all the flags.

| Item  | Description   | Notes  |
|---|---|---|
| name  | Project name   | Must be unique on the host (no two projects can have the same name). It's best if this is the same as the directory name.   |
| type | Project type | php/drupal6/drupal7/drupal8/backdrop/typo3wordpress. Project type "php" does not try to do any CMS configuration or settings file management, and can work with any project|
| docroot | Relative path of the docroot (where index.php or index.html is) from the project root| |
| php_version | 5.6/7.0/7.1/7.2/7.3/7.4/8.0 | It is only possible to provide the major version (like "7.3"), not a minor version like "7.3.2", and it is only possible to use the provided php versions. |
| webimage | docker image to use for webserver | It is unusual to change the default and is not recommended, but the webimage can be overridden with a correctly crafted image, probably derived from drud/ddev-webserver |
| dbimage | docker image to use for db server | It is unusual to change the default and is not recommended, but the dbimage can be overridden with a correctly crafted image, probably derived from drud/ddev-dbserver |
| dbaimage | docker image to use for dba server (phpMyAdmin server) | It is unusual to change the default and is not recommended, but the dbimage can be overridden with a correctly crafted image, probably derived from drud/phpmyadmin |
| mariadb_version | Version of MariaDB to be used |  Defaults to 10.3, but 5.5, 10.0, 10.1, 10.2, 10.3, 10.4, 10.5 are available. Cannot be used with mysql_version. See [Database Server Types](database_types.md) for details and caveats. |
| mysql_version | Version of Oracle MySQL to be used |  Defaults to empty (using MariaDB). 5.5, 5.6, 5.7, and 8.0 are available. Conflicts with mariadb_version. See [Database Server Types](database_types.md) for details and caveats. |
| router_http_port | Port used by the router for http |  Defaults to port 80. This can be changed if there is a conflict on the host over port 80 |
| router_https_port | Port used by the router for https |Defaults to 443, usually only changed if there is a conflicting process using port 443 |
| xdebug_enabled | "true" enables xdebug | Most people use `ddev xdebug` and `ddev xdebug off` instead of configuring this, because xdebug has a significant performance impact. |
| webserver_type | nginx-fpm or apache-fpm | The default is nginx-fpm, and it works best for many projects.|
| timezone | timezone to use in container and in PHP configuration | It can be set to any valid timezone, see [timezone list](https://en.wikipedia.org/wiki/List_of_tz_database_time_zones). For example "Europe/Dublin" or "MST7MDT". The default is UTC. |
| composer_version | version of composer to use in web container and `ddev composer` | It can be left blank to get the default composer v1, or set to "1", for latest composer v1, "2" for latest composer v2, or to an explicit composer version. (Composer v2 is currently the default for ddev.) |
| additional_hostnames | array of extra hostnames | `additional_hostnames: ["somename", "someothername", "*.thirdname"]` would provide http and https URLs for "somename.ddev.site" and "someothername.ddev.site", as well as "one.thirdname.ddev.site" and "two.thirdname.ddev.site". Note that the wildcard/asterisk setting only works if you're using DNS to resolve hostnames (which is the default) and you're connected to the internet. |
| additional_fqdns | extra fully-qualified domain names | `additional_fqdns: ["example.com", "sub1.example.com"]` would provide http and https URLs for "example.com" and "sub1.example.com". Please take care with this because it can cause great confusion and adds extraneous items to your /etc/hosts file. |
| upload_dir | Relative path to upload directory used by `ddev import-files` | |
| working_dir | explicitly specify the working directory used by `ddev exec` and `ddev ssh` | `working_dir: { web:  "/var/www", db: "/etc" }` would set the working directories for the web and db containers. |
| omit_containers | Allows the project to not load specified containers | For example, `omit_containers: [db, dba, ddev-ssh-agent]`. Currently only these containers are supported. Some containers can also be omitted globally in the ~/.ddev/global_config.yaml and the result is additive; all containers named in both places will be omitted. Note that if you omit the "db" container, several standard features of ddev that access the database container will be unusable. |
| web_environment | Inject environment variables into web container | For example, `web_environment: ["SOMEENV=someval", "SOMEOTHERENV=someotherval"]`.  |
| nfs_mount_enabled | Allows using NFS to mount the project into the container for performance reasons | See [nfs_mount_enabled documentation](../performance.md). This requires configuration on the host before it can be used. Note that project-level configuration of nfs_mount_enabled is unusual, and that if it's true in the global config, that overrides the project-specific nfs_mount_enabled|
| fail_on_hook_fail | Decide whether `ddev start` should be interrupted by a failing hook |
| host_https_port | Specify a specific and persistent https port for direct binding to the localhost interface | This is not commonly used, but a specific port can be provided here and the https URL will always remain the same. For example, if you put "59001", the project will always use "<https://127.0.0.1:59001".> for the localhost URL. (Note that the named URL is more commonly used and for most purposes is better.) If this is not set the port will change from `ddev start` to `ddev start` |
| host_webserver_port | Specify a specific and persistent http port for direct binding to the localhost interface | This is not commonly used, but a specific port can be provided here and the https URL will always remain the same. For example, if you put "59000", the project will always use "<http://127.0.0.1:59000".> for the localhost URL. (Note that the named URL is more commonly used and for most purposes is better.) If this is not set the port will change from `ddev start` to `ddev start` |
| host_db_port | localhost binding port for database server | If specified here, the database port will remain consistent. This is useful for configuration of host-side database browsers. Note, though, that `ddev sequelpro` and `ddev mysql` do all this automatically, as does the sample command `ddev mysqlworkbench`. |
| phpmyadmin_port | port used for phpMyAdmin URL | This is sometimes changed from the default 8036 when a port conflict is discovered |
| phpmyadmin_https_port | port used for phpMyAdmin URL (https) | This is sometimes changed from the default 8037 when a port conflict is discovered |
| mailhog_port | port used in MailHog URL | this can be changed from the default 8025 in case of port conflicts |
| mailhog_https_port | port used in MailHog URL | this can be changed from the default 8026 in case of port conflicts |
| webimage_extra_packages| Add extra Debian packages to the ddev-webserver. | For example, `webimage_extra_packages: [php-yaml, php-bcmath]` will add those two packages |
| dbimage_extra_packages| Add extra Debian packages to the ddev-dbserver. | For example, `dbimage_extra_packages: ["less"]` will add those that package. |
| use_dns_when_possible | defaults to true (using DNS instead of editing /etc/hosts) | If set to false, ddev will always update the /etc/hosts file with the project hostname instead of using DNS for name resolution |
| project_tld | defaults to "ddev.site" so project urls become "someproject.ddev.site" | This can be changed to anything that works for you; to keep things the way they were before ddev v1.9, use "ddev.local" |
| ngrok_args | Extra flags for ngrok when using the `ddev share` command | For example, `--subdomain mysite --auth user:pass`. See [ngrok docs on http flags](https://ngrok.com/docs#http) |
| disable_settings_management | defaults to false | If true, ddev will not create or update CMS-specific settings files |  |
| hooks | | See [Extending Commands](../extending-commands.md) for more information. |
| no_project_mount | Skip mounting the project into the web container | If true, the project will not be mounted by ddev into the web container. This is to enable experimentation with alternate file mounting strategies. Advanced users only! |

## global_config.yaml Options

The $HOME/.ddev/global_config.yaml has a few key global config options.

| Item  | Description   | Notes  |
|---|---|---|
| nfs_mount_enabled | Enables NFS mounting globally for all projects | Only a "true" value has any effect. If true, NFS will be used on all projects, regardless of any settings in the individual projects. |
| fail_on_hook_fail | Enables `ddev start` interruption globally for all projects when a hook fails | Decide whether `ddev start` should be interrupted by a failing hook |
| omit_containers | Allows the project to not load specified containers | For example, `omit_containers: [ "dba", "ddev-ssh-agent"]`. Currently only these containers are supported. Note that you cannot omit the "db" container in the global configuration, but you can in the per-project .ddev/config.yaml. |
| web_environment | Inject environment variables into web container | For example, `web_environment: ["SOMEENV=someval", "SOMEOTHERENV=someotherval"]`.  |
| instrumentation_opt_in | Opt in or out of instrumentation reporting | If true, anonymous usage information is sent to ddev via [segment](https://segment.com) |
| router_bind_all_interfaces | Bind on all network interfaces | If true, ddev-router will bind on all network interfaces instead of just localhost, exposing ddev projects to your local network. If you set this to true, you may consider `omit_containers: ["dba"]` so that the phpMyAdmin port is not available.  |
| disable_http2 | Disable http/2 listen in ddev-router | If true, nginx will not listen for http/2, but just use http/1.1 ssl. There are some browsers which don't work well with http/2. |
| internet_detection_timeout_ms | Internet detection timeout | ddev must detect whether the internet is working to determine whether to add hostnames to /etc/hosts. It uses a DNS query and times it out by default at 750ms. In rare cases you may need to increase this value if you have slow but working internet. See [FAQ](../faq.md) and [issue link](https://github.com/drud/ddev/issues/2409#issuecomment-662448025).|
| use_hardened_images | If true, use more secure 'hardened' images for an actual internet deployment | The most important thing about hardened images is that sudo is not included in the web container, and the container is run without elevated privileges. This is generally used with experimental casual hosting feature. |
| use_letsencrypt | If true, enables experimental Let's Encrypt integration | 'ddev global --use-letsencrypt' or `ddev global --use-letsencrypt=false'. This requires letsencrypt_email to be set, and can't work if the system is not available on the internet. Used with experimental casual hosting feature. |
| letsencrypt_email | Email associated with Let's Encrypt for experimental Let's Encrypt feature | Set with`ddev global --letsencrypt-email=me@example.com'. Used with experimental casual hosting feature.|
| developer_mode | Set developer mode | If true, developer_mode is set. This is not currently used. |
