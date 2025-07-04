package dockdockgo

import (
	"dockdockgo/internal/api"
	"dockdockgo/internal/cluster"
	"dockdockgo/internal/storage"
	"dockdockgo/pkg/types"
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

var deployCmd = &cobra.Command{
	Use:   "deploy",
	Short: "Manage deployments",
	Long:  `Create, scale, and manage container deployments across the cluster.`,
}

var deployCreateCmd = &cobra.Command{
	Use:   "create [NAME] [IMAGE]",
	Short: "Create a new deployment",
	Long:  `Create a new deployment with the specified name and image.`,
	Args:  cobra.MinimumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]
		image := args[1]

		replicas, _ := cmd.Flags().GetInt("replicas")
		ports, _ := cmd.Flags().GetStringSlice("port")
		env, _ := cmd.Flags().GetStringSlice("env")
		volumes, _ := cmd.Flags().GetStringSlice("volume")
		placement, _ := cmd.Flags().GetString("placement")
		restartPolicy, _ := cmd.Flags().GetString("restart-policy")

		// Parse ports
		portMappings, err := parsePorts(ports)
		if err != nil {
			fmt.Printf("Error parsing ports: %v\n", err)
			return
		}

		// Parse environment variables
		environment := parseEnvironment(env)

		// Parse volumes
		volumeMappings, err := parseVolumes(volumes)
		if err != nil {
			fmt.Printf("Error parsing volumes: %v\n", err)
			return
		}

		// Parse placement config
		placementConfig, err := parsePlacement(placement)
		if err != nil {
			fmt.Printf("Error parsing placement: %v\n", err)
			return
		}

		// Create deployment spec
		spec := &types.DeploymentSpec{
			Name:          name,
			Image:         image,
			Replicas:      replicas,
			Ports:         portMappings,
			Environment:   environment,
			Volumes:       volumeMappings,
			Placement:     placementConfig,
			RestartPolicy: restartPolicy,
		}

		// Try API first, fallback to direct storage access
		apiHost, apiPort := getAPIEndpoint()
		client := api.NewClient(apiHost, apiPort)
		if client.IsServiceRunning() {
			// Use API
			deployment, err := client.CreateDeployment(spec)
			if err != nil {
				fmt.Printf("Failed to create deployment via API: %v\n", err)
				return
			}
			fmt.Printf("✓ Deployment %s created successfully via API\n", deployment.Name)
			fmt.Printf("  ID: %s\n", deployment.ID)
			fmt.Printf("  Image: %s\n", deployment.Image)
			fmt.Printf("  Replicas: %d\n", deployment.Replicas)
			fmt.Printf("  Status: %s\n", deployment.Status)

			// Trigger cluster sync
			triggerClusterSync(client)
			return
		}

		// Fallback to direct storage access
		storage, err := storage.NewDefaultStorage()
		if err != nil {
			fmt.Printf("Failed to initialize storage: %v\n", err)
			return
		}
		defer storage.Close()

		deploymentManager := cluster.NewDeploymentManager(storage)

		// Create deployment
		deployment, err := deploymentManager.CreateDeployment(spec)
		if err != nil {
			fmt.Printf("Failed to create deployment: %v\n", err)
			return
		}

		fmt.Printf("✓ Deployment %s created successfully\n", deployment.Name)
		fmt.Printf("  ID: %s\n", deployment.ID)
		fmt.Printf("  Image: %s\n", deployment.Image)
		fmt.Printf("  Replicas: %d\n", deployment.Replicas)
		fmt.Printf("  Status: %s\n", deployment.Status)
	},
}

