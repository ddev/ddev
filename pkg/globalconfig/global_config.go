package globalconfig

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	configTypes "github.com/ddev/ddev/pkg/config/types"
	"github.com/ddev/ddev/pkg/globalconfig/types"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/ddev/ddev/pkg/output"
	"github.com/ddev/ddev/pkg/versionconstants"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

// DdevGlobalConfigName is the name of the global config file.
const DdevGlobalConfigName = "global_config.yaml"

var (
	// DdevGlobalConfig is the currently active global configuration struct
	DdevGlobalConfig GlobalConfig
)

type ProjectInfo struct {
	AppRoot       string   `yaml:"approot"`
	UsedHostPorts []string `yaml:"used_host_ports,omitempty,flow"`
}

// GlobalConfig is the struct defining ddev's global config
type GlobalConfig struct {
	OmitContainersGlobal             []string                    `yaml:"omit_containers,flow"`
	PerformanceMode                  configTypes.PerformanceMode `yaml:"performance_mode"`
	InstrumentationOptIn             bool                        `yaml:"instrumentation_opt_in"`
	InstrumentationQueueSize         int                         `yaml:"instrumentation_queue_size,omitempty"`
	InstrumentationReportingInterval time.Duration               `yaml:"instrumentation_reporting_interval,omitempty"`
	RouterBindAllInterfaces          bool                        `yaml:"router_bind_all_interfaces"`
	InternetDetectionTimeout         int64                       `yaml:"internet_detection_timeout_ms"`
	DeveloperMode                    bool                        `yaml:"developer_mode,omitempty"`
	InstrumentationUser              string                      `yaml:"instrumentation_user,omitempty"`
	LastStartedVersion               string                      `yaml:"last_started_version"`
	UseHardenedImages                bool                        `yaml:"use_hardened_images"`
	UseLetsEncrypt                   bool                        `yaml:"use_letsencrypt"`
	LetsEncryptEmail                 string                      `yaml:"letsencrypt_email"`
	AutoRestartContainers            bool                        `yaml:"auto_restart_containers"`
	FailOnHookFailGlobal             bool                        `yaml:"fail_on_hook_fail"`
	WebEnvironment                   []string                    `yaml:"web_environment"`
	DisableHTTP2                     bool                        `yaml:"disable_http2"`
	TableStyle                       string                      `yaml:"table_style"`
	SimpleFormatting                 bool                        `yaml:"simple_formatting"`
	RequiredDockerComposeVersion     string                      `yaml:"required_docker_compose_version,omitempty"`
	UseDockerComposeFromPath         bool                        `yaml:"use_docker_compose_from_path,omitempty"`
	MkcertCARoot                     string                      `yaml:"mkcert_caroot"`
	ProjectTldGlobal                 string                      `yaml:"project_tld"`
	XdebugIDELocation                string                      `yaml:"xdebug_ide_location"`
	NoBindMounts                     bool                        `yaml:"no_bind_mounts"`
	Router                           string                      `yaml:"router"`
	TraefikMonitorPort               string                      `yaml:"traefik_monitor_port,omitempty"`
	WSL2NoWindowsHostsMgt            bool                        `yaml:"wsl2_no_windows_hosts_mgt"`
	RouterHTTPPort                   string                      `yaml:"router_http_port"`
	RouterHTTPSPort                  string                      `yaml:"router_https_port"`
	Messages                         MessagesConfig              `yaml:"messages,omitempty"`
	RemoteConfig                     RemoteConfig                `yaml:"remote_config,omitempty"`
	ProjectList                      map[string]*ProjectInfo     `yaml:"project_info"`
}

// New() returns a default GlobalConfig
func New() GlobalConfig {

	cfg := GlobalConfig{
		RequiredDockerComposeVersion: versionconstants.RequiredDockerComposeVersionDefault,
		InternetDetectionTimeout:     nodeps.InternetDetectionTimeoutDefault,
		TableStyle:                   "default",
		RouterHTTPPort:               nodeps.DdevDefaultRouterHTTPPort,
		RouterHTTPSPort:              nodeps.DdevDefaultRouterHTTPSPort,
		LastStartedVersion:           "v0.0",
		NoBindMounts:                 nodeps.NoBindMountsDefault,
		Router:                       types.RouterTypeDefault,
		MkcertCARoot:                 readCAROOT(),
		ProjectList:                  make(map[string]*ProjectInfo),
		TraefikMonitorPort:           nodeps.TraefikMonitorPortDefault,
	}

	return cfg
}

