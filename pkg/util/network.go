package util

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"hash"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/cheggaaa/pb/v3"
	"github.com/ddev/ddev/pkg/github"
	"github.com/ddev/ddev/pkg/output"
	"github.com/hashicorp/go-retryablehttp"
	"golang.org/x/term"
)

// DownloadFile retrieves a file with retry logic, optional progress bar, and SHA256 verification.
func DownloadFile(destPath string, fileURL string, progressBar bool, shaSumURL string) (err error) {
	return DownloadFileExtended(destPath, fileURL, progressBar, shaSumURL, 2, 20*time.Minute)
}

// DownloadFileExtended retrieves a file with retry logic, optional progress bar, and SHA256 verification.
// It allows specifying the number of retries and timeout duration.
func DownloadFileExtended(destPath string, fileURL string, progressBar bool, shaSumURL string, retries int, timeout time.Duration) (err error) {
	const timeoutMax = 1 * time.Hour

	if output.JSONOutput || !term.IsTerminal(int(os.Stdin.Fd())) {
		progressBar = false
	}

	// Configure retryablehttp client with backoff, retry policy, and global timeout.
	createClient := func(clientTimeout time.Duration, attempt int) (*retryablehttp.Client, time.Duration) {
		client := retryablehttp.NewClient()
		client.RetryMax = retries
		client.RetryWaitMin = 500 * time.Millisecond
		client.RetryWaitMax = 5 * time.Second
		// "context deadline exceeded" error during file copying cannot be retried with retryablehttp
		// See https://github.com/hashicorp/go-retryablehttp/issues/167
		// We use manual retry logic (for loop) around the entire Get+Copy operation instead
		client.CheckRetry = retryablehttp.DefaultRetryPolicy
		client.Backoff = retryablehttp.DefaultBackoff
		client.Logger = nil
		// Double the timeout for each retry attempt, up to timeoutMax
		// 1st attempt = clientTimeout * 2^0
		// 2nd attempt = clientTimeout * 2^1
		// 3rd attempt = clientTimeout * 2^2
		clientTimeout = clientTimeout * time.Duration(1<<attempt)
		if clientTimeout > timeoutMax {
			clientTimeout = timeoutMax
		}
		// Timeout for the entire request
		client.HTTPClient.Timeout = clientTimeout
		client.RequestLogHook = func(_ retryablehttp.Logger, req *http.Request, attempt int) {
			if attempt > 0 {
				// attempt==1 is the first retry, 2 the second, etc
				Debug("Retrying download of %s try #%d", req.URL.String(), attempt)
			}
		}
		return client, clientTimeout
	}

	// Ensure partial files are removed on any error.
	defer func() {
		if err != nil {
			_ = os.Remove(destPath)
		}
	}()

	// Download expected SHA sum if provided with retry logic.
	var expectedSHA string
	if shaSumURL != "" {
		// SHA is a smaller file, use a shorter timeout
		shaSumTimeout := 20 * time.Second
		for attempt := 0; attempt <= retries; attempt++ {
			client, currentTimeout := createClient(shaSumTimeout, attempt)
			Debug("Attempting to download SHASUM URL=%s (attempt %d/%d) with timeout %v", shaSumURL, attempt+1, retries+1, currentTimeout)
			req, reqErr := retryablehttp.NewRequest("GET", shaSumURL, nil)
			if reqErr != nil {
				err = fmt.Errorf("creating request for shaSum URL %s: %w", shaSumURL, reqErr)
				return
			}
			gitHubHeaders := github.GetGitHubHeaders(shaSumURL)
			for key, value := range gitHubHeaders {
				req.Header.Set(key, value)
			}
			resp, getErr := client.Do(req)
			if tokenErr := github.HasInvalidGitHubToken(resp); tokenErr != nil {
				WarningOnce("Warning: %v, retrying without token", tokenErr)
				for key := range gitHubHeaders {
					req.Header.Del(key)
				}
				respNoAuth, err := client.Do(req)
				if err == nil {
					if resp != nil {
						CheckClose(resp.Body)
					}
					resp = respNoAuth
					getErr = err
				}
			}
			if getErr != nil {
				err = fmt.Errorf("downloading shaSum URL %s: %w", shaSumURL, getErr)
				return
			}
			if resp.StatusCode != http.StatusOK {
				CheckClose(resp.Body)
				err = fmt.Errorf("unexpected HTTP status downloading %s: %s", shaSumURL, resp.Status)
				return
			}
			body, readErr := io.ReadAll(resp.Body)
			CheckClose(resp.Body)
			if readErr != nil {
				if errors.Is(readErr, context.DeadlineExceeded) && attempt < retries {
					Debug("SHASUM read attempt %d failed with timeout, retrying...", attempt+1)
					continue
				}
				err = fmt.Errorf("reading shaSum: %w", readErr)
				return
			}
			expectedSHA = strings.TrimSpace(string(body))
			break
		}
	}

	var hasher hash.Hash

	for attempt := 0; attempt <= retries; attempt++ {
		// Create/recreate the destination file for each attempt
		outFile, createErr := os.Create(destPath)
		if createErr != nil {
			err = createErr
			return
		}
		client, currentTimeout := createClient(timeout, attempt)
		// Download the main fileURL.
		Debug("Downloading %s to '%s' (attempt %d/%d) with timeout %v", fileURL, destPath, attempt+1, retries+1, currentTimeout)
		req, reqErr := retryablehttp.NewRequest("GET", fileURL, nil)
		if reqErr != nil {
			_ = outFile.Close()
			err = fmt.Errorf("creating request for file URL %s: %w", fileURL, reqErr)
			return
		}
		gitHubHeaders := github.GetGitHubHeaders(fileURL)
		for key, value := range gitHubHeaders {
			req.Header.Set(key, value)
		}
		resp, getErr := client.Do(req)
		if tokenErr := github.HasInvalidGitHubToken(resp); tokenErr != nil {
			WarningOnce("Warning: %v, retrying without token", tokenErr)
			for key := range gitHubHeaders {
				req.Header.Del(key)
			}
			respNoAuth, err := client.Do(req)
			if err == nil {
				if resp != nil {
					CheckClose(resp.Body)
				}
				resp = respNoAuth
				getErr = err
			}
		}
		if getErr != nil {
			_ = outFile.Close()
			err = fmt.Errorf("downloading file %s: %w", fileURL, getErr)
			return
		}
		if resp.StatusCode != http.StatusOK {
			CheckClose(resp.Body)
			_ = outFile.Close()
			err = fmt.Errorf("download link %s returned wrong status code: got %d want %d", fileURL, resp.StatusCode, http.StatusOK)
			return
		}

		// Wrap reader in progress bar if requested.
		reader := io.Reader(resp.Body)
		var bar *pb.ProgressBar
		if progressBar {
			bar = pb.Full.Start64(resp.ContentLength)
			reader = bar.NewProxyReader(resp.Body)
		}

		// Write file and compute SHA concurrently.
		hasher = sha256.New()
		writer := io.MultiWriter(outFile, hasher)

		_, copyErr := io.Copy(writer, reader)

		CheckClose(resp.Body)
		_ = outFile.Close()

		// Finish progress bar if it was used
		if bar != nil {
			bar.Finish()
		}

		if copyErr != nil {
			if errors.Is(copyErr, context.DeadlineExceeded) && attempt < retries {
				Debug("File copy attempt %d failed with timeout, retrying...", attempt+1)
				continue
			}
			err = copyErr
			return
		}
		// Success - break out of retry loop
		break
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
		output.UserOut.WithFields(output.Fields{
			"URL":      o.URL,
			"headers":  o.Headers,
			"expected": o.ExpectedStatus,
			"got":      resp.StatusCode,
		}).Infof("HTTP Status could not be matched, expected %d, received %d", o.ExpectedStatus, resp.StatusCode)
	}
	return fmt.Errorf("failed to match status code: %d: %v", o.ExpectedStatus, err)
}
