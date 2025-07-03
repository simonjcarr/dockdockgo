package dockdockgo

import (
	"dockdockgo/internal/config"
	"fmt"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage DockDockGo configuration",
	Long:  `Manage DockDockGo configuration including servers, defaults, and settings.`,
}

var configInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize configuration file",
	Run: func(cmd *cobra.Command, args []string) {
		configPath, _ := cmd.Flags().GetString("config")

		cfg := config.Get()
		if err := config.Save(cfg, configPath); err != nil {
			fmt.Printf("Failed to initialize config: %v\n", err)
			return
		}

		if configPath == "" {
			configPath = "~/.dockdockgo/config.yaml"
		}
		fmt.Printf("✓ Configuration initialized at: %s\n", configPath)
	},
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show current configuration",
	Run: func(cmd *cobra.Command, args []string) {
		cfg := config.Get()

		data, err := yaml.Marshal(cfg)
		if err != nil {
			fmt.Printf("Failed to marshal config: %v\n", err)
			return
		}

		fmt.Print(string(data))
	},
}

var configServerCmd = &cobra.Command{
	Use:   "server",
	Short: "Manage server configurations",
}

var configServerAddCmd = &cobra.Command{
	Use:   "add [NAME]",
	Short: "Add a new server configuration",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]
		host, _ := cmd.Flags().GetString("host")
		port, _ := cmd.Flags().GetString("port")
		user, _ := cmd.Flags().GetString("user")
		keyPath, _ := cmd.Flags().GetString("key")
		password, _ := cmd.Flags().GetString("password")
		dockerHost, _ := cmd.Flags().GetString("docker-host")
		maxReplicas, _ := cmd.Flags().GetInt("max-replicas")

		if host == "" {
			fmt.Println("Error: --host is required")
			return
		}

		cfg := config.Get()

		// Check if server already exists
		if _, exists := cfg.GetServer(name); exists {
			fmt.Printf("Server '%s' already exists. Use 'config server update' to modify.\n", name)
			return
		}

		server := &config.ServerConfig{
			Host:        host,
			Port:        port,
			User:        user,
			KeyPath:     keyPath,
			Password:    password,
			DockerHost:  dockerHost,
			MaxReplicas: maxReplicas,
			Labels:      make(map[string]string),
		}

		cfg.AddServer(name, server)

		configPath, _ := cmd.Flags().GetString("config")
		if err := config.Save(cfg, configPath); err != nil {
			fmt.Printf("Failed to save config: %v\n", err)
			return
		}

		fmt.Printf("✓ Server '%s' added successfully\n", name)
	},
}

var configServerListCmd = &cobra.Command{
	Use:   "list",
	Short: "List configured servers",
	Run: func(cmd *cobra.Command, args []string) {
		cfg := config.Get()
		servers := cfg.ListServers()

		if len(servers) == 0 {
			fmt.Println("No servers configured")
			return
		}

		fmt.Printf("%-15s %-20s %-10s %-15s\n", "NAME", "HOST", "PORT", "USER")
		fmt.Println("------------------------------------------------------------")

		for _, name := range servers {
			server, _ := cfg.GetServer(name)
			port := server.Port
			if port == "" {
				port = "22"
			}
			fmt.Printf("%-15s %-20s %-10s %-15s\n", name, server.Host, port, server.User)
		}
	},
}

var configServerRemoveCmd = &cobra.Command{
	Use:   "remove [NAME]",
	Short: "Remove a server configuration",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]
		cfg := config.Get()

		if _, exists := cfg.GetServer(name); !exists {
			fmt.Printf("Server '%s' does not exist\n", name)
			return
		}

		cfg.RemoveServer(name)

		configPath, _ := cmd.Flags().GetString("config")
		if err := config.Save(cfg, configPath); err != nil {
			fmt.Printf("Failed to save config: %v\n", err)
			return
		}

		fmt.Printf("✓ Server '%s' removed successfully\n", name)
	},
}

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(configInitCmd)
	configCmd.AddCommand(configShowCmd)
	configCmd.AddCommand(configServerCmd)

	configServerCmd.AddCommand(configServerAddCmd)
	configServerCmd.AddCommand(configServerListCmd)
	configServerCmd.AddCommand(configServerRemoveCmd)

	// Global config flag
	configCmd.PersistentFlags().StringP("config", "c", "", "Configuration file path")

	// Server add flags
	configServerAddCmd.Flags().StringP("host", "H", "", "Server hostname or IP address (required)")
	configServerAddCmd.Flags().StringP("port", "p", "22", "SSH port")
	configServerAddCmd.Flags().StringP("user", "u", "", "SSH username")
	configServerAddCmd.Flags().StringP("key", "k", "", "SSH private key path")
	configServerAddCmd.Flags().StringP("password", "", "", "SSH password")
	configServerAddCmd.Flags().StringP("docker-host", "", "", "Docker daemon host")
	configServerAddCmd.Flags().IntP("max-replicas", "", 10, "Maximum replicas for this server")

	configServerAddCmd.MarkFlagRequired("host")
}
