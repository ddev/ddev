# xhprof Profiling

DDEV has built-in support for [xhprof](https://www.php.net/manual/en/book.xhprof.php). The official PECL xhprof extension only supports PHP >=7.x.

## Basic xhprof Usage

* Enable xhprof with [`ddev xhprof on`](../usage/commands.md#xhprof) (or `ddev xhprof` or `ddev xhprof enable`) and check its status with `ddev xhprof status`.
* `ddev xhprof on` will show you the URL you can use to see the xhprof analysis, `https://<projectname>.ddev.site/xhprof` shows recent runs. (It’s often useful to keep a tab or window open with this URL and refresh as needed.)
* Use a web browser or other technique to visit a page whose performance you want to study. To eliminate first-time cache-building issues, you may want to hit it twice.
* Visit one of the links provided by `ddev xhprof on` and study the results.
* On the profiler output page, you can drill down to the function that you want to study, or use the graphical “View Full Callgraph” link. Click the column headers to sort by number of runs and inclusive or exclusive wall time, then drill down into the function you want to study and do the same.
* The runs are erased on [`ddev restart`](../usage/commands.md#restart).
* If you’re using Apache with a custom `.ddev/apache/apache-site.conf`, you’ll need to make sure it includes `Alias "/xhprof" "/var/xhprof/xhprof_html"` from DDEV’s [default apache-site.conf](https://github.com/ddev/ddev/blob/main/pkg/ddevapp/webserver_config_assets/apache-site-php.conf).

For a tutorial on how to study the various xhprof reports, see the section “How to use XHPROF UI” in [A Guide to Profiling with XHPROF](https://inviqa.com/blog/profiling-xhprof). It takes a little time to get your eyes used to the reporting. (You don’t need to do any of the installation described in that article!)

## Advanced xhprof Configuration

You can change the contents of the `xhprof_prepend` function in `.ddev/xhprof/xhprof_prepend.php`.

For example, you may want to add a link to the profile run to the bottom of the profiled web page. The provided `xhprof_prepend.php` has comments and a sample function to do that, which works with Drupal 7. If you change it, remove the `#ddev-generated` line from the top, and check it in (`git add -f .ddev/xhprof/xhprof_prepend.php`).

Another example: you could exclude memory profiling so there are fewer columns to study. Change `xhprof_enable(XHPROF_FLAGS_MEMORY);` to `xhprof_enable();` in `.ddev/xhprof/xhprof_prepend.php` and remove the `#ddev-generated` at the top of the file. See the docs on [xhprof_enable()](https://www.php.net/manual/en/function.xhprof-enable.php).

## XHGui

XHGui is a graphical interface for XHProf profiling data that can store the results in the database.

XHGui is a available via the official [ddev-xhgui add-on](https://github.com/ddev/ddev-xhgui), `ddev add-on get ddev/ddev-xhgui`.

## Information Links

* [php.net xhprof](https://www.php.net/manual/en/book.xhprof.php)
* [Old facebook xhprof docs](https://web.archive.org/web/20110514095512/http://mirror.facebook.net/facebook/xhprof/doc.html)
* [rfay screencast on xhprof and blackfire.io](https://www.youtube.com/watch?v=6h2QMAtRjTA)
* [pecl.php.net docs](https://pecl.php.net/package/xhprof)
* [Upstream GitHub repository `lonngxhinH/xhprof`](https://github.com/longxinH/xhprof)
