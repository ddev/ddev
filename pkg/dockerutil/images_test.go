package dockerutil_test

import (
	"testing"

	"github.com/compose-spec/compose-go/v2/types"
	ddevImages "github.com/ddev/ddev/pkg/docker"
	"github.com/ddev/ddev/pkg/dockerutil"
	"github.com/stretchr/testify/require"
)

// TestImageExistsLocallyForPlatform relies on TestMain having pulled the web
// image, which is always for the daemon's own platform.
func TestImageExistsLocallyForPlatform(t *testing.T) {
	serverVersion, err := dockerutil.GetServerVersion()
	require.NoError(t, err)
	daemonPlatform := serverVersion.Os + "/" + serverVersion.Arch

	webImage := ddevImages.GetWebImage()
	for platform, want := range map[string]bool{
		"":               true,
		daemonPlatform:   true,
		"linux/s390x":    false,
		"not a platform": false,
	} {
		exists, err := dockerutil.ImageExistsLocallyForPlatform(webImage, platform)
		require.NoError(t, err)
		require.Equal(t, want, exists, "platform %q", platform)
	}

	for _, platform := range []string{"", daemonPlatform} {
		exists, err := dockerutil.ImageExistsLocallyForPlatform("ddev/does-not-exist:none", platform)
		require.NoError(t, err)
		require.False(t, exists, "platform %q", platform)
	}
}

// TestPullImagesForPlatform uses platforms no DDEV host runs, so the image
// can only be present for one if PullImages pulled it for that platform.
// Whether earlier pulls of the tag survive depends on the image store, so only
// the platform pulled last, sorting last, is checked.
func TestPullImagesForPlatform(t *testing.T) {
	const image = "busybox:1.35"
	require.NoError(t, dockerutil.RemoveImage(image))
	t.Cleanup(func() {
		_ = dockerutil.RemoveImage(image)
	})

	err := dockerutil.PullImages([]types.ServiceConfig{
		{Image: image, Platform: "linux/ppc64le"},
		{Image: image, Platform: "linux/s390x"},
		{Image: image},
		{Image: image, Platform: "linux/ppc64le"},
	}, false)
	require.NoError(t, err)

	for platform, want := range map[string]bool{
		"linux/s390x":   true,
		"linux/riscv64": false,
	} {
		exists, err := dockerutil.ImageExistsLocallyForPlatform(image, platform)
		require.NoError(t, err)
		require.Equal(t, want, exists, "platform %q", platform)
	}

	// The tag is now local for other platforms, which must not skip this pull.
	err = dockerutil.PullImages([]types.ServiceConfig{{Image: image, Platform: "linux/riscv64"}}, false)
	require.NoError(t, err)
	exists, err := dockerutil.ImageExistsLocallyForPlatform(image, "linux/riscv64")
	require.NoError(t, err)
	require.True(t, exists)

	err = dockerutil.PullImages([]types.ServiceConfig{{Image: image, Platform: "not a platform"}}, false)
	require.Error(t, err)
}

func TestBuildPlatformToPull(t *testing.T) {
	serverVersion, err := dockerutil.GetServerVersion()
	require.NoError(t, err)
	daemonPlatform := serverVersion.Os + "/" + serverVersion.Arch

	for _, tc := range []struct {
		buildPlatforms []string
		want           string
	}{
		{nil, ""},
		{[]string{daemonPlatform}, ""},
		{[]string{"linux/s390x", daemonPlatform}, ""},
		{[]string{"linux/s390x"}, "linux/s390x"},
		{[]string{"linux/s390x", "linux/ppc64le"}, "linux/s390x"},
	} {
		require.Equal(t, tc.want, dockerutil.BuildPlatformToPull(tc.buildPlatforms), "build platforms %v", tc.buildPlatforms)
	}
}
