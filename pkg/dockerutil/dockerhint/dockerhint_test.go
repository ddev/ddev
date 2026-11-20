package dockerhint

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDetectProvider(t *testing.T) {
	tests := []struct {
		goos, contextName, host string
		provider                Provider
		instance                string
	}{
		{"darwin", "lima-default", "unix:///Users/me/.lima/default/sock/docker.sock", ProviderLima, "default"},
		{"darwin", "lima-docker", "unix:///Users/me/.lima/docker/sock/docker.sock", ProviderLima, "docker"},
		{"darwin", "colima", "unix:///Users/me/.colima/default/docker.sock", ProviderColima, "default"},
		{"darwin", "colima-work", "unix:///Users/me/.colima/work/docker.sock", ProviderColima, "work"},
		{"darwin", "colima-work", "", ProviderColima, "work"},
		{"darwin", "orbstack", "unix:///Users/me/.orbstack/run/docker.sock", ProviderOrbStack, ""},
		{"darwin", "rancher-desktop", "unix:///Users/me/.rd/docker.sock", ProviderRancher, ""},
		{"darwin", "desktop-linux", "unix:///Users/me/.docker/run/docker.sock", ProviderDockerDesktop, ""},
		{"linux", "desktop-linux", "unix:///home/me/.docker/desktop/docker.sock", ProviderDockerDesktop, ""},
		{"windows", "desktop-linux", "npipe:////./pipe/dockerDesktopLinuxEngine", ProviderDockerDesktop, ""},
		{"windows", "default", "npipe:////./pipe/docker_engine", ProviderDockerDesktop, ""},
		{"darwin", "podman", "unix:///var/folders/xy/T/podman/podman-machine-default-api.sock", ProviderPodman, ""},
		{"linux", "rootless", "unix:///run/user/1000/docker.sock", ProviderRootless, ""},
		{"linux", "default", "unix:///var/run/docker.sock", ProviderDockerCE, ""},
		{"linux", "remote", "tcp://192.168.1.10:2376", ProviderRemote, ""},
		{"darwin", "default", "unix:///var/run/docker.sock", ProviderUnknown, ""},
	}
	for _, tc := range tests {
		provider, instance := DetectProvider(tc.goos, tc.contextName, tc.host)
		require.Equal(t, tc.provider, provider, "host=%s context=%s", tc.host, tc.contextName)
		require.Equal(t, tc.instance, instance, "host=%s context=%s", tc.host, tc.contextName)
	}
}

func TestConnectionHelp(t *testing.T) {
	const docs = "https://docs.ddev.com/en/stable/users/install/docker-installation/#troubleshooting-docker"
	notRunning := errors.New("dial unix /Users/me/.lima/default/sock/docker.sock: connect: no such file or directory")

	// The example from https://github.com/ddev/ddev/issues/8836
	help := ConnectionHelp(Input{
		GOOS:        "darwin",
		ContextName: "lima-default",
		Host:        "unix:///Users/me/.lima/default/sock/docker.sock",
		Err:         notRunning,
		DocsURL:     docs,
	})
	require.Contains(t, help, `Lima instance "default"`)
	require.Contains(t, help, "    limactl start default\n")
	require.Contains(t, help, "    docker context ls\n    docker context use <name>\n")
	require.Contains(t, help, "If that does not help, see "+docs)

	tests := []struct {
		name string
		in   Input
		want []string
		not  []string
	}{
		{
			name: "colima profile",
			in:   Input{GOOS: "darwin", ContextName: "colima-work", Host: "unix:///Users/me/.colima/work/docker.sock", Err: notRunning},
			want: []string{"    colima start --profile work\n"},
		},
		{
			name: "docker desktop on macOS",
			in:   Input{GOOS: "darwin", ContextName: "desktop-linux", Host: "unix:///Users/me/.docker/run/docker.sock", Err: notRunning},
			want: []string{"    open -a Docker\n"},
		},
		{
			name: "docker desktop on Windows",
			in:   Input{GOOS: "windows", ContextName: "desktop-linux", Host: "npipe:////./pipe/dockerDesktopLinuxEngine", Err: notRunning},
			want: []string{`Docker Desktop.exe"`},
		},
		{
			name: "docker ce on Linux",
			in:   Input{GOOS: "linux", ContextName: "default", Host: "unix:///var/run/docker.sock", Err: notRunning},
			want: []string{"    sudo systemctl start docker\n"},
			not:  []string{"WSL"},
		},
		{
			name: "docker ce on WSL2",
			in:   Input{GOOS: "linux", WSL2: true, ContextName: "default", Host: "unix:///var/run/docker.sock", Err: notRunning},
			want: []string{"WSL Integration", "    sudo systemctl start docker\n"},
		},
		{
			name: "socket permission",
			in:   Input{GOOS: "linux", ContextName: "default", Host: "unix:///var/run/docker.sock", Err: errors.New("permission denied while trying to connect to the docker API at unix:///var/run/docker.sock")},
			want: []string{"    sudo usermod -aG docker $USER\n"},
			not:  []string{"systemctl"},
		},
		{
			name: "missing context from DOCKER_CONTEXT",
			in:   Input{GOOS: "darwin", DockerCtxEnv: "gone", Err: errors.New(`context "gone": context not found: open /Users/me/.docker/contexts/meta/x/meta.json: no such file or directory`)},
			want: []string{"DOCKER_CONTEXT=gone", "    unset DOCKER_CONTEXT\n", "    docker context use <name>\n"},
			not:  []string{"start"},
		},
		{
			name: "DOCKER_HOST overrides context",
			in:   Input{GOOS: "windows", ContextName: "default", Host: "tcp://10.0.0.5:2375", DockerHostEnv: "tcp://10.0.0.5:2375", Err: errors.New("connection refused")},
			want: []string{"remote Docker host tcp://10.0.0.5:2375.", "    Remove-Item Env:DOCKER_HOST\n"},
			not:  []string{"docker context ls", `Docker context "default"`},
		},
		{
			name: "unknown provider",
			in:   Input{GOOS: "darwin", ContextName: "default", Host: "unix:///var/run/docker.sock", Err: notRunning},
			want: []string{"Start your Docker provider", "    docker context ls\n"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			help := ConnectionHelp(tc.in)
			for _, s := range tc.want {
				require.Contains(t, help, s)
			}
			for _, s := range tc.not {
				require.NotContains(t, help, s)
			}
		})
	}
}
