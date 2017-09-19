package goodhosts

import (
	"fmt"
	"testing"
)

func TestItemInSlice(t *testing.T) {
	item := "this"
	list := []string{"hello", "brah"}
	result := itemInSlice("goodbye", list)
	if result {
		t.Error(fmt.Sprintf("'%' should not have been found in slice.", item))
	}

	item = "hello"
	result = itemInSlice(item, list)
	if !result {
		t.Error(fmt.Sprintf("'%' should have been found in slice.", item))
	}
}
