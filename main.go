package main

import (
	"dockdockgo/cmd/dockdockgo"
	"os"
)

func main() {
	if err := dockdockgo.Execute(); err != nil {
		os.Exit(1)
	}
}