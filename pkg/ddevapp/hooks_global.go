package ddevapp

import (
	"fmt"
	"maps"
	"reflect"
	"slices"
	"strings"

	"github.com/ddev/ddev/pkg/dockerutil"
	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/moby/moby/api/types/container"
)

// hookTask is a task from either the project's hooks or the global hooks.
type hookTask struct {
	task   YAMLTask
	global bool
}

// loadGlobalHooks copies the hooks in global_config.yaml, already validated
// by ReadGlobalConfig, into app.GlobalHooks, separate from app.Hooks so
// WriteConfig never copies them into a project's config.yaml.
func (app *DdevApp) loadGlobalHooks() {
	app.GlobalHooks = nil
	hooks := globalconfig.DdevGlobalConfig.Hooks
	if len(hooks) == 0 {
		return
	}
	app.GlobalHooks = make(map[string][]YAMLTask, len(hooks))
	for name, tasks := range hooks {
		for _, t := range tasks {
			app.GlobalHooks[name] = append(app.GlobalHooks[name], YAMLTask(t))
		}
	}
}

// tasksForHook returns the tasks to run for hookName: global tasks first,
// then project tasks. A global task equal to a project task is dropped so a
// hook copied into both places runs once, at its project position.
func (app *DdevApp) tasksForHook(hookName string) []hookTask {
	var tasks []hookTask
	if !app.SkipGlobalHooks {
		for _, g := range app.GlobalHooks[hookName] {
			if !slices.ContainsFunc(app.Hooks[hookName], func(p YAMLTask) bool { return reflect.DeepEqual(g, p) }) {
				tasks = append(tasks, hookTask{task: g, global: true})
			}
		}
	}
	for _, p := range app.Hooks[hookName] {
		tasks = append(tasks, hookTask{task: p})
	}
	return tasks
}

// globalExecServiceMissing returns true if t is an exec task whose service
// is omitted or not running, so a global hook can be skipped in projects
// that don't have that service.
func (app *DdevApp) globalExecServiceMissing(t Task) (string, bool) {
	e, ok := t.(ExecTask)
	if !ok {
		return "", false
	}
	if slices.Contains(app.GetOmittedContainers(), e.service) {
		return e.service, true
	}
	if state, _ := dockerutil.GetContainerStateByName(GetContainerName(app, e.service)); state != container.StateRunning {
		return e.service, true
	}
	return "", false
}

// globalHooksSummary describes the global hooks that apply to this project,
// like "2 post-start, 1 pre-start", or returns "" when there are none.
func (app *DdevApp) globalHooksSummary() string {
	if app.SkipGlobalHooks {
		return ""
	}
	var parts []string
	for _, name := range slices.Sorted(maps.Keys(app.GlobalHooks)) {
		if n := len(app.GlobalHooks[name]); n > 0 {
			parts = append(parts, fmt.Sprintf("%d %s", n, name))
		}
	}
	return strings.Join(parts, ", ")
}
