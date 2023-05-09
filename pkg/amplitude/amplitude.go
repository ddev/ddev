package amplitude

import (
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/amplitude/analytics-go/amplitude"
	"github.com/ddev/ddev/pkg/amplitude/loggers"
	"github.com/ddev/ddev/pkg/amplitude/storages"
	"github.com/ddev/ddev/pkg/dockerutil"
	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/ddev/ddev/pkg/util"
	"github.com/ddev/ddev/pkg/version"
	"github.com/ddev/ddev/pkg/versionconstants"
	"github.com/ddev/ddev/third_party/ampli"
	"github.com/denisbrodbeck/machineid"
	"github.com/spf13/cobra"
)

var (
	userID       string
	eventOptions ampli.EventOptions
	initialized  bool
)

// GetUserID returns the unique user id to be used when tracking an event.
func GetUserID() string {
	if userID == "" {
		userID, _ = machineid.ProtectedID("ddev")
	}

	return userID
}

// GetEventOptions returns the EventOptions to be used when tracking an event.
func GetEventOptions() ampli.EventOptions {
	return eventOptions
}

// TrackBinary collects and tracks information about the binary for
// instrumentation.
func TrackBinary() {
	runTime := util.TimeTrack()
	defer runTime()

	// Initialization is currently done before via init() func somewhere while
	// creating the ddevapp. This should be cleaned up.
	// TODO remove once clean up has done.
	InitAmplitude()

	// Early exit if instrumentation is disabled.
	if ampli.Instance.Disabled {
		return
	}

	dockerVersion, _ := dockerutil.GetDockerVersion()
	dockerPlaform, _ := version.GetDockerPlatform()
	timezone, _ := time.Now().In(time.Local).Zone()

	builder := ampli.Binary.Builder().
		Architecture(runtime.GOARCH).
		DockerPlatform(dockerPlaform).
		DockerVersion(dockerVersion).
		Language(os.Getenv("LANG")).
		Os(runtime.GOOS).
		Timezone(timezone).
		Version(versionconstants.DdevVersion)

	wslDistro := nodeps.GetWSLDistro()
	if wslDistro != "" {
		builder.
			WslDistro(wslDistro)
	}

	ampli.Instance.Binary(GetUserID(), builder.Build(), GetEventOptions())
}

// TrackCommand collects and tracks information about the command for
// instrumentation.
func TrackCommand(cmd *cobra.Command, args []string) {
	runTime := util.TimeTrack()
	defer runTime()

	// Initialization is currently done before via init() func somewhere while
	// creating the ddevapp. This should be cleaned up.
	// TODO remove once clean up has done.
	InitAmplitude()

	// Early exit if instrumentation is disabled.
	if ampli.Instance.Disabled {
		return
	}

	builder := ampli.Command.Builder().
		Arguments(args).
		CalledAs(cmd.CalledAs()).
		CommandName(cmd.Name())

	ampli.Instance.Command(GetUserID(), builder.Build(), GetEventOptions())
}

// Flush transmits the queued events if limits are reached.
func Flush() {
	runTime := util.TimeTrack()
	defer runTime()

	// Early exit if instrumentation is disabled or internet not active.
	if ampli.Instance.Disabled || !globalconfig.IsInternetActive() {
		return
	}

	ampli.Instance.Flush()
}

// setIdentity prepares the identity for later use by calling Identify.
func setIdentity() {
	runTime := util.TimeTrack()
	defer runTime()

	lang := os.Getenv("LANG")

	eventOptions = ampli.EventOptions{
		AppVersion: versionconstants.DdevVersion,
		Platform:   runtime.GOARCH,
		OSName:     runtime.GOOS,
		Language:   lang,
	}

	// Early exit if instrumentation is disabled.
	if ampli.Instance.Disabled {
		return
	}

	ampli.Instance.Identify(GetUserID(), GetEventOptions())
}

// InitAmplitude initializes the instrumentation and must be called once before
// the instrumentation functions can be used.
// Initialization is currently done before via init() func somewhere while
// creating the ddevapp. This should be cleaned up.
// TODO make private once clean up has done.
func InitAmplitude() {
	runTime := util.TimeTrack()
	defer runTime()

	if initialized {
		return
	}

	defer func() {
		initialized = true
	}()

	// Disable instrumentation if AmplitudeAPIKey is not available.
	if versionconstants.AmplitudeAPIKey == "" {
		ampli.Instance.Disabled = true
		return
	}

	// Size of the queue. If reached the queued events will be sent.
	queueSize := globalconfig.DdevGlobalConfig.InstrumentationQueueSize
	if queueSize <= 0 {
		queueSize = 50
	}

	// Interval of reporting. If reached since last reporting events will be sent.
	interval := globalconfig.DdevGlobalConfig.InstrumentationReportingInterval
	if interval <= 0 {
		interval = 24 * time.Hour
	}

	ampli.Instance.Load(ampli.LoadOptions{
		Client: ampli.LoadClientOptions{
			APIKey: versionconstants.AmplitudeAPIKey,
			Configuration: amplitude.Config{
				FlushInterval:  interval,
				FlushQueueSize: queueSize,
				Logger:         loggers.NewDdevLogger(),
				StorageFactory: func() amplitude.EventStorage {
					return storages.NewDelayedTransmissionEventStorage(
						queueSize,
						interval,
						filepath.Join(globalconfig.GetGlobalDdevDir(), `.amplitude.cache`),
					)
				},
			},
		},
		Disabled: globalconfig.DdevNoInstrumentation || !globalconfig.DdevGlobalConfig.InstrumentationOptIn,
	})

	setIdentity()
}
