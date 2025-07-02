package dockdockgo

import (
	"dockdockgo/internal/docker"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

var imagesCmd = &cobra.Command{
	Use:   "images",
	Short: "List images",
	Long:  `List Docker images available locally or on remote servers.`,
	Run: func(cmd *cobra.Command, args []string) {
		servers, _ := cmd.Flags().GetStringSlice("servers")

		if len(servers) == 0 {
			// List local images
			if err := listLocalImages(); err != nil {
				fmt.Printf("Failed to list local images: %v\n", err)
				return
			}
		} else {
			fmt.Println("Remote image listing not yet implemented")
		}
	},
}

var pullCmd = &cobra.Command{
	Use:   "pull [IMAGE]",
	Short: "Pull an image from a registry",
	Long:  `Pull an image from a Docker registry to local storage or remote servers.`,
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		image := args[0]
		servers, _ := cmd.Flags().GetStringSlice("servers")

		if len(servers) == 0 {
			// Pull image locally
			if err := pullLocalImage(image); err != nil {
				fmt.Printf("Failed to pull image %s: %v\n", image, err)
				return
			}
			fmt.Printf("✓ Image %s pulled successfully\n", image)
		} else {
			fmt.Println("Remote image pull not yet implemented")
		}
	},
}

var rmiCmd = &cobra.Command{
	Use:   "rmi [IMAGE...]",
	Short: "Remove one or more images",
	Long:  `Remove one or more images from local storage or remote servers.`,
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		force, _ := cmd.Flags().GetBool("force")
		servers, _ := cmd.Flags().GetStringSlice("servers")

		if len(servers) == 0 {
			// Remove local images
			for _, imageID := range args {
				if err := removeLocalImage(imageID, force); err != nil {
					fmt.Printf("Failed to remove image %s: %v\n", imageID, err)
				} else {
					fmt.Printf("✓ Image %s removed\n", imageID)
				}
			}
		} else {
			fmt.Println("Remote image removal not yet implemented")
		}
	},
}

func listLocalImages() error {
	dockerClient, err := docker.NewClient()
	if err != nil {
		return fmt.Errorf("failed to create Docker client: %w", err)
	}
	defer dockerClient.Close()

	images, err := dockerClient.ListImages()
	if err != nil {
		return fmt.Errorf("failed to list images: %w", err)
	}

	if len(images) == 0 {
		fmt.Println("No images found")
		return nil
	}

	// Print header
	fmt.Printf("%-20s %-15s %-12s %-15s %-10s\n", "REPOSITORY", "TAG", "IMAGE ID", "CREATED", "SIZE")

	// Print images
	for _, image := range images {
		id := image.ID[7:19] // Remove "sha256:" prefix and truncate
		created := time.Unix(image.Created, 0).Format("2006-01-02")
		size := fmt.Sprintf("%.1fMB", float64(image.Size)/(1024*1024))

		for _, repoTag := range image.RepoTags {
			if repoTag == "<none>:<none>" {
				continue
			}
			parts := strings.Split(repoTag, ":")
			repo := parts[0]
			tag := "latest"
			if len(parts) > 1 {
				tag = parts[1]
			}

			fmt.Printf("%-20s %-15s %-12s %-15s %-10s\n", repo, tag, id, created, size)
		}

		// Handle images with no tags
		if len(image.RepoTags) == 0 || (len(image.RepoTags) == 1 && image.RepoTags[0] == "<none>:<none>") {
			fmt.Printf("%-20s %-15s %-12s %-15s %-10s\n", "<none>", "<none>", id, created, size)
		}
	}

	return nil
}

func pullLocalImage(image string) error {
	dockerClient, err := docker.NewClient()
	if err != nil {
		return fmt.Errorf("failed to create Docker client: %w", err)
	}
	defer dockerClient.Close()

	fmt.Printf("Pulling image: %s\n", image)
	return dockerClient.PullImage(image)
}

func removeLocalImage(imageID string, force bool) error {
	dockerClient, err := docker.NewClient()
	if err != nil {
		return fmt.Errorf("failed to create Docker client: %w", err)
	}
	defer dockerClient.Close()

	return dockerClient.RemoveImage(imageID, force)
}

func init() {
	rootCmd.AddCommand(imagesCmd)
	rootCmd.AddCommand(pullCmd)
	rootCmd.AddCommand(rmiCmd)
	
	imagesCmd.Flags().StringSliceP("servers", "s", []string{}, "List images on remote servers")
	pullCmd.Flags().StringSliceP("servers", "s", []string{}, "Pull image to remote servers")
	rmiCmd.Flags().StringSliceP("servers", "s", []string{}, "Remove images from remote servers")
	rmiCmd.Flags().BoolP("force", "f", false, "Force removal of images")
}