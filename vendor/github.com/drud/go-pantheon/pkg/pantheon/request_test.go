package pantheon

import (
	"bytes"
	"fmt"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"net/http"
	"testing"
)

const requestPath = "/foo/bar"

// Test request header for default headers and a custom header, we set.
// Test for success when sending an HTTP request body.
func TestHttpRequest(t *testing.T) {
	assert := assert.New(t)
	expectedHeaderValue := "Bar"

	// Define a route on the test server.
	mux.HandleFunc(requestPath, func(w http.ResponseWriter, r *http.Request) {
		// Check the request for default headers.
		assert.Equal(contentType, r.Header.Get("Content-Type"))
		assert.Equal(userAgent, r.Header.Get("User-Agent"))
		// Check the request for custom headers.
		assert.Equal(expectedHeaderValue, r.Header.Get("Foo"))
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			assert.Error(err)
		}

		// Check that the request body is carrying our value.
		actual := string(body)
		assert.Equal("Bar", actual)
		// Write a response body with the value extracted from the request body.
		fmt.Fprint(w, actual)
	})

	// Construct a new request with custom header.
	resp, err := httpRequest("GET", requestPath, []byte("Bar"), map[string]string{"Foo": expectedHeaderValue})
	if err != nil {
		assert.Error(err)
	}

	// Read the byte response.
	// Then convert it to a string.
	var actual string
	raw := bytes.NewBuffer(resp)
	actual = raw.String()
	// Check for equality between the response body and expected value.
	assert.Equal(expectedHeaderValue, actual)
}
