package util

import (
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"github.com/cheggaaa/pb"
	"github.com/drud/ddev/pkg/output"
	log "github.com/sirupsen/logrus"
)

// DownloadFile retrieves a file.
func DownloadFile(fp string, url string, progressBar bool) (err error) {
	// Create the file
	out, err := os.Create(fp)
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

		bar := pb.New(int(resp.ContentLength)).SetUnits(pb.U_BYTES).Prefix(filepath.Base(fp))
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
	tickerInt := o.TickerInterval
	if tickerInt == 0 {
		tickerInt = 20
	}

	giveUp := make(chan bool)
	go func() {
		time.Sleep(time.Second * o.Timeout)
		giveUp <- true
	}()

	client := &http.Client{}
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return errors.New("received http redirect")
	}

	var respCode int
	queryTicker := time.NewTicker(time.Second * tickerInt).C
	for {
		select {
		case <-queryTicker:
			req, err := http.NewRequest("GET", o.URL, nil)
			CheckErr(err)
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
					// Log expected vs. actual if we do not get a match.
					output.UserOut.WithFields(log.Fields{
						"URL":      o.URL,
						"expected": o.ExpectedStatus,
						"got":      resp.StatusCode,
					}).Info("HTTP Status code matched expectations")
					return nil
				}

				// Log expected vs. actual if we do not get a match.
				log.WithFields(log.Fields{
					"URL":      o.URL,
					"expected": o.ExpectedStatus,
					"got":      resp.StatusCode,
				}).Info("HTTP Status could not be matched")

				respCode = resp.StatusCode
			}

		case <-giveUp:
			return fmt.Errorf("timed out after %d seconds. Got status %d, wanted %d", o.Timeout, respCode, o.ExpectedStatus)
		}
	}
}

// IsPortActive checks to see if the given port on localhost is answering.
func IsPortActive(port string) bool {
	conn, err := net.Dial("tcp", ":"+port)
	// If we were able to connect, something is listening on the port.
	if err == nil {
		_ = conn.Close()
		return true
	}
	// If we get ECONNREFUSED the port is not active.
	oe, ok := err.(*net.OpError)
	if ok {
		syscallErr, ok := oe.Err.(*os.SyscallError)

		// On Windows, WSAECONNREFUSED (10061) results instead of ECONNREFUSED. And golang doesn't seem to have it.
		var WSAECONNREFUSED syscall.Errno = 10061

		if ok && (syscallErr.Err == syscall.ECONNREFUSED || syscallErr.Err == WSAECONNREFUSED) {
			return false
		}
	}
	// Otherwise, hmm, something else happened. It's not a fatal or anything.
	Warning("Unable to properly check port status: %v", oe)
	return false
}
