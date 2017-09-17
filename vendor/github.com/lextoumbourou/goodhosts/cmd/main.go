package main

import (
	"fmt"
	"github.com/docopt/docopt-go"
	"github.com/lextoumbourou/goodhosts"
	"os"
)

func check(err error) {
	if err != nil {
		panic(err)
	}
}

func main() {
	usage := `Goodhosts - simple hosts file management.

Usage:
  goodhosts check <ip> <host>...
  goodhosts add <ip> <host>...
  goodhosts (rm|remove) <ip> <host>...
  goodhosts list [--all]
  goodhosts -h | --help
  goodhosts --version

Options:
  --all         Display comments when listing.
  -h --help     Show this screen.
  --version     Show the version.`

	args, _ := docopt.Parse(usage, nil, true, "Goodhosts 2.0.0", false)

	hosts, err := goodhosts.NewHosts()
	check(err)

	if args["list"].(bool) {
		total := 0
		for _, line := range hosts.Lines {
			var lineOutput string

			if line.IsComment() && !args["--all"].(bool) {
				continue
			}

			lineOutput = fmt.Sprintf("%s", line.Raw)
			if line.Err != nil {
				lineOutput = fmt.Sprintf("%s # <<< Malformated!", lineOutput)
			}
			total += 1

			fmt.Println(lineOutput)
		}

		fmt.Printf("\nTotal: %d\n", total)

		return
	}

	if args["check"].(bool) {
		hasErr := false

		ip := args["<ip>"].(string)
		hostEntries := args["<host>"].([]string)

		for _, hostEntry := range hostEntries {
			if !hosts.Has(ip, hostEntry) {
				fmt.Fprintln(os.Stderr, fmt.Sprintf("%s %s is not in the hosts file", ip, hostEntry))
				hasErr = true
			}
		}

		if hasErr {
			os.Exit(1)
		}

		return
	}

	if args["add"].(bool) {
		ip := args["<ip>"].(string)
		hostEntries := args["<host>"].([]string)

		if !hosts.IsWritable() {
			fmt.Fprintln(os.Stderr, "Host file not writable. Try running with elevated privileges.")
			os.Exit(1)
		}

		err = hosts.Add(ip, hostEntries...)
		if err != nil {
			fmt.Fprintln(os.Stderr, fmt.Sprintf("%s", err.Error()))
			os.Exit(2)
		}

		err = hosts.Flush()
		check(err)

		return
	}

	if args["rm"].(bool) || args["remove"].(bool) {
		ip := args["<ip>"].(string)
		hostEntries := args["<host>"].([]string)

		if !hosts.IsWritable() {
			fmt.Fprintln(os.Stderr, "Host file not writable. Try running with elevated privileges.")
			os.Exit(1)
		}

		err = hosts.Remove(ip, hostEntries...)
		if err != nil {
			fmt.Fprintf(os.Stderr, fmt.Sprintf("%s\n", err.Error()))
			os.Exit(2)
		}

		err = hosts.Flush()
		check(err)

		return
	}
}
