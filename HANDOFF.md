# Handoff: using an existing SSH agent from DDEV containers

Context for [#3878](https://github.com/ddev/ddev/issues/3878). This file
will be deleted before merge; only the open items remain.

## TODO: validation and tests

- [x] Platform-specific automated coverage: the live relay tests in
  `TestSSHAgentUpstream` and `TestCmdAuthSSHUpstream`, including fallback when
  `SSH_AUTH_SOCK` is unset, pass on WSL2 arm64 with Docker CE and with Docker
  Desktop, so both the `wsl2-mirrored` and `wsl2-docker-desktop` pipelines run
  them. Native Windows is covered by `TestSSHAgentUpstreamSocketPaths`.
- [x] `TestCmdGlobalConfigSSHAgentUpstream` covers `~/...` and, on native
  Windows, drive-letter and UNC paths through the actual `ddev config global
  --ssh-agent-upstream` CLI path rather than just the internal resolver.
- [ ] Retest the user documentation against the supported provider matrix
  before removing draft status from PR #8861.
