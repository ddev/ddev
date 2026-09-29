# Handoff: using an existing SSH agent from DDEV containers

Context for [#3878](https://github.com/ddev/ddev/issues/3878): users whose
keys live only in an SSH agent need a way to use them from DDEV containers.
The user guide is [Using an Existing SSH Agent](docs/content/users/usage/cli.md).

## Constraints

- Keep `ddev auth ssh` with key files as the default behavior.
- DDEV may use an existing agent socket, but must not install, start, or
  configure an agent on the host.
- An upstream agent lets every container on `ddev_default` request signatures.
  Document that exposure and rely on agents that ask for approval where needed.

## Current implementation

- `ssh_agent_upstream` is a global setting. Empty uses DDEV's agent; `host`
  uses a provider-forwarded or `$SSH_AUTH_SOCK` agent; an absolute or
  home-relative socket path uses that agent.
- Relay mode runs `socat` in `ddev-ssh-agent`. Most sockets mount their parent
  directory so a restarted agent is picked up; the provider socket mounts as a
  file for Colima.
- The relay uses the caller's UID with `keep-id`, and root otherwise to reach
  Docker Desktop's root-owned socket. SELinux relay mode needs `label=disable`.
- An unusable upstream warns and falls back to DDEV's agent. A stopped agent
  leaves the relay healthy but unable to sign until the agent returns.
- `ddev auth ssh` without flags lists upstream keys. `ddev auth ssh -f -`
  accepts an unencrypted private key from standard input.

## Known limitations

- Native Windows agents (1Password, Windows OpenSSH) listen only on
  `\\.\pipe\openssh-ssh-agent`, and Docker Desktop for Windows forwards no
  agent. Any non-empty `ssh_agent_upstream` on native Windows warns and falls
  back to DDEV's agent. A `ddev.exe` pipe-to-TCP relay was considered and
  rejected as an extra, fragile host process.
- WSL2 needs a Linux-visible Unix socket for a Windows agent. A bridge such as
  `mame/wsl2-ssh-agent` can provide one; DDEV must not create it.
- A fixed symlink to a forwarded SSH socket does not work when its target is
  outside the mounted directory.
- Coder workspace work belongs in
  [ddev/coder-ddev#210](https://github.com/ddev/coder-ddev/issues/210).

## TODO: validation and tests

- [ ] Traditional Windows with Docker Desktop: verify that `ddev start` and
  `ddev auth ssh` warn and fall back with `ssh_agent_upstream: host`, and that
  `op read ... | ddev auth ssh -f -` works.
- [ ] WSL2 with Docker Desktop integration: run the same checks and determine
  whether its provider socket reaches a Windows-host agent.
- [x] WSL2 with Docker CE (Windows 11 arm64, 1Password): a `socat` and
  `npiperelay` bridge on `~/.1password/agent.sock` works with `host` and with
  the explicit and `~/` path. Stopping the bridge warns; restarting it
  restores signing without recreating `ddev-ssh-agent`. The x64
  `npiperelay` runs under emulation. The bridge works both as a systemd user
  service (Windows interop works there given the full `npiperelay.exe`
  path) and from a `~/.bashrc` snippet. Still to check: that the service
  comes back after `wsl --shutdown` without opening a shell.
- [ ] coder.ddev.com workspace: with this branch's binary, load the
  Coder-managed git key (from `$CODER_AGENT_URL/api/v2/workspaceagents/me/gitsshkey`)
  into an `ssh-agent -a` on a fixed socket outside `~/.ssh`, set
  `ssh_agent_upstream` to that socket, and check `ddev exec ssh -T
  git@github.com`. Also try piping the key to `ddev auth ssh -f -` with no
  upstream. Earlier tests used an older binary with a compose override, not
  this setting. Workspace setup belongs in ddev/coder-ddev#210.
- [ ] Add platform-specific automated coverage once the supported Windows and
  WSL2 behavior is decided, including fallback when no provider agent exists.
- [ ] Add a CLI configuration test for `~/...` and native Windows drive paths,
  which the schema now accepts alongside Unix absolute paths.
- [ ] Retest the user documentation against the supported provider matrix
  before removing draft status.
