package drudapi

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func expect(t *testing.T, a interface{}, b interface{}) {
	if a != b {
		t.Errorf("Expected %v (type %v) - Got %v (type %v)", b, reflect.TypeOf(b), a, reflect.TypeOf(a))
	}
}

func refute(t *testing.T, a interface{}, b interface{}) {
	if a == b {
		t.Errorf("Did not expect %v (type %v) - Got %v (type %v)", b, reflect.TypeOf(b), a, reflect.TypeOf(a))
	}
}

func getTestServer(code int, body string) *httptest.Server {

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// @todo make sure thigns being sent to tests server are correct
		w.WriteHeader(code)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, body)
	}))

	return server
}

func postTestServer(t *testing.T, body string, reqData map[string]string) *httptest.Server {
	code := 200

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// @todo make sure thigns being sent to tests server are correct
		expect(t, r.Header.Get("Content-Type"), reqData["content-type"])
		expect(t, r.Header.Get("Authorization"), reqData["authorization"])
		expect(t, r.RequestURI, reqData["endpoint"])
		expect(t, r.ContentLength, int64(len(reqData["payload"])))

		w.WriteHeader(code)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, body)
	}))

	return server
}
