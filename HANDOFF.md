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

- Native Windows uses named-pipe agents. Do not promise `host` support until
  Docker Desktop for Windows is tested end to end.
- WSL2 needs a Linux-visible Unix socket for a Windows agent. A bridge such as
  `mame/wsl2-ssh-agent` can provide one; DDEV must not create it.
- A fixed symlink to a forwarded SSH socket does not work when its target is
  outside the mounted directory.
- Coder workspace work belongs in
  [ddev/coder-ddev#210](https://github.com/ddev/coder-ddev/issues/210).

## TODO: validation and tests

- [ ] Traditional Windows with Docker Desktop: test the Windows OpenSSH agent
  and 1Password with `ssh_agent_upstream: host`. Verify `ddev start`,
  `ddev auth ssh`, `ddev exec ssh-add -l`, and GitHub SSH authentication. If
  Docker Desktop does not forward either agent, make `host` fall back with a
  clear warning and document key-file mode as the supported path.
- [ ] WSL2 with Docker Desktop integration: run the same checks and determine
  whether its provider socket reaches a Windows-host agent.
- [ ] WSL2 with Docker CE: use a Windows named-pipe-to-Unix-socket bridge;
  verify `host` and an explicit socket path, then stop and restart the bridge
  without recreating `ddev-ssh-agent`.
- [ ] Add platform-specific automated coverage once the supported Windows and
  WSL2 behavior is decided, including fallback when no provider agent exists.
- [ ] Add a CLI configuration test for `~/...` and native Windows drive paths,
  which the schema now accepts alongside Unix absolute paths.
- [ ] Retest the user documentation against the supported provider matrix
  before removing draft status.
