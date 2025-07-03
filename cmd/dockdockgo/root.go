package dockdockgo

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	version   = "dev"
	commit    = "unknown"
	buildTime = "unknown"
)

var rootCmd = &cobra.Command{
	Use:   "dockdockgo",
	Short: "A docker swarm like application for container orchestration",
	Long: `DockDockGo is a CLI and API application that provides docker swarm-like 
functionality with enhanced remote management capabilities. It allows you to 
deploy and manage containers across multiple remote servers with ease.`,
	Run: func(cmd *cobra.Command, args []string) {
		showVersion, _ := cmd.Flags().GetBool("version")
		if showVersion {
			printVersion()
			return
		}
		fmt.Println("DockDockGo - Container orchestration made simple")
		fmt.Println("Use 'dockdockgo --help' for available commands")
	},
}

func Execute() error {
	return rootCmd.Execute()
}

func SetVersionInfo(v, c, bt string) {
	version = v
	commit = c
	buildTime = bt
}

func printVersion() {
	fmt.Printf("DockDockGo version %s\n", version)
	fmt.Printf("Commit: %s\n", commit)
	fmt.Printf("Built: %s\n", buildTime)
	os.Exit(0)
}

func init() {
	rootCmd.Flags().BoolP("version", "v", false, "Print version information")
}