// Make sure the global configuration has been initialized
func EnsureGlobalConfig() {
	DdevGlobalConfig = New()
	err := ReadGlobalConfig()
	if err != nil {
		output.UserErr.Fatalf("unable to read global config: %v", err)
	}
}

// GetGlobalConfigPath gets the path to global config file
func GetGlobalConfigPath() string {
	return filepath.Join(GetGlobalDdevDir(), DdevGlobalConfigName)
}

// GetDDEVBinDir returns the directory of the mutagen config and binary
func GetDDEVBinDir() string {
	return filepath.Join(GetGlobalDdevDir(), "bin")
}

// GetMutagenPath gets the full path to the mutagen binary
func GetMutagenPath() string {
	mutagenBinary := "mutagen"
	if runtime.GOOS == "windows" {
		mutagenBinary = mutagenBinary + ".exe"
	}
	return filepath.Join(GetDDEVBinDir(), mutagenBinary)
}

// GetMutagenDataDirectory gets the full path to the MUTAGEN_DATA_DIRECTORY
func GetMutagenDataDirectory() string {
	currentMutagenDataDirectory := os.Getenv("MUTAGEN_DATA_DIRECTORY")
	if currentMutagenDataDirectory != "" {
		return currentMutagenDataDirectory
	}
	// If it's not already set, return ~/.ddev_mutagen_data_directory
	// This may be affected by tests that change $HOME
	return GetGlobalDdevDir() + "_" + "mutagen_data_directory"
}

// GetDockerComposePath gets the full path to the docker-compose binary
// Normally this is the one that has been downloaded to ~/.ddev/bin, but if
// UseDockerComposeFromPath, then it will be whatever if found in $PATH
func GetDockerComposePath() (string, error) {
	if DdevGlobalConfig.UseDockerComposeFromPath {
		executableName := "docker-compose"
		path, err := exec.LookPath(executableName)
		if err != nil {
			return "", fmt.Errorf("no docker-compose")
		}
		return path, nil
	}
	composeBinary := "docker-compose"
	if runtime.GOOS == "windows" {
		composeBinary = composeBinary + ".exe"
	}
	return filepath.Join(GetDDEVBinDir(), composeBinary), nil
}

// GetTableStyle returns the configured (string) table style
func GetTableStyle() string {
	return DdevGlobalConfig.TableStyle
}

// ValidateGlobalConfig validates global config
func ValidateGlobalConfig() error {
	if !IsValidOmitContainers(DdevGlobalConfig.OmitContainersGlobal) {
		return fmt.Errorf("invalid omit_containers: %s, must contain only %s", strings.Join(DdevGlobalConfig.OmitContainersGlobal, ","), strings.Join(GetValidOmitContainers(), ",")).(InvalidOmitContainers)
	}

	if !types.IsValidRouterType(DdevGlobalConfig.Router) {
		return fmt.Errorf("Invalid router: %s, valid router types are %v", DdevGlobalConfig.Router, types.GetValidRouterTypes())
	}

	if !IsValidTableStyle(DdevGlobalConfig.TableStyle) {
		DdevGlobalConfig.TableStyle = "default"
	}

	if !IsValidXdebugIDELocation(DdevGlobalConfig.XdebugIDELocation) {
		return fmt.Errorf(`xdebug_ide_location must be IP address or one of %v`, ValidXdebugIDELocations)
	}

	if DdevGlobalConfig.DisableHTTP2 && DdevGlobalConfig.IsTraefikRouter() {
		return fmt.Errorf("disable_http2 and router = traefik are mutually incompatible, as Traefik does not support disabling HTTP2")
	}

	if DdevGlobalConfig.IsTraefikRouter() && (DdevGlobalConfig.UseLetsEncrypt || DdevGlobalConfig.LetsEncryptEmail != "") {
		return fmt.Errorf("use-letsencrypt is not directly supported with traefik. but can be configured with custom config, see https://doc.traefik.io/traefik/https/acme/")
	}

	return nil
}

