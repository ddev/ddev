package util

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	log "github.com/Sirupsen/logrus"
)

// DownloadFile retreives a file from url and stores it into filepath
func DownloadFile(filepath string, url string) (err error) {
	// Create the file
	out, err := os.Create(filepath)
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

	// Write the body to file
	_, err = io.Copy(out, resp.Body)
	return err
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
		return errors.New("Redirect")
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
					log.WithFields(log.Fields{
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
