package dockdockgo

import (
	"fmt"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "dockdockgo",
	Short: "A docker swarm like application for container orchestration",
	Long: `DockDockGo is a CLI and API application that provides docker swarm-like 
functionality with enhanced remote management capabilities. It allows you to 
deploy and manage containers across multiple remote servers with ease.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("DockDockGo - Container orchestration made simple")
		fmt.Println("Use 'dockdockgo --help' for available commands")
	},
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.Flags().BoolP("version", "v", false, "Print version information")
}