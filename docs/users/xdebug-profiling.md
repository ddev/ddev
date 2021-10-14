## Profiling with xdebug

DDEV-Local has built-in support for [xdebug](https://xdebug.org/).

### Basic usage

* Switch XDebug to profiling mode:

  ```
  xdebug.mode=profile
  xdebug.start_with_request=yes
  xdebug.profiler_output_name=trace.%c%p%r%u.out
  ```

* Enable xdebug with `ddev xdebug on`
* Do a HTTP request towards DDEV and the profile will be located inside the `/tmp` directory of the webserver container.
* To move it to the host machine, probably you can copy the needed profile files to the web docroot temporarily: `ddev ssh && mv /tmp/trace* .`
* Analyze it with any call graph viewer, for example [kcachegrind](https://kcachegrind.github.io/html/Home.html).
* After you're ready, execute `ddev xdebug off` to avoid generating unneeded profile files.

### Information Links

* [xdebug](https://xdebug.org/)
* [kcachegrind](https://kcachegrind.github.io/html/Home.html)
