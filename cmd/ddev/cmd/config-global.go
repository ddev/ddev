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

	dirty := false
	if cmd.Flag("instrumentation-opt-in").Changed {
		globalconfig.DdevGlobalConfig.InstrumentationOptIn = instrumentationOptIn
		// Make sure that they don't get prompted again right after they opted out.
		globalconfig.DdevGlobalConfig.LastStartedVersion = version.VERSION
		dirty = true
	}
	if cmd.Flag("omit-containers").Changed {
		omitContainers = strings.Replace(omitContainers, " ", "", -1)
		if omitContainers == "" {
			globalconfig.DdevGlobalConfig.OmitContainersGlobal = []string{}
		} else {
			globalconfig.DdevGlobalConfig.OmitContainersGlobal = strings.Split(omitContainers, ",")
		}
		dirty = true
	}
	if cmd.Flag("nfs-mount-enabled").Changed {
		globalconfig.DdevGlobalConfig.NFSMountEnabledGlobal, _ = cmd.Flags().GetBool("nfs-mount-enabled")
		dirty = true
	}

	if cmd.Flag("router-bind-all-interfaces").Changed {
		globalconfig.DdevGlobalConfig.RouterBindAllInterfaces, _ = cmd.Flags().GetBool("router-bind-all-interfaces")
		dirty = true
	}

	if cmd.Flag("internet-detection-timeout-ms").Changed {
		val, _ := cmd.Flags().GetInt("internet-detection-timeout-ms")
		globalconfig.DdevGlobalConfig.InternetDetectionTimeout = int64(val)
		dirty = true
	}

	if dirty {
		err = globalconfig.ValidateGlobalConfig()
		if err != nil {
			util.Failed("Invalid configuration in %s: %v", globalconfig.GetGlobalConfigPath(), err)
		}
		err = globalconfig.WriteGlobalConfig(globalconfig.DdevGlobalConfig)
		if err != nil {
			util.Failed("Failed to write global config: %v", err)
		}
	}
	util.Success("Global configuration:")
	output.UserOut.Printf("instrumentation-opt-in=%v", globalconfig.DdevGlobalConfig.InstrumentationOptIn)
	output.UserOut.Printf("omit-containers=[%s]", strings.Join(globalconfig.DdevGlobalConfig.OmitContainersGlobal, ","))
	output.UserOut.Printf("nfs-mount-enabled=%v", globalconfig.DdevGlobalConfig.NFSMountEnabledGlobal)

	output.UserOut.Printf("router-bind-all-interfaces=%v", globalconfig.DdevGlobalConfig.RouterBindAllInterfaces)
	output.UserOut.Printf("internet-detection-timeout-ms=%v", globalconfig.DdevGlobalConfig.InternetDetectionTimeout)
}

func init() {
	configGlobalCommand.Flags().StringVarP(&omitContainers, "omit-containers", "", "", "For example, --omit-containers=dba,ddev-ssh-agent")
	configGlobalCommand.Flags().Bool("nfs-mount-enabled", false, "Enable NFS mounting on all projects globally")
	configGlobalCommand.Flags().BoolVarP(&instrumentationOptIn, "instrumentation-opt-in", "", false, "instrmentation-opt-in=true")
	configGlobalCommand.Flags().Bool("router-bind-all-interfaces", false, "router-bind-all-interfaces=true")
	configGlobalCommand.Flags().Int("internet-detection-timeout-ms", 750, "Increase timeout when checking internet timeout, in milliseconds")
	ConfigCommand.AddCommand(configGlobalCommand)
}
