# Handoff: using an existing SSH agent from DDEV containers

Context for [#3878](https://github.com/ddev/ddev/issues/3878). This file
will be deleted before merge; only the open items remain.

## TODO: validation and tests

- [ ] Add platform-specific automated coverage once the supported Windows and
  WSL2 behavior is decided, including fallback when no provider agent exists.
- [x] `TestCmdGlobalConfigSSHAgentUpstream` covers `~/...` and, on native
  Windows, drive-letter and UNC paths through the actual `ddev config global
  --ssh-agent-upstream` CLI path rather than just the internal resolver.
- [ ] Retest the user documentation against the supported provider matrix
  before removing draft status from PR #8861.
