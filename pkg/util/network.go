package util

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"
	"golang.org/x/term"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/cheggaaa/pb"
	"github.com/ddev/ddev/pkg/output"
	log "github.com/sirupsen/logrus"
)

// DownloadFile retrieves a file.
func DownloadFile(destPath string, url string, progressBar bool, shaSumURL string) (err error) {
	if output.JSONOutput || !term.IsTerminal(int(os.Stdin.Fd())) {
		progressBar = false
	}

	// If shaSumURL is provided, download and read the expected SHASUM
	var expectedSHA string
	if shaSumURL != "" {
		resp, err := http.Get(shaSumURL)
		if err != nil {
			return fmt.Errorf("failed to download shaSum URL %s: %v", shaSumURL, err)
		}
		defer CheckClose(resp.Body)

		shaBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("failed to read shaSum file: %v", err)
		}
		expectedSHA = string(bytes.TrimSpace(shaBytes))
	}

	// Create the file
	out, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer CheckClose(out)

	// Get the data
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer CheckClose(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download link %s returned wrong status code: got %v want %v", url, resp.StatusCode, http.StatusOK)
	}

	reader := resp.Body
	if progressBar {
		bar := pb.New(int(resp.ContentLength)).SetUnits(pb.U_BYTES).Prefix(filepath.Base(destPath))
		bar.Start()
		reader = bar.NewProxyReader(resp.Body)
		defer bar.Finish()
	}

	hasher := sha256.New()
	writer := io.MultiWriter(out, hasher)
	if _, err = io.Copy(writer, reader); err != nil {
		return err
	}

	if expectedSHA != "" {
		baseName := filepath.Base(url)
		var matchedSHA string

		lines := bytes.Split([]byte(expectedSHA), []byte{'\n'})
		for _, line := range lines {
			fields := bytes.Fields(line)
			if len(fields) != 2 {
				continue
			}
			filename := bytes.TrimPrefix(fields[1], []byte("*"))
			if bytes.Equal(filename, []byte(baseName)) {
				matchedSHA = string(fields[0])
				break
			}
		}

		if matchedSHA == "" {
			_ = os.Remove(destPath)
			return fmt.Errorf("no matching SHA256 found for %s in shaSum file", baseName)
		}

		actualSHA := fmt.Sprintf("%x", hasher.Sum(nil))
		if actualSHA != matchedSHA {
			_ = os.Remove(destPath)
			return fmt.Errorf("SHA256 mismatch: expected %s, got %s", matchedSHA, actualSHA)
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