var deployListCmd = &cobra.Command{
	Use:   "list",
	Short: "List deployments",
	Long:  `List all deployments in the cluster.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Try API first, fallback to direct storage access
		apiHost, apiPort := getAPIEndpoint()
		client := api.NewClient(apiHost, apiPort)
		var deployments []*types.Deployment
		var err error

		if client.IsServiceRunning() {
			// Use API
			deployments, err = client.ListDeployments()
			if err != nil {
				fmt.Printf("Failed to list deployments via API: %v\n", err)
				return
			}
		} else {
			// Fallback to direct storage access
			storage, err := storage.NewDefaultStorage()
			if err != nil {
				fmt.Printf("Failed to initialize storage: %v\n", err)
				return
			}
			defer storage.Close()

			deploymentManager := cluster.NewDeploymentManager(storage)

			deployments, err = deploymentManager.ListDeployments()
			if err != nil {
				fmt.Printf("Failed to list deployments: %v\n", err)
				return
			}
		}

		if len(deployments) == 0 {
			fmt.Println("No deployments found")
			return
		}

		fmt.Printf("%-20s %-30s %-15s %-10s %-10s\n", "NAME", "IMAGE", "STATUS", "REPLICAS", "AGE")
		fmt.Println("---------------------------------------------------------------------------------")

		for _, deployment := range deployments {
			age := formatAge(deployment.CreatedAt)
			fmt.Printf("%-20s %-30s %-15s %-10d %-10s\n",
				deployment.Name,
				deployment.Image,
				deployment.Status,
				deployment.Replicas,
				age)
		}
	},
}

var deployScaleCmd = &cobra.Command{
	Use:   "scale [NAME] [REPLICAS]",
	Short: "Scale a deployment",
	Long:  `Scale a deployment to the specified number of replicas.`,
	Args:  cobra.MinimumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]
		replicasStr := args[1]

		replicas, err := strconv.Atoi(replicasStr)
		if err != nil {
			fmt.Printf("Invalid replicas count: %s\n", replicasStr)
			return
		}

		if replicas < 0 {
			fmt.Printf("Replicas count cannot be negative\n")
			return
		}

		storage, err := storage.NewDefaultStorage()
		if err != nil {
			fmt.Printf("Failed to initialize storage: %v\n", err)
			return
		}
		defer storage.Close()

		deploymentManager := cluster.NewDeploymentManager(storage)

		// Get deployment by name
		deployment, err := deploymentManager.GetDeploymentByName(name)
		if err != nil {
			fmt.Printf("Deployment %s not found: %v\n", name, err)
			return
		}

		// Scale deployment
		deployment, err = deploymentManager.ScaleDeployment(deployment.ID, replicas)
		if err != nil {
			fmt.Printf("Failed to scale deployment: %v\n", err)
			return
		}

		fmt.Printf("✓ Deployment %s scaled to %d replicas\n", deployment.Name, deployment.Replicas)
	},
}

var deployDeleteCmd = &cobra.Command{
	Use:   "delete [NAME]",
	Short: "Delete a deployment",
	Long:  `Delete a deployment and all its containers.`,
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]

		client, storage, err := getStorageOrAPI()
		if err != nil {
			fmt.Printf("Failed to initialize storage/API: %v\n", err)
			return
		}

		// Use API if available
		if client != nil {
			if err := client.DeleteDeployment(name); err != nil {
				fmt.Printf("Failed to delete deployment via API: %v\n", err)
				return
			}
			fmt.Printf("✓ Deployment %s deleted successfully via API\n", name)

			// Trigger cluster sync
			triggerClusterSync(client)
			return
		}

		// Fallback to direct storage access
		defer storage.Close()
		deploymentManager := cluster.NewDeploymentManager(storage)

		// Get deployment by name
		deployment, err := deploymentManager.GetDeploymentByName(name)
		if err != nil {
			fmt.Printf("Deployment %s not found: %v\n", name, err)
			return
		}

		// Delete deployment
		if err := deploymentManager.DeleteDeployment(deployment.ID); err != nil {
			fmt.Printf("Failed to delete deployment: %v\n", err)
			return
		}

		fmt.Printf("✓ Deployment %s deleted successfully\n", deployment.Name)
	},
}

var deployStatusCmd = &cobra.Command{
	Use:   "status [NAME]",
	Short: "Show deployment status",
	Long:  `Show detailed status of a deployment including container information.`,
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]

		// Try API first, fallback to direct storage access
		apiHost, apiPort := getAPIEndpoint()
		client := api.NewClient(apiHost, apiPort)

		if client.IsServiceRunning() {
			// Use API for status
			deployment, err := client.GetDeploymentByName(name)
			if err != nil {
				fmt.Printf("Deployment %s not found via API: %v\n", name, err)
				return
			}

			// Display deployment info via API
			displayDeploymentStatus(deployment)
			return
		}

		// Fallback to direct storage access
		storage, err := storage.NewDefaultStorage()
		if err != nil {
			fmt.Printf("Failed to initialize storage: %v\n", err)
			return
		}
		defer storage.Close()

		deploymentManager := cluster.NewDeploymentManager(storage)

		// Get deployment by name
		deployment, err := deploymentManager.GetDeploymentByName(name)
		if err != nil {
			fmt.Printf("Deployment %s not found: %v\n", name, err)
			return
		}

		// Get containers for this deployment
		containers, err := storage.ListContainersByDeployment(deployment.ID)
		if err != nil {
			fmt.Printf("Failed to list containers: %v\n", err)
			return
		}

		// Display deployment info
		fmt.Printf("Deployment: %s\n", deployment.Name)
		fmt.Printf("ID: %s\n", deployment.ID)
		fmt.Printf("Image: %s\n", deployment.Image)
		fmt.Printf("Status: %s\n", deployment.Status)
		fmt.Printf("Replicas: %d\n", deployment.Replicas)
		fmt.Printf("Created: %s\n", deployment.CreatedAt.Format("2006-01-02 15:04:05"))
		fmt.Printf("Updated: %s\n", deployment.UpdatedAt.Format("2006-01-02 15:04:05"))

		if len(deployment.Ports) > 0 {
			fmt.Printf("Ports:\n")
			for _, port := range deployment.Ports {
				fmt.Printf("  %d:%d/%s\n", port.HostPort, port.ContainerPort, port.Protocol)
			}
		}

		if len(deployment.Environment) > 0 {
			fmt.Printf("Environment:\n")
			for key, value := range deployment.Environment {
				fmt.Printf("  %s=%s\n", key, value)
			}
		}

		// Display container info
		if len(containers) > 0 {
			fmt.Printf("\nContainers:\n")
			fmt.Printf("%-20s %-15s %-15s %-15s\n", "NAME", "STATUS", "NODE", "STARTED")
			fmt.Println("---------------------------------------------------------------------")

			for _, container := range containers {
				startedStr := "N/A"
				if container.StartedAt != nil {
					startedStr = formatAge(*container.StartedAt)
				}

				fmt.Printf("%-20s %-15s %-15s %-15s\n",
					container.Name,
					container.Status,
					container.NodeID,
					startedStr)
			}
		}
	},
}

func parsePorts(ports []string) ([]types.PortMapping, error) {
	var portMappings []types.PortMapping

	for _, port := range ports {
		mapping, err := parsePortMapping(port)
		if err != nil {
			return nil, err
		}
		portMappings = append(portMappings, mapping)
	}

	return portMappings, nil
}

func parsePortMapping(portStr string) (types.PortMapping, error) {
	parts := strings.Split(portStr, ":")
	if len(parts) != 2 {
		return types.PortMapping{}, fmt.Errorf("invalid port format: %s (expected host:container)", portStr)
	}

	hostPort, err := strconv.Atoi(parts[0])
	if err != nil {
		return types.PortMapping{}, fmt.Errorf("invalid host port: %s", parts[0])
	}

	containerPortStr := parts[1]
	protocol := "tcp"

	if strings.Contains(containerPortStr, "/") {
		protocolParts := strings.Split(containerPortStr, "/")
		containerPortStr = protocolParts[0]
		protocol = protocolParts[1]
	}

	containerPort, err := strconv.Atoi(containerPortStr)
	if err != nil {
		return types.PortMapping{}, fmt.Errorf("invalid container port: %s", containerPortStr)
	}

	return types.PortMapping{
		HostPort:      hostPort,
		ContainerPort: containerPort,
		Protocol:      protocol,
	}, nil
}

func parseEnvironment(env []string) map[string]string {
	environment := make(map[string]string)

	for _, envVar := range env {
		parts := strings.SplitN(envVar, "=", 2)
		if len(parts) == 2 {
			environment[parts[0]] = parts[1]
		} else {
			environment[parts[0]] = ""
		}
	}

	return environment
}

func parseVolumes(volumes []string) ([]types.VolumeMapping, error) {
	var volumeMappings []types.VolumeMapping

	for _, volume := range volumes {
		parts := strings.Split(volume, ":")
		if len(parts) < 2 {
			return nil, fmt.Errorf("invalid volume format: %s (expected host:container[:ro])", volume)
		}

		readOnly := false
		if len(parts) > 2 && parts[2] == "ro" {
			readOnly = true
		}

		volumeMappings = append(volumeMappings, types.VolumeMapping{
			HostPath:      parts[0],
			ContainerPath: parts[1],
			ReadOnly:      readOnly,
		})
	}

	return volumeMappings, nil
}

func parsePlacement(placement string) (*types.PlacementConfig, error) {
	if placement == "" {
		return nil, nil
	}

	// Simple implementation - can be enhanced
	config := &types.PlacementConfig{
		Strategy: placement,
	}

	return config, nil
}

func displayDeploymentStatus(deployment *types.Deployment) {
	// Display deployment info
	fmt.Printf("Deployment: %s\n", deployment.Name)
	fmt.Printf("ID: %s\n", deployment.ID)
	fmt.Printf("Image: %s\n", deployment.Image)
	fmt.Printf("Status: %s\n", deployment.Status)
	fmt.Printf("Replicas: %d\n", deployment.Replicas)
	fmt.Printf("Created: %s\n", deployment.CreatedAt.Format("2006-01-02 15:04:05"))
	fmt.Printf("Updated: %s\n", deployment.UpdatedAt.Format("2006-01-02 15:04:05"))

	if len(deployment.Ports) > 0 {
		fmt.Printf("Ports:\n")
		for _, port := range deployment.Ports {
			fmt.Printf("  %d:%d/%s\n", port.HostPort, port.ContainerPort, port.Protocol)
		}
	}

	if len(deployment.Environment) > 0 {
		fmt.Printf("Environment:\n")
		for key, value := range deployment.Environment {
			fmt.Printf("  %s=%s\n", key, value)
		}
	}

	fmt.Printf("\nNote: Container details require direct database access.\n")
	fmt.Printf("Use API or check with 'docker ps' for running containers.\n")
}

func init() {
	rootCmd.AddCommand(deployCmd)
	deployCmd.AddCommand(deployCreateCmd)
	deployCmd.AddCommand(deployListCmd)
	deployCmd.AddCommand(deployScaleCmd)
	deployCmd.AddCommand(deployDeleteCmd)
	deployCmd.AddCommand(deployStatusCmd)

	// Deploy create flags
	deployCreateCmd.Flags().IntP("replicas", "r", 1, "Number of replicas")
	deployCreateCmd.Flags().StringSliceP("port", "p", []string{}, "Port mappings (host:container)")
	deployCreateCmd.Flags().StringSliceP("env", "e", []string{}, "Environment variables")
	deployCreateCmd.Flags().StringSliceP("volume", "v", []string{}, "Volume mounts (host:container[:ro])")
	deployCreateCmd.Flags().StringP("placement", "", "spread", "Placement strategy (spread, pack, binpack)")
	deployCreateCmd.Flags().StringP("restart-policy", "", "unless-stopped", "Restart policy")
}

// triggerClusterSync triggers cluster synchronization after deployment changes
func triggerClusterSync(client *api.Client) {
	fmt.Println("  → Triggering cluster synchronization...")

	if client == nil {
		fmt.Println("  ⚠️  Cannot sync - API client not available")
		return
	}

	if err := client.TriggerClusterSync(); err != nil {
		fmt.Printf("  ⚠️  Cluster sync failed: %v\n", err)
		return
	}

	fmt.Println("  ✅ Cluster sync triggered successfully")
}

// cleanupFailedDeployment removes a failed deployment from the database
func cleanupFailedDeployment(name string) error {
	storage, err := storage.NewDefaultStorage()
	if err != nil {
		return fmt.Errorf("failed to initialize storage: %w", err)
	}

	// Try to get the deployment
	deployment, err := storage.GetDeploymentByName(name)
	if err != nil {
		return fmt.Errorf("deployment %s not found: %w", name, err)
	}

	// Delete associated containers
	containers, err := storage.ListContainersByDeployment(deployment.ID)
	if err == nil {
		for _, container := range containers {
			if err := storage.DeleteContainer(container.ID); err != nil {
				fmt.Printf("Warning: Failed to delete container %s: %v\n", container.Name, err)
			}
		}
	}

	// Delete the deployment
	if err := storage.DeleteDeployment(deployment.ID); err != nil {
		return fmt.Errorf("failed to delete deployment: %w", err)
	}

	fmt.Printf("✓ Cleaned up failed deployment: %s\n", name)
	return nil
}

// getStorageOrAPI returns either an API client or storage instance, preferring API
func getStorageOrAPI() (client *api.Client, storageInst *storage.Storage, err error) {
	// Try API first
	client = api.NewClient("localhost", "8080")

	// Test if API is available by calling health endpoint
	_, err = client.ListDeployments()
	if err == nil {
		// API is working
		return client, nil, nil
	}

	// API not available, fall back to direct storage
	fmt.Println("  ⚠️  API server not running, using direct database access")
	storageInst, err = storage.NewDefaultStorage()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to initialize storage: %w", err)
	}

	return nil, storageInst, nil
}
