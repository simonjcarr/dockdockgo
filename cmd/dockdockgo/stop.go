package dockdockgo

import (
	"dockdockgo/internal/docker"
	"fmt"

	"github.com/spf13/cobra"
)

var stopCmd = &cobra.Command{
	Use:   "stop [CONTAINER...]",
	Short: "Stop one or more running containers",
	Long:  `Stop one or more running containers locally or on remote servers.`,
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		servers, _ := cmd.Flags().GetStringSlice("servers")

		if len(servers) == 0 {
			// Stop local containers
			for _, containerID := range args {
				if err := stopLocalContainer(containerID); err != nil {
					fmt.Printf("Failed to stop container %s: %v\n", containerID, err)
				} else {
					fmt.Printf("✓ Container %s stopped\n", containerID)
				}
			}
		} else {
			fmt.Println("Remote container stop not yet implemented")
		}
	},
}

var rmCmd = &cobra.Command{
	Use:   "rm [CONTAINER...]",
	Short: "Remove one or more containers",
	Long:  `Remove one or more containers locally or on remote servers.`,
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		force, _ := cmd.Flags().GetBool("force")
		servers, _ := cmd.Flags().GetStringSlice("servers")

		if len(servers) == 0 {
			// Remove local containers
			for _, containerID := range args {
				if err := removeLocalContainer(containerID, force); err != nil {
					fmt.Printf("Failed to remove container %s: %v\n", containerID, err)
				} else {
					fmt.Printf("✓ Container %s removed\n", containerID)
				}
			}
		} else {
			fmt.Println("Remote container removal not yet implemented")
		}
	},
}

var restartCmd = &cobra.Command{
	Use:   "restart [CONTAINER...]",
	Short: "Restart one or more containers",
	Long:  `Restart one or more containers locally or on remote servers.`,
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		servers, _ := cmd.Flags().GetStringSlice("servers")

		if len(servers) == 0 {
			// Restart local containers
			for _, containerID := range args {
				if err := restartLocalContainer(containerID); err != nil {
					fmt.Printf("Failed to restart container %s: %v\n", containerID, err)
				} else {
					fmt.Printf("✓ Container %s restarted\n", containerID)
				}
			}
		} else {
			fmt.Println("Remote container restart not yet implemented")
		}
	},
}

func stopLocalContainer(containerID string) error {
	dockerClient, err := docker.NewClient()
	if err != nil {
		return fmt.Errorf("failed to create Docker client: %w", err)
	}
	defer dockerClient.Close()

	return dockerClient.StopContainer(containerID)
}

func removeLocalContainer(containerID string, force bool) error {
	dockerClient, err := docker.NewClient()
	if err != nil {
		return fmt.Errorf("failed to create Docker client: %w", err)
	}
	defer dockerClient.Close()

	return dockerClient.RemoveContainer(containerID, force)
}

func restartLocalContainer(containerID string) error {
	dockerClient, err := docker.NewClient()
	if err != nil {
		return fmt.Errorf("failed to create Docker client: %w", err)
	}
	defer dockerClient.Close()

	return dockerClient.RestartContainer(containerID)
}

func init() {
	rootCmd.AddCommand(stopCmd)
	rootCmd.AddCommand(rmCmd)
	rootCmd.AddCommand(restartCmd)
	
	stopCmd.Flags().StringSliceP("servers", "s", []string{}, "Stop containers on remote servers")
	rmCmd.Flags().StringSliceP("servers", "s", []string{}, "Remove containers on remote servers")
	rmCmd.Flags().BoolP("force", "f", false, "Force removal of running containers")
	restartCmd.Flags().StringSliceP("servers", "s", []string{}, "Restart containers on remote servers")
}