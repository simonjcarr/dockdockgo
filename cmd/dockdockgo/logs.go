package dockdockgo

import (
	"dockdockgo/internal/docker"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
)

var logsCmd = &cobra.Command{
	Use:   "logs [CONTAINER]",
	Short: "Fetch the logs of a container",
	Long:  `Fetch the logs of a container running locally or on remote servers.`,
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		containerID := args[0]
		follow, _ := cmd.Flags().GetBool("follow")
		servers, _ := cmd.Flags().GetStringSlice("servers")

		if len(servers) == 0 {
			// Get logs from local container
			if err := getLocalContainerLogs(containerID, follow); err != nil {
				fmt.Printf("Failed to get container logs: %v\n", err)
				return
			}
		} else {
			fmt.Println("Remote container logs not yet implemented")
		}
	},
}

func getLocalContainerLogs(containerID string, follow bool) error {
	dockerClient, err := docker.NewClient()
	if err != nil {
		return fmt.Errorf("failed to create Docker client: %w", err)
	}
	defer dockerClient.Close()

	logReader, err := dockerClient.GetContainerLogs(containerID, follow)
	if err != nil {
		return fmt.Errorf("failed to get container logs: %w", err)
	}
	defer logReader.Close()

	// Stream logs to stdout
	_, err = io.Copy(os.Stdout, logReader)
	return err
}

func init() {
	rootCmd.AddCommand(logsCmd)
	
	logsCmd.Flags().BoolP("follow", "f", false, "Follow log output")
	logsCmd.Flags().StringSliceP("servers", "s", []string{}, "Get logs from containers on remote servers")
}