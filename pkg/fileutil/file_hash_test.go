package fileutil_test

import (
	"crypto/sha1"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/ddev/ddev/pkg/fileutil"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/ddev/ddev/pkg/testcommon"
	"github.com/ddev/ddev/pkg/util"
	"github.com/stretchr/testify/require"
)

// TestFileHash is a unit test for FileHash()
func TestFileHash(t *testing.T) {
	testCases := []struct {
		content       string
		optionalExtra string
	}{
		{"This is a test file for FileHash function.", ""},
		{"Another test case with different content.", ""},
		{"Test file with special characters: !@#$%^&*()_+-=[]{};':\"|,.<>?", ""},
		{"Test file with provided extra content", "random extra content"},
	}

	for i, tc := range testCases {
		t.Run("", func(t *testing.T) {
			tmpDir := testcommon.CreateTmpDir("TestFileHash_" + util.RandString(10))
			t.Cleanup(func() {
				_ = os.RemoveAll(tmpDir)
			})
			testFile := filepath.Join(tmpDir, fmt.Sprintf("TestFileHash_%d", i))
			err := fileutil.TemplateStringToFile(tc.content, nil, testFile)
			require.NoError(t, err)

			// Use FileHash first
			result, err := fileutil.FileHash(testFile, tc.optionalExtra)
			require.NoError(t, err)

			// FileHash also hashes the file's path and the extra string
			canonicalFileName := testFile
			if nodeps.IsWindows() {
				canonicalFileName = util.WindowsPathToCygwinPath(testFile)
			}
			expectedHash := fmt.Sprintf("%x", sha1.Sum([]byte(tc.content+canonicalFileName+tc.optionalExtra)))

			require.Equal(t, expectedHash, result)
		})
	}
}
