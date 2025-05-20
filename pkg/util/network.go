package util

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"golang.org/x/term"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/cheggaaa/pb"
	"github.com/ddev/ddev/pkg/output"
	retryablehttp "github.com/hashicorp/go-retryablehttp"
	log "github.com/sirupsen/logrus"
)

// DownloadFile retrieves a file with retry logic, optional progress bar, and SHA256 verification.
func DownloadFile(destPath string, fileURL string, progressBar bool, shaSumURL string) (err error) {
	if output.JSONOutput || !term.IsTerminal(int(os.Stdin.Fd())) {
		progressBar = false
	}

	// Configure retryablehttp client with backoff, retry policy, and global timeout.
	client := retryablehttp.NewClient()
	client.RetryMax = 4
	client.RetryWaitMin = 500 * time.Millisecond
	client.RetryWaitMax = 5 * time.Second
	client.CheckRetry = func(ctx context.Context, resp *http.Response, err error) (bool, error) {
		// Default retry policy only retries on
		// - connection reset
		// - connection refused
		// - No Response
		// - net.Error with Temporary() == true
		if err != nil && strings.Contains(err.Error(), "context deadline exceeded") {
			return true, nil
		}
		return retryablehttp.DefaultRetryPolicy(ctx, resp, err)
	}
	client.Backoff = retryablehttp.DefaultBackoff
	client.Logger = nil
	client.HTTPClient.Timeout = 5 * time.Minute
	client.RequestLogHook = func(_ retryablehttp.Logger, req *http.Request, attempt int) {
		if attempt > 0 {
			// attempt==1 is the first retry, 2 the second, etc
			Debug("Retrying download of %s try #%d", req.URL.String(), attempt)
		}
	}

	// Ensure partial files are removed on any error.
	defer func() {
		if err != nil {
			_ = os.Remove(destPath)
		}
	}()

	// Download expected SHA sum if provided.
	var expectedSHA string
	if shaSumURL != "" {
		Debug("Attempting to download SHASUM URL=%s", shaSumURL)
		resp, getErr := client.Get(shaSumURL)
		if getErr != nil {
			err = fmt.Errorf("downloading shaSum URL %s: %w", shaSumURL, getErr)
			return
		}
		defer CheckClose(resp.Body)
		if resp.StatusCode != http.StatusOK {
			err = fmt.Errorf("unexpected HTTP status downloading %s: %s", shaSumURL, resp.Status)
			return
		}
		body, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			err = fmt.Errorf("reading shaSum: %w", readErr)
			return
		}
		expectedSHA = strings.TrimSpace(string(body))
	}

	// Create the destination file.
	outFile, createErr := os.Create(destPath)
	if createErr != nil {
		err = createErr
		return
	}
	defer func() {
		if closeErr := outFile.Close(); closeErr != nil && err == nil {
			err = closeErr
		}
	}()

	// Download the main fileURL.
	Debug("Downloading %s to %s", fileURL, destPath)
	resp, getErr := client.Get(fileURL)
	if getErr != nil {
		err = fmt.Errorf("downloading file %s: %w", fileURL, getErr)
		return
	}
	defer CheckClose(resp.Body)
	if resp.StatusCode != http.StatusOK {
		err = fmt.Errorf("download link %s returned wrong status code: got %d want %d", fileURL, resp.StatusCode, http.StatusOK)
		return
	}

	// Wrap reader in progress bar if requested.
	reader := io.Reader(resp.Body)
	if progressBar {
		bar := pb.New(int(resp.ContentLength)).SetUnits(pb.U_BYTES).Prefix(filepath.Base(destPath))
		bar.Start()
		reader = bar.NewProxyReader(resp.Body)
		defer bar.Finish()
	}

	// Write file and compute SHA concurrently.
	hasher := sha256.New()
	writer := io.MultiWriter(outFile, hasher)
	if _, copyErr := io.Copy(writer, reader); copyErr != nil {
		err = copyErr
		return
	}

	// Verify SHA if provided.
	if expectedSHA != "" {
		baseName := filepath.Base(fileURL)
		var matchedSHA string
		for _, line := range strings.Split(expectedSHA, "\n") {
			fields := strings.Fields(line)
			if len(fields) != 2 {
				continue
			}
			filename := strings.TrimPrefix(fields[1], "*")
			if filename == baseName {
				matchedSHA = fields[0]
				break
			}
		}
		if matchedSHA == "" {
			err = fmt.Errorf("no matching SHA256 found for %s in shaSum file", baseName)
			return
		}
		actualSHA := fmt.Sprintf("%x", hasher.Sum(nil))
		if actualSHA != matchedSHA {
			err = fmt.Errorf("SHA256 mismatch: expected %s, got %s", matchedSHA, actualSHA)
			return
		}
	}

	return nil
}

// HTTPOptions defines the URL and other common HTTP options for EnsureHTTPStatus.
type HTTPOptions struct {
	URL            string
	Username       string
	Password       string
	Timeout        time.Duration
	TickerInterval time.Duration
	ExpectedStatus int
	Headers        map[string]string
}

// NewHTTPOptions returns a new HTTPOptions struct with some sane defaults.
func NewHTTPOptions(URL string) *HTTPOptions {
	o := HTTPOptions{
		URL:            URL,
		TickerInterval: 20,
		Timeout:        60,
		ExpectedStatus: http.StatusOK,
		Headers:        make(map[string]string),
	}
	return &o
}

// EnsureHTTPStatus will verify a URL responds with a given response code within the Timeout period (in seconds)
func EnsureHTTPStatus(o *HTTPOptions) error {
	client := &http.Client{
		Timeout: o.Timeout * time.Second,
	}
	client.CheckRedirect = func(_ *http.Request, _ []*http.Request) error {
		return errors.New("received http redirect")
	}

	req, err := http.NewRequest("GET", o.URL, nil)
	if err != nil {
		return err
	}
	if o.Username != "" && o.Password != "" {
		req.SetBasicAuth(o.Username, o.Password)
	}

	if len(o.Headers) > 0 {
		for header, value := range o.Headers {
			if header == "Host" {
				req.Host = value
				continue
			}
			req.Header.Add(header, value)
		}
	}
	// Make the request
	resp, err := client.Do(req)

	if err == nil {
		defer CheckClose(resp.Body)
		if o.ExpectedStatus != 0 && resp.StatusCode == o.ExpectedStatus {
			return nil
		}
		// Log expected vs. actual if we do not get a match.
		output.UserOut.WithFields(log.Fields{
			"URL":      o.URL,
			"headers":  o.Headers,
			"expected": o.ExpectedStatus,
			"got":      resp.StatusCode,
		}).Infof("HTTP Status could not be matched, expected %d, received %d", o.ExpectedStatus, resp.StatusCode)
	}
	return fmt.Errorf("failed to match status code: %d: %v", o.ExpectedStatus, err)
}
