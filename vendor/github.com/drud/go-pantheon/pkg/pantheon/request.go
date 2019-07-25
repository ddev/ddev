package pantheon

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"path"
	"strings"
)

// APIHost in the Hostname + basepath for the pantheon API endpoint.
var APIHost = "https://terminus.pantheon.io:443/api"

// Default HTTP header values
const (
	contentType = "application/json"
	userAgent   = "Terminus/1.9 (php_version=7.1.5&script=bin/terminus)"
)

// RequestEntity provides an interface for making requests to the Pantheon API and marshaling/unmarshaling JSON data. Any object which
// wishes to use the Request type must implement this interface.
type RequestEntity interface {
	Path(method string, auth AuthSession) string // Path returns the request path of the entity for a given HTTP method.
	Unmarshal(data []byte) error                 // Unmarshal is responsible for converting the []byte response back into the struct.
	JSON() ([]byte, error)                       // JSON() is responsible for preparing the struct for HTTP transport. It is responsible for removing any fields which should not be included in the request.
}

// setRequestHeaders sets default headers that should be used for all HTTP requests.
func setRequestHeaders(req *http.Request) {
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("User-Agent", userAgent)
}

// httpRequest performs a HTTP request for a given resource.
func httpRequest(requestType string, requestPath string, body []byte, headers map[string]string) ([]byte, error) {
	var req *http.Request
	var err error
	u, err := url.Parse(APIHost)
	if err != nil {
		return nil, err
	}

	u.Path = path.Join(u.Path, requestPath)
	req, err = http.NewRequest(strings.ToUpper(requestType), u.String(), bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}

	setRequestHeaders(req)

	if len(headers) > 0 {
		for key, value := range headers {
			req.Header.Set(key, value)
		}
	}
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode-200 > 100 {
		return nil, fmt.Errorf("%s: %d", resp.Status, resp.StatusCode)
	}

	return respBody, nil
}