// ReadGlobalConfig reads the global config file into DdevGlobalConfig
// Or creates the file
func ReadGlobalConfig() error {
	globalConfigFile := GetGlobalConfigPath()

	// Can't use fileutil.FileExists() here because of import cycle.
	if _, err := os.Stat(globalConfigFile); err != nil {
		// ~/.ddev doesn't exist and running as root (only ddev hostname could do this)
		// Then create global config.
		if os.Geteuid() == 0 {
			logrus.Warning("not reading global config file because running with root privileges")
			return nil
		}
		if os.IsNotExist(err) {
			DdevGlobalConfig = New()
			err := WriteGlobalConfig(DdevGlobalConfig)
			if err != nil {
				return err
			}
		} else {
			return err
		}
	}

	source, err := os.ReadFile(globalConfigFile)
	if err != nil {
		return fmt.Errorf("unable to read ddev global config file %s: %v", source, err)
	}

	// ReadConfig config values from file.
	err = yaml.Unmarshal(source, &DdevGlobalConfig)
	if err != nil {
		return err
	}

	caRootEnv := os.Getenv("CAROOT")
	if GetCAROOT() == "" || !fileExists(filepath.Join(DdevGlobalConfig.MkcertCARoot, "rootCA.pem")) || (caRootEnv != "" && caRootEnv != DdevGlobalConfig.MkcertCARoot) {
		DdevGlobalConfig.MkcertCARoot = readCAROOT()
	}

	// If they set the internetdetectiontimeout below default, just reset to default
	// and ignore the setting.
	if DdevGlobalConfig.InternetDetectionTimeout < nodeps.InternetDetectionTimeoutDefault {
		DdevGlobalConfig.InternetDetectionTimeout = nodeps.InternetDetectionTimeoutDefault
	}

	// It's possible to have had pre-existing `router_http_port: ""` or `traefik_monitor_port`, if
	// so we have to override that.
	if DdevGlobalConfig.RouterHTTPPort == "" {
		DdevGlobalConfig.RouterHTTPPort = nodeps.DdevDefaultRouterHTTPPort
	}
	if DdevGlobalConfig.RouterHTTPSPort == "" {
		DdevGlobalConfig.RouterHTTPSPort = nodeps.DdevDefaultRouterHTTPSPort
	}
	if DdevGlobalConfig.TraefikMonitorPort == "" {
		DdevGlobalConfig.TraefikMonitorPort = nodeps.TraefikMonitorPortDefault
	}

	// Remove dba
	if nodeps.ArrayContainsString(DdevGlobalConfig.OmitContainersGlobal, "dba") {
		DdevGlobalConfig.OmitContainersGlobal = nodeps.RemoveItemFromSlice(DdevGlobalConfig.OmitContainersGlobal, "dba")
	}

	err = ValidateGlobalConfig()
	if err != nil {
		return err
	}

	return nil
}

