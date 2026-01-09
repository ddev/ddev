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
	require.NoError(t, err)
	// Volume should now have at least 1MB of data
	require.Greater(t, sizeBytes, int64(1024*1024-1), "Volume should contain at least 1MB of data")
	require.NotEmpty(t, sizeHuman)
	require.NotEqual(t, "0B", sizeHuman, "Volume should not be empty")

	// Test non-existent volume
	sizeBytes, sizeHuman, err = dockerutil.GetVolumeSize("nonexistent_volume_xyz")
	require.NoError(t, err) // Should not error, just return 0
	require.Equal(t, int64(0), sizeBytes)
	require.Equal(t, "0B", sizeHuman)
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
	require.NoError(t, err)
	require.NotNil(t, volumeSizes)

	// Our test volume should be in the results
	volSize, exists := volumeSizes[testVolume]
	require.True(t, exists, "Test volume should exist in results")
	require.Equal(t, testVolume, volSize.Name)
	require.GreaterOrEqual(t, volSize.SizeBytes, int64(0))
	require.NotEmpty(t, volSize.SizeHuman)
}

// TestVolumeExists tests the VolumeExists function
func TestVolumeExists(t *testing.T) {
	testVolume := t.Name()
	_ = dockerutil.RemoveVolume(testVolume)

	t.Cleanup(func() {
		_ = dockerutil.RemoveVolume(testVolume)
	})

	// Volume should not exist initially
	require.False(t, dockerutil.VolumeExists(testVolume))

	// Create the volume
	_, err := dockerutil.CreateVolume(testVolume, "local", map[string]string{}, nil)
	require.NoError(t, err)

	// Volume should exist now
	require.True(t, dockerutil.VolumeExists(testVolume))

	// Remove the volume
	err = dockerutil.RemoveVolume(testVolume)
	require.NoError(t, err)

	// Volume should not exist anymore
	require.False(t, dockerutil.VolumeExists(testVolume))
}

// TestVolumeLabels tests the VolumeLabels function
func TestVolumeLabels(t *testing.T) {
	testVolume := t.Name()
	_ = dockerutil.RemoveVolume(testVolume)

	t.Cleanup(func() {
		_ = dockerutil.RemoveVolume(testVolume)
	})

	// Test with labels
	labels := map[string]string{
		"com.ddev.site-name":    "test-site",
		"com.ddev.platform":     "ddev",
		"com.ddev.custom-label": "custom-value",
	}
	_, err := dockerutil.CreateVolume(testVolume, "local", map[string]string{}, labels)
	require.NoError(t, err)

	// Get labels from the volume
	retrievedLabels, err := dockerutil.VolumeLabels(testVolume)
	require.NoError(t, err)
	require.NotNil(t, retrievedLabels)

	// Verify all labels are present
	require.Equal(t, "test-site", retrievedLabels["com.ddev.site-name"])
	require.Equal(t, "ddev", retrievedLabels["com.ddev.platform"])
	require.Equal(t, "custom-value", retrievedLabels["com.ddev.custom-label"])

	// Test non-existent volume
	_, err = dockerutil.VolumeLabels("nonexistent_volume_xyz")
	require.Error(t, err)
}

// TestPurgeDirectoryContentsInVolume tests purging directory contents in a volume
func TestPurgeDirectoryContentsInVolume(t *testing.T) {
	testVolume := t.Name()
	_ = dockerutil.RemoveVolume(testVolume)

	t.Cleanup(func() {
		_ = dockerutil.RemoveVolume(testVolume)
	})

	// Create a volume
	_, err := dockerutil.CreateVolume(testVolume, "local", map[string]string{}, nil)
	require.NoError(t, err)

	// Create some files in the volume
	_, _, err = dockerutil.RunSimpleContainer(
		versionconstants.UtilitiesImage,
		"",
		[]string{"sh", "-c", "mkdir -p /mnt/v/subdir1 /mnt/v/subdir2 && echo 'file1' > /mnt/v/subdir1/file1.txt && echo 'file2' > /mnt/v/subdir2/file2.txt"},
		nil,
		nil,
		[]string{testVolume + ":/mnt/v"},
		"0",
		true,
		false,
		nil,
		nil,
		nil,
	)
	require.NoError(t, err)

	files, err := dockerutil.ListFilesInVolume(testVolume, "subdir1")
	require.NoError(t, err)
	require.Len(t, files, 1)
	require.Contains(t, files, "file1.txt")
	files, err = dockerutil.ListFilesInVolume(testVolume, "subdir2")
	require.Len(t, files, 1)
	require.Contains(t, files, "file2.txt")

	// Purge directory contents
	err = dockerutil.PurgeDirectoryContentsInVolume(testVolume, []string{"subdir1", "subdir2"}, "0")
	require.NoError(t, err)

	files, err = dockerutil.ListFilesInVolume(testVolume, "subdir1")
	require.NoError(t, err)
	require.Len(t, files, 0)
	files, err = dockerutil.ListFilesInVolume(testVolume, "subdir2")
	require.Len(t, files, 0)
}

