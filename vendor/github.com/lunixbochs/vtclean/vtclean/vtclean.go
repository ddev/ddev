package main

import (
	"flag"
	"github.com/lunixbochs/vtclean"
	"io"
	"os"
)

func main() {
	color := flag.Bool("color", false, "enable color")
	flag.Parse()

	stdout := vtclean.NewWriter(os.Stdout, *color)
	defer stdout.Close()
	io.Copy(stdout, os.Stdin)
}
