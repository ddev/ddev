package util_test

import (
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/ddev/ddev/pkg/util"
	"github.com/stretchr/testify/require"
)

// TestDownloadFile tests downloading a file with no sha, good sha, bad sha
func TestDownloadFile(t *testing.T) {
	testData := "hello world\n"
	sum := fmt.Sprintf("%x", sha256.Sum256([]byte(testData)))
	fileName := "testfile.txt"

	// HTTP test server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/" + fileName:
			_, _ = io.WriteString(w, testData)
		case "/sha256sums.txt":
			_, _ = fmt.Fprintf(w, "%s *%s\n", sum, fileName)
		case "/badsha256sums.txt":
			_, _ = fmt.Fprintf(w, "%s *%s\n", "deadbeef", fileName)
		default:
			http.NotFound(w, r)
		}
	}))
	defer ts.Close()

	tmpDir := t.TempDir()

	t.Run("no shasum", func(t *testing.T) {
		dest := filepath.Join(tmpDir, "plain.txt")
		err := util.DownloadFile(dest, ts.URL+"/"+fileName, false, "")
		require.NoError(t, err)
		content, err := os.ReadFile(dest)
		require.NoError(t, err)
		require.Equal(t, testData, string(content))
	})

	t.Run("correct shasum", func(t *testing.T) {
		dest := filepath.Join(tmpDir, "verified.txt")
		err := util.DownloadFile(dest, ts.URL+"/"+fileName, false, ts.URL+"/sha256sums.txt")
		require.NoError(t, err)
		content, err := os.ReadFile(dest)
		require.NoError(t, err)
		require.Equal(t, testData, string(content))
	})

	t.Run("incorrect shasum", func(t *testing.T) {
		dest := filepath.Join(tmpDir, "bad.txt")
		err := util.DownloadFile(dest, ts.URL+"/"+fileName, false, ts.URL+"/badsha256sums.txt")
		require.Error(t, err, "expected error due to SHA256 mismatch")
		require.NoFileExistsf(t, dest, "expected file %s to be deleted after sha mismatch, but it exists", dest)
	})
}
