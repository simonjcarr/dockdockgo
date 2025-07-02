package dockdockgo

import (
	"dockdockgo/internal/docker"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

var psCmd = &cobra.Command{
	Use:   "ps",
	Short: "List containers",
	Long:  `List containers running locally or on remote servers.`,
	Run: func(cmd *cobra.Command, args []string) {
		all, _ := cmd.Flags().GetBool("all")
		servers, _ := cmd.Flags().GetStringSlice("servers")

		if len(servers) == 0 {
			// List local containers
			if err := listLocalContainers(all); err != nil {
				fmt.Printf("Failed to list local containers: %v\n", err)
				return
			}
		} else {
			fmt.Println("Remote container listing not yet implemented")
		}
	},
}

func listLocalContainers(all bool) error {
	dockerClient, err := docker.NewClient()
	if err != nil {
		return fmt.Errorf("failed to create Docker client: %w", err)
	}
	defer dockerClient.Close()

	containers, err := dockerClient.ListContainers(all)
	if err != nil {
		return fmt.Errorf("failed to list containers: %w", err)
	}

	if len(containers) == 0 {
		fmt.Println("No containers found")
		return nil
	}

	// Print header
	fmt.Printf("%-12s %-20s %-15s %-10s %-15s\n", "CONTAINER ID", "IMAGE", "COMMAND", "STATUS", "NAMES")

	// Print containers
	for _, container := range containers {
		id := container.ID[:12]
		image := container.Image
		command := container.Command
		if len(command) > 15 {
			command = command[:12] + "..."
		}
		status := container.Status
		names := strings.Join(container.Names, ",")
		if len(names) > 0 && names[0] == '/' {
			names = names[1:] // Remove leading slash
		}

		fmt.Printf("%-12s %-20s %-15s %-10s %-15s\n", id, image, command, status, names)
	}

	return nil
}

func init() {
	rootCmd.AddCommand(psCmd)
	
	psCmd.Flags().BoolP("all", "a", false, "Show all containers (default shows just running)")
	psCmd.Flags().StringSliceP("servers", "s", []string{}, "List containers on remote servers")
}