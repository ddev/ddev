package dockerutil_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ddev/ddev/pkg/dockerutil"
	"github.com/ddev/ddev/pkg/versionconstants"
	"github.com/moby/moby/client"
	asrt "github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCreateVolume does a trivial test of creating a trivial Docker volume.
func TestCreateVolume(t *testing.T) {
	assert := asrt.New(t)
	// Make sure there's no existing volume.
	//nolint: errcheck
	dockerutil.RemoveVolume("junker99")
	vol, err := dockerutil.CreateVolume("junker99", "local", map[string]string{}, nil)
	require.NoError(t, err)

	//nolint: errcheck
	defer dockerutil.RemoveVolume("junker99")
	require.NotNil(t, vol)
	assert.Equal("junker99", vol.Name)
}

// TestRemoveVolume makes sure we can remove a volume successfully
func TestRemoveVolume(t *testing.T) {
	assert := asrt.New(t)
	ctx, apiClient, err := dockerutil.GetDockerClient()
	if err != nil {
		t.Fatalf("Could not get docker client: %v", err)
	}

	testVolume := "junker999"
	spareVolume := "someVolumeThatCanNeverExit"

	_ = dockerutil.RemoveVolume(testVolume)

	vol, err := dockerutil.CreateVolume(testVolume, "local", map[string]string{}, nil)
	assert.NoError(err)

	volumes, err := apiClient.VolumeList(ctx, client.VolumeListOptions{Filters: client.Filters{}.Add("name", testVolume)})
	assert.NoError(err)
	require.Len(t, volumes.Items, 1)
	assert.Equal(testVolume, volumes.Items[0].Name)

	require.NotNil(t, vol)
	assert.Equal(testVolume, vol.Name)
	err = dockerutil.RemoveVolume(testVolume)
	assert.NoError(err)

	volumes, err = apiClient.VolumeList(ctx, client.VolumeListOptions{Filters: client.Filters{}.Add("name", testVolume)})
	assert.NoError(err)
	assert.Empty(volumes.Items)

	// Make sure spareVolume doesn't exist, then make sure removal
	// of nonexistent volume doesn't result in error
	_ = dockerutil.RemoveVolume(spareVolume)
	err = dockerutil.RemoveVolume(spareVolume)
	assert.NoError(err)
}

// TestCopyIntoVolume makes sure CopyToVolume copies a local directory into a volume
func TestCopyIntoVolume(t *testing.T) {
	assert := asrt.New(t)
	err := dockerutil.RemoveVolume(t.Name())
	assert.NoError(err)

	pwd, _ := os.Getwd()
	t.Cleanup(func() {
		err = dockerutil.RemoveVolume(t.Name())
		assert.NoError(err)
	})

	err = dockerutil.CopyIntoVolume(filepath.Join(pwd, "testdata", t.Name()), t.Name(), "", "0", "", true)
	require.NoError(t, err)

	// Make sure that the content is the same, and that .test.sh is executable
	// On Windows the upload can result in losing executable bit
	_, out, err := dockerutil.RunSimpleContainer(versionconstants.UtilitiesImage, "", []string{"sh", "-c", "cd /mnt/" + t.Name() + " && ls -R .test.sh * && ./.test.sh"}, nil, nil, []string{t.Name() + ":/mnt/" + t.Name()}, "25", true, false, nil, nil, nil)
	assert.NoError(err)
	assert.Equal(`.test.sh
root.txt

subdir1:
subdir1.txt
hi this is a test file
`, out)

	err = dockerutil.CopyIntoVolume(filepath.Join(pwd, "testdata", t.Name()), t.Name(), "somesubdir", "501", "", true)
	assert.NoError(err)
	_, out, err = dockerutil.RunSimpleContainer(versionconstants.UtilitiesImage, "", []string{"sh", "-c", "cd /mnt/" + t.Name() + "/somesubdir  && pwd && ls -R"}, nil, nil, []string{t.Name() + ":/mnt/" + t.Name()}, "0", true, false, nil, nil, nil)
	assert.NoError(err)
	assert.Equal(`/mnt/TestCopyIntoVolume/somesubdir
.:
root.txt
subdir1

./subdir1:
subdir1.txt
`, out)

	// Now try a file
	err = dockerutil.CopyIntoVolume(filepath.Join(pwd, "testdata", t.Name(), "root.txt"), t.Name(), "", "0", "", true)
	assert.NoError(err)

	// Make sure that the content is the same, and that .test.sh is executable
	_, out, err = dockerutil.RunSimpleContainer(versionconstants.UtilitiesImage, "", []string{"cat", "/mnt/" + t.Name() + "/root.txt"}, nil, nil, []string{t.Name() + ":/mnt/" + t.Name()}, "25", true, false, nil, nil, nil)
	assert.NoError(err)
	assert.Equal("root.txt here\n", out)

	// Copy destructively and make sure that stuff got destroyed
	err = dockerutil.CopyIntoVolume(filepath.Join(pwd, "testdata", t.Name()+"2"), t.Name(), "", "0", "", true)
	require.NoError(t, err)

	_, _, err = dockerutil.RunSimpleContainer(versionconstants.UtilitiesImage, "", []string{"ls", "/mnt/" + t.Name() + "/subdir1/subdir1.txt"}, nil, nil, []string{t.Name() + ":/mnt/" + t.Name()}, "25", true, false, nil, nil, nil)
	require.Error(t, err)

	_, _, err = dockerutil.RunSimpleContainer(versionconstants.UtilitiesImage, "", []string{"ls", "/mnt/" + t.Name() + "/subdir1/only-the-new-stuff.txt"}, nil, nil, []string{t.Name() + ":/mnt/" + t.Name()}, "25", true, false, nil, nil, nil)
	require.NoError(t, err)
}

