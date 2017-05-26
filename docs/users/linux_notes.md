<h1>Linux Notes</h1>

On Linux there are minor behavior differences due to the significant architecture differences of Docker for Linux.

* File mounts are direct on Linux, and Docker always mounts filesystems as user uid 1000. As a result, if you are **not** using the default Linux user 1000 and you need to create files using the webserver, you may need to make directories in your filesystem writeable by all users. For example `find sites/default/files -type d | xargs chmod ugo+rwx`.
