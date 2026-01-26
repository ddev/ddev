package storage_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ddev/ddev/pkg/config/remoteconfig/storage"
	"github.com/ddev/ddev/pkg/config/remoteconfig/types"
	"github.com/stretchr/testify/require"
)

// TestAddonFileStorageReadWrite tests basic read/write operations
func TestAddonFileStorageReadWrite(t *testing.T) {
	tmpDir := t.TempDir()
	cacheFile := filepath.Join(tmpDir, ".addon-data")

	addonStorage := storage.NewAddonFileStorage(cacheFile)

	// Create test addon data
	testData := &types.AddonData{
		UpdatedDateTime:     time.Now(),
		TotalAddonsCount:    2,
		OfficialAddonsCount: 1,
		ContribAddonsCount:  1,
		Addons: []types.Addon{
			{
				User:          "ddev",
				Repo:          "ddev-new-addon",
				DefaultBranch: "main",
				TagName:       types.FlexibleString{Value: "v1.0.0", IsSet: true},
			},
			{
				User:          "ddev",
				Repo:          "ddev-extra-addon",
				DefaultBranch: "master",
				TagName:       types.FlexibleString{Value: "v2.0.0", IsSet: true},
			},
		},
	}

	// Write data
	err := addonStorage.Write(testData)
	require.NoError(t, err, "Write should succeed")

	// Verify file exists
	require.FileExists(t, cacheFile, "Cache file should exist")

	// Read data back
	readData, err := addonStorage.Read()
	require.NoError(t, err, "Read should succeed")
	require.NotNil(t, readData, "Read data should not be nil")

	// Verify data matches
	require.Equal(t, testData.TotalAddonsCount, readData.TotalAddonsCount, "TotalAddonsCount should match")
	require.Equal(t, testData.OfficialAddonsCount, readData.OfficialAddonsCount, "OfficialAddonsCount should match")
	require.Len(t, readData.Addons, 2, "Should have 2 addons")
	require.Equal(t, "ddev", readData.Addons[0].User, "First addon user should match")
	require.Equal(t, "ddev-new-addon", readData.Addons[0].Repo, "First addon repo should match")
	require.Equal(t, "v1.0.0", readData.Addons[0].TagName.Value, "First addon tag should match")
}

// TestAddonFileStorageReadNonExistent tests reading when file doesn't exist
func TestAddonFileStorageReadNonExistent(t *testing.T) {
	tmpDir := t.TempDir()
	cacheFile := filepath.Join(tmpDir, "nonexistent", ".addon-data")

	addonStorage := storage.NewAddonFileStorage(cacheFile)

	// Reading non-existent file should succeed with empty data
	data, err := addonStorage.Read()
	require.NoError(t, err, "Read of non-existent file should succeed")
	require.NotNil(t, data, "Data should not be nil")
	require.Empty(t, data.Addons, "Addons should be empty")
}

// TestAddonFileStorageReadCorrupted tests reading corrupted file
func TestAddonFileStorageReadCorrupted(t *testing.T) {
	tmpDir := t.TempDir()
	cacheFile := filepath.Join(tmpDir, ".addon-data")

	// Write corrupted data
	err := os.WriteFile(cacheFile, []byte("corrupted binary data that's not gob"), 0644)
	require.NoError(t, err, "Writing corrupted file should succeed")

	addonStorage := storage.NewAddonFileStorage(cacheFile)

	// Reading corrupted file should return error
	_, err = addonStorage.Read()
	require.Error(t, err, "Read of corrupted file should fail")
}

// TestAddonFileStorageReadPermissionDenied tests reading when permissions are denied
func TestAddonFileStorageReadPermissionDenied(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("Skipping permission test when running as root")
	}

	tmpDir := t.TempDir()
	cacheFile := filepath.Join(tmpDir, ".addon-data")

	// Create a file with no read permissions
	err := os.WriteFile(cacheFile, []byte("test"), 0000)
	require.NoError(t, err, "Creating file should succeed")

	addonStorage := storage.NewAddonFileStorage(cacheFile)

	// Reading should fail due to permissions
	_, err = addonStorage.Read()
	require.Error(t, err, "Read should fail due to permissions")

	// Clean up - restore permissions so cleanup can delete the file
	_ = os.Chmod(cacheFile, 0644)
}

