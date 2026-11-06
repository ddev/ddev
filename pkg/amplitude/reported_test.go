package amplitude

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ddev/ddev/third_party/ampli"
	"github.com/stretchr/testify/require"
)

func TestAlreadyReported(t *testing.T) {
	file := filepath.Join(t.TempDir(), reportedFile)
	day1 := "2026-11-05"
	day2 := "2026-11-06"

	php83 := projectEvent("8.3")
	php84 := projectEvent("8.4")

	require.False(t, alreadyReported(file, day1, "project:p1", php83, false))
	require.True(t, alreadyReported(file, day1, "project:p1", php83, false))

	// Changed properties are sent, and switching back is not sent again.
	require.False(t, alreadyReported(file, day1, "project:p1", php84, false))
	require.True(t, alreadyReported(file, day1, "project:p1", php83, false))

	require.False(t, alreadyReported(file, day1, "project:p2", php83, false))

	require.False(t, alreadyReported(file, day2, "project:p1", php83, false))
	require.True(t, alreadyReported(file, day2, "project:p1", php83, false))
	require.False(t, alreadyReported(file, day2, "project:p2", php83, false))

	identify := ampli.Identify.Builder().DdevEnvironment("").DockerPlatform("docker-desktop").DockerVersion("28.0.0").Timezone("UTC").Build()
	require.False(t, alreadyReported(file, day2, "identify", identify, false))
	require.True(t, alreadyReported(file, day2, "identify", identify, false))

	newDocker := ampli.Identify.Builder().DdevEnvironment("").DockerPlatform("docker-desktop").DockerVersion("29.0.0").Timezone("UTC").Build()
	require.False(t, alreadyReported(file, day2, "identify", newDocker, false))
}

func TestAlreadyReportedCorruptFile(t *testing.T) {
	file := filepath.Join(t.TempDir(), reportedFile)
	require.NoError(t, os.WriteFile(file, []byte("not json"), 0644))

	event := projectEvent("8.3")
	require.False(t, alreadyReported(file, "2026-11-06", "project:p1", event, false))
	require.True(t, alreadyReported(file, "2026-11-06", "project:p1", event, false))
}

func projectEvent(phpVersion string) ampli.ProjectEvent {
	return ampli.Project.Builder().Id("p1").PerformanceMode("none").PhpVersion(phpVersion).ProjectType("php").WebserverType("nginx-fpm").XhProfMode("prepend").Build()
}

func TestAlreadyReportedUnchanged(t *testing.T) {
	file := filepath.Join(t.TempDir(), reportedFile)
	docker28 := identifyEvent("28.0.0")
	docker29 := identifyEvent("29.0.0")

	require.False(t, alreadyReported(file, "2026-11-05", "identify", docker28, true))
	require.True(t, alreadyReported(file, "2026-11-05", "identify", docker28, true))
	require.True(t, alreadyReported(file, "2026-11-06", "identify", docker28, true))

	require.False(t, alreadyReported(file, "2026-11-06", "identify", docker29, true))
	require.True(t, alreadyReported(file, "2026-11-20", "identify", docker29, true))

	// Only the latest state counts, so a change back is sent again.
	require.False(t, alreadyReported(file, "2026-11-20", "identify", docker28, true))

	// Daily hashes and latest hashes are kept apart.
	require.False(t, alreadyReported(file, "2026-11-20", "project:p1", projectEvent("8.3"), false))
	require.True(t, alreadyReported(file, "2026-11-20", "identify", docker28, true))
}

func identifyEvent(dockerVersion string) ampli.IdentifyEvent {
	return ampli.Identify.Builder().DdevEnvironment("").DockerPlatform("docker-desktop").DockerVersion(dockerVersion).Timezone("UTC").Build()
}
