package dockdockgo

import (
	"dockdockgo/internal/orchestrator"
	"fmt"

	"github.com/spf13/cobra"
)

var runCmd = &cobra.Command{
	Use:   "run [IMAGE]",
	Short: "Run a container from an image",
	Long: `Run a container from a local or remote image. Supports all standard 
docker run options and can deploy to multiple remote servers.`,
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		image := args[0]
		servers, _ := cmd.Flags().GetStringSlice("servers")
		replicas, _ := cmd.Flags().GetInt("replicas")
		ports, _ := cmd.Flags().GetStringSlice("port")
		env, _ := cmd.Flags().GetStringSlice("env")
		name, _ := cmd.Flags().GetString("name")
		detach, _ := cmd.Flags().GetBool("detach")
		installDocker, _ := cmd.Flags().GetBool("install-docker")
		sshKey, _ := cmd.Flags().GetString("ssh-key")
		sshUser, _ := cmd.Flags().GetString("ssh-user")
		sshPassword, _ := cmd.Flags().GetString("ssh-password")

		// Set default name if not provided
		if name == "" {
			name = "dockdockgo-container"
		}

		config := &orchestrator.DeploymentConfig{
			Image:         image,
			Name:          name,
			Servers:       servers,
			Replicas:      replicas,
			Ports:         ports,
			Environment:   env,
			SSHKey:        sshKey,
			SSHUser:       sshUser,
			SSHPassword:   sshPassword,
			InstallDocker: installDocker,
			Detach:        detach,
		}

		orch := orchestrator.NewOrchestrator()
		if err := orch.DeployContainer(config); err != nil {
			fmt.Printf("Deployment failed: %v\n", err)
			return
		}

		fmt.Println("✓ Deployment completed successfully")
	},
}

func init() {
	rootCmd.AddCommand(runCmd)

	runCmd.Flags().StringSliceP("servers", "s", []string{}, "List of remote servers to deploy to")
	runCmd.Flags().IntP("replicas", "r", 1, "Number of replicas to run")
	runCmd.Flags().StringSliceP("port", "p", []string{}, "Port mappings (host:container)")
	runCmd.Flags().StringSliceP("env", "e", []string{}, "Environment variables")
	runCmd.Flags().StringP("name", "n", "", "Container name")
	runCmd.Flags().BoolP("detach", "d", false, "Run container in background")
	runCmd.Flags().BoolP("install-docker", "", false, "Automatically install Docker on remote servers without confirmation")
	runCmd.Flags().StringP("ssh-key", "", "", "Path to SSH private key file")
	runCmd.Flags().StringP("ssh-user", "", "", "SSH username for remote servers")
	runCmd.Flags().StringP("ssh-password", "", "", "SSH password for remote servers")
}
