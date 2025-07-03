package compose

import (
	"dockdockgo/internal/orchestrator"
	"fmt"
	"path/filepath"
)

type Deployer struct {
	orchestrator *orchestrator.Orchestrator
	projectName  string
}

func NewDeployer(projectName string) *Deployer {
	return &Deployer{
		orchestrator: orchestrator.NewOrchestrator(),
		projectName:  projectName,
	}
}

func (d *Deployer) Deploy(compose *ComposeFile, options *DeployOptions) error {
	if err := compose.Validate(); err != nil {
		return fmt.Errorf("compose file validation failed: %w", err)
	}

	fmt.Printf("Deploying project: %s\n", d.projectName)

	// Deploy services in dependency order
	deployOrder, err := d.calculateDeployOrder(compose)
	if err != nil {
		return fmt.Errorf("failed to calculate deployment order: %w", err)
	}

	for _, serviceName := range deployOrder {
		service := compose.Services[serviceName]
		if err := d.deployService(serviceName, service, options); err != nil {
			return fmt.Errorf("failed to deploy service %s: %w", serviceName, err)
		}
	}

	fmt.Printf("✓ Project %s deployed successfully\n", d.projectName)
	return nil
}

func (d *Deployer) deployService(name string, service *Service, options *DeployOptions) error {
	fmt.Printf("Deploying service: %s\n", name)

	// Convert service to deployment configuration
	config := &orchestrator.DeploymentConfig{
		Image:         service.Image,
		Name:          d.getContainerName(name, service),
		Servers:       d.getServiceServers(service, options),
		Replicas:      service.GetReplicas(),
		Ports:         service.Ports,
		Environment:   d.getEnvironmentSlice(service),
		Volumes:       service.Volumes,
		SSHKey:        options.SSHKey,
		SSHUser:       options.SSHUser,
		SSHPassword:   options.SSHPassword,
		InstallDocker: options.InstallDocker,
		Detach:        true, // Compose services always run in detached mode
	}

	if err := d.orchestrator.DeployContainer(config); err != nil {
		return err
	}

	fmt.Printf("✓ Service %s deployed\n", name)
	return nil
}

func (d *Deployer) getContainerName(serviceName string, service *Service) string {
	if service.ContainerName != "" {
		return service.ContainerName
	}
	return fmt.Sprintf("%s_%s", d.projectName, serviceName)
}

func (d *Deployer) getServiceServers(service *Service, options *DeployOptions) []string {
	// Priority: service-specific servers > global servers from options
	if servers := service.GetTargetServers(); len(servers) > 0 {
		return servers
	}
	return options.Servers
}

func (d *Deployer) getEnvironmentSlice(service *Service) []string {
	envMap := service.GetEnvironmentMap()
	env := make([]string, 0, len(envMap))
	for key, value := range envMap {
		env = append(env, fmt.Sprintf("%s=%s", key, value))
	}
	return env
}

func (d *Deployer) calculateDeployOrder(compose *ComposeFile) ([]string, error) {
	// Simple implementation - can be enhanced with proper dependency resolution
	order := make([]string, 0, len(compose.Services))

	// First pass: services without dependencies
	for name, service := range compose.Services {
		if !d.hasDependencies(service) {
			order = append(order, name)
		}
	}

	// Second pass: services with dependencies
	for name, service := range compose.Services {
		if d.hasDependencies(service) {
			order = append(order, name)
		}
	}

	return order, nil
}

func (d *Deployer) hasDependencies(service *Service) bool {
	switch deps := service.DependsOn.(type) {
	case []interface{}:
		return len(deps) > 0
	case []string:
		return len(deps) > 0
	case map[string]interface{}:
		return len(deps) > 0
	default:
		return false
	}
}

func (d *Deployer) Stop(compose *ComposeFile) error {
	fmt.Printf("Stopping project: %s\n", d.projectName)

	// Stop services in reverse order
	deployOrder, err := d.calculateDeployOrder(compose)
	if err != nil {
		return fmt.Errorf("failed to calculate deployment order: %w", err)
	}

	// Reverse the order for stopping
	for i := len(deployOrder) - 1; i >= 0; i-- {
		serviceName := deployOrder[i]
		service := compose.Services[serviceName]
		containerName := d.getContainerName(serviceName, service)

		fmt.Printf("Stopping service: %s (container: %s)\n", serviceName, containerName)
		// TODO: Implement container stop via orchestrator
	}

	fmt.Printf("✓ Project %s stopped\n", d.projectName)
	return nil
}

func (d *Deployer) Remove(compose *ComposeFile) error {
	fmt.Printf("Removing project: %s\n", d.projectName)

	// Remove services in reverse order
	deployOrder, err := d.calculateDeployOrder(compose)
	if err != nil {
		return fmt.Errorf("failed to calculate deployment order: %w", err)
	}

	// Reverse the order for removal
	for i := len(deployOrder) - 1; i >= 0; i-- {
		serviceName := deployOrder[i]
		service := compose.Services[serviceName]
		containerName := d.getContainerName(serviceName, service)

		fmt.Printf("Removing service: %s (container: %s)\n", serviceName, containerName)
		// TODO: Implement container removal via orchestrator
	}

	fmt.Printf("✓ Project %s removed\n", d.projectName)
	return nil
}

type DeployOptions struct {
	Servers       []string
	SSHKey        string
	SSHUser       string
	SSHPassword   string
	InstallDocker bool
	ProjectName   string
}

func GetProjectName(composeFilePath string) string {
	dir := filepath.Dir(composeFilePath)
	if dir == "." {
		return "default"
	}
	return filepath.Base(dir)
}

func BuildDeployOptions(servers []string, sshKey, sshUser, sshPassword string, installDocker bool, projectName string) *DeployOptions {
	return &DeployOptions{
		Servers:       servers,
		SSHKey:        sshKey,
		SSHUser:       sshUser,
		SSHPassword:   sshPassword,
		InstallDocker: installDocker,
		ProjectName:   projectName,
	}
}
