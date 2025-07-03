package dockdockgo

import (
	"dockdockgo/internal/api"
	"dockdockgo/internal/storage"
	"dockdockgo/pkg/types"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
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

		// Get existing nodes to check for duplicates
		existingNodes, err := storage.ListNodes()
		if err != nil {
			fmt.Printf("Failed to list existing nodes: %v\n", err)
			return
		}

		for _, hostname := range args {
			// Check if server already exists
			var existingNode *types.Node
			for _, node := range existingNodes {
				if node.Hostname == hostname || node.IPAddress == hostname {
					existingNode = node
					break
				}
			}

			if existingNode != nil {
				// Update existing node
				existingNode.Role = role
				existingNode.Port = port
				existingNode.Labels = nodeLabels
				existingNode.LastHeartbeat = time.Now()

				if err := storage.SaveNode(existingNode); err != nil {
					fmt.Printf("Failed to update server %s: %v\n", hostname, err)
					continue
				}

				fmt.Printf("✓ Server %s updated in cluster\n", hostname)
				fmt.Printf("  ID: %s\n", existingNode.ID)
				fmt.Printf("  Role: %s (updated)\n", existingNode.Role)
				fmt.Printf("  Port: %d (updated)\n", existingNode.Port)
				fmt.Printf("  Status: %s\n", existingNode.Status)
			} else {
				// Create new node
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
		}
	},
}

