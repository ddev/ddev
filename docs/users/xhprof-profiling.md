## Profiling with xhprof

DDEV-Local has built-in support for [xhprof](https://www.php.net/manual/en/book.xhprof.php). The official PECL xhprof extension does not support PHP5.6, but only PHP 7.\* and PHP 8.\*.

### Basic xhprof Usage

* Enable xhprof with `ddev xhprof on` (or `ddev xhprof` or `ddev xhprof enable`) and see status with `ddev xhprof status`
* `ddev xhprof on` will show you the URL you can use to see the xhprof analysis,  `https://<projectname>.ddev.site/xhprof` shows recent runs. It's often useful to just have a tab or window open with this URL and refresh it as needed.
* Use a web browser or other technique to visit a page whose performance you want to study. To eliminate first-time cache-building issues, you may want to hit it twice.
* Visit one of the links provided by `ddev xhprof on` and study the results.
* On the profiler output page you can drill down to the function that you want to study, or use the graphical "View Full Callgraph" link. Click the column headers to sort by number of runs and inclusive or exclusive wall time, then drill down into the function you really want to study and do the same.
* The runs are erased on `ddev restart`.
* If you are using webserver_type apache-fpm and you have a custom .ddev/apache/apache-site.conf, you'll need to make sure it has the `Alias "/xhprof" "/var/xhprof/xhprof_html"` in it that the [provided apache-site.conf](https://github.com/drud/ddev/blob/master/pkg/ddevapp/webserver_config_assets/apache-site-php.conf) has.

For a tutorial on how to study the various xhprof reports, see the section "How to use XHPROF UI" in [A Guide to Profiling with XHPROF](https://inviqa.com/blog/profiling-xhprof). It takes a little time to get your eyes used to the reporting. (You do not need to do any of the installation described in that article, of course.)

### Advanced xhprof configuration

You can change the contents of the xhprof_prepend function - it's in `.ddev/xhprof/xhprof_prepend.php`.

For example, you may want to add a link to the profile run to the bottom of the profiled web page; the provided xhprof_prepend.php has comments and a sample function to do that, which works with Drupal 7. If you change it, remove the `#ddev-generated` line from the top, and check it in (`git add -f .ddev/xhprof/xhprof_prepend.php`)
