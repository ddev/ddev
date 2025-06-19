package main

import (
	"github.com/ddev/ddev/pkg/hostname"
	"github.com/ddev/ddev/pkg/util"
)

func main() {
	err := hostname.AddHostEntry("something.example.com", "127.0.0.1")
	if err != nil {
		util.Failed("Failed to add host entry: %v", err)
	}
}
