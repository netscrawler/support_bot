package main

import (
	"fmt"
	"os"
	"runtime"
	"strings"

	"support_bot/internal/cli"
)

var (
	Version   = "v0.0.0"
	Commit    = "unknown"
	BuildTime = "unknown"
)

func main() {
	var err error
	if len(os.Args) < 2 {
		err = cli.Run(Version, Commit, BuildTime, nil)
	} else {
		switch os.Args[1] {
		case "run":
			err = cli.Run(Version, Commit, BuildTime, os.Args[2:])
		case "version":
			version()
		case "ctl":
			cli.Ctl()
		case "config":
			cli.Config()
		case "help", "-h", "--help":
			cli.Help()
		default:
			if strings.HasPrefix(os.Args[1], "-") {
				err = cli.Run(Version, Commit, BuildTime, os.Args[1:])
			} else {
				cli.Help()
				os.Exit(1)
			}
		}
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func version() {
	fmt.Printf(
		"Version: %s\nCommit: %s\nBuildTime: %s\nRuntime: %s",
		Version,
		Commit,
		BuildTime,
		runtime.Version(),
	)
	os.Exit(0)
}
