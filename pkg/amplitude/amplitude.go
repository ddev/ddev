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
	userID      string
	initialized bool
	identified  bool
)

const cacheFile = ".amplitude.cache"

// GetUserID returns the unique user id to be used when tracking an event.
func GetUserID() string {
	if userID == "" {
		userID, _ = machineid.ProtectedID("ddev")
	}

	return userID
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
		CommandName(cmd.Name()).
		CommandPath(cmd.CommandPath())

	ampli.Instance.Command(GetUserID(), builder.Build())
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

// FlushForce transmits the queued events even if limits are not reached.
func FlushForce() {
	runTime := util.TimeTrack()
	defer runTime()

	// Early exit if instrumentation is disabled or internet not active.
	if ampli.Instance.Disabled || !globalconfig.IsInternetActive() {
		return
	}

	backupInstrumentationQueueSize := globalconfig.DdevGlobalConfig.InstrumentationQueueSize

	defer func() {
		globalconfig.DdevGlobalConfig.InstrumentationQueueSize = backupInstrumentationQueueSize
		ampli.Instance.Client = nil
		initialized = false

		InitAmplitude()
	}()

	globalconfig.DdevGlobalConfig.InstrumentationQueueSize = 1
	ampli.Instance.Client = nil
	initialized = false

	InitAmplitude()

	Flush()
}

// Clean removes the cache file.
func Clean() {
	_ = os.Remove(getCacheFileName())
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
						getCacheFileName(),
					)
				},
			},
		},
		Disabled: globalconfig.DdevNoInstrumentation || !globalconfig.DdevGlobalConfig.InstrumentationOptIn,
	})

	identify()
}

// getCacheFileName returns the cache filename.
func getCacheFileName() string {
	return filepath.Join(globalconfig.GetGlobalDdevDir(), cacheFile)
}

// identify collects information about this installation.
func identify() {
	runTime := util.TimeTrack()
	defer runTime()

	// Early exit if instrumentation is disabled.
	if ampli.Instance.Disabled {
		return
	}

	// Avoid multiple calls.
	if identified {
		return
	}

	defer func() {
		identified = true
	}()

	// Identify this installation.
	dockerVersion, _ := dockerutil.GetDockerVersion()
	dockerPlaform, _ := version.GetDockerPlatform()
	timezone, _ := time.Now().In(time.Local).Zone()

	builder := ampli.Identify.Builder().
		DockerPlatform(dockerPlaform).
		DockerVersion(dockerVersion).
		Language(os.Getenv("LANG")).
		Os(runtime.GOOS).
		Platform(runtime.GOARCH).
		Timezone(timezone).
		Version(versionconstants.DdevVersion)

	if wslDistro := nodeps.GetWSLDistro(); wslDistro != "" {
		builder.
			WslDistro(wslDistro)
	}

	ampli.Instance.Identify(GetUserID(), builder.Build())
}
