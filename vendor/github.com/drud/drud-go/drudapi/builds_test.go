package drudapi

import (
	"log"
	"testing"
)

func TestGetBuild200(t *testing.T) {
	expectedResp := `{
    "_updated": "Mon, 23 May 2016 20:23:34 GMT",
    "name": "funbuild",
    "branch": "master",
    "state": "success",
	"logs": "wooot!",
    "_created": "Mon, 23 May 2016 20:23:34 GMT",
    "_id": "574366c6e2638a001f430115",
    "_etag": "9906d3a8584f0fabbd96451013b38d20fff5f5d3"
}`
	server := getTestServer(200, expectedResp)
	defer server.Close()

	r := Request{
		Host: server.URL,
		Auth: &Credentials{
			AdminToken: "dgdfg",
		},
	}

	bd := &Build{
		Name: "funbuild",
	}

	err := r.Get(bd)
	if err != nil {
		log.Fatal(err)
	}

	expect(t, bd.Name, "funbuild")
	expect(t, bd.Branch, "master")
	expect(t, bd.State, "success")
	expect(t, bd.Logs, "wooot!")

}

// TestPostBuild tests that when posting a build the server receives data that is formed
// correctly and that given a proper response the build will know its id and etag
func TestPostBuild(t *testing.T) {
	expectedResp := `{
    "_updated": "Mon, 13 Jun 2016 16:21:25 GMT",
    "name": "testybuild",
	"state": "failed",
	"logs": "booo!",
    "_created": "Mon, 13 Jun 2016 16:21:25 GMT",
    "_status": "OK",
    "_id": "98734598723094857023985",
    "_etag": "qwertyuiop12345"
}`

	reqData := map[string]string{
		"content-type":  "application/json",
		"authorization": "token dfgdfg",
		"endpoint":      "/builds",
	}

	c := &Client{
		Name:  "testyclient",
		Email: "client@testy.com",
		Phone: "3192832323",
	}

	bd := &Build{
		Name:   "testybuild",
		Client: *c,
	}

	reqData["payload"] = string(bd.JSON())

	server := postTestServer(t, expectedResp, reqData)
	defer server.Close()

	r := Request{
		Host: server.URL,
		Auth: &Credentials{
			AdminToken: "dfgdfg",
		},
	}

	err := r.Post(bd)
	if err != nil {
		log.Fatal(err)
	}

	expect(t, bd.Etag, "qwertyuiop12345")
	expect(t, bd.ID, "98734598723094857023985")
	expect(t, bd.State, "failed")
	expect(t, bd.Logs, "booo!")

}