// TestFormatBytes tests the byte formatting function
func TestFormatBytes(t *testing.T) {
	assert := asrt.New(t)

	tests := []struct {
		name     string
		bytes    int64
		expected string
	}{
		{"Zero bytes", 0, "0B"},
		{"Small bytes", 512, "512B"},
		{"Exactly 1KB", 1024, "1.0KB"},
		{"Multiple KB", 2560, "2.5KB"},
		{"Exactly 1MB", 1024 * 1024, "1.0MB"},
		{"Fractional MB", 1024*1024 + 512*1024, "1.5MB"},
		{"Large MB", 157286400, "150.0MB"},
		{"Exactly 1GB", 1024 * 1024 * 1024, "1.0GB"},
		{"Fractional GB", int64(1024*1024*1024) + int64(512*1024*1024), "1.5GB"},
		{"Multiple GB", int64(5) * 1024 * 1024 * 1024, "5.0GB"},
		{"Exactly 1TB", int64(1024) * 1024 * 1024 * 1024, "1.0TB"},
		{"Large volume", 982700000, "937.2MB"}, // Approximate d11 volume size
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := dockerutil.FormatBytes(tt.bytes)
			assert.Equal(tt.expected, result, "FormatBytes(%d) should return %s, got %s", tt.bytes, tt.expected, result)
		})
	}
}

// TestGetVolumeSize tests getting the size of a Docker volume using the Docker API
func TestGetVolumeSize(t *testing.T) {
	if dockerutil.IsPodman() {
		t.Skip("Podman does not support docker system df volume sizing")
	}
	assert := asrt.New(t)

	testVolume := "test_volume_size_check"
	_ = dockerutil.RemoveVolume(testVolume)

	// Create a volume
	_, err := dockerutil.CreateVolume(testVolume, "local", map[string]string{}, nil)
	require.NoError(t, err)

	t.Cleanup(func() {
		err = dockerutil.RemoveVolume(testVolume)
		assert.NoError(err)
	})

	// Write some data to the volume to ensure it has a measurable size
	// Create a 1MB file in the volume
	_, _, err = dockerutil.RunSimpleContainer(
		versionconstants.UtilitiesImage,
		"",
		[]string{"sh", "-c", "dd if=/dev/zero of=/mnt/testvolume/testfile bs=1M count=1"},
		nil,
		nil,
		[]string{testVolume + ":/mnt/testvolume"},
		"0",
		true,
		false,
		nil,
		nil,
		nil,
	)
	require.NoError(t, err)

	// Get the volume size
	sizeBytes, sizeHuman, err := dockerutil.GetVolumeSize(testVolume)
	assert.NoError(err)
	// Volume should now have at least 1MB of data
	assert.Greater(sizeBytes, int64(1024*1024-1), "Volume should contain at least 1MB of data")
	assert.NotEmpty(sizeHuman)
	assert.NotEqual("0B", sizeHuman, "Volume should not be empty")

	// Test non-existent volume
	sizeBytes, sizeHuman, err = dockerutil.GetVolumeSize("nonexistent_volume_xyz")
	assert.NoError(err) // Should not error, just return 0
	assert.Equal(int64(0), sizeBytes)
	assert.Equal("0B", sizeHuman)
}

// TestParseDockerSystemDf tests parsing Docker system df output via API
func TestParseDockerSystemDf(t *testing.T) {
	if dockerutil.IsPodman() {
		t.Skip("Podman does not support docker system df")
	}
	assert := asrt.New(t)

	testVolume := "test_parse_df_volume"
	_ = dockerutil.RemoveVolume(testVolume)

	// Create a test volume
	_, err := dockerutil.CreateVolume(testVolume, "local", map[string]string{}, nil)
	require.NoError(t, err)

	t.Cleanup(func() {
		err = dockerutil.RemoveVolume(testVolume)
		assert.NoError(err)
	})

	// Parse Docker system df
	volumeSizes, err := dockerutil.ParseDockerSystemDf()
	assert.NoError(err)
	assert.NotNil(volumeSizes)

	// Our test volume should be in the results
	volSize, exists := volumeSizes[testVolume]
	assert.True(exists, "Test volume should exist in results")
	assert.Equal(testVolume, volSize.Name)
	assert.GreaterOrEqual(volSize.SizeBytes, int64(0))
	assert.NotEmpty(volSize.SizeHuman)
}
