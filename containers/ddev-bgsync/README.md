# bg-sync (deprecated)

Forked from https://github.com/cweagans/docker-bg-sync at
https://github.com/cweagans/docker-bg-sync/commit/4e39642ab414f2ba0b55eb3933203115e6ed2c0b

Thanks!

====================

This container continuously syncs files between two directories. This is useful
for avoiding the filesystem slowness on Docker Desktop for Mac, for instance. It's also
generally useful for any other time where you have a slow filesystem as a source
of files that need to be read inside of a container.


## Environment variables

This container uses values from a handful of environment variables. These are
documented below.

  * **`SYNC_SOURCE`** (default: `/source`): The path inside the container which
    will be used as the source of the file sync. Most of the time, you probably
    shouldn't change the value of this variable. Instead, just bind-mount your
    files into the container at `/source` and call it a day.
  * **`SYNC_DESTINATION`** (default: `/destination`): When files are changed in
    `SYNC_SOURCE`, they will be copied over to the equivalent paths in `SYNC_DESTINATION`.
    If you are using bg-sync to avoid filesystem slowness, you should set this
    path to whatever path the volume is at in your application container. In the
    example above, for instance, this would be `/var/www/myapp`.
  * **`SYNC_PREFER`** (default: `/source`): Control the conflict strategy to apply
    when there are conflits. By default the contents from the source folder are
    left unchanged but there is also the "newer" option to pick up the most
    recent files.
  * **`SYNC_VERBOSE`** (default: "0"): Set this variable to "1" to get more log
    output from Unison.
  * **`SYNC_MAX_INOTIFY_WATCHES`** (default: ''): If set, the sync script will
    attempt to increase the value of `fs.inotify.max_user_watches`. **IMPORTANT**:
    This requires that you run this container as a priviliged container. Otherwise,
    the inotify limit increase *will not work*. As always, when running a third
    party container as a priviliged container, look through the source thoroughly
    first to make sure it won't do anything nefarious. `sync.sh` should be pretty
    understandable. Go on - read it. I'll wait.
  * **`SYNC_EXTRA_UNISON_PROFILE_OPTS`** (default: ''): The value of this variable
    will be appended to the end of the Unison profile that's automatically generated
    when this container is started. Ensure that the syntax is valid. If you have
    more than one option that you want to add, simply make this a multiline string.
    **IMPORTANT**: The *ability* to add extra lines to your Unison profile is
    supported by the bg-sync project. The *results* of what might happen because
    of this configuration is *not*. Use this option at your own risk.
  * **`SYNC_NODELETE_SOURCE`** (default: '1'): Set this variable to "0" to allow
    Unison to sync deletions to the source directory. This could cause unpredictable
    behaviour with your source files.