var serverListCmd = &cobra.Command{
	Use:   "list",
	Short: "List cluster servers",
	Long:  `List all servers in the DockDockGo cluster.`,
	Run: func(cmd *cobra.Command, args []string) {
		nodes, err := getNodesList()
		if err != nil {
			fmt.Printf("Error: %v\n", err)
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

// Cluster commands
var clusterCmd = &cobra.Command{
	Use:   "cluster",
	Short: "Manage cluster lifecycle",
	Long:  `Initialize, join, and manage DockDockGo cluster operations.`,
}

var clusterInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a new cluster",
	Long:  `Initialize a new DockDockGo cluster on this node. This node will become the cluster master.`,
	Run: func(cmd *cobra.Command, args []string) {
		advertiseAddr, _ := cmd.Flags().GetString("advertise-addr")

		// Try API-first approach
		client := api.NewClient("localhost", "8080")
		if client.IsServiceRunning() {
			fmt.Printf("Using DockDockGo API service...\n")

			response, err := client.ClusterInit(advertiseAddr)
			if err != nil {
				fmt.Printf("Failed to initialize cluster via API: %v\n", err)
				fmt.Printf("Falling back to direct database access...\n")
			} else {
				fmt.Printf("✓ Cluster initialized successfully\n")
				fmt.Printf("  Master Node ID: %s\n", response.NodeID)
				fmt.Printf("  Hostname: %s\n", response.Hostname)
				fmt.Printf("  IP Address: %s\n", response.IPAddress)
				fmt.Printf("  Port: %d\n", response.Port)
				fmt.Printf("\nTo add workers to this cluster, run on other nodes:\n")
				fmt.Printf("  %s\n", response.JoinCommand)
				return
			}
		}

		// Fallback to direct database access
		fmt.Printf("API service not available, using direct database access...\n")
		storage, err := storage.NewDefaultStorage()
		if err != nil {
			fmt.Printf("Failed to initialize storage: %v\n", err)
			fmt.Printf("\nTroubleshooting:\n")
			fmt.Printf("- Ensure no other DockDockGo processes are running\n")
			fmt.Printf("- Check database permissions in /var/lib/dockdockgo/\n")
			fmt.Printf("- Try stopping the DockDockGo service: sudo systemctl stop dockdockgo\n")
			return
		}
		defer storage.Close()

		// Get hostname and IP
		hostname, err := os.Hostname()
		if err != nil {
			fmt.Printf("Failed to get hostname: %v\n", err)
			return
		}

		// Use advertise-addr or hostname as IP
		ipAddress := advertiseAddr
		if ipAddress == "" {
			ipAddress = hostname
		}

		// Create master node
		masterNode := &types.Node{
			ID:            uuid.New().String(),
			Hostname:      hostname,
			IPAddress:     ipAddress,
			Port:          8443,
			Status:        types.NodeOnline,
			Role:          "master",
			Version:       "1.0.0", // TODO: Get actual version
			Labels:        map[string]string{"cluster.role": "master"},
			LastHeartbeat: time.Now(),
			JoinedAt:      time.Now(),
		}

		// Check if cluster already exists
		nodes, err := storage.ListNodes()
		if err != nil {
			fmt.Printf("Failed to check existing cluster: %v\n", err)
			return
		}

		if len(nodes) > 0 {
			fmt.Printf("Cluster already initialized. Found %d existing nodes.\n", len(nodes))
			fmt.Printf("Use 'dockdockgo server list' to see cluster status.\n")
			return
		}

		// Save master node
		if err := storage.SaveNode(masterNode); err != nil {
			fmt.Printf("Failed to initialize cluster: %v\n", err)
			return
		}

		fmt.Printf("✓ Cluster initialized successfully\n")
		fmt.Printf("  Master Node ID: %s\n", masterNode.ID)
		fmt.Printf("  Hostname: %s\n", masterNode.Hostname)
		fmt.Printf("  IP Address: %s\n", masterNode.IPAddress)
		fmt.Printf("  Port: %d\n", masterNode.Port)
		fmt.Printf("\nTo add workers to this cluster, run on other nodes:\n")
		fmt.Printf("  dockdockgo cluster join %s\n", ipAddress)
	},
}

var clusterJoinCmd = &cobra.Command{
	Use:   "join [MASTER_ADDRESS]",
	Short: "Join an existing cluster",
	Long:  `Join this node to an existing DockDockGo cluster by connecting to the master node.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		masterAddr := args[0]
		role, _ := cmd.Flags().GetString("role")

		fmt.Printf("Attempting to join cluster at %s...\n", masterAddr)

		// Try API-first approach
		client := api.NewClient("localhost", "8080")
		if client.IsServiceRunning() {
			fmt.Printf("Using DockDockGo API service...\n")

			response, err := client.ClusterJoin(masterAddr, role)
			if err != nil {
				fmt.Printf("Failed to join cluster via API: %v\n", err)
				fmt.Printf("Falling back to direct database access...\n")
			} else {
				fmt.Printf("✓ Successfully joined cluster\n")
				fmt.Printf("  This Node ID: %s\n", response.NodeID)
				fmt.Printf("  Role: %s\n", response.Role)
				fmt.Printf("  Master: %s\n", response.Master)
				fmt.Printf("  Cluster Nodes: %d\n", response.ClusterNodes)
				fmt.Printf("\nRun 'dockdockgo server list' to see all cluster nodes.\n")
				return
			}
		}

		// Fallback to direct database access
		fmt.Printf("API service not available, using direct database access...\n")
		storage, err := storage.NewDefaultStorage()
		if err != nil {
			fmt.Printf("Failed to initialize storage: %v\n", err)
			fmt.Printf("\nTroubleshooting:\n")
			fmt.Printf("- Ensure no other DockDockGo processes are running\n")
			fmt.Printf("- Check database permissions in /var/lib/dockdockgo/\n")
			fmt.Printf("- Try stopping the DockDockGo service: sudo systemctl stop dockdockgo\n")
			return
		}
		defer storage.Close()

		// Get hostname
		hostname, err := os.Hostname()
		if err != nil {
			fmt.Printf("Failed to get hostname: %v\n", err)
			return
		}

		// Check if this node already exists
		existingNodes, err := storage.ListNodes()
		if err != nil {
			fmt.Printf("Failed to check existing nodes: %v\n", err)
			return
		}

		var existingNode *types.Node
		for _, node := range existingNodes {
			if node.Hostname == hostname {
				existingNode = node
				break
			}
		}

		var thisNode *types.Node
		if existingNode != nil {
			// Update existing node
			existingNode.Role = role
			existingNode.Status = types.NodeOnline
			existingNode.LastHeartbeat = time.Now()
			if err := storage.SaveNode(existingNode); err != nil {
				fmt.Printf("Failed to update this node: %v\n", err)
				return
			}
			thisNode = existingNode
		} else {
			// Try to fetch cluster state from master
			clusterState, err := fetchClusterState(masterAddr)
			if err != nil {
				fmt.Printf("Failed to connect to master node: %v\n", err)
				fmt.Printf("Make sure the master node is running and accessible at %s:8080\n", masterAddr)
				return
			}

			// Save all nodes from cluster state
			for _, node := range clusterState {
				if err := storage.SaveNode(node); err != nil {
					fmt.Printf("Warning: Failed to save node %s: %v\n", node.Hostname, err)
				}
			}

			// Create this node and add it to the cluster
			thisNode = &types.Node{
				ID:            uuid.New().String(),
				Hostname:      hostname,
				IPAddress:     hostname, // TODO: Get actual IP
				Port:          8443,
				Status:        types.NodeOnline,
				Role:          role,
				Version:       "1.0.0", // TODO: Get actual version
				Labels:        map[string]string{"cluster.role": role},
				LastHeartbeat: time.Now(),
				JoinedAt:      time.Now(),
			}

			if err := storage.SaveNode(thisNode); err != nil {
				fmt.Printf("Failed to save this node: %v\n", err)
				return
			}
		}

		// Get final cluster count
		finalNodes, _ := storage.ListNodes()

		fmt.Printf("✓ Successfully joined cluster\n")
		fmt.Printf("  This Node ID: %s\n", thisNode.ID)
		fmt.Printf("  Role: %s\n", thisNode.Role)
		fmt.Printf("  Master: %s\n", masterAddr)
		fmt.Printf("  Cluster Nodes: %d\n", len(finalNodes))
		fmt.Printf("\nRun 'dockdockgo server list' to see all cluster nodes.\n")
	},
}

func getNodesList() ([]*types.Node, error) {
	// Try API-first approach
	client := api.NewClient("localhost", "8080")
	if client.IsServiceRunning() {
		nodes, err := client.ListNodes()
		if err == nil {
			return nodes, nil
		}
		fmt.Printf("Failed to list servers via API: %v\n", err)
		fmt.Printf("Falling back to direct database access...\n")
	}

	// Fallback to direct database access
	storage, err := storage.NewDefaultStorage()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize storage: %v\n\nTroubleshooting:\n- Ensure no other DockDockGo processes are running\n- Check database permissions in /var/lib/dockdockgo/\n- Try stopping the DockDockGo service: sudo systemctl stop dockdockgo", err)
	}
	defer storage.Close()

	nodes, err := storage.ListNodes()
	if err != nil {
		return nil, fmt.Errorf("failed to list servers: %v", err)
	}

	return nodes, nil
}

func fetchClusterState(masterAddr string) ([]*types.Node, error) {
	// Try to fetch cluster state from master node's API
	url := fmt.Sprintf("http://%s:8080/api/v1/nodes", masterAddr)

	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to master API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("master API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var nodes []*types.Node
	if err := json.Unmarshal(body, &nodes); err != nil {
		return nil, fmt.Errorf("failed to parse cluster state: %w", err)
	}

	return nodes, nil
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

	// Add cluster commands
	rootCmd.AddCommand(clusterCmd)
	clusterCmd.AddCommand(clusterInitCmd)
	clusterCmd.AddCommand(clusterJoinCmd)

	// Server add flags
	serverAddCmd.Flags().IntP("port", "p", 8443, "gRPC port for cluster communication")
	serverAddCmd.Flags().StringP("role", "r", "worker", "Server role (master, worker)")
	serverAddCmd.Flags().StringSliceP("label", "l", []string{}, "Server labels (key=value)")

	// Server remove flags
	serverRemoveCmd.Flags().BoolP("force", "f", false, "Force removal even with running containers")

	// Cluster init flags
	clusterInitCmd.Flags().StringP("advertise-addr", "a", "", "IP address to advertise to other nodes (defaults to hostname)")

	// Cluster join flags
	clusterJoinCmd.Flags().StringP("role", "r", "worker", "Role for this node in the cluster (worker, master)")
}
