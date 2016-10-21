package utils

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"

	log "github.com/Sirupsen/logrus"
)

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

// Returns a new HTTPOptions struct with some sane defaults.
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

	queryTicker := time.NewTicker(time.Second * tickerInt).C
	for {
		select {
		case <-queryTicker:
			req, err := http.NewRequest("GET", o.URL, nil)
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
				defer resp.Body.Close()
				if resp.StatusCode == o.ExpectedStatus {
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
			}

		case <-giveUp:
			return fmt.Errorf("No deployment found after waiting %d seconds", o.Timeout)
		}
	}
}

// IsTCPPortAvailable checks a port to see if anythign is listening
// hostPort is a string like 'localhost:80'
func IsTCPPortAvailable(port int) bool {
	conn, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		return false
	}
	conn.Close()
	return true
}
