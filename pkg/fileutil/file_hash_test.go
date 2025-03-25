package fileutil_test

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/ddev/ddev/pkg/dockerutil"
	"github.com/ddev/ddev/pkg/fileutil"
	"github.com/ddev/ddev/pkg/testcommon"
	"github.com/ddev/ddev/pkg/util"
	"github.com/ddev/ddev/pkg/versionconstants"
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

			// But we have to add the filepath to the testFile before
			// we can use the externalComputeSha1Sum successfully
			canonicalFileName := testFile
			if runtime.GOOS == "windows" {
				canonicalFileName = util.WindowsPathToCygwinPath(testFile)
			}
			f, err := os.OpenFile(testFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			require.NoError(t, err)
			_, err = f.WriteString(canonicalFileName)
			require.NoError(t, err)
			if tc.optionalExtra != "" {
				_, err = f.WriteString(tc.optionalExtra)
				require.NoError(t, err)
			}
			_ = f.Close()

			expectedHash, err := externalComputeSha1Sum(testFile)
			require.NoError(t, err)

			require.Equal(t, expectedHash, result)
		})
	}
}

// externalComputeSha1Sum uses external tool (sha1sum for example) to compute shasum
func externalComputeSha1Sum(filePath string) (string, error) {
	// Use a canonical filename in unix-style format so that we don't
	// get caught by differences in filename format on Windows.
	if runtime.GOOS == "windows" {
		filePath = util.WindowsPathToCygwinPath(filePath)
	}
	dir := path.Dir(filePath)
	_, out, err := dockerutil.RunSimpleContainer(versionconstants.BusyboxImage, "", []string{"sha1sum", filePath}, nil, nil, []string{dir + ":" + dir}, "0", true, false, nil, nil, nil)

	if err != nil {
		return "", err
	}

	hash := strings.Split(out, " ")[0]
	return hash, nil
}
