# Handoff: using an existing SSH agent from DDEV containers

Context for whoever picks up
[#3878](https://github.com/ddev/ddev/issues/3878): users whose keys live in an
agent rather than in `~/.ssh` (1Password, YubiKey, forwarded agents, Coder
workspaces) cannot use `ddev auth ssh`. This file records what was learned and
tested on 2026-09-23 and 2026-09-24. The work is on branch
`20260923_rfay_ssh_agent_upstream`; the user guide is "Using an Existing SSH
Agent" in `docs/content/users/usage/cli.md`.

## Constraints

- Do not break current practice: `ddev auth ssh` with key files stays the
  default and behaves exactly as today.
- DDEV does not change host configuration (editing `/etc/hosts` is the only
  exception). DDEV may *use* an agent socket the user already has, but must
  not install, launch, or configure one on the host. The `ddev 1password`
  command in the add-on below writes a macOS LaunchAgent, so that approach
  cannot move into core.

## How ddev-ssh-agent works by default

- `ddev-ssh-agent` runs `ssh-agent` in the container, listening on
  `/tmp/.ssh-agent/socket` inside the `ddev-ssh-agent_socket_dir` volume.
- Every web container mounts that volume at `/home/.ssh-agent` and sets
  `SSH_AUTH_SOCK=/home/.ssh-agent/socket`
  ([app_compose_template.yaml:161](pkg/ddevapp/app_compose_template.yaml#L161),
  [:256](pkg/ddevapp/app_compose_template.yaml#L256)).
- `ddev auth ssh` bind-mounts private key *files* into a short-lived container
  and runs `ssh-add` against that socket
  ([auth-ssh.go](cmd/ddev/cmd/auth-ssh.go)). A key that exists only in an agent
  cannot be added this way, and the agent protocol does not allow exporting
  private keys, so the only general fix is to proxy the agent socket.
- `CreateSSHAuthComposeFile` in [ssh_auth.go](pkg/ddevapp/ssh_auth.go)
  merges user overrides from `~/.ddev/ssh-auth-compose.*.yaml`, which made
  the global, no-code prototype below possible.
- `EnsureSSHAgentContainer` recreates the container only when the image or
  the upstream label changes, so an edited override needs
  `docker rm -f ddev-ssh-agent` or `ddev poweroff`.

## SSH agent basics

An SSH agent is a long-running process that holds private keys and signs
authentication challenges on request. Clients never see the keys; they talk to
the agent over a Unix socket (a named pipe on Windows) using the
[agent protocol](https://datatracker.ietf.org/doc/draft-ietf-sshm-ssh-agent/).
Because the protocol has no "export key" operation, an agent-only key can be
used from a container only by giving the container a path to the socket.

How clients find an agent:

- `$SSH_AUTH_SOCK` names the socket. `ssh`, `ssh-add`, `git`, and most
  libraries read it.
- `IdentityAgent` in `~/.ssh/config` overrides `$SSH_AUTH_SOCK`, but only for
  `ssh` itself ([ssh_config(5)](https://man.openbsd.org/ssh_config#IdentityAgent)).
  `ssh-add` ignores it, so `ssh-add -l` can report an empty agent while `ssh`
  authenticates through 1Password.
- `ssh -A` (or `ForwardAgent yes`) forwards the local agent to a remote host,
  where sshd creates a new socket and sets `$SSH_AUTH_SOCK` for that login
  only. OpenSSH 9 and earlier use `/tmp/ssh-*/agent.*`; OpenSSH 10 (Ubuntu
  26.04) uses `~/.ssh/agent/s.*.sshd.*`.

Agents users are likely to have:

- **macOS built-in agent.** launchd starts `ssh-agent` on demand and sets
  `$SSH_AUTH_SOCK` to `/var/run/com.apple.launchd.*/Listeners` for every GUI
  and terminal process. Keys are forgotten at logout.
  `ssh-add --apple-use-keychain <key>` stores the passphrase in the Keychain,
  and `ssh-add --apple-load-keychain` reloads those keys, or `UseKeychain yes`
  plus `AddKeysToAgent yes` in `~/.ssh/config` loads them on first use
  (`man ssh-add` on macOS). KeePassXC also loads keys into this agent.
- **Agents with their own socket**, selected with `IdentityAgent` or by
  exporting `SSH_AUTH_SOCK`:
  [1Password](https://developer.1password.com/docs/ssh/agent/)
  (`~/Library/Group Containers/2BUA8C4S2C.com.1password/t/agent.sock` on macOS,
  `~/.1password/agent.sock` on Linux),
  [Secretive](https://github.com/maxgoedjen/secretive) (Secure Enclave keys),
  Bitwarden, Strongbox, and
  [gpg-agent](https://www.gnupg.org/documentation/manuals/gnupg/Agent-Options.html)
  with `enable-ssh-support`, the usual YubiKey route. These generally refuse
  `ssh-add` of new keys.
- **Linux desktop agents.** GNOME Keyring or gcr-ssh-agent on Ubuntu and
  Fedora, KWallet with `ssh-agent` on KDE, or a user-started `ssh-agent`
  ([Arch wiki](https://wiki.archlinux.org/title/SSH_keys#SSH_agents)).
- **Windows.** The OpenSSH agent service and 1Password use the named pipe
  `\\.\pipe\openssh-ssh-agent`; Pageant has its own protocol.

Everyday commands
([ssh-agent(1)](https://man.openbsd.org/ssh-agent),
[ssh-add(1)](https://man.openbsd.org/ssh-add)):

```bash
ssh-add -l                       # list keys in the agent $SSH_AUTH_SOCK names
ssh-add                          # add ~/.ssh/id_rsa, id_ecdsa, id_ed25519, ...
ssh-add ~/.ssh/other_key         # add one key
ssh-add -d ~/.ssh/other_key      # remove one key; -D removes all
eval "$(ssh-agent -s)"           # start a private agent for this shell
ssh-agent -a /path/agent.sock    # start one on a fixed socket
SSH_AUTH_SOCK=~/.1password/agent.sock ssh-add -l   # query a specific agent
ssh -T git@github.com            # test authentication
```

## What Docker providers forward on macOS

Docker Desktop, OrbStack, and Colima each expose a host agent inside their VM
at `/run/host-services/ssh-auth.sock`, following Docker Desktop's convention
([Docker docs](https://docs.docker.com/desktop/features/networking/networking-how-tos/),
[OrbStack docs](https://docs.orbstack.dev/docker/#ssh-agent-forwarding),
[Colima FAQ](https://github.com/abiosoft/colima/blob/main/docs/FAQ.md)).
Which agent is behind it differs, and was tested here:

| Provider | Agent forwarded | Socket in the VM |
| --- | --- | --- |
| OrbStack | `IdentityAgent` from `~/.ssh/config` if set, else launchd `$SSH_AUTH_SOCK`; read at OrbStack startup | 0666 |
| Docker Desktop | launchd `$SSH_AUTH_SOCK` only; `IdentityAgent` ignored | `root:root` 0660 |
| Colima (`colima start --ssh-agent`) | launchd `$SSH_AUTH_SOCK` only, through Lima's ssh forwarding | symlink to `/tmp/ssh-*/agent.*` |
| Rancher Desktop | none: no `/run/host-services`, no `SSH_AUTH_SOCK` in `rdctl shell` | — |
| Lima with [`ssh.forwardAgent: true`](https://lima-vm.io/docs/config/) | launchd `$SSH_AUTH_SOCK`, through the hostagent's persistent SSH connection | `/tmp/ssh-*/agent.*`, random per VM boot |
| Podman (rootless, SELinux enforcing) | only per `podman machine ssh` session, which uses `~/.ssh/config` and so reaches 1Password; each session gets a new `~/.ssh/agent/s.*` socket that disappears when it ends | — |

Consequences:

- A shell `export SSH_AUTH_SOCK=...` never reaches these GUI or background
  providers. On Docker Desktop and Colima, 1Password users must repoint the
  launchd socket itself, which is what 1Password's
  ["Configure SSH_AUTH_SOCK globally for every client"](https://www.1password.dev/ssh/agent/compatibility/)
  LaunchAgent does (and what the `ddev 1password` add-on command installs).
  DDEV cannot do this, per the constraints above.
- OrbStack's docs say 1Password's agent will not work, but here it did,
  because OrbStack now follows `IdentityAgent`.

### Why not rely on OrbStack alone

OrbStack does the most: it finds the agent the user's `ssh` would use and
forwards it with open permissions. That still leaves gaps DDEV has to cover:

- Only OrbStack behaves this way. Docker Desktop, Colima, and Lima ignore
  `IdentityAgent`, Linux has no host-services socket at all, and a container
  never gets any of them unless something mounts the socket and sets
  `SSH_AUTH_SOCK`.
- DDEV's containers use the `ddev-ssh-agent` socket volume, not
  `/run/host-services`, so without the relay OrbStack's forwarding is unused.
  The ddev-1password add-on mounts it per project and only into `web`.
- OrbStack reads `~/.ssh/config` only at startup, so switching agents needs an
  OrbStack restart, which users will not expect.
- `ddev auth ssh` with key files, the default, must keep working for users
  with no agent.

## Existing per-project workaround

[anotherjames/ddev-1password](https://github.com/anotherjames/ddev-1password)
bind-mounts `/run/host-services/ssh-auth.sock` into `web`, sets
`SSH_AUTH_SOCK`, and adds a post-start hook
`sudo chmod o+rw /run/host-services/ssh-auth.sock` (the socket is root-owned,
mode 660, on Docker Desktop). It must be installed in every project, only
`web` gets the agent, and it is macOS-oriented. Reported working on Docker
Desktop, OrbStack, and Colima started with `--ssh-agent`. A user asked in the
issue for a way to do this from `~/.ddev` instead.

## Tested: relay mode via a global override (WSL2 + 1Password)

Environment: WSL2 Ubuntu 26.04 on Windows arm64, native docker-ce 29.8,
1Password's agent on the Windows side (`\\.\pipe\openssh-ssh-agent`). Before
this, the user reached it only through Windows `ssh.exe`, which containers
cannot use.

1. The user, not DDEV, provides a Linux socket. Here that was
   [mame/wsl2-ssh-agent](https://github.com/mame/wsl2-ssh-agent), which relays
   a Unix socket to the Windows pipe through `powershell.exe` and needs no
   sudo, socat, or Windows binaries:

   ```bash
   go install github.com/mame/wsl2-ssh-agent@latest
   wsl2-ssh-agent -foreground -socket ~/tmp/wsl-agent/agent.sock &
   SSH_AUTH_SOCK=~/tmp/wsl-agent/agent.sock ssh-add -l   # lists 1Password keys
   ```

2. `~/.ddev/ssh-auth-compose.host-agent.yaml` turns `ddev-ssh-agent` into a
   relay:

   ```yaml
   services:
     ddev-ssh-agent:
       volumes:
         - ${HOME}/tmp/wsl-agent:/upstream
       command: ["socat", "UNIX-LISTEN:/tmp/.ssh-agent/socket,perm=0666,fork,unlink-early", "UNIX-CONNECT:/upstream/agent.sock"]
       healthcheck:
         test: "killall -0 socat"
   ```

3. Results in a fresh `php` project:
   - `ddev exec ssh-add -l` listed both 1Password keys, and
     `ddev exec ssh -T git@github.com` authenticated.
   - No project files, no chmod hook, and no root were needed; on native
     Linux the socket is owned by the same UID the container runs as.
   - Stopping the host relay made the container's agent fail; restarting it
     recovered without any DDEV restart, because the override mounts the
     socket's *directory*, not the socket file.

Gotchas found:

- `command` must be a YAML list. [entry.sh:79](containers/ddev-ssh-agent/files/entry.sh#L79)
  runs `exec $@` unquoted, so a `bash -c '...'` string is word-split and the
  container exits immediately.
- The stock healthcheck requires both `socat` and `ssh-agent`
  ([healthcheck.sh:18](containers/ddev-ssh-agent/files/healthcheck.sh#L18)),
  so relay mode needs its own.
- Mount the parent directory, not the socket file: a file bind mount pins the
  old inode and goes stale when the upstream socket is recreated.
- On Docker Desktop, the socket is root:root 660, so the relay needs
  `user: "0:0"`. Confirmed later on macOS; see the results below.

## Coder workspaces (coder.ddev.com)

- There is normally no SSH agent. git over SSH works through Coder's GitSSH
  wrapper: the [ddev/coder-ddev](https://github.com/ddev/coder-ddev) template
  startup script (`freeform/template.tf`) sets
  `GIT_SSH_COMMAND="<coder binary> gitssh"` and `core.sshCommand`.
  `coder gitssh` fetches the user's Coder-managed key from the Coder server
  using the workspace agent's credentials and runs `ssh -i` with a temporary
  file.
- Consequences: only git uses the key, plain `ssh` does not, and DDEV
  containers have neither the `coder` binary nor the credentials, so git
  inside `web` (for example Composer with private repositories) fails.
- An agent exists only in a session opened with forwarding, such as
  `coder ssh -A`, and only for that session. VS Code for Web and the web
  terminal have none.
- Checked in two workspaces (`freeform` and `drupal-contrib` templates):
  `GIT_SSH_COMMAND=/tmp/coder.*/coder gitssh --` is set, `SSH_AUTH_SOCK` is
  unset, and Docker runs inside the workspace, so workspace paths are Docker
  host paths as on native Linux. A workspace shell has `CODER_AGENT_URL` and
  `CODER_AGENT_TOKEN`, which fetch the Coder-managed key directly:

  ```bash
  curl -fsS -H "Coder-Session-Token: $CODER_AGENT_TOKEN" \
    "${CODER_AGENT_URL%/}/api/v2/workspaceagents/me/gitsshkey" \
    | jq -r .private_key | ssh-add -
  ```

- Tested proposal 3 by hand in both workspaces, with released DDEV (v1.25.3
  and v1.25.4) and the global override above in place of the core setting:
  start `ssh-agent -a ~/tmp/agent/agent.sock`, load the Coder key as shown,
  and relay to that socket. `ddev exec ssh -T git@github.com` authenticated
  with the Coder key. With this branch, the equivalent is
  `ddev config global --ssh-agent-upstream=$HOME/tmp/agent/agent.sock`.
- Scripting gotchas: `coder ssh ws -- bash -lc '...'` word-splits the
  command, and feeding a script through `bash -s` lets `ddev exec` swallow the
  rest of stdin unless it reads from `/dev/null`.

## Proposals

1. **Upstream-agent mode for ddev-ssh-agent (core, opt-in).** Implemented;
   see "Implemented: proposal 1" below. Add a global
   setting such as `ssh_agent_upstream`:
   - empty (default): today's behavior
   - a socket path: mount its parent directory and run socat as in the test
     above
   - `host`: use `$SSH_AUTH_SOCK`, or `/run/host-services/ssh-auth.sock` on
     Docker Desktop, OrbStack, or Colima (detection helpers exist in
     `pkg/dockerutil/providers.go`)

   Changing only ddev-ssh-agent gives every project and every service on
   `ddev_default` the agent at once, with no project changes, because web
   containers already share the socket volume. In relay mode, `ddev auth ssh`
   could re-point the relay at the current `$SSH_AUTH_SOCK` (forwarded-agent
   sockets change every session) and print `ssh-add -l`. `ddev describe`
   should show which mode is active. The docs should say that every container
   can then ask the agent to sign; 1Password's per-use approval limits that.

   A later option is a small Go agent (`golang.org/x/crypto/ssh/agent`) that
   combines ddev's own keyring with the upstream agent, so keys added with
   `ddev auth ssh -f` keep working next to the host agent.

2. **`ddev auth ssh -f -` (key from stdin).** Implemented. The key is read
   and checked for a PEM header before `ddev-ssh-agent` starts (starting it
   can consume stdin), then piped to `ssh-add -` through
   `dockerutil.ExecWithStdin`, which closes stdin so `ssh-add` sees the end of
   the key. Keys with a passphrase can't work, since there is no terminal to
   ask for it. In relay mode the key goes to the upstream agent, which a plain
   `ssh-agent` accepts and 1Password refuses. `TestCmdAuthSSHStdin` covers it.

3. **coder-ddev template change (outside this repo).** Filed as
   [ddev/coder-ddev#210](https://github.com/ddev/coder-ddev/issues/210). The template's startup
   script belongs to the workspace owner, so the host-config constraint does
   not apply. The recommended approach: start a regular `ssh-agent` on a fixed
   socket, load the Coder key, and export `SSH_AUTH_SOCK`. That makes plain
   `ssh` work in the workspace, and proposal 1 then covers containers with no
   Coder-specific code in DDEV.

Not covered: native Windows (named-pipe agents such as Pageant or the Windows
OpenSSH agent) without WSL. Key-file mode remains the answer there.

## Tested: macOS + OrbStack + 1Password

- OrbStack's `/run/host-services/ssh-auth.sock` serves the 1Password keys
  (OrbStack follows `IdentityAgent` in `~/.ssh/config`, not the launchd
  `$SSH_AUTH_SOCK`, which was empty), and it is mode 0666, so no chmod.
- Bind-mounting 1Password's macOS socket directory directly does not work:
  the listing fails with "Operation not permitted" and connecting is refused.
  On macOS only the provider's host-services socket works.
- The override above, pointed at `/run/host-services`, worked: `ssh-add -l`
  and `ssh -T git@github.com` succeeded in `web`.

## Implemented: proposal 1

- Global `ssh_agent_upstream` (`ddev config global --ssh-agent-upstream`):
  empty, `host`, or an absolute socket path (`~` expands), resolved by
  `SSHAgentUpstreamSocket` in [ssh_auth.go](pkg/ddevapp/ssh_auth.go). `host`
  means `/run/host-services/ssh-auth.sock` on Docker Desktop, OrbStack, and
  Colima; the Lima instance's forwarded socket, read with
  `limactl shell <instance> printenv SSH_AUTH_SOCK`; and `$SSH_AUTH_SOCK` on
  Linux and WSL2. Other macOS providers are rejected.
- In relay mode the compose template runs socat with a `killall -0 socat`
  healthcheck and `security_opt: label=disable`. The relay runs as `0:0`,
  or as the user's UID when DDEV uses `keep-id` (rootless Podman on Linux).
  `sshAgentUpstreamMount` mounts `/run/host-services/ssh-auth.sock` as a file
  and any other socket by its directory.
- The container carries a `com.ddev.ssh-agent-upstream` label, and a mismatch
  recreates it, so changing the setting or `$SSH_AUTH_SOCK` takes effect on
  the next start.
- When the setting cannot be used, `ddev start` warns and runs DDEV's own
  agent; `ddev auth ssh` fails with the reason and the fix.
- `ddev auth ssh` without flags lists the upstream agent's keys; with `-f` or
  `-d` it still tries `ssh-add`, which 1Password refuses. When the upstream
  agent is down it fails with "Make sure that agent is running and holds your
  keys."
- Docs: the "Using an Existing SSH Agent" section in `cli.md`, and the
  `ssh_agent_upstream` entry in `config.md`.

## Automated tests

| Test | Where it does real work | Covers |
| --- | --- | --- |
| `TestSSHAgentUpstreamMount`, `TestSSHAgentUpstreamSocketPaths` (`ssh_auth_internal_test.go`) | Everywhere, no Docker | File mount for the provider socket, directory mount otherwise, `~` expansion, relative paths rejected |
| `TestSSHAgentUpstream` | Rendering and fallback everywhere; live relay on Linux with native Docker | Invalid setting falls back; relay to a host `ssh-agent -a`; agent stop fails, restart recovers with the same container; `host` via `$SSH_AUTH_SOCK`; clearing the setting restores DDEV's agent |
| `TestSSHAgentUpstreamHost` | macOS | Throwaway key in the macOS agent is visible on OrbStack, Docker Desktop, and Colima with `--ssh-agent`; fallback where nothing is forwarded; skips when OrbStack's `IdentityAgent` points elsewhere or Colima lacks `--ssh-agent` |
| `TestCmdAuthSSHUpstream` | Linux with native Docker | `ddev auth ssh` lists upstream keys, and explains a stopped agent |

Run so far: Ubuntu 24.04 docker-ce, Ubuntu 26.04 rootless Podman and rootless
Docker (all pass); macOS Docker Desktop and Colima (pass), OrbStack (skips
here because `IdentityAgent` points at 1Password). Both upstream tests write
the global config back in cleanup, because code they call saves it.

For CI on macOS: the test needs a macOS agent session (`SSH_AUTH_SOCK`), Colima
must be started with `--ssh-agent`, and Lima needs `ssh.forwardAgent: true`,
or those cases only exercise the fallback.

Manual only: 1Password (a GUI app with unlock and approval prompts), OrbStack
following `IdentityAgent` (needs a `~/.ssh/config` edit and an OrbStack
restart), Docker Desktop, Colima, and Lima ignoring `IdentityAgent`, forwarded
`ssh -A` logins, Coder workspaces, and Windows.

## macOS results so far

| Provider | Agent | Result |
| --- | --- | --- |
| OrbStack | 1Password via `IdentityAgent` | Works, including GitHub auth from `web` |
| OrbStack | Apple's agent (no `IdentityAgent`) | Works after restarting OrbStack |
| OrbStack | `SSH_AUTH_SOCK` exported in the shell only | Ignored; containers get Apple's agent |
| OrbStack | 1Password quit, then restarted | Clear error while down; recovers with no DDEV restart (directory mount; retest with the file mount) |
| Docker Desktop | Apple's agent | Works; the root relay is required |
| Docker Desktop | 1Password via `IdentityAgent` | Containers get Apple's agent instead |
| Colima `--ssh-agent` | Apple's agent | Works once the socket file, not its directory, is mounted |
| OrbStack | gpg-agent (`enable-ssh-support`) via `IdentityAgent` | Works: keys listed and `ssh-keygen -Y sign` succeeds from `web` |
| Lima (rootless Docker) with `forwardAgent` | Apple's agent | Works; DDEV reads the socket path with `limactl shell <instance> printenv SSH_AUTH_SOCK`, and a VM restart's new path is picked up on the next start |
| Podman (rootless) | 1Password, via a user-held `podman machine ssh` session and an explicit socket path | Relay works, including GitHub auth from `web`, once `label=disable` is set; not practical as a supported setup |
| Rancher Desktop, Lima without `forwardAgent` | any | `host` rejected; `ddev start` warns and falls back to DDEV's own agent, `ddev auth ssh` fails with the fix |

Findings:

- `/run/host-services/ssh-auth.sock` is mounted as a file, because Colima's is
  a symlink that a directory mount does not resolve. Other socket paths are
  mounted by directory, so an agent that recreates its socket keeps working.
- Switching from the relay back to DDEV's own agent works while projects keep
  running, because web containers mount the socket volume, not the socket.
- An unusable upstream, such as `host` after switching to Rancher Desktop,
  must not block projects, so `ddev start` warns and runs DDEV's own agent.
- SELinux blocks the relay from the upstream socket (`Permission denied` on
  Podman's enforcing VM), so relay mode sets `security_opt: label=disable`.
  Docker ignores it where SELinux is off. It is also enough on Fedora 44.
- The ddev-ssh-agent healthcheck only checks socat, so the container stays
  healthy while the upstream agent is down.

## Linux results so far

Ubuntu 24.04.5 desktop (arm64, Parallels), docker-ce 29.8.1, UID 1001:

| Agent | Setting | Result |
| --- | --- | --- |
| Forwarded from macOS with `ssh -A` (1Password there) | `host` | Works, including GitHub auth from `web` |
| Same, from a later SSH login | `host` | Relay fails once the first login ends; `ddev auth ssh` or `ddev start` from the new login recreates it |
| gcr-ssh-agent, Ubuntu's desktop agent at `/run/user/<uid>/gcr/ssh` | `host` with the desktop `SSH_AUTH_SOCK` | Works |
| gcr-ssh-agent | explicit `/run/user/<uid>/gcr/ssh` | Works, and the path is stable across logins |
| Private agent from `eval $(ssh-agent -s)`, no forwarding | `host`, from that shell | Works; only that shell and its children know the socket |
| Private agent on a fixed socket, `ssh-agent -a ~/.ssh/agent.sock` | explicit path | Works, and the path is stable across logins |
| Forwarded, through a fixed symlink such as `~/.ssh/rc` maintains | explicit symlink path | Fails: the directory mount does not contain the link's absolute target |

Findings:

- A forwarded socket lives only as long as the SSH login that created it.
  `host` follows `$SSH_AUTH_SOCK`, so each new login needs `ddev auth ssh`,
  and a start from one login repoints the relay away from another login.
- The common tmux pattern of a fixed symlink to the current forwarded socket
  does not work. Supporting it would mean mounting the link and its target
  at their real paths, which for `/tmp/ssh-*` means the host's whole `/tmp`.
- A private `ssh-agent` outlives the shell that started it, unlike a forwarded
  socket, but other shells cannot find it. Starting it with `-a` on a fixed
  path is the robust form, and is what proposal 3 suggests for Coder.
- Ubuntu 24.04 runs gcr-ssh-agent; the older gnome-keyring socket at
  `/run/user/<uid>/keyring/ssh` also exists. For desktop users an explicit
  `/run/user/<uid>/gcr/ssh` is the most robust setting.

Ubuntu 26.04.1 (arm64, Parallels), rootless Podman 6.1.2 through the Docker
CLI, no SELinux. DDEV applies `userns_mode: keep-id` here:

| Agent | Setting | Result |
| --- | --- | --- |
| 1Password over a socat bridge to the Mac (`~/.1password-agent.sock`) | `host` | Works, including GitHub auth from `web`; stopping the bridge gives the clear error, restarting it recovers with the same container |
| OpenSSH `ssh-agent.service` (`/run/user/<uid>/openssh_agent`, 0600) | explicit path | Failed while the relay ran as root ("Connection reset by peer"); works now that it runs as the user's UID under keep-id |
| gcr-ssh-agent | explicit path | Works |
| Forwarded with `ssh -A` | `host` | Works; OpenSSH 10 puts the socket at `~/.ssh/agent/s.*.sshd.*` instead of `/tmp/ssh-*` |
| None (`SSH_AUTH_SOCK` unset) | `host` | `ddev auth ssh` says `SSH_AUTH_SOCK` is not set |

Findings:

- Under keep-id, container root is a subordinate UID on the host. Root in
  that namespace can still open the user's files, but OpenSSH's `ssh-agent`
  checks the peer's UID and drops anyone other than its owner or root. So
  with keep-id the relay runs as the user's UID; elsewhere it stays root,
  which Docker Desktop's `root:root` 0660 socket needs.
- A socket directly in `$HOME`, like this bridge's, makes the directory
  mount expose all of `$HOME` to the relay container. It works; a socket in
  its own directory is tidier.
- `TestSSHAgentUpstream` (including its live section) and
  `TestCmdAuthSSHUpstream` pass here.

Same VM with rootless Docker 29.8.1 (`linux-docker-rootless`, no keep-id, so
the relay runs as container root, which rootless Docker maps to the user):
the 1Password bridge with `host` (including GitHub auth from `web`), OpenSSH's
0600 agent, gcr, a forwarded `ssh -A` agent, the bridge stop and restart, and
the no-agent error all behave as on Podman, and both automated tests pass.
Note that `make testcmd` runs `ddev poweroff` in its `TestMain`, which stops
every project on the machine.

Fedora 44 Workstation (arm64, Parallels), SELinux enforcing, with both rootful
docker-ce 29.8.1 (`linux-docker`) and rootless Podman 5.8.7 with crun 1.28
(`podman-rootless-selinux`, keep-id). Each engine gave the same results:

| Agent | Setting | Result |
| --- | --- | --- |
| Forwarded with `ssh -A` (`~/.ssh/agent/s.*.sshd.*`) | `host` | Works, including GitHub auth from `web` |
| gcr-ssh-agent, Fedora's GNOME agent | explicit `/run/user/<uid>/gcr/ssh` | Works |
| Private agent, `ssh-agent -a ~/tmp/privagent/agent.sock` | explicit path | Works; killing and restarting the agent recovers with the same relay container |
| DDEV's own agent | unset | `ddev auth ssh -f -` adds a key from stdin |
| Private agent | explicit path | `ddev auth ssh -f -` adds the key to the upstream agent, and `web` sees it |

`TestSSHAgentUpstream` (including its live section), `TestCmdAuthSSHUpstream`,
and `TestCmdAuthSSHStdin` pass on both engines. `label=disable` is enough for
SELinux on both.

On rootless Podman, about one project start in five fails with `crun: write
to /proc/sys/net/ipv4/ping_group_range (are all the IDs mapped in the user
namespace?)`, always on the db container. Released v1.25.4 fails the same
way with no upstream set, so it isn't caused by this change; retrying the
start works.

## Environments to test

Each environment should cover `ddev auth ssh`, `ddev exec ssh-add -l`, and
`ddev exec ssh -T git@github.com`, plus the agent being stopped and restarted.

macOS providers, each with Apple's agent and with an `IdentityAgent` agent:

- OrbStack, Docker Desktop, and Colima with `--ssh-agent` (done)
- Rancher Desktop (done: no forwarding), Lima (done with `forwardAgent`),
  and Podman (done: no persistent forwarding). socktainer (Apple
  container) is out of scope.

macOS agents, with at least one provider:

- Apple's agent (done), 1Password (done)
- gpg-agent with `enable-ssh-support` (done on OrbStack with a software key;
  a YubiKey still untested)
- Secretive, Bitwarden, or Strongbox: another socket-path agent, to confirm
  nothing is 1Password-specific

Linux, with native Docker and `ssh_agent_upstream=host`:

- Ubuntu desktop with gcr-ssh-agent and a forwarded agent (done; see above)
- Fedora 44 with SELinux enforcing, docker-ce and rootless Podman (done)
- Arch or Debian
- A forwarded agent over `ssh -A` (done on Ubuntu)
- 1Password's Linux agent via an explicit socket path
- Docker Desktop for Linux, which may offer `/run/host-services`
- Rootless Podman and rootless Docker (done on Ubuntu 26.04)

WSL2: the relay setup above (done), repeated with the core setting instead of
the override.

Traditional Windows: the Windows OpenSSH agent and 1Password use a named pipe
that the Docker Desktop VM cannot mount. See whether Docker Desktop for Windows
forwards an agent at all before deciding whether it is in scope.

## Next steps

- Remaining matrix: Docker Desktop for
  Linux (does it offer `/run/host-services`, and is `/tmp` shared into its
  VM?), WSL2 with the core setting, Secretive or Bitwarden, a YubiKey through
  gpg-agent, and retesting 1Password quit-and-restart on OrbStack now that the
  provider socket is a file mount.
- Decide whether traditional Windows is in scope.
- Decide whether to support a fixed symlink to a forwarded socket (the tmux
  pattern); it would need the link's target directory mounted as well.
- Consider showing the mode in `ddev describe`, and a healthcheck that notices
  a dead upstream.
- Put proposal 3 into the coder-ddev template's startup script
  ([ddev/coder-ddev#210](https://github.com/ddev/coder-ddev/issues/210)).
