package docker

import (
	"testing"

	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/stretchr/testify/require"
)

func TestResolveImageTag(t *testing.T) {
	require.Equal(t, "v1.25.4", resolveImageTag("c202e92108", "v1.25.4"))
	require.Equal(t, "c202e92108", resolveImageTag("c202e92108", "20260814_rfay_docker_update_phase_2"))
	require.Equal(t, "c202e92108", resolveImageTag("c202e92108", ""))
	require.Equal(t, "abc123", resolveImageTag("abc123", "v1.25.4-15-gabcdef1"))

	require.Equal(t, "v1.25.4-rc1", resolveImageTag("c202e92108", "v1.25.4-rc1"))
	require.Equal(t, "v1.25.4-alpha1", resolveImageTag("c202e92108", "v1.25.4-alpha1"))
	require.Equal(t, "v1.26.0-beta2", resolveImageTag("c202e92108", "v1.26.0-beta2"))
	require.Equal(t, "c202e92108", resolveImageTag("c202e92108", "v1.25.4-dirty"))
	require.Equal(t, "c202e92108", resolveImageTag("c202e92108", "v1.25.4-preview1"))
}

func TestImageRepoDockerOrg(t *testing.T) {
	require.Equal(t, "ddev/ddev-webserver", imageRepo("ddev/ddev-webserver"))

	t.Setenv(dockerOrgEnvVar, "ddevhq")
	require.Equal(t, "ddevhq/ddev-webserver", imageRepo("ddev/ddev-webserver"))
	require.Equal(t, "postgres", imageRepo("postgres"))

	require.Regexp(t, `^ddevhq/ddev-webserver:`, GetWebImage())
	require.Regexp(t, `^ddevhq/ddev-dbserver-mariadb-11\.8:`, GetDBImage(nodeps.MariaDB, nodeps.MariaDB118))
	require.Regexp(t, `^ddevhq/ddev-ssh-agent:`, GetSSHAuthImage())
	require.Regexp(t, `^ddevhq/ddev-traefik-router:`, GetRouterImage())
	require.Regexp(t, `^ddevhq/ddev-xhgui:`, GetXhguiImage())
	require.Equal(t, "postgres:17", GetDBImage(nodeps.Postgres, nodeps.Postgres17))

	t.Setenv(dockerOrgEnvVar, "")
	require.Equal(t, "ddev/ddev-webserver", imageRepo("ddev/ddev-webserver"))
}
