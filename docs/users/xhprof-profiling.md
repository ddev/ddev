## Profiling with xhprof

DDEV-Local has built-in support for [xhprof](https://www.php.net/manual/en/book.xhprof.php). The official PECL xhprof extension does not support PHP5.6, but only PHP7.*and PHP8.*.

### Basic xhprof Usage

* Enable xhprof with `ddev xhprof enable` (or `ddev xhprof` or `ddev xhprof on`) and see status with `ddev xhprof status`
* Use a web browser or other technique to visit a page whose performance you want to study.
  
* At the bottom of the page there will be a link to "xhprof profiler output" so you can study the page.
* On the profiler output page you can drill down to the function that you want to study, or use the graphical "View Full Callgraph" link.
* Visit `<project_url>/xhprof/` (for example, `https://project.ddev.site/xhprof/`) to see and study all the runs you have captured.

For a tutorial on how to study the various xhprof reports, see the section "How to use XPROF UI" in [A Guide to Profiling with Xprof](https://inviqa.com/blog/profiling-xhprof). It takes a little time to get your eyes used to the reporting. (You do not need to do any of the installation described in that article, of course.)
