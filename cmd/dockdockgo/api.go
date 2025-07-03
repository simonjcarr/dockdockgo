package dockdockgo

import (
	"dockdockgo/internal/api"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var apiCmd = &cobra.Command{
	Use:   "api",
	Short: "API server management",
	Long:  `Start and manage the DockDockGo API server for remote management.`,
}

var apiStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the API server",
	Run: func(cmd *cobra.Command, args []string) {
		port, _ := cmd.Flags().GetString("port")
		host, _ := cmd.Flags().GetString("host")

		server := api.NewServer(host, port)
		if err := server.Start(); err != nil {
			fmt.Printf("Failed to start API server: %v\n", err)
			os.Exit(1)
		}
	},
}

var apiTokenCmd = &cobra.Command{
	Use:   "token",
	Short: "Generate API access token",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Generating new API token...")
	},
}

func init() {
	rootCmd.AddCommand(apiCmd)
	apiCmd.AddCommand(apiStartCmd)
	apiCmd.AddCommand(apiTokenCmd)

	apiStartCmd.Flags().StringP("port", "p", "8080", "Port to listen on")
	apiStartCmd.Flags().StringP("host", "H", "0.0.0.0", "Host to bind to")
	apiStartCmd.Flags().BoolP("tls", "t", false, "Enable TLS")
	apiStartCmd.Flags().StringP("cert", "c", "", "TLS certificate file")
	apiStartCmd.Flags().StringP("key", "k", "", "TLS private key file")
}
