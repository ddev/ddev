package drudapi

import (
	"log"
	"testing"
)

func TestGetClient200(t *testing.T) {
	expectedResp := `{
    "_updated": "Mon, 23 May 2016 20:23:34 GMT",
    "name": "1fee",
    "email": "me@there.com",
    "phone": "123-123-1234",
    "_links": {
        "self": {
            "href": "client/1fee",
            "title": "Client"
        },
        "parent": {
            "href": "/",
            "title": "home"
        },
        "collection": {
            "href": "client",
            "title": "client"
        }
    },
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

	c := &Client{
		Name: "1fee",
	}

	err := r.Get(c)
	if err != nil {
		log.Fatal(err)
	}

	expect(t, c.Name, "1fee")
	expect(t, c.Email, "me@there.com")
	expect(t, c.Phone, "123-123-1234")
	expect(t, c.Created, "Mon, 23 May 2016 20:23:34 GMT")
	expect(t, c.ID, "574366c6e2638a001f430115")
	expect(t, c.Etag, "9906d3a8584f0fabbd96451013b38d20fff5f5d3")

}

// TestPostClient tests that when posting a client the server receives data that is formed
// correctly and that given a proper response the client will know its id and etag
func TestPostClient(t *testing.T) {
	expectedResp := `{
    "_updated": "Mon, 13 Jun 2016 16:21:25 GMT",
    "name": "testyclient",
	"email": "client@testy.com",
	"phone": "3192832323",
    "_links": {
        "self": {
            "href": "client/testy",
            "title": "Client"
        }
    },
    "_created": "Mon, 13 Jun 2016 16:21:25 GMT",
    "_status": "OK",
    "_id": "98734598723094857023985",
    "_etag": "qwertyuiop12345"
}`

	reqData := map[string]string{
		"content-type":  "application/json",
		"authorization": "token dfgdfg",
		"endpoint":      "/client",
	}

	c := &Client{
		Name:  "testyclient",
		Email: "client@testy.com",
		Phone: "3192832323",
	}

	reqData["payload"] = string(c.JSON())

	server := postTestServer(t, expectedResp, reqData)
	defer server.Close()

	r := Request{
		Host: server.URL,
		Auth: &Credentials{
			AdminToken: "dfgdfg",
		},
	}

	err := r.Post(c)
	if err != nil {
		log.Fatal(err)
	}

	expect(t, c.Etag, "qwertyuiop12345")
	expect(t, c.ID, "98734598723094857023985")

}