// WriteGlobalConfig writes the global config into ~/.ddev.
func WriteGlobalConfig(config GlobalConfig) error {
	err := ValidateGlobalConfig()
	if err != nil {
		return err
	}

	cfgCopy := config
	// Remove some items that are defaults
	if cfgCopy.RequiredDockerComposeVersion == versionconstants.RequiredDockerComposeVersionDefault {
		cfgCopy.RequiredDockerComposeVersion = ""
	}

	// Overwrite PerformanceMode with effective value if empty.
	if cfgCopy.PerformanceMode == configTypes.PerformanceModeEmpty {
		cfgCopy.PerformanceMode = cfgCopy.GetPerformanceMode()
	}

	cfgbytes, err := yaml.Marshal(cfgCopy)
	if err != nil {
		return err
	}

	// Append current image information
	instructions := `
# You can turn off usage of the
# ddev-ssh-agent and ddev-router containers with
# omit_containers["ddev-ssh-agent", "ddev-router"]

# You can opt in or out of sending instrumentation to the ddev developers with
# instrumentation_opt_in: true # or false
#
# Maximum number of collected events in the local queue. If reached, collected
# data is sent.
# instrumentation_queue_size: 100
#
# Reporting interval in hours. If the last report was longer ago, collected
# data is sent.
# instrumentation_reporting_interval: 24

# performance_mode: "<default for the OS>"
# DDEV offers performance optimization strategies to improve the filesystem
# performance depending on your host system. Can be overridden with the project
# config.
#
# Possible values are:
#   - "none":    disables performance optimization.
#   - "mutagen": enables Mutagen.
#   - "nfs":     enables NFS.
#
# See https://ddev.readthedocs.io/en/latest/users/install/performance/#mutagen
# and https://ddev.readthedocs.io/en/latest/users/install/performance/#nfs.

# You can set the global project_tld. This way any project will use this tld. If not
# set the local project_tld is used, or the default of ddev.
# project_tld: ""
#
# You can inject environment variables into the web container with:
# web_environment:
# - SOMEENV=somevalue
# - SOMEOTHERENV=someothervalue

# Adjust the default table style used in ddev list and describe
# table_style: default
# table_style: bold
# table_style: bright

# Require simpler formatting where possible
# simpler_formatting: false

# In unusual cases the default value to wait to detect internet availability is too short.
# You can adjust this value higher to make it less likely that ddev will declare internet
# unavailable, but ddev may wait longer on some commands. This should not be set below the default 3000
# ddev will ignore low values, as they're not useful
# internet_detection_timeout_ms: 3000

# You can enable 'ddev start' to be interrupted by a failing hook with
# fail_on_hook_fail: true

# router: traefik # or nginx-proxy
# Traefik router is default, but you can switch to the legacy "nginx-proxy" router.

# router_http_port: <port>  # Port to be used for http (defaults to 80)
# router_https_port: <port> # Port for https (defaults to 443)

# disable_http2: false
# Disable http2 on ddev-router if true

# instrumentation_user: <your_username> # can be used to give ddev specific info about who you are
# developer_mode: true # (defaults to false) is not used widely at this time.
# router_bind_all_interfaces: false  # (defaults to false)
#    If true, ddev-router will bind http/s and MailHog ports on all
#    network interfaces instead of just localhost, so others on your local network can
#    access those ports. Note that this exposes the MailHog ports as well, which
#    can be a major security issue, so choose wisely.

# use_hardened_images: false
# With hardened images a container that is exposed to the internet is
# a harder target, although not as hard as a fully-secured host.
# sudo is removed, mailhog is removed, and since the web container
# is run only as the owning user, only project files might be changed
# if a CMS or PHP bug allowed creating or altering files, and
# permissions should not allow escalation.
#
# xdebug_ide_location: 
# In some cases, especially WSL2, the IDE may be set up different ways
# For example, if in WSL2 PhpStorm is running the Linux version inside WSL2
# or if using JetBrains Gateway
# then set xdebug_ide_location: WSL2
# If using vscode language server, which listens inside the container
# then set xdebug_ide_location: container

# Lets Encrypt:
# This integration is entirely experimental; your mileage may vary.
# * Your host must be directly internet-connected.
# * DNS for the hostname must be set to point to the host in question
# * You must have router_bind_all_interfaces: true or else the Let's Encrypt certbot
#   process will not be able to process the IP address of the host (and nobody will be able to access your site)
# * You will need to add a startup script to start your sites after a host reboot.
# * If using several sites at a single top-level domain, you'll probably want to set
#   project_tld to that top-level domain. Otherwise, you can use additional-hostnames or
#   additional_fqdns.
#
# use_letsencrypt: false
# (Experimental, only useful on an internet-based server)
# Set to true if certificates are to be obtained via certbot on https://letsencrypt.org/

# letsencrypt_email: <email>
# Email to be used for experimental letsencrypt certificates

# auto_restart_containers: false
# Experimental
# If true, attempt to automatically restart projects/containers after reboot or docker restart.

# fail_on_hook_fail: false
# Decide whether 'ddev start' should be interrupted by a failing hook

# traefik_monitor_port: 10999
# Change this only if you're having conflicts with some 
# service that needs port 10999

# wsl2_no_windows_hosts_mgt: false
# On WSL2 by default the Windows-side hosts file (normally C:\Windows\system32\drivers\etc\hosts)
# is used for hosts file management, but doing that requires running sudo and ddev.exe on
# Windows side; you may not want this if you're running your browser in WSL2 or for
# various other reasons.

# required_docker_compose_version: ""
# This can be used to override the default required docker-compose version
# It should normally be left alone, but can be set to, for example, "v2.1.1"

# use_docker_compose_from_path: false
# This can be set to true to allow ddev to use whatever docker-compose is
# found in the $PATH instead of using the private docker-compose downloaded
# to ~/.ddev/bin/docker-compose.
# Please don't use this unless directed to do so

# messages:
#   ticker_interval: 20 // Interval in hours to show ticker messages, -1 disables the ticker
# Controls the display of the ticker messages.

# remote_config: # Intended for debugging only, should not be changed.
#   update_interval: 10 // Interval in hours to download the remote config
#   remote:
#     owner: ddev
#     repo: remote-config
#     ref: main
#     filepath: remote-config.jsonc
# Controls the download of the remote config. Please do not change.
`
	cfgbytes = append(cfgbytes, instructions...)

	err = os.WriteFile(GetGlobalConfigPath(), cfgbytes, 0644)
	if err != nil {
		return err
	}

	return nil
}

