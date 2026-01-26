package types_test

import (
	"encoding/json"
	"testing"

	"github.com/ddev/ddev/pkg/config/remoteconfig/types"
	"github.com/stretchr/testify/require"
)

// TestFlexibleStringUnmarshalJSON tests the FlexibleString JSON unmarshaling
func TestFlexibleStringUnmarshalJSON(t *testing.T) {
	tests := []struct {
		name          string
		jsonInput     string
		expectedValue string
		expectedIsSet bool
		expectError   bool
	}{
		{
			name:          "string value",
			jsonInput:     `"v1.2.3"`,
			expectedValue: "v1.2.3",
			expectedIsSet: true,
			expectError:   false,
		},
		{
			name:          "empty string",
			jsonInput:     `""`,
			expectedValue: "",
			expectedIsSet: true,
			expectError:   false,
		},
		{
			name:          "null value",
			jsonInput:     `null`,
			expectedValue: "",
			expectedIsSet: false,
			expectError:   false,
		},
		{
			name:          "small integer",
			jsonInput:     `42`,
			expectedValue: "42",
			expectedIsSet: true,
			expectError:   false,
		},
		{
			name:          "large integer",
			jsonInput:     `1234567890`,
			expectedValue: "1234567890",
			expectedIsSet: true,
			expectError:   false,
		},
		{
			name:          "very large integer (avoid scientific notation)",
			jsonInput:     `9999999999999`,
			expectedValue: "9999999999999",
			expectedIsSet: true,
			expectError:   false,
		},
		{
			name:          "float with decimal",
			jsonInput:     `3.14`,
			expectedValue: "3.14",
			expectedIsSet: true,
			expectError:   false,
		},
		{
			name:          "zero",
			jsonInput:     `0`,
			expectedValue: "0",
			expectedIsSet: true,
			expectError:   false,
		},
		{
			name:          "negative integer",
			jsonInput:     `-42`,
			expectedValue: "-42",
			expectedIsSet: true,
			expectError:   false,
		},
		{
			name:          "negative float",
			jsonInput:     `-3.14`,
			expectedValue: "-3.14",
			expectedIsSet: true,
			expectError:   false,
		},
		{
			name:          "boolean value (should error)",
			jsonInput:     `true`,
			expectedValue: "",
			expectedIsSet: false,
			expectError:   true,
		},
		{
			name:          "array value (should error)",
			jsonInput:     `[1,2,3]`,
			expectedValue: "",
			expectedIsSet: false,
			expectError:   true,
		},
		{
			name:          "object value (should error)",
			jsonInput:     `{"key":"value"}`,
			expectedValue: "",
			expectedIsSet: false,
			expectError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var fs types.FlexibleString
			err := json.Unmarshal([]byte(tt.jsonInput), &fs)

			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expectedValue, fs.Value, "Value should match")
				require.Equal(t, tt.expectedIsSet, fs.IsSet, "IsSet flag should match")
			}
		})
	}
}

// TestFlexibleStringInAddonStruct tests FlexibleString within the Addon struct
func TestFlexibleStringInAddonStruct(t *testing.T) {
	tests := []struct {
		name          string
		jsonInput     string
		expectedValue string
		expectedIsSet bool
	}{
		{
			name: "string tag name",
			jsonInput: `{
				"title": "Test Addon",
				"user": "ddev",
				"repo": "test-addon",
				"tag_name": "v1.0.0"
			}`,
			expectedValue: "v1.0.0",
			expectedIsSet: true,
		},
		{
			name: "numeric tag name",
			jsonInput: `{
				"title": "Test Addon",
				"user": "ddev",
				"repo": "test-addon",
				"tag_name": 123
			}`,
			expectedValue: "123",
			expectedIsSet: true,
		},
		{
			name: "null tag name",
			jsonInput: `{
				"title": "Test Addon",
				"user": "ddev",
				"repo": "test-addon",
				"tag_name": null
			}`,
			expectedValue: "",
			expectedIsSet: false,
		},
		{
			name: "missing tag name field",
			jsonInput: `{
				"title": "Test Addon",
				"user": "ddev",
				"repo": "test-addon"
			}`,
			expectedValue: "",
			expectedIsSet: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var addon types.Addon
			err := json.Unmarshal([]byte(tt.jsonInput), &addon)
			require.NoError(t, err)
			require.Equal(t, tt.expectedValue, addon.TagName.Value, "TagName.Value should match")
			require.Equal(t, tt.expectedIsSet, addon.TagName.IsSet, "TagName.IsSet should match")
		})
	}
}

