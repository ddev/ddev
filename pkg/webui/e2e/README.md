# Browser tests for `ddev serve`

These [Playwright](https://playwright.dev/) tests drive the dashboard in
`pkg/webui/assets` in Chromium. The Go tests in `pkg/webui` cover the server
side; these cover the page.

| Project | What it runs against | Needs Docker | Run with |
| --- | --- | --- | --- |
| `mocked` | The real page files served by `mocked/mock-backend.js`, a fake API whose event streams the tests control | No | `make webui-e2e` |
| `real` | A real `ddev serve`, which adds, starts, snapshots, and deletes a PHP project | Yes | `make webui-e2e-real` |

The mocked tests fail on any script error or Content-Security-Policy
violation in the page. The mock sends the same policy as
`pkg/webui/server.go`, so keep the two in step.

The real test keeps its project in a separate `DDEV_XDG_CONFIG_HOME` under
`~/tmp`, so your own project list is not touched. It still shares the DDEV
router with your projects. Run it by hand with:

```bash
cd pkg/webui/e2e
npm ci
npx playwright install chromium
DDEV_WEBUI_E2E=true DDEV_BINARY=/path/to/ddev npx playwright test --project=real
```

`npx playwright test --project=mocked --ui` opens Playwright's interactive
runner, and `npx playwright show-trace test-results/*/trace.zip` replays a
failure.

`.github/workflows/webui-e2e.yml` runs both projects when `pkg/webui` or
`ddev serve` change.
