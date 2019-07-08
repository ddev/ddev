package cmd

import (
	"github.com/denisbrodbeck/machineid"
	"github.com/drud/ddev/pkg/ddevapp"
	"github.com/drud/ddev/pkg/globalconfig"
	"github.com/drud/ddev/pkg/nodeps"
	"github.com/drud/ddev/pkg/version"
	"github.com/getsentry/raven-go"
	"gopkg.in/segmentio/analytics-go.v3"
	"strings"

	"github.com/drud/ddev/pkg/dockerutil"
	"github.com/drud/ddev/pkg/output"
	"github.com/drud/ddev/pkg/util"
	"github.com/spf13/cobra"
)

var startAll bool

// StartCmd provides the ddev start command
var StartCmd = &cobra.Command{
	Use:     "start [projectname ...]",
	Aliases: []string{"add"},
	Short:   "Start a ddev project.",
	Long: `Start initializes and configures the web server and database containers
to provide a local development environment. You can run 'ddev start' from a
project directory to start that project, or you can start stopped projects in
any directory by running 'ddev start projectname [projectname ...]'`,
	PreRun: func(cmd *cobra.Command, args []string) {
		dockerutil.EnsureDdevNetwork()
	},
	Run: func(cmd *cobra.Command, args []string) {
		projects, err := getRequestedProjects(args, startAll)
		if err != nil {
			util.Failed("Failed to get project(s): %v", err)
		}

		// Look for version change and opt-in Sentry if it has changed.
		err = checkDdevVersionAndOptInInstrumentation()
		if err != nil {
			util.Failed(err.Error())
		}

		for _, project := range projects {
			if err := ddevapp.CheckForMissingProjectFiles(project); err != nil {
				util.Failed("Failed to start %s: %v", project.GetName(), err)
			}

			output.UserOut.Printf("Starting %s...", project.GetName())
			if err := project.Start(); err != nil {
				util.Failed("Failed to start %s: %v", project.GetName(), err)
				continue
			}

			util.Success("Successfully started %s", project.GetName())
			util.Success("Project can be reached at %s", strings.Join(project.GetAllURLs(), ", "))
			if project.WebcacheEnabled {
				util.Warning("All contents were copied to fast docker filesystem,\nbut bidirectional sync operation may not be fully functional for a few minutes.")
			}

			reportInstrumentation(projects)
		}
	},
	PostRun: func(cmd *cobra.Command, args []string) {
	},
}

func reportInstrumentation(projects []*ddevapp.DdevApp) {
	instrumentationNotSetUpWarning()

	if globalconfig.DdevGlobalConfig.InstrumentationOptIn && nodeps.IsInternetActive() {
		_ = raven.CaptureMessageAndWait("Usage: ddev start", map[string]string{"severity-level": "info", "report-type": "usage"})

		client := analytics.New(version.SegmentKey)
		hashedID, _ := machineid.ProtectedID("ddev")

		err := ddevapp.SegmentUser(client, hashedID)
		if err != nil {
			util.Warning("error sending hashedID to segment: %v", err)
		}

		for _, project := range projects {
			err = ddevapp.SegmentEvent(client, hashedID, "start", project)
			if err != nil {
				util.Warning("error sending event to segment: %v", err)
			}
		}
		err = client.Close()
		if err != nil {
			util.Warning("segment analytics client.close() failed: %v", err)
		}

	}

}

func init() {
	StartCmd.Flags().BoolVarP(&startAll, "all", "a", false, "Start all stopped projects")
	RootCmd.AddCommand(StartCmd)
}

func instrumentationNotSetUpWarning() {
	if version.SentryDSN == "" && globalconfig.DdevGlobalConfig.InstrumentationOptIn {
		output.UserOut.Warning("Instrumentation is opted in, but SentryDSN is not available.")
	}
	if version.SegmentKey == "" && globalconfig.DdevGlobalConfig.InstrumentationOptIn {
		output.UserOut.Warning("Instrumentation is opted in, but SegmentKey is not available.")
	}
}

// checkDdevVersionAndOptInInstrumentation() reads global config and checks to see if current version is different
// from the last saved version. If it is, prompt to request anon ddev usage stats
// and update the info.
func checkDdevVersionAndOptInInstrumentation() error {
	if !output.JSONOutput && version.COMMIT != globalconfig.DdevGlobalConfig.LastUsedVersion && globalconfig.DdevGlobalConfig.InstrumentationOptIn == false && !globalconfig.DdevNoSentry {
		allowStats := util.Confirm("It looks like you have a new ddev release.\nMay we send anonymous ddev usage statistics and errors?\nTo know what we will see please take a look at\nhttps://ddev.readthedocs.io/en/latest/users/cli-usage/#opt-in-usage-information\nPermission to beam up?")
		if allowStats {
			globalconfig.DdevGlobalConfig.InstrumentationOptIn = true
		}
		globalconfig.DdevGlobalConfig.LastUsedVersion = version.VERSION
		err := globalconfig.WriteGlobalConfig(globalconfig.DdevGlobalConfig)
		if err != nil {
			return err
		}
	}
	return nil
}