// TestFlexibleStringMarshalJSON tests the FlexibleString JSON marshaling
func TestFlexibleStringMarshalJSON(t *testing.T) {
	tests := []struct {
		name           string
		flexibleString types.FlexibleString
		expectedJSON   string
	}{
		{
			name:           "set string value",
			flexibleString: types.FlexibleString{Value: "v1.2.3", IsSet: true},
			expectedJSON:   `"v1.2.3"`,
		},
		{
			name:           "set empty string",
			flexibleString: types.FlexibleString{Value: "", IsSet: true},
			expectedJSON:   `""`,
		},
		{
			name:           "not set",
			flexibleString: types.FlexibleString{Value: "", IsSet: false},
			expectedJSON:   `null`,
		},
		{
			name:           "not set with value",
			flexibleString: types.FlexibleString{Value: "ignored", IsSet: false},
			expectedJSON:   `null`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jsonBytes, err := json.Marshal(tt.flexibleString)
			require.NoError(t, err)
			require.Equal(t, tt.expectedJSON, string(jsonBytes))
		})
	}
}

// TestFlexibleStringRoundTrip tests marshaling and unmarshaling FlexibleString
func TestFlexibleStringRoundTrip(t *testing.T) {
	tests := []struct {
		name     string
		original types.FlexibleString
	}{
		{
			name:     "set string value",
			original: types.FlexibleString{Value: "v1.2.3", IsSet: true},
		},
		{
			name:     "set empty string",
			original: types.FlexibleString{Value: "", IsSet: true},
		},
		{
			name:     "not set",
			original: types.FlexibleString{Value: "", IsSet: false},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Marshal to JSON
			jsonBytes, err := json.Marshal(tt.original)
			require.NoError(t, err)

			// Unmarshal back
			var roundTrip types.FlexibleString
			err = json.Unmarshal(jsonBytes, &roundTrip)
			require.NoError(t, err)

			// Compare
			require.Equal(t, tt.original.Value, roundTrip.Value)
			require.Equal(t, tt.original.IsSet, roundTrip.IsSet)
		})
	}
}

// TestAddonRoundTrip tests marshaling and unmarshaling Addon with FlexibleString
func TestAddonRoundTrip(t *testing.T) {
	original := types.Addon{
		Title:         "ddev/ddev-redis",
		User:          "ddev",
		Repo:          "ddev-redis",
		DefaultBranch: "main",
		TagName:       types.FlexibleString{Value: "v1.0.0", IsSet: true},
		Type:          "official",
	}

	// Marshal to JSON
	jsonBytes, err := json.Marshal(original)
	require.NoError(t, err)

	// Unmarshal back
	var roundTrip types.Addon
	err = json.Unmarshal(jsonBytes, &roundTrip)
	require.NoError(t, err)

	// Compare
	require.Equal(t, original.Title, roundTrip.Title)
	require.Equal(t, original.User, roundTrip.User)
	require.Equal(t, original.Repo, roundTrip.Repo)
	require.Equal(t, original.TagName.Value, roundTrip.TagName.Value)
	require.Equal(t, original.TagName.IsSet, roundTrip.TagName.IsSet)
	require.Equal(t, original.Type, roundTrip.Type)
}

// TestAddonDataFindAddon tests the FindAddon method
func TestAddonDataFindAddon(t *testing.T) {
	addonData := types.AddonData{
		Addons: []types.Addon{
			{User: "ddev", Repo: "ddev-redis", TagName: types.FlexibleString{Value: "v1.0.0", IsSet: true}},
			{User: "ddev", Repo: "ddev-extra-addon", TagName: types.FlexibleString{Value: "v2.0.0", IsSet: true}},
			{User: "other", Repo: "test-addon", TagName: types.FlexibleString{Value: "", IsSet: false}},
		},
	}

	tests := []struct {
		name        string
		ownerRepo   string
		shouldFind  bool
		expectedTag string
	}{
		{
			name:        "find existing addon",
			ownerRepo:   "ddev/ddev-redis",
			shouldFind:  true,
			expectedTag: "v1.0.0",
		},
		{
			name:        "find second addon",
			ownerRepo:   "ddev/ddev-extra-addon",
			shouldFind:  true,
			expectedTag: "v2.0.0",
		},
		{
			name:        "find addon with null tag",
			ownerRepo:   "other/test-addon",
			shouldFind:  true,
			expectedTag: "",
		},
		{
			name:       "non-existent addon",
			ownerRepo:  "ddev/ddev-nonexistent",
			shouldFind: false,
		},
		{
			name:       "malformed owner/repo",
			ownerRepo:  "invalid",
			shouldFind: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			addon := addonData.FindAddon(tt.ownerRepo)

			if tt.shouldFind {
				require.NotNil(t, addon, "Should find addon")
				require.Equal(t, tt.expectedTag, addon.TagName.Value, "Tag should match")
			} else {
				require.Nil(t, addon, "Should not find addon")
			}
		})
	}
}
