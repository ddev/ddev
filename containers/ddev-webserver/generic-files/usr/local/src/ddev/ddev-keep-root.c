// ddev-keep-root.c
//
// #ddev-generated
//
// Small LD_PRELOAD library that stops a process from dropping privileges. It
// turns setuid/setgid/initgroups/... into no-ops that return success, so the
// process keeps its current user (in practice, root).
//
// It only does this when the DDEV_KEEP_ROOT environment variable is set.
// Otherwise each call goes to the real libc function, so the library is safe
// to load into any process.
//
// DDEV needs this for Docker rootless. There the host user maps to container
// UID 0, so bind-mounted files (including .git) look root-owned and the web
// container runs as 0:0. Apache (Debian) refuses to run as root unless it was
// built with -DBIG_SECURITY_HOLE. Instead of rebuilding Apache, start.sh
// LD_PRELOADs this library for apache2 and keeps User/Group at a non-root user
// (so Apache's startup root check passes); the library then blocks the
// privilege drop, so Apache stays root and can read and write the root-owned
// files.

#define _GNU_SOURCE
#include <dlfcn.h>
#include <grp.h>
#include <stdlib.h>
#include <sys/types.h>
#include <unistd.h>

static int ddev_keep_root(void) {
    const char *v = getenv("DDEV_KEEP_ROOT");
    return v != NULL && (*v == '1' || *v == 't' || *v == 'T' || *v == 'y' || *v == 'Y');
}

// Each interceptor returns success and does nothing when DDEV_KEEP_ROOT is set,
// otherwise it calls the real libc function found via dlsym(RTLD_NEXT).
#define DDEV_NOOP_WHEN_KEEP_ROOT(name, args_decl, args_call) \
    int name args_decl { \
        static int (*real) args_decl = NULL; \
        if (ddev_keep_root()) \
            return 0; \
        if (real == NULL) \
            real = (int (*) args_decl) dlsym(RTLD_NEXT, #name); \
        return real args_call; \
    }

DDEV_NOOP_WHEN_KEEP_ROOT(setuid, (uid_t uid), (uid))
DDEV_NOOP_WHEN_KEEP_ROOT(seteuid, (uid_t uid), (uid))
DDEV_NOOP_WHEN_KEEP_ROOT(setgid, (gid_t gid), (gid))
DDEV_NOOP_WHEN_KEEP_ROOT(setegid, (gid_t gid), (gid))
DDEV_NOOP_WHEN_KEEP_ROOT(setreuid, (uid_t ruid, uid_t euid), (ruid, euid))
DDEV_NOOP_WHEN_KEEP_ROOT(setregid, (gid_t rgid, gid_t egid), (rgid, egid))
DDEV_NOOP_WHEN_KEEP_ROOT(setresuid, (uid_t ruid, uid_t euid, uid_t suid), (ruid, euid, suid))
DDEV_NOOP_WHEN_KEEP_ROOT(setresgid, (gid_t rgid, gid_t egid, gid_t sgid), (rgid, egid, sgid))
DDEV_NOOP_WHEN_KEEP_ROOT(setgroups, (size_t size, const gid_t *list), (size, list))
DDEV_NOOP_WHEN_KEEP_ROOT(initgroups, (const char *user, gid_t group), (user, group))