// GetGlobalDdevDir returns ~/.ddev, the global caching directory
func GetGlobalDdevDir() string {
	userHome, err := os.UserHomeDir()
	if err != nil {
		logrus.Fatal("could not get home directory for current user. is it set?")
	}
	ddevDir := filepath.Join(userHome, ".ddev")

	// Create the directory if it is not already present.
	if _, err := os.Stat(ddevDir); os.IsNotExist(err) {
		// If they happen to be running as root/sudo, we won't create the directory
		// but act like we did. This should only happen for ddev hostname, which
		// doesn't need config or access to this dir anyway.
		if os.Geteuid() == 0 {
			return ddevDir
		}
		err = os.MkdirAll(ddevDir, 0755)
		if err != nil {
			logrus.Fatalf("Failed to create required directory %s, err: %v", ddevDir, err)
		}
	}
	// config.yaml is not allowed in ~/.ddev, can only result in disaster
	// so remove it if it happens to be discovered globally
	badFile := filepath.Join(ddevDir, "config.yaml")
	if _, err := os.Stat(badFile); err == nil {
		_ = os.Remove(filepath.Join(badFile))
	}
	return ddevDir
}

// IsValidOmitContainers is a helper function to determine if the OmitContainers array is valid
func IsValidOmitContainers(containerList []string) bool {
	for _, containerName := range containerList {
		if _, ok := ValidOmitContainers[containerName]; !ok {
			return false
		}
	}
	return true
}

// GetValidOmitContainers is a helper function that returns a list of valid containers for OmitContainers.
func GetValidOmitContainers() []string {
	s := make([]string, 0, len(ValidOmitContainers))

	for p := range ValidOmitContainers {
		s = append(s, p)
	}

	return s
}

// HostPostIsAllocated returns the project name that has allocated
// the port, or empty string.
func HostPostIsAllocated(port string) string {
	for project, item := range DdevGlobalConfig.ProjectList {
		if nodeps.ArrayContainsString(item.UsedHostPorts, port) {
			return project
		}
	}
	return ""
}

// CheckHostPortsAvailable checks GlobalDdev UsedHostPorts to see if requested ports are available.
func CheckHostPortsAvailable(projectName string, ports []string) error {
	for _, port := range ports {
		allocatedProject := HostPostIsAllocated(port)
		if allocatedProject != projectName && allocatedProject != "" {
			return fmt.Errorf("host port %s has already been allocated to project %s", port, allocatedProject)
		}
	}
	return nil
}

