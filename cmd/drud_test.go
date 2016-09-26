package cmd

import (
	"os"
	"reflect"
	"testing"
)

var (
	binary = "drud" // The drud binary to use.
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

func setup() {
	if os.Getenv("DRUD_BINARY") != "" {
		binary = os.Getenv("DRUD_BINARY")
	}
}
