package cmd

import (
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"

	configTypes "github.com/ddev/ddev/pkg/config/types"
	"github.com/ddev/ddev/pkg/fileutil"
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
		omitContainers = strings.ReplaceAll(omitContainers, " ", "")
		if omitContainers == "" || omitContainers == `""` || omitContainers == `''` {
			globalconfig.DdevGlobalConfig.OmitContainersGlobal = []string{}
		} else {
			globalconfig.DdevGlobalConfig.OmitContainersGlobal = strings.Split(omitContainers, ",")
		}
		dirty = true
	}
	if cmd.Flag("omit-project-name-by-default").Changed {
		v, _ := cmd.Flags().GetBool("omit-project-name-by-default")
		globalconfig.DdevGlobalConfig.OmitProjectNameByDefault = v
		dirty = true
	}
	if cmd.Flag("web-environment").Changed {
		env := strings.TrimSpace(webEnvironmentGlobal)
		if env == "" || env == `""` || env == `''` {
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

	if cmd.Flag(configTypes.FlagXHProfModeName).Changed {
		xhprofMode, _ := cmd.Flags().GetString(configTypes.FlagXHProfModeName)

		if err := configTypes.CheckValidXHProfMode(xhprofMode, configTypes.ConfigTypeGlobal); err != nil {
			util.Error("%s. Not changing value of xhprof_mode option.", err)
		} else {
			globalconfig.DdevGlobalConfig.XHProfMode = xhprofMode
			dirty = true
		}
	}

	if cmd.Flag(configTypes.FlagXHProfModeResetName).Changed {
		xhprofModeReset, _ := cmd.Flags().GetBool(configTypes.FlagXHProfModeResetName)

		if xhprofModeReset {
			globalconfig.DdevGlobalConfig.XHProfMode = configTypes.XHProfModeEmpty
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

	if cmd.Flag("mailpit-http-port").Changed {
		val, _ := cmd.Flags().GetString("mailpit-http-port")
		globalconfig.DdevGlobalConfig.RouterMailpitHTTPPort = val
		dirty = true
	}
	if cmd.Flag("mailpit-https-port").Changed {
		val, _ := cmd.Flags().GetString("mailpit-https-port")
		globalconfig.DdevGlobalConfig.RouterMailpitHTTPSPort = val
		dirty = true
	}

	if cmd.Flag("traefik-monitor-port").Changed {
		val, _ := cmd.Flags().GetString("traefik-monitor-port")
		globalconfig.DdevGlobalConfig.TraefikMonitorPort = val
		dirty = true
	}

	if cmd.Flag("share-default-provider").Changed {
		val, _ := cmd.Flags().GetString("share-default-provider")
		globalconfig.DdevGlobalConfig.ShareDefaultProvider = val
		dirty = true
	}

	if cmd.Flag("no-tui").Changed {
		val, _ := cmd.Flags().GetBool("no-tui")
		globalconfig.DdevGlobalConfig.NoTUI = val
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

	v := reflect.ValueOf(globalconfig.DdevGlobalConfig)
	typeOfVal := v.Type()

	keys := make([]string, 0, v.NumField())
	valMap := map[string]string{}
	for i := 0; i < v.NumField(); i++ {
		tag := typeOfVal.Field(i).Tag.Get("yaml")
		parts := strings.Split(tag, ",")
		tag = parts[0]
		//name := typeOfVal.Field(i).Name
		fieldValue := v.Field(i).Interface()
		if tag != "build info" && tag != "web_environment" && tag != "project_info" && tag != "remote_config" && tag != "messages" && tag != "router" {
			tagWithDashes := strings.ReplaceAll(tag, "_", "-")
			valMap[tagWithDashes] = fmt.Sprintf("%v", fieldValue)
			keys = append(keys, tagWithDashes)
		}
	}

	// Add remote config URLs to the display
	valMap["remote-config-url"] = globalconfig.DdevGlobalConfig.RemoteConfig.RemoteConfigURL
	keys = append(keys, "remote-config-url")
	valMap["sponsorship-data-url"] = globalconfig.DdevGlobalConfig.RemoteConfig.SponsorshipDataURL
	keys = append(keys, "sponsorship-data-url")
	valMap["addon-data-url"] = globalconfig.DdevGlobalConfig.RemoteConfig.AddonDataURL
	keys = append(keys, "addon-data-url")
	valMap["remote-config-update-interval"] = fmt.Sprintf("%d", globalconfig.DdevGlobalConfig.RemoteConfig.UpdateInterval)
	keys = append(keys, "remote-config-update-interval")

	sort.Strings(keys)
	if !output.JSONOutput {
		for _, label := range keys {
			output.UserOut.Printf("%s=%v", label, valMap[label])
		}
	} else {
		output.UserOut.WithField("raw", valMap).Println("")
	}
}

func init() {
	configGlobalCommand.Flags().StringVarP(&omitContainers, "omit-containers", "", "", `For example, --omit-containers=ddev-ssh-agent or --omit-containers=""`)
	_ = configGlobalCommand.RegisterFlagCompletionFunc("omit-containers", configCompletionFuncWithCommas(globalconfig.GetValidOmitContainers()))
	configGlobalCommand.Flags().StringVarP(&webEnvironmentGlobal, "web-environment", "", "", `Set the environment variables in the web container: --web-environment="TYPO3_CONTEXT=Development,SOMEENV=someval"`)
	configGlobalCommand.Flags().StringVarP(&webEnvironmentGlobal, "web-environment-add", "", "", `Append environment variables to the web container: --web-environment-add="TYPO3_CONTEXT=Development,SOMEENV=someval"`)
	configGlobalCommand.Flags().BoolVarP(&instrumentationOptIn, "instrumentation-opt-in", "", true, "Whether to allow instrumentation reporting with --instrumentation-opt-in=true")
	_ = configGlobalCommand.RegisterFlagCompletionFunc("instrumentation-opt-in", configCompletionFunc([]string{"true", "false"}))
	configGlobalCommand.Flags().Bool("router-bind-all-interfaces", false, "Bind host router ports on all interfaces, not only on the localhost network interface")
	_ = configGlobalCommand.RegisterFlagCompletionFunc("router-bind-all-interfaces", configCompletionFunc([]string{"true", "false"}))
	configGlobalCommand.Flags().Int("internet-detection-timeout-ms", nodeps.InternetDetectionTimeoutDefault, "Increase timeout when checking internet timeout, in milliseconds")
	_ = configGlobalCommand.RegisterFlagCompletionFunc("internet-detection-timeout-ms", configCompletionFunc([]string{strconv.Itoa(nodeps.InternetDetectionTimeoutDefault)}))
	configGlobalCommand.Flags().Bool("use-letsencrypt", false, "Enables experimental Let's Encrypt integration, 'ddev config global --use-letsencrypt' or 'ddev config global --use-letsencrypt=false'")
	_ = configGlobalCommand.RegisterFlagCompletionFunc("use-letsencrypt", configCompletionFunc([]string{"true", "false"}))
	configGlobalCommand.Flags().String("letsencrypt-email", "", "Email associated with Let's Encrypt, 'ddev config global --letsencrypt-email=me@example.com'")
	configGlobalCommand.Flags().Bool("simple-formatting", false, "If true, use simple formatting for tables and implicitly set 'NO_COLOR=1'")
	_ = configGlobalCommand.RegisterFlagCompletionFunc("simple-formatting", configCompletionFunc([]string{"true", "false"}))
	configGlobalCommand.Flags().Bool("use-hardened-images", false, "If true, use more secure 'hardened' images for an actual internet deployment")
	_ = configGlobalCommand.RegisterFlagCompletionFunc("use-hardened-images", configCompletionFunc([]string{"true", "false"}))
	configGlobalCommand.Flags().Bool("fail-on-hook-fail", false, "If true, 'ddev start' will fail when a hook fails")
	_ = configGlobalCommand.RegisterFlagCompletionFunc("fail-on-hook-fail", configCompletionFunc([]string{"true", "false"}))
	configGlobalCommand.Flags().String(configTypes.FlagPerformanceModeName, configTypes.FlagPerformanceModeDefault, configTypes.FlagPerformanceModeDescription(configTypes.ConfigTypeGlobal))
	configGlobalCommand.Flags().Bool(configTypes.FlagPerformanceModeResetName, false, configTypes.FlagPerformanceModeResetDescription(configTypes.ConfigTypeGlobal))
	_ = configGlobalCommand.RegisterFlagCompletionFunc(configTypes.FlagPerformanceModeName, configCompletionFunc(configTypes.ValidPerformanceModeOptions(configTypes.ConfigTypeGlobal)))

	configGlobalCommand.Flags().String(configTypes.FlagXHProfModeName, configTypes.FlagXHProfModeDefault, configTypes.FlagXHProfModeDescription(configTypes.ConfigTypeGlobal))
	_ = configGlobalCommand.RegisterFlagCompletionFunc(configTypes.FlagXHProfModeName, configCompletionFunc(configTypes.ValidXHProfModeOptions(configTypes.ConfigTypeGlobal)))
	configGlobalCommand.Flags().Bool(configTypes.FlagXHProfModeResetName, false, configTypes.FlagXHProfModeResetDescription(configTypes.ConfigTypeGlobal))

	configGlobalCommand.Flags().Bool("omit-project-name-by-default", true, "If true, do not automatically write the 'name' field in the .ddev/config.yaml file")
	_ = configGlobalCommand.RegisterFlagCompletionFunc("omit-project-name-by-default", configCompletionFunc([]string{"true", "false"}))

	configGlobalCommand.Flags().String("table-style", "default", fmt.Sprintf(`Table style for "ddev list" and "ddev describe", possible values are "%s"`, strings.Join(globalconfig.ValidTableStyleList(), `", "`)))
	_ = configGlobalCommand.RegisterFlagCompletionFunc("table-style", configCompletionFunc(globalconfig.ValidTableStyleList()))
	configGlobalCommand.Flags().String("required-docker-compose-version", "", "Override default docker-compose version (used only in development testing)")
	_ = configGlobalCommand.Flags().MarkHidden("required-docker-compose-version")
	configGlobalCommand.Flags().String("project-tld", nodeps.DdevDefaultTLD, "Set the default top-level domain to be used for all projects, can be overridden by project configuration")
	_ = configGlobalCommand.RegisterFlagCompletionFunc("project-tld", configCompletionFunc([]string{nodeps.DdevDefaultTLD}))
	configGlobalCommand.Flags().Bool("use-docker-compose-from-path", false, fmt.Sprintf("If true, use docker-compose from path instead of private %s (used only in development testing)", fileutil.ShortHomeJoin(globalconfig.GetDDEVBinDir(), "docker-compose")))
	_ = configGlobalCommand.Flags().MarkHidden("use-docker-compose-from-path")
	configGlobalCommand.Flags().Bool("no-bind-mounts", false, "If true, don't use bind-mounts. Useful for environments like remote Docker where bind-mounts are impossible")
	_ = configGlobalCommand.RegisterFlagCompletionFunc("no-bind-mounts", configCompletionFunc([]string{"true", "false"}))
	configGlobalCommand.Flags().String("xdebug-ide-location", "", "For less usual IDE locations specify where the IDE is running for Xdebug to reach it (for advanced use only)")
	configGlobalCommand.Flags().Bool("wsl2-no-windows-hosts-mgt", false, "WSL2 only; make DDEV ignore Windows-side hosts file (for advanced use only)")
	configGlobalCommand.Flags().String("router-http-port", nodeps.DdevDefaultRouterHTTPPort, "The default router HTTP port for all projects, can be overridden by project configuration")
	_ = configGlobalCommand.RegisterFlagCompletionFunc("router-http-port", configCompletionFunc([]string{nodeps.DdevDefaultRouterHTTPPort}))
	configGlobalCommand.Flags().String("router-https-port", nodeps.DdevDefaultRouterHTTPSPort, "The default router HTTPS port for all projects, can be overridden by project configuration")
	_ = configGlobalCommand.RegisterFlagCompletionFunc("router-https-port", configCompletionFunc([]string{nodeps.DdevDefaultRouterHTTPSPort}))
	configGlobalCommand.Flags().String("mailpit-http-port", nodeps.DdevDefaultMailpitHTTPPort, "The default Mailpit HTTP port for all projects, can be overridden by project configuration")
	_ = configGlobalCommand.RegisterFlagCompletionFunc("mailpit-http-port", configCompletionFunc([]string{nodeps.DdevDefaultMailpitHTTPPort}))
	configGlobalCommand.Flags().String("mailpit-https-port", nodeps.DdevDefaultMailpitHTTPSPort, "The default Mailpit HTTPS port for all projects, can be overridden by project configuration")
	_ = configGlobalCommand.RegisterFlagCompletionFunc("mailpit-https-port", configCompletionFunc([]string{nodeps.DdevDefaultMailpitHTTPSPort}))
	configGlobalCommand.Flags().String("router", globalconfigTypes.RouterTypeTraefik, fmt.Sprintf("The only valid router types are %s", strings.Join(globalconfigTypes.GetValidRouterTypes(), ", ")))
	_ = configGlobalCommand.Flags().MarkDeprecated("router", "\nThe only router used now is traefik, so --router is no longer needed")
	_ = configGlobalCommand.Flags().MarkHidden("router")
	configGlobalCommand.Flags().String("traefik-monitor-port", nodeps.TraefikMonitorPortDefault, `Can be used to change the Traefik monitor port in case of port conflicts, for example "ddev config global --traefik-monitor-port=11999"`)
	_ = configGlobalCommand.RegisterFlagCompletionFunc("traefik-monitor-port", configCompletionFunc([]string{nodeps.TraefikMonitorPortDefault}))
	configGlobalCommand.Flags().String("share-default-provider", "", `The default share provider for all projects (ngrok, cloudflared, or custom), can be overridden by project configuration`)
	_ = configGlobalCommand.RegisterFlagCompletionFunc("share-default-provider", configCompletionFunc([]string{"ngrok", "cloudflared"}))
	configGlobalCommand.Flags().Bool("no-tui", false, "If true, disable the interactive TUI dashboard when running bare 'ddev'")
	_ = configGlobalCommand.RegisterFlagCompletionFunc("no-tui", configCompletionFunc([]string{"true", "false"}))
	ConfigCommand.AddCommand(configGlobalCommand)
}
