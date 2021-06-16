## Profiling with xhprof

DDEV-Local has built-in support for [xhprof](https://www.php.net/manual/en/book.xhprof.php). The official PECL xhprof extension does not support PHP5.6, but only PHP7.*and PHP8.*.

### Basic xhprof Usage

* Enable xhprof with `ddev xhprof enable` (or `ddev xhprof` or `ddev xhprof on`) and see status with `ddev xhprof status`
* Use a web browser or other technique to visit a page whose performance you want to study.
  
* On some CMSs, there will be a link to "xhprof profiler output" so you can study the page. (For example, on Drupal 7 at least with the default theme, it's at the bottom of the page, on TYPO3 v10 it's in the upper left.) On other CMSs the output is suppressed and you'll need to visit `/xhprof/` (for example, `https://project.ddev.site/xhprof/`) and click the first listed link.
* On the profiler output page you can drill down to the function that you want to study, or use the graphical "View Full Callgraph" link. Click the column headers to sort by number of runs and inclusive or exclusive wall time, then drill down into the function you really want to study and do the same.
* Visit `<project_url>/xhprof/` (for example, `https://project.ddev.site/xhprof/`) to see and study all the runs you have captured. (These are erased on `ddev restart`.)

For a tutorial on how to study the various xhprof reports, see the section "How to use XHPROF UI" in [A Guide to Profiling with XHPROF](https://inviqa.com/blog/profiling-xhprof). It takes a little time to get your eyes used to the reporting. (You do not need to do any of the installation described in that article, of course.)
