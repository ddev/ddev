package utils

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"

	log "github.com/Sirupsen/logrus"
)

// Verify a deployment exists at a URL within the timeout period (in seconds)
func EnsureHTTPStatus(targetURL string, username string, password string, timeout int, expectedStatus int) error {
	giveUp := make(chan bool)
	go func() {
		time.Sleep(time.Second * time.Duration(timeout))
		giveUp <- true
	}()
	client := &http.Client{}
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return errors.New("Redirect")
	}
	queryTicker := time.NewTicker(time.Second * 20).C
	for {
		select {
		case <-queryTicker:
			req, err := http.NewRequest("GET", targetURL, nil)
			if username != "" && password != "" {
				req.SetBasicAuth(username, password)
			}

			// Make the request
			res, err := client.Do(req)
			if err == nil {
				if res.StatusCode == expectedStatus {
					// Log expected vs. actual if we do not get a match.
					log.WithFields(log.Fields{
						"URL":      targetURL,
						"expected": expectedStatus,
						"got":      res.StatusCode,
					}).Info("HTTP Status code matched expectations")
					return nil
				}

				// Log expected vs. actual if we do not get a match.
				log.WithFields(log.Fields{
					"URL":      targetURL,
					"expected": expectedStatus,
					"got":      res.StatusCode,
				}).Info("HTTP Status could not be matched")
			}

		case <-giveUp:
			return fmt.Errorf("No deployment found after waiting %d seconds", timeout)
		}
	}
}

// IsTCPPortAvailable checks a port to see if anythign is listening
// hostPort is a string like 'localhost:80'
func IsTCPPortAvailable(hostPort string) bool {
	conn, err := net.Listen("tcp", hostPort)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}
