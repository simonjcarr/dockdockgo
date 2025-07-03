package dockdockgo

import (
	"dockdockgo/internal/storage"
	"dockdockgo/pkg/types"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Manage cluster servers",
	Long:  `Add, remove, and manage servers in the DockDockGo cluster.`,
}

var serverAddCmd = &cobra.Command{
	Use:   "add [HOSTNAME...]",
	Short: "Add servers to the cluster",
	Long:  `Add one or more servers to the DockDockGo cluster.`,
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		port, _ := cmd.Flags().GetInt("port")
		role, _ := cmd.Flags().GetString("role")
		labels, _ := cmd.Flags().GetStringSlice("label")

		storage, err := storage.NewDefaultStorage()
		if err != nil {
			fmt.Printf("Failed to initialize storage: %v\n", err)
			return
		}
		defer storage.Close()

		// Parse labels
		nodeLabels := parseLabels(labels)

		for _, hostname := range args {
			node := &types.Node{
				ID:            uuid.New().String(),
				Hostname:      hostname,
				IPAddress:     hostname, // TODO: Resolve IP address
				Port:          port,
				Status:        types.NodeOffline,
				Role:          role,
				Version:       "1.0.0", // TODO: Get actual version
				Labels:        nodeLabels,
				LastHeartbeat: time.Now(),
				JoinedAt:      time.Now(),
			}

			if err := storage.SaveNode(node); err != nil {
				fmt.Printf("Failed to add server %s: %v\n", hostname, err)
				continue
			}

			fmt.Printf("✓ Server %s added to cluster\n", hostname)
			fmt.Printf("  ID: %s\n", node.ID)
			fmt.Printf("  Role: %s\n", node.Role)
			fmt.Printf("  Port: %d\n", node.Port)

			// TODO: Connect to server and install DockDockGo agent
			fmt.Printf("  Status: %s (pending agent installation)\n", node.Status)
		}
	},
}

var serverListCmd = &cobra.Command{
	Use:   "list",
	Short: "List cluster servers",
	Long:  `List all servers in the DockDockGo cluster.`,
	Run: func(cmd *cobra.Command, args []string) {
		storage, err := storage.NewDefaultStorage()
		if err != nil {
			fmt.Printf("Failed to initialize storage: %v\n", err)
			return
		}
		defer storage.Close()

		nodes, err := storage.ListNodes()
		if err != nil {
			fmt.Printf("Failed to list servers: %v\n", err)
			return
		}

		if len(nodes) == 0 {
			fmt.Println("No servers found")
			return
		}

		fmt.Printf("%-20s %-15s %-10s %-10s %-15s %-10s\n", "HOSTNAME", "IP", "PORT", "ROLE", "STATUS", "AGE")
		fmt.Println("--------------------------------------------------------------------------------")

		for _, node := range nodes {
			age := formatAge(node.JoinedAt)
			fmt.Printf("%-20s %-15s %-10d %-10s %-15s %-10s\n",
				node.Hostname,
				node.IPAddress,
				node.Port,
				node.Role,
				node.Status,
				age)
		}
	},
}

var serverRemoveCmd = &cobra.Command{
	Use:   "remove [HOSTNAME...]",
	Short: "Remove servers from the cluster",
	Long:  `Remove one or more servers from the DockDockGo cluster.`,
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		force, _ := cmd.Flags().GetBool("force")

		storage, err := storage.NewDefaultStorage()
		if err != nil {
			fmt.Printf("Failed to initialize storage: %v\n", err)
			return
		}
		defer storage.Close()

		for _, hostname := range args {
			// Find node by hostname
			nodes, err := storage.ListNodes()
			if err != nil {
				fmt.Printf("Failed to list servers: %v\n", err)
				continue
			}

			var nodeToRemove *types.Node
			for _, node := range nodes {
				if node.Hostname == hostname {
					nodeToRemove = node
					break
				}
			}

			if nodeToRemove == nil {
				fmt.Printf("Server %s not found\n", hostname)
				continue
			}

			// Check if server has running containers
			containers, err := storage.ListContainersByNode(nodeToRemove.ID)
			if err != nil {
				fmt.Printf("Failed to check containers on server %s: %v\n", hostname, err)
				continue
			}

			runningContainers := 0
			for _, container := range containers {
				if container.Status == types.ContainerRunning {
					runningContainers++
				}
			}

			if runningContainers > 0 && !force {
				fmt.Printf("Server %s has %d running containers. Use --force to remove anyway.\n", hostname, runningContainers)
				continue
			}

			// Remove the node
			if err := storage.DeleteNode(nodeToRemove.ID); err != nil {
				fmt.Printf("Failed to remove server %s: %v\n", hostname, err)
				continue
			}

			fmt.Printf("✓ Server %s removed from cluster\n", hostname)

			if runningContainers > 0 {
				fmt.Printf("  Warning: %d containers were running on this server\n", runningContainers)
			}
		}
	},
}

