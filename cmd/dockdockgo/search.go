package dockdockgo

import (
	"dockdockgo/internal/docker"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

var searchCmd = &cobra.Command{
	Use:   "search [TERM]",
	Short: "Search for images in registries",
	Long: `Search for container images in local storage and remote registries. 
Defaults to DockerHub if no registry is specified.`,
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		searchTerm := args[0]
		registry, _ := cmd.Flags().GetString("registry")
		local, _ := cmd.Flags().GetBool("local")
		limit, _ := cmd.Flags().GetInt("limit")

		if local {
			if err := searchLocalImages(searchTerm); err != nil {
				fmt.Printf("Failed to search local images: %v\n", err)
				return
			}
		} else {
			if registry == "" {
				registry = "docker.io"
			}
			if err := searchRemoteImages(searchTerm, limit); err != nil {
				fmt.Printf("Failed to search remote images: %v\n", err)
				return
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(searchCmd)

	searchCmd.Flags().StringP("registry", "r", "", "Registry to search (defaults to DockerHub)")
	searchCmd.Flags().BoolP("local", "l", false, "Search local images only")
	searchCmd.Flags().IntP("limit", "", 25, "Limit number of results")
}

func searchLocalImages(searchTerm string) error {
	dockerClient, err := docker.NewClient()
	if err != nil {
		return fmt.Errorf("failed to create Docker client: %w", err)
	}
	defer dockerClient.Close()

	images, err := dockerClient.ListImages()
	if err != nil {
		return fmt.Errorf("failed to list images: %w", err)
	}

	fmt.Printf("Searching local images for: %s\n", searchTerm)
	fmt.Printf("%-40s %-15s %-12s\n", "REPOSITORY", "TAG", "IMAGE ID")

	found := false
	for _, image := range images {
		for _, repoTag := range image.RepoTags {
			if strings.Contains(strings.ToLower(repoTag), strings.ToLower(searchTerm)) {
				parts := strings.Split(repoTag, ":")
				repo := parts[0]
				tag := "latest"
				if len(parts) > 1 {
					tag = parts[1]
				}
				id := image.ID[7:19] // Remove "sha256:" prefix and truncate

				fmt.Printf("%-40s %-15s %-12s\n", repo, tag, id)
				found = true
			}
		}
	}

	if !found {
		fmt.Printf("No local images found matching: %s\n", searchTerm)
	}

	return nil
}

func searchRemoteImages(searchTerm string, limit int) error {
	dockerClient, err := docker.NewClient()
	if err != nil {
		return fmt.Errorf("failed to create Docker client: %w", err)
	}
	defer dockerClient.Close()

	fmt.Printf("Searching Docker Hub for: %s\n", searchTerm)
	results, err := dockerClient.SearchImages(searchTerm)
	if err != nil {
		return fmt.Errorf("failed to search images: %w", err)
	}

	if len(results) == 0 {
		fmt.Printf("No images found matching: %s\n", searchTerm)
		return nil
	}

	fmt.Printf("%-30s %-60s %-10s %-10s\n", "NAME", "DESCRIPTION", "STARS", "OFFICIAL")

	count := 0
	for _, result := range results {
		if count >= limit {
			break
		}

		description := result.Description
		if len(description) > 60 {
			description = description[:57] + "..."
		}

		official := "No"
		if result.IsOfficial {
			official = "Yes"
		}

		fmt.Printf("%-30s %-60s %-10d %-10s\n", result.Name, description, result.StarCount, official)
		count++
	}

	return nil
}
