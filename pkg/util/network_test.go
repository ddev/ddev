package util_test

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/output"
	"github.com/ddev/ddev/pkg/util"
	log "github.com/sirupsen/logrus"
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

// TestDownloadFileRetryLogic tests the retry logic structure of DownloadFile
func TestDownloadFileRetryLogic(t *testing.T) {
	testData := "hello world\n"
	fileName := "testfile.txt"
	attempts := 0

	// Simulate server that fails first few times then succeeds
	handlerThatFailsThenSucceeds := func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts <= 2 { // First 2 attempts fail
			t.Logf("simulating server error (attempt %d)", attempts)
			http.Error(w, "server temporarily unavailable", http.StatusInternalServerError)
			return
		}
		t.Logf("responding 200 OK (attempt %d)", attempts)
		_, _ = io.WriteString(w, testData)
	}

	tmpDir := t.TempDir()

	t.Run("server errors then success with retries", func(t *testing.T) {
		attempts = 0
		ts := httptest.NewServer(http.HandlerFunc(handlerThatFailsThenSucceeds))
		defer ts.Close()

		dest := filepath.Join(tmpDir, "retry-success.txt")
		err := util.DownloadFile(dest, ts.URL+"/"+fileName, false, "")
		require.NoError(t, err)
		require.Equal(t, 3, attempts, "expected exactly 3 attempts (2 failures + 1 success)")

		content, err := os.ReadFile(dest)
		require.NoError(t, err)
		require.Equal(t, testData, string(content))
	})

	t.Run("server errors exhaust all retries", func(t *testing.T) {
		attempts = 0
		// Server always returns error
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			attempts++
			t.Logf("simulating server error (attempt %d)", attempts)
			http.Error(w, "server permanently unavailable", http.StatusInternalServerError)
		}))
		defer ts.Close()

		dest := filepath.Join(tmpDir, "retry-fail.txt")
		// Set DdevDebug directly since the package variable is cached at init time
		origDebug := globalconfig.DdevDebug
		globalconfig.DdevDebug = true
		t.Cleanup(func() { globalconfig.DdevDebug = origDebug })
		// Also set log level since UserOut is initialized at package load time
		origLevel := output.UserOut.GetLevel()
		output.UserOut.SetLevel(log.DebugLevel)
		t.Cleanup(func() { output.UserOut.SetLevel(origLevel) })

		restoreOutput := util.CaptureUserOut()
		err := util.DownloadFile(dest, ts.URL+"/"+fileName, false, "")
		out := restoreOutput()

		require.Error(t, err)
		require.Contains(t, err.Error(), "giving up after")
		require.Contains(t, out, fmt.Sprintf("Retrying download of %s try #%d", ts.URL+"/"+fileName, 1), "expected log output for first retry, but got: %s", out)
		require.Contains(t, out, fmt.Sprintf("Retrying download of %s try #%d", ts.URL+"/"+fileName, 2), "expected log output for second retry, but got: %s", out)
		require.Equal(t, 3, attempts, "expected exactly 3 attempts (1 try * 3 retryablehttp attempts)")
		require.NoFileExistsf(t, dest, "expected file %s to be deleted after failure", dest)
	})

	t.Run("shasum server errors then success with retries", func(t *testing.T) {
		attempts = 0
		sum := fmt.Sprintf("%x", sha256.Sum256([]byte(testData)))

		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/" + fileName:
				t.Log("serving main file")
				_, _ = io.WriteString(w, testData)
			case "/sha256sums.txt":
				attempts++
				if attempts <= 2 { // First 2 attempts fail
					t.Logf("simulating shasum server error (attempt %d)", attempts)
					http.Error(w, "server temporarily unavailable", http.StatusInternalServerError)
					return
				}
				t.Log("serving shasum file")
				_, _ = fmt.Fprintf(w, "%s *%s\n", sum, fileName)
			default:
				http.NotFound(w, r)
			}
		}))
		defer ts.Close()

		dest := filepath.Join(tmpDir, "shasum-retry.txt")
		err := util.DownloadFile(dest, ts.URL+"/"+fileName, false, ts.URL+"/sha256sums.txt")
		require.NoError(t, err)
		require.Equal(t, 3, attempts, "expected exactly 3 attempts for shasum (2 failures + 1 success)")

		content, err := os.ReadFile(dest)
		require.NoError(t, err)
		require.Equal(t, testData, string(content))
	})

	t.Run("context deadline exceeded during client.Get", func(t *testing.T) {
		attempts = 0
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			attempts++
			t.Logf("handling request (attempt %d)", attempts)
			ticker := time.NewTicker(2 * time.Second)
			defer ticker.Stop()
			select {
			case <-r.Context().Done():
				t.Log("connection dropped from client")
				return
			case <-ticker.C:
			}
			t.Log("responding 200 OK")
			w.WriteHeader(http.StatusOK)
			_, _ = io.WriteString(w, testData)
		}))
		defer ts.Close()

		dest := filepath.Join(tmpDir, "timeout-retry.txt")
		// Use very short timeout to trigger context deadline exceeded
		err := util.DownloadFileExtended(dest, ts.URL+"/"+fileName, false, "", 1, 1*time.Second)
		require.ErrorIs(t, err, context.DeadlineExceeded, `expected "context deadline exceeded" error`)
		require.Equal(t, 2, attempts, "expected 2 attempts from retryablehttp")
		require.NoFileExistsf(t, dest, "expected file %s to be deleted after failure", dest)
	})

	t.Run("context deadline exceeded during file copy", func(t *testing.T) {
		attempts = 0
		// Server responds quickly but sends data slowly to trigger timeout during io.Copy
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			attempts++
			t.Logf("serving slow file copy (attempt %d)", attempts)

			// Write headers immediately so HTTP request succeeds
			w.Header().Set("Content-Length", "1000000") // 1MB
			w.WriteHeader(http.StatusOK)

			// Send data very slowly to trigger timeout during io.Copy
			flusher, ok := w.(http.Flusher)
			if !ok {
				t.Fatal("ResponseWriter doesn't support flushing")
			}

			for i := 0; i < 1000; i++ {
				select {
				case <-r.Context().Done():
					t.Log("client disconnected during slow transfer")
					return
				default:
					_, _ = w.Write(make([]byte, 1000)) // Write 1KB chunks
					flusher.Flush()
					time.Sleep(100 * time.Millisecond) // Add delay to trigger client timeout
				}
			}
		}))
		defer ts.Close()

		dest := filepath.Join(tmpDir, "slow-copy.txt")
		// Set DdevDebug directly since the package variable is cached at init time
		origDebug := globalconfig.DdevDebug
		globalconfig.DdevDebug = true
		t.Cleanup(func() { globalconfig.DdevDebug = origDebug })
		// Also set log level since UserOut is initialized at package load time
		origLevel := output.UserOut.GetLevel()
		output.UserOut.SetLevel(log.DebugLevel)
		t.Cleanup(func() { output.UserOut.SetLevel(origLevel) })

		restoreOutput := util.CaptureUserOut()
		err := util.DownloadFileExtended(dest, ts.URL+"/"+fileName, false, "", 2, 1*time.Second)
		out := restoreOutput()

		require.ErrorIs(t, err, context.DeadlineExceeded, `expected "context deadline exceeded" error`)
		require.Contains(t, out, "with timeout 1s", "expected log output for initial attempt with 1s timeout (2^0), but got: %s", out)
		require.Contains(t, out, "with timeout 2s", "expected log output for retry attempt with 2s timeout (2^1), but got: %s", out)
		require.Contains(t, out, "with timeout 4s", "expected log output for final retry attempt with 4s timeout (2^2), but got: %s", out)
		require.Equal(t, 3, attempts, "expected 3 attempts (1 initial + 2 retries)")
		require.NoFileExistsf(t, dest, "expected file %s to be deleted after failure", dest)
	})
}