var serverStatusCmd = &cobra.Command{
	Use:   "status [HOSTNAME]",
	Short: "Show server status",
	Long:  `Show detailed status of a server including resource usage and containers.`,
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		hostname := args[0]

		storage, err := storage.NewDefaultStorage()
		if err != nil {
			fmt.Printf("Failed to initialize storage: %v\n", err)
			return
		}
		defer storage.Close()

		// Find node by hostname
		nodes, err := storage.ListNodes()
		if err != nil {
			fmt.Printf("Failed to list servers: %v\n", err)
			return
		}

		var node *types.Node
		for _, n := range nodes {
			if n.Hostname == hostname {
				node = n
				break
			}
		}

		if node == nil {
			fmt.Printf("Server %s not found\n", hostname)
			return
		}

		// Get containers on this node
		containers, err := storage.ListContainersByNode(node.ID)
		if err != nil {
			fmt.Printf("Failed to list containers: %v\n", err)
			return
		}

		// Display server info
		fmt.Printf("Server: %s\n", node.Hostname)
		fmt.Printf("ID: %s\n", node.ID)
		fmt.Printf("IP Address: %s\n", node.IPAddress)
		fmt.Printf("Port: %d\n", node.Port)
		fmt.Printf("Role: %s\n", node.Role)
		fmt.Printf("Status: %s\n", node.Status)
		fmt.Printf("Version: %s\n", node.Version)
		fmt.Printf("Joined: %s\n", node.JoinedAt.Format("2006-01-02 15:04:05"))
		fmt.Printf("Last Heartbeat: %s\n", node.LastHeartbeat.Format("2006-01-02 15:04:05"))

		if len(node.Labels) > 0 {
			fmt.Printf("Labels:\n")
			for key, value := range node.Labels {
				fmt.Printf("  %s=%s\n", key, value)
			}
		}

		if node.Resources != nil {
			fmt.Printf("Resources:\n")
			fmt.Printf("  CPU Cores: %d\n", node.Resources.CPUCores)
			fmt.Printf("  CPU Usage: %.1f%%\n", node.Resources.CPUUsagePercent)
			fmt.Printf("  Memory: %d/%d MB (%.1f%%)\n",
				node.Resources.MemoryUsageMB,
				node.Resources.MemoryTotalMB,
				float64(node.Resources.MemoryUsageMB)/float64(node.Resources.MemoryTotalMB)*100)
			fmt.Printf("  Disk: %d/%d GB (%.1f%%)\n",
				node.Resources.DiskUsageGB,
				node.Resources.DiskTotalGB,
				float64(node.Resources.DiskUsageGB)/float64(node.Resources.DiskTotalGB)*100)
			fmt.Printf("  Containers: %d/%d\n", node.Resources.ContainerCount, node.Resources.MaxContainers)
		}

		// Display container info
		if len(containers) > 0 {
			fmt.Printf("\nContainers (%d):\n", len(containers))
			fmt.Printf("%-20s %-20s %-15s %-15s\n", "NAME", "DEPLOYMENT", "STATUS", "STARTED")
			fmt.Println("------------------------------------------------------------------------")

			for _, container := range containers {
				startedStr := "N/A"
				if container.StartedAt != nil {
					startedStr = formatAge(*container.StartedAt)
				}

				fmt.Printf("%-20s %-20s %-15s %-15s\n",
					container.Name,
					container.DeploymentID,
					container.Status,
					startedStr)
			}
		} else {
			fmt.Printf("\nNo containers running on this server\n")
		}
	},
}

func parseLabels(labels []string) map[string]string {
	labelMap := make(map[string]string)

	for _, label := range labels {
		parts := strings.SplitN(label, "=", 2)
		if len(parts) == 2 {
			labelMap[parts[0]] = parts[1]
		} else {
			labelMap[parts[0]] = ""
		}
	}

	return labelMap
}

func formatAge(t time.Time) string {
	duration := time.Since(t)

	if duration < time.Minute {
		return fmt.Sprintf("%ds", int(duration.Seconds()))
	} else if duration < time.Hour {
		return fmt.Sprintf("%dm", int(duration.Minutes()))
	} else if duration < 24*time.Hour {
		return fmt.Sprintf("%dh", int(duration.Hours()))
	} else {
		return fmt.Sprintf("%dd", int(duration.Hours()/24))
	}
}

func init() {
	rootCmd.AddCommand(serverCmd)
	serverCmd.AddCommand(serverAddCmd)
	serverCmd.AddCommand(serverListCmd)
	serverCmd.AddCommand(serverRemoveCmd)
	serverCmd.AddCommand(serverStatusCmd)

	// Server add flags
	serverAddCmd.Flags().IntP("port", "p", 8443, "gRPC port for cluster communication")
	serverAddCmd.Flags().StringP("role", "r", "worker", "Server role (master, worker)")
	serverAddCmd.Flags().StringSliceP("label", "l", []string{}, "Server labels (key=value)")

	// Server remove flags
	serverRemoveCmd.Flags().BoolP("force", "f", false, "Force removal even with running containers")
}
