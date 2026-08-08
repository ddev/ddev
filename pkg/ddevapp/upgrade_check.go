package ddevapp

import "sync"

// UpgradeCheck is run once per process, immediately before DDEV starts any
// container. The CLI installs the DDEV version check here: it offers a
// `ddev poweroff` after an upgrade and records `last_started_version`.
// It stays a no-op for tests and other library consumers, which must never get
// a surprise poweroff.
var UpgradeCheck = func() {}

var (
	upgradeCheckMu   sync.Mutex
	upgradeCheckDone bool
)

// RunUpgradeCheck runs UpgradeCheck at most once per process. It is called from
// every path that creates containers, not just `ddev start`, so that
// `last_started_version` is recorded no matter which command did the starting
// (`ddev pull`, `ddev auth ssh`, `ddev exec`, a custom command, …). Otherwise
// the upgrade prompt keeps reappearing on the next `ddev start`.
// See https://github.com/ddev/ddev/issues/8520
func RunUpgradeCheck() {
	upgradeCheckMu.Lock()
	alreadyDone := upgradeCheckDone
	upgradeCheckDone = true
	upgradeCheckMu.Unlock()

	if alreadyDone {
		return
	}
	UpgradeCheck()
}