// GetFreePort gets an ephemeral port currently available, but also not
// listed in DdevGlobalConfig.UsedHostPorts
func GetFreePort(localIPAddr string) (string, error) {
	// Limit tries arbitrarily. It will normally succeed on first try.
	for i := 1; i < 1000; i++ {
		// From https://github.com/phayes/freeport/blob/master/freeport.go#L8
		// Ignores that the actual listener may be on a docker toolbox interface,
		// so this is just a heuristic.
		addr, err := net.ResolveTCPAddr("tcp", "127.0.0.1:0")
		if err != nil {
			return "", err
		}

		l, err := net.ListenTCP("tcp", addr)
		if err != nil {
			return "", err
		}
		port := strconv.Itoa(l.Addr().(*net.TCPAddr).Port)
		// nolint: errcheck
		l.Close()

		// In the case of Docker Toolbox, the actual listening IP may be something else
		// like 192.168.99.100, so check that to make sure it's not currently occupied.
		conn, _ := net.Dial("tcp", localIPAddr+":"+port)
		if conn != nil {
			continue
		}

		if HostPostIsAllocated(port) != "" {
			continue
		}
		return port, nil
	}
	return "-1", fmt.Errorf("GetFreePort() failed to find a free port")

}

// ReservePorts adds the ProjectInfo if necessary and assigns the reserved ports
func ReservePorts(projectName string, ports []string) error {
	// If the project doesn't exist, add it.
	_, ok := DdevGlobalConfig.ProjectList[projectName]
	if !ok {
		DdevGlobalConfig.ProjectList[projectName] = &ProjectInfo{}
	}
	DdevGlobalConfig.ProjectList[projectName].UsedHostPorts = ports
	err := WriteGlobalConfig(DdevGlobalConfig)
	return err
}

// SetProjectAppRoot sets the approot in the ProjectInfo of global config
func SetProjectAppRoot(projectName string, appRoot string) error {
	// If the project doesn't exist, add it.
	_, ok := DdevGlobalConfig.ProjectList[projectName]
	if !ok {
		DdevGlobalConfig.ProjectList[projectName] = &ProjectInfo{}
	}
	// Can't use fileutil.FileExists because of import cycle.
	if _, err := os.Stat(appRoot); err != nil {
		return fmt.Errorf("project %s project root %s does not exist", projectName, appRoot)
	}
	if DdevGlobalConfig.ProjectList[projectName].AppRoot != "" && DdevGlobalConfig.ProjectList[projectName].AppRoot != appRoot {
		return fmt.Errorf("project %s project root is already set to %s, refusing to change it to %s; you can `ddev stop --unlist %s` and start again if the listed project root is in error", projectName, DdevGlobalConfig.ProjectList[projectName].AppRoot, appRoot, projectName)
	}
	DdevGlobalConfig.ProjectList[projectName].AppRoot = appRoot
	err := WriteGlobalConfig(DdevGlobalConfig)
	return err
}

// GetProject returns a project given name provided,
// or nil if not found.
func GetProject(projectName string) *ProjectInfo {
	project, ok := DdevGlobalConfig.ProjectList[projectName]
	if !ok {
		return nil
	}
	return project
}

// RemoveProjectInfo removes the ProjectInfo line for a project
func RemoveProjectInfo(projectName string) error {
	_, ok := DdevGlobalConfig.ProjectList[projectName]
	if ok {
		delete(DdevGlobalConfig.ProjectList, projectName)
		err := WriteGlobalConfig(DdevGlobalConfig)
		if err != nil {
			return err
		}
	}
	return nil
}

// GetGlobalProjectList returns the global project list map
func GetGlobalProjectList() map[string]*ProjectInfo {
	return DdevGlobalConfig.ProjectList
}

// GetCAROOT is just a wrapper on global config
func GetCAROOT() string {
	return DdevGlobalConfig.MkcertCARoot
}

// readCAROOT() verifies that the mkcert command is available and its CA keys readable.
// 1. Find out CAROOT
// 2. Look there to see if key/crt are readable
// 3. If not, see if mkcert is even available, return empty

