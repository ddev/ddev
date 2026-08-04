# HANDOFF: Puppeteer Chrome deps blocked on this environment

Temporary note — not part of the ddev codebase, delete once resolved. Context for
picking this back up in a different session/environment.

## What I was doing

Trying to run `perf/metrics/03-drupal-install/install-timer.js` standalone (see
`perf/README.md`), one metric out of the `perf/` benchmark harness added in
PR #8622 / #8608-era work. `npm ci` succeeded and Puppeteer downloaded its bundled
Chrome + chrome-headless-shell into `~/.cache/puppeteer/`, but launching either
binary fails.

## Root cause

Not a missing Chrome *install* — it's missing **OS-level shared libraries** the
Chrome binary is dynamically linked against. Confirmed with:

```bash
ldd ~/.cache/puppeteer/chrome/linux-*/chrome-linux64/chrome | grep "not found"
```

16 missing libs on this container (Ubuntu 24.04 "noble"):
`libatk-1.0.so.0`, `libatk-bridge-2.0.so.0`, `libcups.so.2`, `libxcb.so.1`,
`libxkbcommon.so.0`, `libasound.so.2`, `libgbm.so.1`, `libX11.so.6`, `libXext.so.6`,
`libcairo.so.2`, `libpango-1.0.so.0`, `libXcomposite.so.1`, `libXdamage.so.1`,
`libXfixes.so.3`, `libXrandr.so.2`, `libatspi.so.0`.

`chrome-headless-shell` (also downloaded) is missing 13 of the same 16 — switching
to it does **not** avoid this.

## Fix (Ubuntu 24.04 / this container)

```bash
sudo apt-get update
sudo apt-get install -y \
  libatk1.0-0t64 libatk-bridge2.0-0t64 libcups2t64 libasound2t64 \
  libxkbcommon0 libgbm1 libx11-6 libxext6 libxcb1 libcairo2 \
  libpango-1.0-0 libxcomposite1 libxdamage1 libxfixes3 libxrandr2 \
  libatspi2.0-0
```

Package names carry a `t64` suffix on Ubuntu 24.04 specifically (the 64-bit
`time_t` ABI transition renamed several of them: `libatk1.0-0` → `libatk1.0-0t64`,
etc.). Older/other Ubuntu or Debian releases use the un-suffixed names.

**Status as of writing**: this `apt-get install` is running now but very slow —
`apt-get update` alone hung 2+ minutes against `http://archive.ubuntu.com/ubuntu/`
(configured in `/etc/apt/sources.list.d/ubuntu.sources`), even though a direct
`curl -sI https://archive.ubuntu.com` responded instantly. So the slowness is in
the apt fetch path specifically (proxy/NAT/mirror), not a general network outage.
If picking this up fresh, try pointing at a regional/CDN mirror
(e.g. `mirror://mirrors.ubuntu.com/mirrors.txt`) if it's still slow.

## Open question: is docker-in-docker relevant here?

This coder.ddev.com workspace is docker-in-docker. Raised: should the apt install
happen on the **host** instead of in this container?

My read: **no** — install it in whichever container/VM actually executes
`node install-timer.js` and owns `~/.cache/puppeteer/`, which is *this* container
(a nested Ubuntu 24.04 userland with its own libc/library set, independent of
whatever OS the outer Docker host runs). Installing these libs on the host
wouldn't make them visible inside this container's mount namespace, since it's
not a bind-mounted `/usr/lib` — it's this container's own root filesystem.
The `ddev` project's own containers (webserver/dbserver) are irrelevant here too;
Puppeteer runs on the *host side* of a DDEV project (this devcontainer), driving a
browser against the already-running `ddev-webserver` container over the network,
not inside any ddev container.

The real fix for *recurring* pain, if this container gets rebuilt/recreated
often: bake these `apt-get install` packages into whatever image/Dockerfile
defines this coder devcontainer, instead of installing them ad hoc every session.
Ask whoever owns the coder.ddev.com workspace image definition.

## Possible fast-follow: switch from Puppeteer to Playwright

Playwright is the more actively-developed, better-resourced of the two now
(auto-waiting, multi-browser, richer debugging/tracing), while Puppeteer is
narrower in scope and Chrome-team-maintained. Concretely relevant to the mess
above: Playwright ships `npx playwright install-deps`, which detects the distro
from `/etc/os-release` and installs exactly the OS shared libs a browser needs —
it would have avoided this whole detour entirely, on any distro, not just this
container.

Not done as part of unblocking the local run today, since `install-timer.js` is
small and already covered by the PR's documented manual testing, and swapping
libraries mid-PR means rewriting and re-verifying it, not just a dependency bump.
Worth considering as a separate fast-follow PR once #8622 lands.

## To resume

1. Check whether the running `apt-get install` (see `ps aux | grep apt-get`) finished.
2. If it did: `~/.cache/puppeteer/chrome/linux-*/chrome-linux64/chrome --version`
   should print a version instead of a shared-library error.
3. Then re-run the standalone metric per `perf/README.md`:
   ```bash
   export DDEV_PERF_PROJECT_DIR=~/workspace/d11
   export DDEV_PERF_SITE_URL=https://d11.ddev.site/
   bash perf/lib/reset-drupal.sh
   cd perf/metrics/03-drupal-install
   DDEV_PERF_SITE_URL="$DDEV_PERF_SITE_URL" node install-timer.js
   ```
