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
	// webEnvironmentGlobal allows user to set value of environment in web container
	webEnvironmentGlobal string
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
		globalconfig.DdevGlobalConfig.LastStartedVersion = version.DdevVersion
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
	if cmd.Flag("web-environment").Changed {
		env := strings.Replace(webEnvironmentGlobal, " ", "", -1)
		if env == "" {
			globalconfig.DdevGlobalConfig.WebEnvironment = []string{}
		} else {
			globalconfig.DdevGlobalConfig.WebEnvironment = strings.Split(env, ",")
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

	if cmd.Flag("disable-http2").Changed {
		val, _ := cmd.Flags().GetBool("disable-http2")
		globalconfig.DdevGlobalConfig.DisableHTTP2 = val
		dirty = true
	}

	if cmd.Flag("use-letsencrypt").Changed {
		val, _ := cmd.Flags().GetBool("use-letsencrypt")
		globalconfig.DdevGlobalConfig.UseLetsEncrypt = val
		dirty = true
	}

	if cmd.Flag("letsencrypt-email").Changed {
		val, _ := cmd.Flags().GetString("letsencrypt-email")
		globalconfig.DdevGlobalConfig.LetsEncryptEmail = val
		dirty = true
	}

	if cmd.Flag("auto-restart-containers").Changed {
		val, _ := cmd.Flags().GetBool("auto-restart-containers")
		globalconfig.DdevGlobalConfig.AutoRestartContainers = val
		dirty = true
	}

	if cmd.Flag("use-hardened-images").Changed {
		val, _ := cmd.Flags().GetBool("use-hardened-images")
		globalconfig.DdevGlobalConfig.UseHardenedImages = val
		dirty = true
	}

	if cmd.Flag("fail-on-hook-fail").Changed {
		val, _ := cmd.Flags().GetBool("fail-on-hook-fail")
		globalconfig.DdevGlobalConfig.FailOnHookFailGlobal = val
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
	output.UserOut.Printf("web-environment=[%s]", strings.Join(globalconfig.DdevGlobalConfig.WebEnvironment, ","))
	output.UserOut.Printf("nfs-mount-enabled=%v", globalconfig.DdevGlobalConfig.NFSMountEnabledGlobal)

	output.UserOut.Printf("router-bind-all-interfaces=%v", globalconfig.DdevGlobalConfig.RouterBindAllInterfaces)
	output.UserOut.Printf("internet-detection-timeout-ms=%v", globalconfig.DdevGlobalConfig.InternetDetectionTimeout)
	output.UserOut.Printf("disable-http2=%v", globalconfig.DdevGlobalConfig.DisableHTTP2)
	output.UserOut.Printf("use-letsencrypt=%v", globalconfig.DdevGlobalConfig.UseLetsEncrypt)
	output.UserOut.Printf("letsencrypt-email=%v", globalconfig.DdevGlobalConfig.LetsEncryptEmail)
	output.UserOut.Printf("auto-restart-containers=%v", globalconfig.DdevGlobalConfig.AutoRestartContainers)
	output.UserOut.Printf("use-hardened-images=%v", globalconfig.DdevGlobalConfig.UseHardenedImages)
	output.UserOut.Printf("fail-on-hook-fail=%v", globalconfig.DdevGlobalConfig.FailOnHookFailGlobal)
}

func init() {
	configGlobalCommand.Flags().StringVarP(&omitContainers, "omit-containers", "", "", "For example, --omit-containers=dba,ddev-ssh-agent")
	configGlobalCommand.Flags().StringVarP(&webEnvironmentGlobal, "web-environment", "", "", `Add environment variables to web container: --web-environment="TYPO3_CONTEXT=Development,SOMEENV=someval"`)
	configGlobalCommand.Flags().Bool("nfs-mount-enabled", false, "Enable NFS mounting on all projects globally")
	configGlobalCommand.Flags().BoolVarP(&instrumentationOptIn, "instrumentation-opt-in", "", false, "instrmentation-opt-in=true")
	configGlobalCommand.Flags().Bool("router-bind-all-interfaces", false, "router-bind-all-interfaces=true")
	configGlobalCommand.Flags().Int("internet-detection-timeout-ms", 750, "Increase timeout when checking internet timeout, in milliseconds")
	configGlobalCommand.Flags().Bool("disable-http2", false, "Optionally disable http2 in ddev-router, 'ddev global --disable-http2' or `ddev global --disable-http2=false'")
	configGlobalCommand.Flags().Bool("use-letsencrypt", false, "Enables experimental Let's Encrypt integration, 'ddev global --use-letsencrypt' or `ddev global --use-letsencrypt=false'")
	configGlobalCommand.Flags().String("letsencrypt-email", "", "Email associated with Let's Encrypt, `ddev global --letsencrypt-email=me@example.com'")
	configGlobalCommand.Flags().Bool("auto-restart-containers", false, "If true, automatically restart containers after a reboot or docker restart")
	configGlobalCommand.Flags().Bool("use-hardened-images", false, "If true, use more secure 'hardened' images for an actual internet deployment.")
	configGlobalCommand.Flags().Bool("fail-on-hook-fail", false, "If true, 'ddev start' will fail when a hook fails.")

	ConfigCommand.AddCommand(configGlobalCommand)
}
