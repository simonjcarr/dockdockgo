package main

import (
	"dockdockgo/cmd/dockdockgo"
	"os"
)

var (
	Version   = "dev"
	Commit    = "unknown"
	BuildTime = "unknown"
)

func main() {
	dockdockgo.SetVersionInfo(Version, Commit, BuildTime)
	if err := dockdockgo.Execute(); err != nil {
		os.Exit(1)
	}
}
