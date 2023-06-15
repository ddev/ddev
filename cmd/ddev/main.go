package main

import (
	"os"
	"sync"

	"github.com/ddev/ddev/cmd/ddev/cmd"
	"github.com/ddev/ddev/pkg/amplitude"
	"github.com/ddev/ddev/pkg/output"
	"github.com/ddev/ddev/pkg/util"
)

func main() {
	defer util.TimeTrack()()

	// Initialization is currently done before via init() func somewhere while
	// creating the ddevapp. This should be cleaned up.
	amplitude.InitAmplitude()
	defer amplitude.CheckSetUp()

	// Starting the submission asynchronously to reduce the user impact.
	var submittingEvents sync.Mutex
	go func(mutex *sync.Mutex) {
		mutex.Lock()
		defer mutex.Unlock()

		amplitude.Flush()
	}(&submittingEvents)

	// Prevent running as root for most cases
	// We really don't want ~/.ddev to have root ownership, breaks things.
	if os.Geteuid() == 0 && len(os.Args) > 1 && os.Args[1] != "hostname" {
		util.Failed("ddev is not designed to be run with root privileges, please run as normal user and without sudo")
	}

	cmd.Execute()

	// Waiting for submission of events to be finished.
	if !submittingEvents.TryLock() {
		output.UserOut.Println("Submitting usage statistics...")

		submittingEvents.Lock()
		defer submittingEvents.Unlock()

		output.UserOut.Println("Usage statistics submitted.")
	}
}
