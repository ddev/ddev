package fileutil_test

import (
	"fmt"
	"github.com/ddev/ddev/pkg/dockerutil"
	"github.com/ddev/ddev/pkg/fileutil"
	"github.com/ddev/ddev/pkg/testcommon"
	"github.com/ddev/ddev/pkg/util"
	"github.com/ddev/ddev/pkg/versionconstants"
	asrt "github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"
)

// TestFileHash is a unit test for FileHash()
func TestFileHash(t *testing.T) {
	assert := asrt.New(t)

	testCases := []struct {
		content string
	}{
		{"This is a test file for FileHash function."},
		{"Another test case with different content."},
		{"Test file with special characters: !@#$%^&*()_+-=[]{};':\"|,.<>?"},
	}

	for i, tc := range testCases {
		t.Run("", func(t *testing.T) {
			tmpDir := testcommon.CreateTmpDir("TestFileHash_" + util.RandString(10))
			t.Cleanup(func() {
				err := os.RemoveAll(tmpDir)
				assert.NoError(err)
			})
			testFile := filepath.Join(tmpDir, fmt.Sprintf("TestFileHash_%d", i))
			err := fileutil.TemplateStringToFile(tc.content, nil, testFile)
			require.NoError(t, err)

			expectedHash, err := externalComputeSha1Sum(testFile)
			require.NoError(t, err)

			result, err := fileutil.FileHash(testFile)
			require.NoError(t, err)

			require.Equal(t, expectedHash, result)
		})
	}
}

// externalComputeSha1Sum uses external tool (sha1sum for example) to compute shasum
func externalComputeSha1Sum(filePath string) (string, error) {
	dir := filepath.Dir(filePath)
	fileName := filepath.Base(filePath)
	_, out, err := dockerutil.RunSimpleContainer(versionconstants.BusyboxImage, "", []string{"sha1sum", path.Join("/var/tmp/checkdir/", fileName)}, nil, nil, []string{dir + ":" + "/var/tmp/checkdir"}, "0", true, false, nil, nil)

	if err != nil {
		return "", err
	}

	hash := strings.Split(out, " ")[0]
	return hash, nil
}
