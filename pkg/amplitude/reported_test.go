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

	require.False(t, alreadyReported(file, day1, "project:p1", php83))
	require.True(t, alreadyReported(file, day1, "project:p1", php83))

	// Changed properties are sent, and switching back is not sent again.
	require.False(t, alreadyReported(file, day1, "project:p1", php84))
	require.True(t, alreadyReported(file, day1, "project:p1", php83))

	require.False(t, alreadyReported(file, day1, "project:p2", php83))

	require.False(t, alreadyReported(file, day2, "project:p1", php83))
	require.True(t, alreadyReported(file, day2, "project:p1", php83))
	require.False(t, alreadyReported(file, day2, "project:p2", php83))

	identify := ampli.Identify.Builder().DdevEnvironment("").DockerPlatform("docker-desktop").DockerVersion("28.0.0").Timezone("UTC").Build()
	require.False(t, alreadyReported(file, day2, "identify", identify))
	require.True(t, alreadyReported(file, day2, "identify", identify))

	newDocker := ampli.Identify.Builder().DdevEnvironment("").DockerPlatform("docker-desktop").DockerVersion("29.0.0").Timezone("UTC").Build()
	require.False(t, alreadyReported(file, day2, "identify", newDocker))
}

func TestAlreadyReportedCorruptFile(t *testing.T) {
	file := filepath.Join(t.TempDir(), reportedFile)
	require.NoError(t, os.WriteFile(file, []byte("not json"), 0644))

	event := projectEvent("8.3")
	require.False(t, alreadyReported(file, "2026-11-06", "project:p1", event))
	require.True(t, alreadyReported(file, "2026-11-06", "project:p1", event))
}

func projectEvent(phpVersion string) ampli.ProjectEvent {
	return ampli.Project.Builder().Id("p1").PerformanceMode("none").PhpVersion(phpVersion).ProjectType("php").WebserverType("nginx-fpm").XhProfMode("prepend").Build()
}
