package globalconfig

import (
	"fmt"
	"slices"
)

// ValidHookNames are the hook names allowed in config.yaml and global_config.yaml.
var ValidHookNames = []string{
	"pre-start",
	"post-start",
	"pre-import-db",
	"post-import-db",
	"pre-import-files",
	"post-import-files",
	"pre-composer",
	"post-composer",
	"pre-stop",
	"post-stop",
	"pre-config",
	"post-config",
	"pre-describe",
	"post-describe",
	"pre-exec",
	"post-exec",
	"pre-pause",
	"post-pause",
	"pre-pull",
	"post-pull",
	"pre-push",
	"post-push",
	"pre-share",
	"post-share",
	"pre-snapshot",
	"post-snapshot",
	"pre-delete-snapshot",
	"post-delete-snapshot",
	"pre-restore-snapshot",
	"post-restore-snapshot",
}

// ValidHookTasks are the task types a hook can run.
var ValidHookTasks = []string{
	"exec",
	"exec-host",
	"composer",
}

// ValidateHooks checks that every hook name and task type in hooks is supported.
func ValidateHooks(hooks map[string][]map[string]any) error {
	for hookName, tasks := range hooks {
		if !slices.Contains(ValidHookNames, hookName) {
			return fmt.Errorf("invalid hook %s", hookName)
		}
		for _, task := range tasks {
			if !slices.ContainsFunc(ValidHookTasks, func(name string) bool {
				_, ok := task[name]
				return ok
			}) {
				return fmt.Errorf("invalid task '%s' defined for hook %s", task, hookName)
			}
		}
	}
	return nil
}