func readCAROOT() string {
	_, err := exec.LookPath("mkcert")
	if err != nil {
		return ""
	}

	out, err := exec.Command("mkcert", "-CAROOT").Output()
	if err != nil {
		return ""
	}
	root := strings.Trim(string(out), "\r\n")
	if !fileIsReadable(filepath.Join(root, "rootCA-key.pem")) || !fileExists(filepath.Join(root, "rootCA.pem")) {
		return ""
	}

	return root
}

// fileIsReadable checks to make sure a file exists and is readable
// Copied from fileutil because of import cycles
func fileIsReadable(name string) bool {
	file, err := os.OpenFile(name, os.O_RDONLY, 0666)
	if err != nil {
		return false
	}
	file.Close()
	return true
}

// fileExists checks a file's existence
// Copied from fileutil because of import cycles
func fileExists(name string) bool {
	if _, err := os.Stat(name); err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}
	return true
}

// IsInternetActiveAlreadyChecked just flags whether it's been checked
var IsInternetActiveAlreadyChecked = false

// IsInternetActiveResult is the result of the check
var IsInternetActiveResult = false

// IsInternetActiveNetResolver wraps the standard DNS resolver.
// In order to override net.DefaultResolver with a stub, we have to define an
// interface on our own since there is none from the standard library.
var IsInternetActiveNetResolver interface {
	LookupHost(ctx context.Context, host string) (addrs []string, err error)
} = net.DefaultResolver

// IsInternetActive checks to see if we have a viable
// internet connection. It just tries a quick DNS query.
// This requires that the named record be query-able.
// This check will only be made once per command run.
func IsInternetActive() bool {
	// if this was already checked, return the result
	if IsInternetActiveAlreadyChecked {
		return IsInternetActiveResult
	}

	timeout := time.Duration(DdevGlobalConfig.InternetDetectionTimeout) * time.Millisecond
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Using a random URL is more conclusive, but it's more intrusive because
	// DNS may take some time, and it's really annoying.
	testURL := "test.ddev.site"
	addrs, err := IsInternetActiveNetResolver.LookupHost(ctx, testURL)

	// Internet is active (active == true) if both err and ctx.Err() were nil
	active := err == nil && ctx.Err() == nil
	if os.Getenv("DDEV_DEBUG") != "" {
		if active == false {
			output.UserErr.Println("Internet connection not detected, DNS may not work, see https://ddev.readthedocs.io/en/stable/users/basics/faq/ for info.")
		}
		output.UserErr.Debugf("IsInternetActive(): err=%v ctx.Err()=%v addrs=%v IsInternetactive==%v, testURL=%v internet_detection_timeout_ms=%dms\n", err, ctx.Err(), addrs, active, testURL, DdevGlobalConfig.InternetDetectionTimeout)
	}

	// remember the result to not call this twice
	IsInternetActiveAlreadyChecked = true
	IsInternetActiveResult = active

	return active
}

// IsTraefikRouter returns true if the router is traefik
func (c *GlobalConfig) IsTraefikRouter() bool {
	return c.Router == types.RouterTypeTraefik
}

// DockerComposeVersion is filled with the version we find for docker-compose
var DockerComposeVersion = ""

// GetRequiredDockerComposeVersion returns the version of docker-compose we need
// based on the compiled version, or overrides in globalconfig, like
// required_docker_compose_version and use_docker_compose_from_path
// In the case of UseDockerComposeFromPath there is no required version, so this
// will return empty string.
func GetRequiredDockerComposeVersion() string {
	v := DdevGlobalConfig.RequiredDockerComposeVersion
	switch {
	case DdevGlobalConfig.UseDockerComposeFromPath:
		v = ""
	case DdevGlobalConfig.RequiredDockerComposeVersion != "":
		v = DdevGlobalConfig.RequiredDockerComposeVersion
	}
	return v
}

// Return the traefik router URL
func GetRouterURL() string {
	routerURL := ""
	// Until we figure out how to configure this, use static value
	if DdevGlobalConfig.IsTraefikRouter() {
		routerURL = "http://127.0.0.1:" + DdevGlobalConfig.TraefikMonitorPort
	}
	return routerURL
}
