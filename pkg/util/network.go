package util

import (
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
func DownloadFile(destPath string, url string, progressBar bool) (err error) {
	if output.JSONOutput || !term.IsTerminal(int(os.Stdin.Fd())) {
		progressBar = false
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

		// create proxy reader
		reader = bar.NewProxyReader(resp.Body)
		// Writer the body to file
		_, err = io.Copy(out, reader)
		bar.Finish()
	} else {
		_, err = io.Copy(out, reader)
	}

	if err != nil {
		return err
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
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
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