// TestAddonFileStorageMultipleReads tests that data is cached after first read
func TestAddonFileStorageMultipleReads(t *testing.T) {
	tmpDir := t.TempDir()
	cacheFile := filepath.Join(tmpDir, ".addon-data")

	addonStorage := storage.NewAddonFileStorage(cacheFile)

	testData := &types.AddonData{
		TotalAddonsCount: 1,
		Addons: []types.Addon{
			{User: "test", Repo: "addon"},
		},
	}

	// Write initial data
	err := addonStorage.Write(testData)
	require.NoError(t, err, "Initial write should succeed")

	// First read
	data1, err := addonStorage.Read()
	require.NoError(t, err, "First read should succeed")
	require.Equal(t, 1, data1.TotalAddonsCount, "First read should get correct data")

	// Modify file on disk (simulating external change)
	modifiedData := &types.AddonData{
		TotalAddonsCount: 99,
		Addons:           []types.Addon{},
	}
	storage2 := storage.NewAddonFileStorage(cacheFile)
	err = storage2.Write(modifiedData)
	require.NoError(t, err, "External write should succeed")

	// Second read from original addonStorage - should still return cached data
	data2, err := addonStorage.Read()
	require.NoError(t, err, "Second read should succeed")
	require.Equal(t, 1, data2.TotalAddonsCount, "Second read should return cached data, not modified data")
}

// TestAddonFileStorageWriteCreatesDirs tests that Write creates parent directories
func TestAddonFileStorageWriteCreatesDirs(t *testing.T) {
	tmpDir := t.TempDir()
	cacheFile := filepath.Join(tmpDir, "nested", "dirs", ".addon-data")

	addonStorage := storage.NewAddonFileStorage(cacheFile)

	testData := &types.AddonData{
		TotalAddonsCount: 1,
		Addons:           []types.Addon{{User: "test", Repo: "addon"}},
	}

	// Write should create parent directories
	err := addonStorage.Write(testData)
	require.NoError(t, err, "Write should succeed and create parent dirs")
	require.DirExists(t, filepath.Dir(cacheFile), "Parent directory should exist")
	require.FileExists(t, cacheFile, "Cache file should exist")
}

// TestAddonFileStorageEmptyAddons tests handling of empty addon list
func TestAddonFileStorageEmptyAddons(t *testing.T) {
	tmpDir := t.TempDir()
	cacheFile := filepath.Join(tmpDir, ".addon-data")

	addonStorage := storage.NewAddonFileStorage(cacheFile)

	testData := &types.AddonData{
		TotalAddonsCount: 0,
		Addons:           []types.Addon{},
	}

	// Write empty addon list
	err := addonStorage.Write(testData)
	require.NoError(t, err, "Write with empty addons should succeed")

	// Read back
	readData, err := addonStorage.Read()
	require.NoError(t, err, "Read should succeed")
	require.Empty(t, readData.Addons, "Addons should be empty")
	require.Equal(t, 0, readData.TotalAddonsCount, "TotalAddonsCount should be 0")
}

// TestAddonFileStorageFlexibleStringPersistence tests that FlexibleString fields persist correctly
func TestAddonFileStorageFlexibleStringPersistence(t *testing.T) {
	tmpDir := t.TempDir()
	cacheFile := filepath.Join(tmpDir, ".addon-data")

	addonStorage := storage.NewAddonFileStorage(cacheFile)

	testData := &types.AddonData{
		TotalAddonsCount: 3,
		Addons: []types.Addon{
			{User: "test", Repo: "addon1", TagName: types.FlexibleString{Value: "v1.0.0", IsSet: true}},
			{User: "test", Repo: "addon2", TagName: types.FlexibleString{Value: "123", IsSet: true}},
			{User: "test", Repo: "addon3", TagName: types.FlexibleString{Value: "", IsSet: false}},
		},
	}

	// Write data
	err := addonStorage.Write(testData)
	require.NoError(t, err, "Write should succeed")

	// Read back
	readData, err := addonStorage.Read()
	require.NoError(t, err, "Read should succeed")

	// Verify FlexibleString fields
	require.Equal(t, "v1.0.0", readData.Addons[0].TagName.Value, "String tag should persist")
	require.True(t, readData.Addons[0].TagName.IsSet, "IsSet should be true for string tag")

	require.Equal(t, "123", readData.Addons[1].TagName.Value, "Numeric tag should persist")
	require.True(t, readData.Addons[1].TagName.IsSet, "IsSet should be true for numeric tag")

	require.Equal(t, "", readData.Addons[2].TagName.Value, "Empty tag should persist")
	require.False(t, readData.Addons[2].TagName.IsSet, "IsSet should be false for null tag")
}
