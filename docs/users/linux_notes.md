<h1>Linux Notes</h1>

On Linux there are minor behavior differences due to the significant architecture differences of Docker for Linux.

* File mounts are direct on Linux, and Docker always mounts file systems as user uid 1000. As a result, if you are **not** using the default Linux user 1000 and you need to create files using the web server, you may need to make directories in your file system writable by all users. For example `find sites/default/files -type d | xargs chmod ugo+rwx`.
