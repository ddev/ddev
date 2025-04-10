package util

import (
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
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
			io.WriteString(w, testData)
		case "/sha256sums.txt":
			fmt.Fprintf(w, "%s *%s\n", sum, fileName)
		case "/badsha256sums.txt":
			fmt.Fprintf(w, "%s *%s\n", "deadbeef", fileName)
		default:
			http.NotFound(w, r)
		}
	}))
	defer ts.Close()

	tmpDir := t.TempDir()

	t.Run("no shasum", func(t *testing.T) {
		dest := filepath.Join(tmpDir, "plain.txt")
		err := DownloadFile(dest, ts.URL+"/"+fileName, false, "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("correct shasum", func(t *testing.T) {
		dest := filepath.Join(tmpDir, "verified.txt")
		err := DownloadFile(dest, ts.URL+"/"+fileName, false, ts.URL+"/sha256sums.txt")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("incorrect shasum", func(t *testing.T) {
		dest := filepath.Join(tmpDir, "bad.txt")
		err := DownloadFile(dest, ts.URL+"/"+fileName, false, ts.URL+"/badsha256sums.txt")
		if err == nil || err.Error() == "" {
			t.Fatal("expected SHA256 mismatch error, got nil")
		}
	})
}
