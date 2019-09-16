package cmd

import (
	"github.com/drud/ddev/pkg/globalconfig"
	"github.com/drud/ddev/pkg/output"
	"github.com/drud/ddev/pkg/util"
	"github.com/drud/ddev/pkg/version"
	"github.com/spf13/cobra"
	"strings"
)

var (
	instrumentationOptIn bool
	// omitContainers allows user to set value of omit_containers
	omitContainers string
)

// configGlobalCommand is the the `ddev config global` command
var configGlobalCommand *cobra.Command = &cobra.Command{
	Use:     "global [flags]",
	Short:   "Change global configuration",
	Example: "ddev config global --instrumentation-opt-in=false\nddev config global --omit-containers=dba,ddev-ssh-agent",
	Run:     handleGlobalConfig,
}

// handleGlobalConfig handles all the flag processing for global config
func handleGlobalConfig(cmd *cobra.Command, args []string) {
	err := globalconfig.ReadGlobalConfig()
	if err != nil {
		util.Failed("Unable to read global config file: %v", err)
	}

	if cmd.Flag("instrumentation-opt-in").Changed {
		globalconfig.DdevGlobalConfig.InstrumentationOptIn = instrumentationOptIn
		// Make sure that they don't get prompted again right after they opted out.
		globalconfig.DdevGlobalConfig.LastUsedVersion = version.VERSION
	}
	if cmd.Flag("omit-containers").Changed {
		omitContainers = strings.Replace(omitContainers, " ", "", -1)
		if omitContainers == "" {
			globalconfig.DdevGlobalConfig.OmitContainers = []string{}
		} else {
			globalconfig.DdevGlobalConfig.OmitContainers = strings.Split(omitContainers, ",")
		}
	}
	if cmd.Flag("router-bind-all-interfaces").Changed {
		globalconfig.DdevGlobalConfig.RouterBindAllInterfaces, _ = cmd.Flags().GetBool("router-bind-all-interfaces")
	}
	err = globalconfig.ValidateGlobalConfig()
	if err != nil {
		util.Failed("Invalid configuration in %s: %v", globalconfig.GetGlobalConfigPath(), err)
	}
	err = globalconfig.WriteGlobalConfig(globalconfig.DdevGlobalConfig)
	if err != nil {
		util.Failed("Failed to write global config: %v", err)
	}

	util.Success("Global configuration:")
	output.UserOut.Printf("instrumentation-opt-in=%v", globalconfig.DdevGlobalConfig.InstrumentationOptIn)
	output.UserOut.Printf("omit-containers=[%s]", strings.Join(globalconfig.DdevGlobalConfig.OmitContainers, ","))
	output.UserOut.Printf("router-bind-all-interfaces=%v", globalconfig.DdevGlobalConfig.RouterBindAllInterfaces)
}

func init() {
	configGlobalCommand.Flags().StringVarP(&omitContainers, "omit-containers", "", "", "omit-containers=dba,ddev-ssh-agent")
	configGlobalCommand.Flags().BoolVarP(&instrumentationOptIn, "instrumentation-opt-in", "", false, "instrmentation-opt-in=true")
	configGlobalCommand.Flags().Bool("router-bind-all-interfaces", false, "router-bind-all-interfaces-in=true")

	ConfigCommand.AddCommand(configGlobalCommand)
}
