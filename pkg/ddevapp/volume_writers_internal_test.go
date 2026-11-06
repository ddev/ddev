package ddevapp

import (
	"fmt"
	"maps"
	"slices"
	"testing"

	"github.com/compose-spec/compose-go/v2/types"
	"github.com/ddev/ddev/pkg/dockerutil"
	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/docker/compose/v5/pkg/api"
	"github.com/stretchr/testify/require"
)

// knownSharedVolumes are the volumes still attached by more than one service
// in a way some providers refuse, with what keeps them that way.
var knownSharedVolumes = map[string]string{
	"ddev-ssh-agent_socket_dir": "a unix socket can't reach across VMs, so this needs a transport change",
}

// volumeAttachment is one service's mount of a named volume.
type volumeAttachment struct {
	service  string
	readOnly bool
}

// TestNamedVolumesHaveOneWriter renders two projects, the router, and the
// ssh-agent, and fails when a named volume is attached read-write by more than
// one service, or read-write by one and read-only by another. Docker allows
// both, but Apple container refuses them, so this catches a regression on any
// provider.
func TestNamedVolumesHaveOneWriter(t *testing.T) {
	origNoBindMounts := globalconfig.DdevGlobalConfig.NoBindMounts
	t.Cleanup(func() { globalconfig.DdevGlobalConfig.NoBindMounts = origNoBindMounts })

	for _, noBindMounts := range []bool{false, true} {
		t.Run(fmt.Sprintf("no_bind_mounts=%v", noBindMounts), func(t *testing.T) {
			globalconfig.DdevGlobalConfig.NoBindMounts = noBindMounts
			attachments := map[string][]volumeAttachment{}

			var apps []*DdevApp
			for _, suffix := range []string{"a", "b"} {
				app, err := NewApp(t.TempDir(), true)
				require.NoError(t, err)
				app.Name = fmt.Sprintf("volwriters-%v-%s", noBindMounts, suffix)
				t.Cleanup(app.RemoveGlobalProjectInfo)
				app.Type = nodeps.AppTypePHP
				require.NoError(t, app.WriteConfig())
				_ = app.DockerEnv()
				require.NoError(t, app.WriteDockerComposeYAML())
				addVolumeAttachments(attachments, app.ComposeYaml)
				apps = append(apps, app)
			}

			routerCompose, err := generateRouterCompose(apps)
			require.NoError(t, err)
			router, err := dockerutil.LoadComposeProject([]string{routerCompose}, api.ProjectLoadOptions{ProjectName: RouterComposeProjectName})
			require.NoError(t, err)
			addVolumeAttachments(attachments, router)

			sshCompose, err := apps[0].CreateSSHAuthComposeFile()
			require.NoError(t, err)
			sshAgent, err := dockerutil.LoadComposeProject([]string{sshCompose}, api.ProjectLoadOptions{ProjectName: SSHAuthName})
			require.NoError(t, err)
			addVolumeAttachments(attachments, sshAgent)

			var problems []string
			stillShared := map[string]bool{}
			for _, name := range slices.Sorted(maps.Keys(attachments)) {
				if !sharedUnsafely(attachments[name]) {
					continue
				}
				if _, ok := knownSharedVolumes[name]; ok {
					stillShared[name] = true
					continue
				}
				problems = append(problems, fmt.Sprintf("%s: %v", name, attachments[name]))
			}
			require.Empty(t, problems, "named volumes with more than one writer, or a writer and readers")
			// Apple container omits ddev-ssh-agent, so nothing shares its volume.
			if dockerutil.IsAppleContainer() {
				return
			}
			for key := range knownSharedVolumes {
				require.True(t, stillShared[key], "%s is no longer shared, so remove it from knownSharedVolumes", key)
			}
		})
	}
}

// addVolumeAttachments records every named volume mount in project.
func addVolumeAttachments(attachments map[string][]volumeAttachment, project *types.Project) {
	for _, service := range project.Services {
		for _, v := range service.Volumes {
			if v.Type != types.VolumeTypeVolume || v.Source == "" {
				continue
			}
			name := v.Source
			if vol, ok := project.Volumes[v.Source]; ok && vol.Name != "" {
				name = vol.Name
			}
			attachments[name] = append(attachments[name], volumeAttachment{service: project.Name + "/" + service.Name, readOnly: v.ReadOnly})
		}
	}
}

// sharedUnsafely reports a read-write attachment alongside any other one.
func sharedUnsafely(attached []volumeAttachment) bool {
	writers := 0
	for _, a := range attached {
		if !a.readOnly {
			writers++
		}
	}
	return writers > 1 || (writers == 1 && len(attached) > 1)
}
