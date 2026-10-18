package ddevapp

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	composeTypes "github.com/compose-spec/compose-go/v2/types"
	"github.com/ddev/ddev/pkg/dockerutil"
	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/nodeps"
)

// historyMountTarget is where a service's history directory is mounted.
const historyMountTarget = "/mnt/ddev-history"

// historyAnyUID is the DDEV_HISTORY_UID value meaning every container uid
// writes files the host user owns.
const historyAnyUID = "any"

// historyEnv maps each client's history variable to its file in historyMountTarget.
var historyEnv = map[string]string{
	"HISTFILE":          "bash_history",
	"MYSQL_HISTFILE":    "mysql_history",
	"NODE_REPL_HISTORY": "node_repl_history",
	"PSQL_HISTORY":      "psql_history",
	"PYTHON_HISTORY":    "python_history",
	"REDISCLI_HISTFILE": "rediscli_history",
	"SQLITE_HISTORY":    "sqlite_history",
}

// GetHistoryDir returns the host directory holding every service's history.
func (app *DdevApp) GetHistoryDir() string {
	return app.GetConfigPath(".history")
}

// ensureHistoryDir creates the history directory as the host user, so that
// Docker never creates it as root, and keeps its contents out of git.
func (app *DdevApp) ensureHistoryDir() error {
	dir := app.GetHistoryDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, ".gitignore"), []byte("# Shell history can hold passwords and tokens\n*\n"), 0644)
}

// historyUID returns the container uid whose bind-mounted files the host user
// owns in service, or historyAnyUID when the provider maps ownership for all.
func historyUID(service composeTypes.ServiceConfig) string {
	if !nodeps.IsLinux() {
		return historyAnyUID
	}
	if dockerutil.IsRootless() && service.UserNSMode != "keep-id" {
		return "0"
	}
	uid, _, _ := dockerutil.GetContainerUser()
	return uid
}

// addHistoryMount bind-mounts the service's history directory and points
// client history files into it, only when the files written there will
// belong to the host user.
func (app *DdevApp) addHistoryMount(name string, service *composeTypes.ServiceConfig) error {
	if globalconfig.DdevGlobalConfig.NoBindMounts {
		return nil
	}
	var xDdev XDdevExtension
	if _, err := service.Extensions.Get("x-ddev", &xDdev); err == nil && xDdev.OmitHistory {
		return nil
	}
	for _, vol := range service.Volumes {
		if vol.Target == historyMountTarget {
			return nil
		}
	}
	uid := historyUID(*service)
	serviceUID, _, _ := strings.Cut(service.User, ":")
	if uid != historyAnyUID && serviceUID != uid {
		return nil
	}

	if err := app.ensureHistoryDir(); err != nil {
		return fmt.Errorf("unable to create history directory: %v", err)
	}
	source := filepath.Join(app.GetHistoryDir(), name)
	if err := os.MkdirAll(source, 0755); err != nil {
		return fmt.Errorf("unable to create history directory: %v", err)
	}
	service.Volumes = append(service.Volumes, composeTypes.ServiceVolumeConfig{
		Type:   composeTypes.VolumeTypeBind,
		Source: source,
		Target: historyMountTarget,
		Bind:   &composeTypes.ServiceVolumeBind{CreateHostPath: false},
	})

	if service.Environment == nil {
		service.Environment = composeTypes.MappingWithEquals{}
	}
	service.Environment["DDEV_HISTORY_UID"] = &uid
	for envVar, file := range historyEnv {
		if service.Environment[envVar] == nil {
			p := historyMountTarget + "/" + file
			service.Environment[envVar] = &p
		}
	}
	return nil
}

// historyChownArgs returns the bind mount and command that give the host user
// back anything in the history directory a container wrote as another uid.
// The uid is the one the host user maps to inside the repair container.
func (app *DdevApp) historyChownArgs() (mount string, cmd string, ok bool) {
	if !nodeps.IsLinux() || globalconfig.DdevGlobalConfig.NoBindMounts {
		return "", "", false
	}
	if _, err := os.Stat(app.GetHistoryDir()); err != nil {
		return "", "", false
	}
	uid, gid, _ := dockerutil.GetContainerUser()
	if dockerutil.IsRootless() && !dockerutil.UseKeepID() {
		uid, gid = "0", "0"
	}
	const target = "/mnt/ddev-history-host"
	mount = app.GetHistoryDir() + ":" + target
	if dockerutil.IsSELinux() {
		mount += ":z"
	}
	return mount, fmt.Sprintf("chown -R %s:%s %s", uid, gid, target), true
}