// TestListFilesInVolume tests listing files in a volume subdirectory
func TestListFilesInVolume(t *testing.T) {
	testVolume := t.Name()
	_ = dockerutil.RemoveVolume(testVolume)

	t.Cleanup(func() {
		_ = dockerutil.RemoveVolume(testVolume)
	})

	// Create a volume
	_, err := dockerutil.CreateVolume(testVolume, "local", map[string]string{}, nil)
	require.NoError(t, err)

	// Create some files in a subdirectory
	_, _, err = dockerutil.RunSimpleContainer(
		versionconstants.UtilitiesImage,
		"",
		[]string{"sh", "-c", "mkdir -p /mnt/v/testdir && echo 'a' > /mnt/v/testdir/file1.txt && echo 'b' > /mnt/v/testdir/file2.txt && echo 'c' > /mnt/v/testdir/file3.conf"},
		nil,
		nil,
		[]string{testVolume + ":/mnt/v"},
		"0",
		true,
		false,
		nil,
		nil,
		nil,
	)
	require.NoError(t, err)

	// List files in the subdirectory
	files, err := dockerutil.ListFilesInVolume(testVolume, "testdir")
	require.NoError(t, err)
	require.Len(t, files, 3)
	require.Contains(t, files, "file1.txt")
	require.Contains(t, files, "file2.txt")
	require.Contains(t, files, "file3.conf")

	// List files in non-existent directory (should return empty, not error)
	files, err = dockerutil.ListFilesInVolume(testVolume, "nonexistent")
	require.NoError(t, err)
	require.Empty(t, files)

	// List files in root of volume
	_, _, err = dockerutil.RunSimpleContainer(
		versionconstants.UtilitiesImage,
		"",
		[]string{"sh", "-c", "echo 'root' > /mnt/v/rootfile.txt"},
		nil,
		nil,
		[]string{testVolume + ":/mnt/v"},
		"0",
		true,
		false,
		nil,
		nil,
		nil,
	)
	require.NoError(t, err)

	files, err = dockerutil.ListFilesInVolume(testVolume, "")
	require.NoError(t, err)
	require.Contains(t, files, "rootfile.txt")
	require.Contains(t, files, "testdir")
}

// TestRemoveFilesFromVolume tests removing specific files from a volume
func TestRemoveFilesFromVolume(t *testing.T) {
	testVolume := t.Name()
	_ = dockerutil.RemoveVolume(testVolume)

	t.Cleanup(func() {
		_ = dockerutil.RemoveVolume(testVolume)
	})

	// Create a volume
	_, err := dockerutil.CreateVolume(testVolume, "local", map[string]string{}, nil)
	require.NoError(t, err)

	// Create some files in a subdirectory
	_, _, err = dockerutil.RunSimpleContainer(
		versionconstants.UtilitiesImage,
		"",
		[]string{"sh", "-c", "mkdir -p /mnt/v/config && echo 'a' > /mnt/v/config/keep.txt && echo 'b' > /mnt/v/config/remove1.txt && echo 'c' > /mnt/v/config/remove2.txt"},
		nil,
		nil,
		[]string{testVolume + ":/mnt/v"},
		"0",
		true,
		false,
		nil,
		nil,
		nil,
	)
	require.NoError(t, err)

	// Remove specific files
	err = dockerutil.RemoveFilesFromVolume(testVolume, "config", []string{"remove1.txt", "remove2.txt"})
	require.NoError(t, err)

	// Verify only keep.txt remains
	files, err := dockerutil.ListFilesInVolume(testVolume, "config")
	require.NoError(t, err)
	require.Len(t, files, 1)
	require.Contains(t, files, "keep.txt")

	// Removing empty list should not error
	err = dockerutil.RemoveFilesFromVolume(testVolume, "config", []string{})
	require.NoError(t, err)

	// Removing non-existent files should not error (rm -f behavior)
	err = dockerutil.RemoveFilesFromVolume(testVolume, "config", []string{"nonexistent.txt"})
	require.NoError(t, err)

	// keep.txt should still be there
	files, err = dockerutil.ListFilesInVolume(testVolume, "config")
	require.NoError(t, err)
	require.Len(t, files, 1)
	require.Contains(t, files, "keep.txt")
}
