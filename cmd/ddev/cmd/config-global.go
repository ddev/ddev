package cmd

import (
	"fmt"
	"sort"
	"strings"

	configTypes "github.com/ddev/ddev/pkg/config/types"
	"github.com/ddev/ddev/pkg/globalconfig"
	globalconfigTypes "github.com/ddev/ddev/pkg/globalconfig/types"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/ddev/ddev/pkg/output"
	"github.com/ddev/ddev/pkg/util"
	"github.com/ddev/ddev/pkg/versionconstants"
	"github.com/spf13/cobra"
)

var (
	instrumentationOptIn bool
	// omitContainers allows user to set value of omit_containers
	omitContainers string
	// webEnvironmentGlobal allows user to set value of environment in web container
	webEnvironmentGlobal string
)

// configGlobalCommand is the the `ddev config global` command
var configGlobalCommand = &cobra.Command{
	Use:     "global [flags]",
	Short:   "Change global configuration",
	Example: "ddev config global --instrumentation-opt-in=false\nddev config global --omit-containers=ddev-ssh-agent",
	Run:     handleGlobalConfig,
}

// handleGlobalConfig handles all the flag processing for global config
func handleGlobalConfig(cmd *cobra.Command, _ []string) {
	err := globalconfig.ReadGlobalConfig()
	if err != nil {
		util.Failed("Unable to read global config file: %v", err)
	}

	dirty := false
	if cmd.Flag("instrumentation-opt-in").Changed {
		globalconfig.DdevGlobalConfig.InstrumentationOptIn = instrumentationOptIn
		// Make sure that they don't get prompted again right after they opted out.
		globalconfig.DdevGlobalConfig.LastStartedVersion = versionconstants.DdevVersion
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
		env := strings.TrimSpace(webEnvironmentGlobal)
		if env == "" {
			globalconfig.DdevGlobalConfig.WebEnvironment = []string{}
		} else {
			globalconfig.DdevGlobalConfig.WebEnvironment = strings.Split(env, ",")
		}
		dirty = true
	}

	if cmd.Flag("web-environment-add").Changed {
		env := strings.TrimSpace(webEnvironmentGlobal)
		if env == "" {
			globalconfig.DdevGlobalConfig.WebEnvironment = []string{}
		} else {
			envspl := strings.Split(env, ",")
			conc := append(globalconfig.DdevGlobalConfig.WebEnvironment, envspl...)
			// Convert to a hashmap to remove duplicate values.
			hashmap := make(map[string]string)
			for i := 0; i < len(conc); i++ {
				hashmap[conc[i]] = conc[i]
			}
			keys := []string{}
			for key := range hashmap {
				keys = append(keys, key)
			}
			globalconfig.DdevGlobalConfig.WebEnvironment = keys
			sort.Strings(globalconfig.DdevGlobalConfig.WebEnvironment)
		}
		dirty = true
	}

	if cmd.Flag("nfs-mount-enabled").Changed {
		if v, _ := cmd.Flags().GetBool("nfs-mount-enabled"); v {
			globalconfig.DdevGlobalConfig.SetPerformanceMode(configTypes.PerformanceModeNFS)
			dirty = true
		}
	}

	if cmd.Flag("mutagen-enabled").Changed {
		if v, _ := cmd.Flags().GetBool("mutagen-enabled"); v {
			globalconfig.DdevGlobalConfig.SetPerformanceMode(configTypes.PerformanceModeMutagen)
			dirty = true
		}
	}

	if cmd.Flag(configTypes.FlagPerformanceModeName).Changed {
		performanceMode, _ := cmd.Flags().GetString(configTypes.FlagPerformanceModeName)

		if err := configTypes.CheckValidPerformanceMode(performanceMode, configTypes.ConfigTypeGlobal); err != nil {
			util.Error("%s. Not changing value of performance_mode option.", err)
		} else {
			globalconfig.DdevGlobalConfig.SetPerformanceMode(performanceMode)
			dirty = true
		}
	}

	if cmd.Flag(configTypes.FlagPerformanceModeResetName).Changed {
		performanceModeReset, _ := cmd.Flags().GetBool(configTypes.FlagPerformanceModeResetName)

		if performanceModeReset {
			globalconfig.DdevGlobalConfig.SetPerformanceMode(configTypes.PerformanceModeEmpty)
			dirty = true
		}
	}

	if cmd.Flag("xdebug-ide-location").Changed {
		globalconfig.DdevGlobalConfig.XdebugIDELocation, _ = cmd.Flags().GetString("xdebug-ide-location")
		dirty = true
	}

	if cmd.Flag("router-bind-all-interfaces").Changed {
		globalconfig.DdevGlobalConfig.RouterBindAllInterfaces, _ = cmd.Flags().GetBool("router-bind-all-interfaces")
		dirty = true
	}

	if cmd.Flag("simple-formatting").Changed {
		globalconfig.DdevGlobalConfig.SimpleFormatting, _ = cmd.Flags().GetBool("simple-formatting")
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

	if cmd.Flag("table-style").Changed {
		val, _ := cmd.Flags().GetString("table-style")
		if nodeps.ArrayContainsString(globalconfig.ValidTableStyleList(), val) {
			globalconfig.DdevGlobalConfig.TableStyle = val
			dirty = true
		} else {
			util.Error("table-style=%s is not valid. Valid options include %s. Not changing table-style.\n", val, strings.Join(globalconfig.ValidTableStyleList(), ", "))
		}
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

	if cmd.Flag("required-docker-compose-version").Changed {
		val, _ := cmd.Flags().GetString("required-docker-compose-version")
		globalconfig.DdevGlobalConfig.RequiredDockerComposeVersion = val
		dirty = true
	}
	if cmd.Flag("project-tld").Changed {
		val, _ := cmd.Flags().GetString("project-tld")
		globalconfig.DdevGlobalConfig.ProjectTldGlobal = val
		dirty = true
	}

	if cmd.Flag("use-docker-compose-from-path").Changed {
		val, _ := cmd.Flags().GetBool("use-docker-compose-from-path")
		globalconfig.DdevGlobalConfig.UseDockerComposeFromPath = val
		dirty = true
	}

	if cmd.Flag("no-bind-mounts").Changed {
		val, _ := cmd.Flags().GetBool("no-bind-mounts")
		globalconfig.DdevGlobalConfig.NoBindMounts = val
		dirty = true
	}

	if cmd.Flag("use-traefik").Changed {
		if v, _ := cmd.Flags().GetBool("use-traefik"); v {
			globalconfig.DdevGlobalConfig.Router = globalconfigTypes.RouterTypeTraefik
			dirty = true
		}
	}

	if cmd.Flag("router").Changed {
		val, _ := cmd.Flags().GetString("router")
		globalconfig.DdevGlobalConfig.Router = val
		dirty = true
	}
	if cmd.Flag("wsl2-no-windows-hosts-mgt").Changed {
		val, _ := cmd.Flags().GetBool("wsl2-no-windows-hosts-mgt")
		globalconfig.DdevGlobalConfig.WSL2NoWindowsHostsMgt = val
		dirty = true
	}
	if cmd.Flag("router-http-port").Changed {
		val, _ := cmd.Flags().GetString("router-http-port")
		globalconfig.DdevGlobalConfig.RouterHTTPPort = val
		dirty = true
	}
	if cmd.Flag("router-https-port").Changed {
		val, _ := cmd.Flags().GetString("router-https-port")
		globalconfig.DdevGlobalConfig.RouterHTTPSPort = val
		dirty = true
	}
	if cmd.Flag("traefik-monitor-port").Changed {
		val, _ := cmd.Flags().GetString("traefik-monitor-port")
		globalconfig.DdevGlobalConfig.TraefikMonitorPort = val
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
	output.UserOut.Println("Global configuration:")
	output.UserOut.Printf("instrumentation-opt-in=%v", globalconfig.DdevGlobalConfig.InstrumentationOptIn)
	output.UserOut.Printf("omit-containers=[%s]", strings.Join(globalconfig.DdevGlobalConfig.OmitContainersGlobal, ","))
	output.UserOut.Printf("web-environment=[%s]", strings.Join(globalconfig.DdevGlobalConfig.WebEnvironment, ","))
	output.UserOut.Printf("%s=%v", configTypes.FlagPerformanceModeName, globalconfig.DdevGlobalConfig.GetPerformanceMode())

	output.UserOut.Printf("router-bind-all-interfaces=%v", globalconfig.DdevGlobalConfig.RouterBindAllInterfaces)
	output.UserOut.Printf("internet-detection-timeout-ms=%v", globalconfig.DdevGlobalConfig.InternetDetectionTimeout)
	output.UserOut.Printf("disable-http2=%v", globalconfig.DdevGlobalConfig.DisableHTTP2)
	output.UserOut.Printf("use-letsencrypt=%v", globalconfig.DdevGlobalConfig.UseLetsEncrypt)
	output.UserOut.Printf("letsencrypt-email=%v", globalconfig.DdevGlobalConfig.LetsEncryptEmail)
	output.UserOut.Printf("table-style=%v", globalconfig.DdevGlobalConfig.TableStyle)
	output.UserOut.Printf("simple-formatting=%v", globalconfig.DdevGlobalConfig.SimpleFormatting)
	output.UserOut.Printf("auto-restart-containers=%v", globalconfig.DdevGlobalConfig.AutoRestartContainers)
	output.UserOut.Printf("use-hardened-images=%v", globalconfig.DdevGlobalConfig.UseHardenedImages)
	output.UserOut.Printf("fail-on-hook-fail=%v", globalconfig.DdevGlobalConfig.FailOnHookFailGlobal)
	output.UserOut.Printf("required-docker-compose-version=%v", globalconfig.DdevGlobalConfig.RequiredDockerComposeVersion)
	output.UserOut.Printf("use-docker-compose-from-path=%v", globalconfig.DdevGlobalConfig.UseDockerComposeFromPath)
	output.UserOut.Printf("project-tld=%v", globalconfig.DdevGlobalConfig.ProjectTldGlobal)
	output.UserOut.Printf("xdebug-ide-location=%v", globalconfig.DdevGlobalConfig.XdebugIDELocation)
	output.UserOut.Printf("no-bind-mounts=%v", globalconfig.DdevGlobalConfig.NoBindMounts)
	output.UserOut.Printf("router=%v", globalconfig.DdevGlobalConfig.Router)
	output.UserOut.Printf("wsl2-no-windows-hosts-mgt=%v", globalconfig.DdevGlobalConfig.WSL2NoWindowsHostsMgt)
	output.UserOut.Printf("router-http-port=%v", globalconfig.DdevGlobalConfig.RouterHTTPPort)
	output.UserOut.Printf("router-https-port=%v", globalconfig.DdevGlobalConfig.RouterHTTPSPort)
	output.UserOut.Printf("traefik-monitor-port=%v", globalconfig.DdevGlobalConfig.TraefikMonitorPort)
}

func init() {
	configGlobalCommand.Flags().StringVarP(&omitContainers, "omit-containers", "", "", "For example, --omit-containers=ddev-ssh-agent")
	configGlobalCommand.Flags().StringVarP(&webEnvironmentGlobal, "web-environment", "", "", `Set the environment variables in the web container: --web-environment="TYPO3_CONTEXT=Development,SOMEENV=someval"`)
	configGlobalCommand.Flags().StringVarP(&webEnvironmentGlobal, "web-environment-add", "", "", `Append environment variables to the web container: --web-environment-add="TYPO3_CONTEXT=Development,SOMEENV=someval"`)
	configGlobalCommand.Flags().Bool("nfs-mount-enabled", false, "Enable NFS mounting on all projects globally")
	_ = configGlobalCommand.Flags().MarkDeprecated("nfs-mount-enabled", fmt.Sprintf("please use --%s instead", configTypes.FlagPerformanceModeName))
	configGlobalCommand.Flags().BoolVarP(&instrumentationOptIn, "instrumentation-opt-in", "", false, "instrumentation-opt-in=true")
	configGlobalCommand.Flags().Bool("router-bind-all-interfaces", false, "router-bind-all-interfaces=true")
	configGlobalCommand.Flags().Int("internet-detection-timeout-ms", 3000, "Increase timeout when checking internet timeout, in milliseconds")
	configGlobalCommand.Flags().Bool("disable-http2", false, "Optionally disable http2 in deprecated nginx-proxy ddev-router, 'ddev global --disable-http2' or `ddev global --disable-http2=false'")
	configGlobalCommand.Flags().Bool("use-letsencrypt", false, "Enables experimental Let's Encrypt integration, 'ddev global --use-letsencrypt' or `ddev global --use-letsencrypt=false'")
	configGlobalCommand.Flags().String("letsencrypt-email", "", "Email associated with Let's Encrypt, `ddev global --letsencrypt-email=me@example.com'")
	configGlobalCommand.Flags().Bool("auto-restart-containers", false, "If true, automatically restart containers after a reboot or docker restart")
	configGlobalCommand.Flags().Bool("simple-formatting", false, "If true, use simple formatting and no color for tables")
	configGlobalCommand.Flags().Bool("use-hardened-images", false, "If true, use more secure 'hardened' images for an actual internet deployment.")
	configGlobalCommand.Flags().Bool("fail-on-hook-fail", false, "If true, 'ddev start' will fail when a hook fails.")
	configGlobalCommand.Flags().Bool("mutagen-enabled", false, "If true, web container will use mutagen caching/asynchronous updates.")
	_ = configGlobalCommand.Flags().MarkDeprecated("mutagen-enabled", fmt.Sprintf("please use --%s instead", configTypes.FlagPerformanceModeName))
	configGlobalCommand.Flags().String(configTypes.FlagPerformanceModeName, configTypes.FlagPerformanceModeDefault, configTypes.FlagPerformanceModeDescription(configTypes.ConfigTypeGlobal))
	configGlobalCommand.Flags().Bool(configTypes.FlagPerformanceModeResetName, true, configTypes.FlagPerformanceModeResetDescription(configTypes.ConfigTypeGlobal))
	configGlobalCommand.Flags().String("table-style", "", "Table style for list and describe, see ~/.ddev/global_config.yaml for values")
	configGlobalCommand.Flags().String("required-docker-compose-version", "", "Override default docker-compose version (used only in development testing)")
	_ = configGlobalCommand.Flags().MarkHidden("required-docker-compose-version")
	configGlobalCommand.Flags().String("project-tld", "", "Override default project tld")
	configGlobalCommand.Flags().Bool("use-docker-compose-from-path", true, "If true, use docker-compose from path instead of private ~/.ddev/bin/docker-compose (used only in development testing)")
	_ = configGlobalCommand.Flags().MarkHidden("use-docker-compose-from-path")
	configGlobalCommand.Flags().Bool("no-bind-mounts", true, "If true, don't use bind-mounts - useful for environments like remote docker where bind-mounts are impossible")
	configGlobalCommand.Flags().String("xdebug-ide-location", "", "For less usual IDE locations specify where the IDE is running for Xdebug to reach it")
	configGlobalCommand.Flags().Bool("use-traefik", true, "If true, use traefik for ddev-router")
	_ = configGlobalCommand.Flags().MarkDeprecated("use-traefik", "please use --router instead")
	configGlobalCommand.Flags().String("router", globalconfigTypes.RouterTypeTraefik, fmt.Sprintf("Valid router types are %s, default is %s", strings.Join(globalconfigTypes.GetValidRouterTypes(), ", "), globalconfigTypes.RouterTypeDefault))
	configGlobalCommand.Flags().Bool("wsl2-no-windows-hosts-mgt", true, "WSL2 only; make DDEV ignore Windows-side hosts file")
	configGlobalCommand.Flags().String("router-http-port", "", "The default router HTTP port for all projects")
	configGlobalCommand.Flags().String("router-https-port", "", "The default router HTTPS port for all projects")
	configGlobalCommand.Flags().String("traefik-monitor-port", "", "The Traefik monitor port; can be changed in case of port conflicts")
	ConfigCommand.AddCommand(configGlobalCommand)
}
