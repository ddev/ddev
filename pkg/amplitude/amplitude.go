package amplitude

import (
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"github.com/amplitude/analytics-go/amplitude"
	"github.com/ddev/ddev/pkg/amplitude/loggers"
	"github.com/ddev/ddev/pkg/amplitude/storages"
	"github.com/ddev/ddev/pkg/dockerutil"
	"github.com/ddev/ddev/pkg/environment"
	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/ddev/ddev/pkg/output"
	"github.com/ddev/ddev/pkg/util"
	"github.com/ddev/ddev/pkg/version"
	"github.com/ddev/ddev/pkg/versionconstants"
	"github.com/ddev/ddev/third_party/ampli"
	"github.com/denisbrodbeck/machineid"
	"github.com/spf13/cobra"
)

// Local cache variables to speed up the implementation.
var (
	deviceID    string
	initialized bool
	identified  bool
	mutex       sync.Mutex
)

// serverURL overrides the Amplitude endpoint when set; tests point it at a
// local server.
var serverURL string

// cacheFile is the name of the cache file in ~/.ddev.
const cacheFile = ".amplitude.cache"

// GetDeviceID returns the unique device id to be used when tracking an event.
func GetDeviceID() string {
	if deviceID == "" {
		deviceID, _ = machineid.ProtectedID("ddev")
	}

	return deviceID
}

// GetEventOptions returns default options to be used when tracking an event.
func GetEventOptions() (options ampli.EventOptions) {
	options = ampli.EventOptions{
		DeviceID:   GetDeviceID(),
		AppVersion: versionconstants.DdevVersion,
		Platform:   runtime.GOARCH,
		OSName:     runtime.GOOS,
		Language:   os.Getenv("LANG"),
		ProductID:  "ddev cli",
	}

	options.SetTime(time.Now())

	return
}

// TrackCommand counts the command for instrumentation. The counts are sent
// as one Command Summary event per command and week, on the first command
// of a later week.
func TrackCommand(cmd *cobra.Command) {
	// defer util.TimeTrack()()

	InitAmplitude()

	if !IsEnabled() {
		return
	}

	// Events reach the queue in a goroutine that main() doesn't wait for
	// offline or after os.Exit. So roll over only online, and wait here.
	previous := countCommand(getCommandsFileName(), weekStart(time.Now()), cmd.CommandPath(), globalconfig.IsInternetActive)
	if previous != nil {
		sendCommandSummaries(*previous)
		Flush()
	}
}

// Flush transmits the queued events if limits are reached.
func Flush() {
	// defer util.TimeTrack()()

	// Early exit if instrumentation is disabled or internet not active.
	if IsDisabled() {
		return
	}

	if !mutex.TryLock() {
		return
	}

	defer mutex.Unlock()

	ampli.Instance.Flush()
}

// FlushForce transmits the queued events even if limits are not reached.
func FlushForce() {
	// defer util.TimeTrack()()

	// Early exit if instrumentation is disabled or internet not active.
	if IsDisabled() {
		return
	}

	mutex.Lock()

	backupInstrumentationQueueSize := globalconfig.DdevGlobalConfig.InstrumentationQueueSize

	defer func() {
		globalconfig.DdevGlobalConfig.InstrumentationQueueSize = backupInstrumentationQueueSize
		ampli.Instance.Client = nil
		initialized = false

		InitAmplitude()

		mutex.Unlock()
	}()

	globalconfig.DdevGlobalConfig.InstrumentationQueueSize = 1
	ampli.Instance.Client = nil
	initialized = false

	InitAmplitude()

	ampli.Instance.Flush()
}

// Clean removes the cache file, the record of reported events, the command
// counts, and their lock and temporary files.
func Clean() {
	for _, file := range []string{getCacheFileName(), getReportedFileName(), getCommandsFileName()} {
		_ = os.Remove(file)
		_ = os.Remove(file + ".lock")
		tmpFiles, _ := filepath.Glob(file + ".*.tmp")
		for _, tmpFile := range tmpFiles {
			_ = os.Remove(tmpFile)
		}
	}
}

// CheckSetUp shows a warning to the user if the API key is not available.
func CheckSetUp() {
	if !output.JSONOutput && globalconfig.DdevGlobalConfig.InstrumentationOptIn && !globalconfig.DdevNoInstrumentation && versionconstants.AmplitudeAPIKey == "" {
		util.WarningOnce("Instrumentation is opted in, but AmplitudeAPIKey is not available. This usually means you have a locally-built ddev binary or one from a PR build. It's not an error. Please report it if you're using an official release build.")
	}
}

// IsEnabled returns true once instrumentation is initialized and not
// disabled. Unlike ampli's InitializedAndEnabled(), it prints no warning when
// instrumentation was never initialized, as in builds without an API key.
func IsEnabled() bool {
	return ampli.Instance.Client != nil && !ampli.Instance.Disabled
}

// IsDisabled returns true if instrumentation is disabled or no internet is available.
func IsDisabled() bool {
	return ampli.Instance.Disabled || !globalconfig.IsInternetActive()
}

// InitAmplitude initializes the instrumentation once the global config is
// loaded, and does nothing when called earlier. Setup() in cmd/ddev/cmd
// calls it right after loading the global config.
func InitAmplitude() {
	// defer util.TimeTrack()()

	if initialized {
		return
	}

	// Before the global config loads, the opt-in reads as false. Initializing
	// then would disable instrumentation for the whole process, so wait.
	if !globalconfig.IsGlobalConfigLoaded() {
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
		queueSize = 100
	}

	// Interval of reporting. If reached since last reporting events will be sent.
	interval := globalconfig.DdevGlobalConfig.InstrumentationReportingInterval * time.Hour
	if interval <= 0 {
		interval = 24 * time.Hour
	}

	logger := loggers.NewDdevLogger(globalconfig.DdevDebug, globalconfig.DdevVerbose)

	ampli.Instance.Load(ampli.LoadOptions{
		Client: ampli.LoadClientOptions{
			APIKey: versionconstants.AmplitudeAPIKey,
			Configuration: amplitude.Config{
				Logger:    logger,
				ServerURL: serverURL,
				StorageFactory: func() amplitude.EventStorage {
					return storages.NewDelayedTransmissionEventStorage(
						logger,
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
	// defer util.TimeTrack()()

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
	ddevEnvironment := environment.GetDDEVEnvironment()

	builder := ampli.Identify.Builder().
		DdevEnvironment(ddevEnvironment).
		DockerPlatform(dockerPlaform).
		DockerVersion(dockerVersion).
		Timezone(timezone)

	if globalconfig.DdevGlobalConfig.InstrumentationUser != "" {
		builder.User(globalconfig.DdevGlobalConfig.InstrumentationUser)
	}

	if wslDistro := nodeps.GetWSLDistro(); wslDistro != "" {
		builder.WslDistro(wslDistro)
	}

	event := builder.Build()
	// User properties persist in Amplitude, so only a change needs sending.
	if AlreadyReportedUnchanged("identify", event) {
		return
	}

	ampli.Instance.Identify("", event, GetEventOptions())
}
