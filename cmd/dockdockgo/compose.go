package dockdockgo

import (
	"dockdockgo/internal/compose"
	"fmt"

	"github.com/spf13/cobra"
)

var composeCmd = &cobra.Command{
	Use:   "compose",
	Short: "Manage multi-container applications",
	Long:  `Deploy and manage multi-container applications using docker-compose files with extended server targeting.`,
}

var composeUpCmd = &cobra.Command{
	Use:   "up [COMPOSE_FILE]",
	Short: "Deploy services defined in a compose file",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		composeFile := "docker-compose.yml"
		if len(args) > 0 {
			composeFile = args[0]
		}
		
		servers, _ := cmd.Flags().GetStringSlice("servers")
		detach, _ := cmd.Flags().GetBool("detach")
		projectName, _ := cmd.Flags().GetString("project-name")
		sshKey, _ := cmd.Flags().GetString("ssh-key")
		sshUser, _ := cmd.Flags().GetString("ssh-user")
		sshPassword, _ := cmd.Flags().GetString("ssh-password")
		installDocker, _ := cmd.Flags().GetBool("install-docker")
		
		// Parse compose file
		composeData, err := compose.ParseComposeFile(composeFile)
		if err != nil {
			fmt.Printf("Failed to parse compose file: %v\n", err)
			return
		}
		
		// Set project name if not provided
		if projectName == "" {
			projectName = compose.GetProjectName(composeFile)
		}
		
		// Create deployer and deploy
		deployer := compose.NewDeployer(projectName)
		options := compose.BuildDeployOptions(servers, sshKey, sshUser, sshPassword, installDocker, projectName)
		
		if err := deployer.Deploy(composeData, options); err != nil {
			fmt.Printf("Deployment failed: %v\n", err)
			return
		}
		
		if !detach {
			fmt.Println("Services are running. Press Ctrl+C to stop.")
			// TODO: Implement log following for non-detached mode
		}
	},
}

var composeDownCmd = &cobra.Command{
	Use:   "down [COMPOSE_FILE]",
	Short: "Stop and remove services",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		composeFile := "docker-compose.yml"
		if len(args) > 0 {
			composeFile = args[0]
		}
		
		projectName, _ := cmd.Flags().GetString("project-name")
		
		// Parse compose file
		composeData, err := compose.ParseComposeFile(composeFile)
		if err != nil {
			fmt.Printf("Failed to parse compose file: %v\n", err)
			return
		}
		
		// Set project name if not provided
		if projectName == "" {
			projectName = compose.GetProjectName(composeFile)
		}
		
		// Create deployer and stop services
		deployer := compose.NewDeployer(projectName)
		if err := deployer.Remove(composeData); err != nil {
			fmt.Printf("Failed to stop services: %v\n", err)
			return
		}
	},
}

func init() {
	rootCmd.AddCommand(composeCmd)
	composeCmd.AddCommand(composeUpCmd)
	composeCmd.AddCommand(composeDownCmd)
	
	composeUpCmd.Flags().StringSliceP("servers", "s", []string{}, "Override server targets")
	composeUpCmd.Flags().BoolP("detach", "d", true, "Run in background")
	composeUpCmd.Flags().StringP("project-name", "p", "", "Project name (defaults to directory name)")
	composeUpCmd.Flags().StringP("ssh-key", "", "", "SSH private key file")
	composeUpCmd.Flags().StringP("ssh-user", "", "", "SSH username for remote servers")
	composeUpCmd.Flags().StringP("ssh-password", "", "", "SSH password for remote servers")
	composeUpCmd.Flags().BoolP("install-docker", "", false, "Automatically install Docker on remote servers")
	
	composeDownCmd.Flags().StringP("project-name", "p", "", "Project name (defaults to directory name)")
}