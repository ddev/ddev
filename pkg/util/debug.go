package util

import (
	"runtime"
	"strconv"
	"time"

	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/sirupsen/logrus"
)

// TimeTrack determines the amount of time a function takes to return. Timer
// starts when it is called. The printed name is determined from the calling
// function. It returns an anonymous function that, when called, will print
// the elapsed run time.
//
// It only tracks if DDEV_VERBOSE is set.
//
// Usage:
//
//	defer util.TimeTrack()()
//
// or
//
//	tracker := util.TimeTrack()
//	...
//	tracker()
func TimeTrack() func() {
	if globalconfig.DdevVerbose {
		// Determine name from calling function.
		var name string

		if counter, _, _, success := runtime.Caller(1); !success {
			name = "<failed to determine caller name>"
		} else {
			name = runtime.FuncForPC(counter).Name() + "()"
		}

		return timeTrack(&name)
	}

	return func() {}
}

// TimeTrackC determines the amount of time a function takes to return. Timer
// starts when it is called. The customName parameter is printed. It returns an
// anonymous function that, when called, will print the elapsed run time.
//
// It only tracks if DDEV_VERBOSE is set.
//
// Usage:
//
//	defer util.TimeTrackC("a custom name")()
//
// or
//
//	tracker := util.TimeTrackC("a custom name")
//	...
//	tracker()
func TimeTrackC(customName string) func() {
	if globalconfig.DdevVerbose {
		return timeTrack(&customName)
	}

	return func() {}
}

// timeTrack is the internal helper for the exported time track functions.
func timeTrack(name *string) func() {
	start := time.Now()

	// Print message and return func. Printf is avoided to minimize impact.
	logrus.Print("PERF: enter " + *name + " at " + start.Format("15:04:05.000000000"))

	return func() {
		logrus.Print("PERF: exit " + *name + " at " + time.Now().Format("15:04:05.000000000") + " after " + strconv.FormatInt(time.Since(start).Milliseconds(), 10) + "ms")
	}
}
