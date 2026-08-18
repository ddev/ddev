# HANDOFF: finishing the Amplitude side of #8599 (add-on repository reporting)

Temporary note — not part of the ddev codebase, delete once resolved (same
convention as previous HANDOFF.md files, e.g. `73bd61fed`). Context for
picking this back up on a normal computer with a real browser.

Branch: `20260818_rfay_amplitude_addon_repository` (pushed to `rfay`, not yet
merged). PR: https://github.com/ddev/ddev/pull/8716

## What this is

Implements [#8599](https://github.com/ddev/ddev/issues/8599): report each
installed add-on's GitHub `owner/repo` to Amplitude (`Add-on Repositories`),
index-aligned with the existing `Add-ons` (names) property, so add-on usage
can be traced back to a specific repo instead of an ambiguous short name.

## Code side: done

- `pkg/ddevapp/addons.go`: `GetInstalledAddonRepositories(app)`, parallel to
  `GetInstalledAddonNames`, using the existing `IsGithubRef` helper to blank
  out non-GitHub installs (directory/tarball/URL sources).
- `pkg/ddevapp/amplitude_project.go`: `TrackProject()` sends
  `.AddOnRepositories(...)` alongside `.AddOns(...)`.
- `docs/content/users/usage/diagnostics.md`: documented the new property.
- `pkg/ddevapp/addons_test.go`: `TestGetInstalledAddonRepositories` — passing.
- `make`, `gofmt`, `make staticrequired` (golangci-lint + markdownlint) all
  clean in this environment. `zensical` isn't installed here, so that one
  static-check step wasn't run.

## Amplitude side: blocked on this environment, needs a normal computer

Randy created a new Amplitude branch `add-on-repo` and added the
`Add-on Repositories` property in the Data GUI at
`https://data.amplitude.com/ddev/DDEV`, per
`docs/content/developers/building-contributing.md`'s Instrumentation section.

The remaining step — regenerating `third_party/ampli/ampli.go` for real via
the `ampli` CLI — can't be completed from this sandbox:

- Installed `ampli` fine (`npm install -g @amplitude/ampli`).
- `ampli login` requires an interactive OAuth flow in a real browser tied to
  a human Amplitude account. It hangs waiting for a callback that never
  arrives here.
- `~/.amplitude_api_key.txt` is the SDK **write key** (used at build time to
  *send* events, per the docs' "Examining data on Amplitude.com" section) —
  not a personal API token. Tried it with `ampli pull -t <key>`; rejected as
  invalid, as expected — it's the wrong kind of credential.
- There's no non-interactive way to authenticate `ampli` found so far (no
  env var, no device code printed to stdout).

In the meantime, `third_party/ampli/ampli.go` was hand-edited to add the
`AddOnRepositories` builder method/property in the same style as the
existing generated `AddOns` property (interface method + impl, alphabetized
by identifier the way the codegen does it — `AddOnRepositories` sorts before
`AddOns`). The Tracking Plan `Version`/`VersionID` constants in the file
header and `Load()` were deliberately left untouched rather than fabricated,
since only a real `ampli pull` knows the correct values.

### What to do on a normal computer

1. `npm install -g @amplitude/ampli` if not already installed.
2. `ampli login` (opens a browser; log into the `ddev` org).
3. From the repo root, on this branch:

   ```bash
   ampli checkout add-on-repo
   ampli pull
   ```

4. Diff the regenerated `third_party/ampli/ampli.go` against the hand-edited
   version currently on this branch — the `Add-on Repositories` property name,
   `AddOnRepositories` builder method, and the Go call site in
   `pkg/ddevapp/amplitude_project.go` should already line up; the real diff
   should just be the `Version`/`VersionID`/`Build` metadata and possibly
   generator-formatting details.
5. `make && go test -v -run TestGetInstalledAddonRepositories ./pkg/ddevapp/...`
   to confirm nothing broke.
6. Merge the `add-on-repo` branch to `main` in the Amplitude backend (Activity
   tab → Merge), then `ampli checkout main` to leave the workspace on `main`.
7. Commit the regenerated `third_party/ampli/ampli.go` on this branch, push,
   and let PR #8716 proceed through review as normal.
8. Delete this file once the above is done.
