# Xdebug Profiling

Although DDEV has more sophisticated profiling capabilities with [xhprof](xhprof-profiling.md) and [blackfire.io](blackfire-profiling.md) it also has  built-in support for [xdebug profiling](https://xdebug.org/).

## Basic usage

* Create the directory `.ddev/xdebug`, which is where the output files will be dumped.
* Switch XDebug to profiling mode by adding this in `.ddev/php/xdebug.ini`

  ```ini
  xdebug.mode=profile
  xdebug.start_with_request=yes
  xdebug.output_dir=/var/www/html/.ddev/xdebug
  xdebug.profiler_output_name=trace.%c%p%r%u.out
  ```

* Enable xdebug with `ddev xdebug on`
* Do a HTTP request to the DDEV project and the profile will be located in `.ddev/xdebug` directory.
* Analyze it with any call graph viewer, for example [kcachegrind](https://kcachegrind.github.io/html/Home.html).
* When you're done, execute `ddev xdebug off` to avoid generating unneeded profile files.

## Information Links

* [xdebug profiling docs](https://xdebug.org/docs/profiler)
* [kcachegrind](https://kcachegrind.github.io/html/Home.html)
