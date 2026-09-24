# Handoff: using an existing SSH agent from DDEV containers

Context for whoever picks up
[#3878](https://github.com/ddev/ddev/issues/3878): users whose keys live in an
agent rather than in `~/.ssh` (1Password, YubiKey, forwarded agents, Coder
workspaces) cannot use `ddev auth ssh`. This file records what was learned and
tested on 2026-09-23. No DDEV code has changed yet.

## Constraints

- Do not break current practice: `ddev auth ssh` with key files stays the
  default and behaves exactly as today.
- DDEV does not change host configuration (editing `/etc/hosts` is the only
  exception). DDEV may *use* an agent socket the user already has, but must
  not install, launch, or configure one on the host. The `ddev 1password`
  command in the add-on below writes a macOS LaunchAgent, so that approach
  cannot move into core.

## How it works today

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
- `CreateSSHAuthComposeFile` already merges user overrides from
  `~/.ddev/ssh-auth-compose.*.yaml`
  ([ssh_auth.go:174](pkg/ddevapp/ssh_auth.go#L174)). This makes a global,
  no-code prototype possible.
- `EnsureSSHAgentContainer` recreates the container only when the image
  changes ([ssh_auth.go:52](pkg/ddevapp/ssh_auth.go#L52)), so an edited
  override needs `docker rm -f ddev-ssh-agent` or `ddev poweroff`.

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
- On Docker Desktop, the socket is root:root 660, so the relay would need
  `user: "0:0"`. Not tested here.

## Coder workspaces (coder.ddev.com)

- There is normally no SSH agent. git over SSH works through Coder's GitSSH
  wrapper: the [ddev/coder-ddev](https://github.com/ddev/coder-ddev) template
  startup script (`freeform/template.tf`) sets
  `GIT_SSH_COMMAND="<coder binary> gitssh"` and `core.sshCommand`.
  `coder gitssh` fetches the user's Coder-managed key from the Coder server
  using the workspace agent's credentials and runs `ssh -i` with a temporary
  file. That is Coder's general design; the details were not checked inside a
  workspace.
- Consequences: only git uses the key, plain `ssh` does not, and DDEV
  containers have neither the `coder` binary nor the credentials, so git
  inside `web` (for example Composer with private repositories) fails.
- An agent exists only in a session opened with forwarding, such as
  `coder ssh -A`, and only for that session. VS Code for Web and the web
  terminal have none.
- To check in a workspace:

  ```bash
  echo "$GIT_SSH_COMMAND"; git config --global core.sshCommand
  env | grep '^CODER_'
  echo "SSH_AUTH_SOCK=$SSH_AUTH_SOCK"
  ```

  Still unknown: whether a workspace shell has the agent token and URL needed
  to fetch the key directly.

## Proposals

1. **Upstream-agent mode for ddev-ssh-agent (core, opt-in).** Add a global
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

2. **`ddev auth ssh -f -` (key from stdin).** Pipe the key into `ssh-add -` in
   the auth container instead of bind-mounting a file. This covers keys held
   in secret managers, CI variables, and the Coder GitSSH key. It still has to
   pass the PEM check in `fileIsPrivateKey`.

3. **coder-ddev template change (outside this repo).** The template's startup
   script belongs to the workspace owner, so the host-config constraint does
   not apply. The recommended approach: start a regular `ssh-agent` on a fixed
   socket, load the Coder key, and export `SSH_AUTH_SOCK`. That makes plain
   `ssh` work in the workspace, and proposal 1 then covers containers with no
   Coder-specific code in DDEV.

Not covered: native Windows (named-pipe agents such as Pageant or the Windows
OpenSSH agent) without WSL. Key-file mode remains the answer there.

## Next steps

- Try the override on macOS with Docker Desktop (add `user: "0:0"`),
  OrbStack, and Colima with `--ssh-agent`, and on Linux with a forwarded
  agent (`ssh -A`).
- Confirm how `coder gitssh` gets the key and whether a workspace shell can
  fetch it.
- Then implement proposal 1, with proposal 2 as a separate small change.
