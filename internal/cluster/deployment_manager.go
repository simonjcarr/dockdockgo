package cluster

import (
	"dockdockgo/internal/docker"
	"dockdockgo/internal/storage"
	"dockdockgo/pkg/types"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/google/uuid"
)

type DeploymentManager struct {
	storage      *storage.Storage
	scheduler    *Scheduler
	dockerClient *docker.Client
}

func NewDeploymentManager(storage *storage.Storage) *DeploymentManager {
	dockerClient, err := docker.NewClient()
	if err != nil {
		// Log error but continue - we'll handle this when scheduling containers
		fmt.Printf("Warning: Failed to create Docker client: %v\n", err)
		fmt.Printf("Docker daemon may not be running or accessible.\n")
	} else {
		// Test Docker connection
		if err := dockerClient.Ping(); err != nil {
			fmt.Printf("Warning: Docker ping failed: %v\n", err)
			fmt.Printf("Docker daemon may not be responding correctly.\n")
		}
	}

	return &DeploymentManager{
		storage:      storage,
		scheduler:    NewScheduler(storage),
		dockerClient: dockerClient,
	}
}

func (dm *DeploymentManager) CreateDeployment(spec *types.DeploymentSpec) (*types.Deployment, error) {
	// Validate spec
	if err := dm.validateDeploymentSpec(spec); err != nil {
		return nil, fmt.Errorf("invalid deployment spec: %w", err)
	}

	// Check if deployment with same name already exists
	if _, err := dm.storage.GetDeploymentByName(spec.Name); err == nil {
		return nil, fmt.Errorf("deployment with name %s already exists", spec.Name)
	}

	// Create deployment
	deployment := &types.Deployment{
		ID:            uuid.New().String(),
		Name:          spec.Name,
		Image:         spec.Image,
		Command:       spec.Command,
		Entrypoint:    spec.Entrypoint,
		Environment:   spec.Environment,
		Ports:         spec.Ports,
		Volumes:       spec.Volumes,
		Replicas:      spec.Replicas,
		Placement:     spec.Placement,
		HealthCheck:   spec.HealthCheck,
		RestartPolicy: spec.RestartPolicy,
		Status:        types.DeploymentPending,
		Containers:    make(map[string]*types.Container),
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	// Set default restart policy
	if deployment.RestartPolicy == "" {
		deployment.RestartPolicy = "unless-stopped"
	}

	// Save deployment
	if err := dm.storage.SaveDeployment(deployment); err != nil {
		return nil, fmt.Errorf("failed to save deployment: %w", err)
	}

	// Schedule initial containers
	if err := dm.scheduleContainers(deployment, deployment.Replicas); err != nil {
		deployment.Status = types.DeploymentFailed
		dm.storage.SaveDeployment(deployment)
		return nil, fmt.Errorf("failed to schedule containers: %w", err)
	}

	deployment.Status = types.DeploymentRunning
	dm.storage.SaveDeployment(deployment)

	return deployment, nil
}

func (dm *DeploymentManager) ScaleDeployment(deploymentID string, replicas int) (*types.Deployment, error) {
	deployment, err := dm.storage.GetDeployment(deploymentID)
	if err != nil {
		return nil, fmt.Errorf("deployment not found: %w", err)
	}

	deployment.Replicas = replicas
	deployment.Status = types.DeploymentScaling
	deployment.UpdatedAt = time.Now()

	if err := dm.storage.SaveDeployment(deployment); err != nil {
		return nil, fmt.Errorf("failed to save deployment: %w", err)
	}

	// Get current containers
	containers, err := dm.storage.ListContainersByDeployment(deploymentID)
	if err != nil {
		return nil, fmt.Errorf("failed to list containers: %w", err)
	}

	runningContainers := 0
	for _, container := range containers {
		if container.Status == types.ContainerRunning {
			runningContainers++
		}
	}

	if replicas > runningContainers {
		// Scale up
		containersToAdd := replicas - runningContainers
		if err := dm.scheduleContainers(deployment, containersToAdd); err != nil {
			return nil, fmt.Errorf("failed to scale up: %w", err)
		}
	} else if replicas < runningContainers {
		// Scale down
		containersToRemove := runningContainers - replicas
		if err := dm.removeContainers(deployment, containersToRemove); err != nil {
			return nil, fmt.Errorf("failed to scale down: %w", err)
		}
	}

	deployment.Status = types.DeploymentRunning
	deployment.UpdatedAt = time.Now()
	dm.storage.SaveDeployment(deployment)

	return deployment, nil
}

func (dm *DeploymentManager) DeleteDeployment(deploymentID string) error {
	_, err := dm.storage.GetDeployment(deploymentID)
	if err != nil {
		return fmt.Errorf("deployment not found: %w", err)
	}

	// Stop all containers
	containers, err := dm.storage.ListContainersByDeployment(deploymentID)
	if err != nil {
		return fmt.Errorf("failed to list containers: %w", err)
	}

	for _, container := range containers {
		if err := dm.stopContainer(container); err != nil {
			// Log error but continue
			fmt.Printf("Warning: failed to stop container %s: %v\n", container.ID, err)
		}
		dm.storage.DeleteContainer(container.ID)
	}

	// Delete deployment
	return dm.storage.DeleteDeployment(deploymentID)
}

func (dm *DeploymentManager) GetDeployment(deploymentID string) (*types.Deployment, error) {
	return dm.storage.GetDeployment(deploymentID)
}

func (dm *DeploymentManager) GetDeploymentByName(name string) (*types.Deployment, error) {
	return dm.storage.GetDeploymentByName(name)
}

func (dm *DeploymentManager) ListDeployments() ([]*types.Deployment, error) {
	return dm.storage.ListDeployments()
}

func (dm *DeploymentManager) validateDeploymentSpec(spec *types.DeploymentSpec) error {
	if spec.Name == "" {
		return fmt.Errorf("deployment name is required")
	}
	if spec.Image == "" {
		return fmt.Errorf("deployment image is required")
	}
	if spec.Replicas < 0 {
		return fmt.Errorf("replicas cannot be negative")
	}
	if spec.Replicas == 0 {
		spec.Replicas = 1
	}

	// Validate port mappings
	for _, port := range spec.Ports {
		if port.ContainerPort <= 0 || port.ContainerPort > 65535 {
			return fmt.Errorf("invalid container port: %d", port.ContainerPort)
		}
		if port.HostPort < 0 || port.HostPort > 65535 {
			return fmt.Errorf("invalid host port: %d", port.HostPort)
		}
		if port.Protocol != "tcp" && port.Protocol != "udp" {
			port.Protocol = "tcp" // default to tcp
		}
	}

	return nil
}

func (dm *DeploymentManager) scheduleContainers(deployment *types.Deployment, count int) error {
	for i := 0; i < count; i++ {
		container, err := dm.createContainer(deployment, i)
		if err != nil {
			return fmt.Errorf("failed to create container %d: %w", i, err)
		}

		// Schedule container to a node
		node, err := dm.scheduler.ScheduleContainer(container, deployment.Placement)
		if err != nil {
			return fmt.Errorf("failed to schedule container %s: %w", container.ID, err)
		}

		container.NodeID = node.ID
		container.Status = types.ContainerPending

		// Save container
		if err := dm.storage.SaveContainer(container); err != nil {
			return fmt.Errorf("failed to save container: %w", err)
		}

		// Add container to deployment
		deployment.Containers[container.ID] = container

		// Start the container
		if err := dm.startContainer(container, deployment, node); err != nil {
			fmt.Printf("❌ Failed to start container %s: %v\n", container.Name, err)
			container.Status = types.ContainerFailed
			dm.storage.SaveContainer(container)
			
			// Return error on first failure to provide immediate feedback
			return fmt.Errorf("failed to start container %s: %w", container.Name, err)
		}

		fmt.Printf("✅ Successfully started container %s on node %s\n", container.Name, node.Hostname)
	}

	return nil
}

func (dm *DeploymentManager) createContainer(deployment *types.Deployment, index int) (*types.Container, error) {
	containerName := deployment.Name
	if deployment.Replicas > 1 {
		containerName = fmt.Sprintf("%s-%d", deployment.Name, index+1)
	}

	// Adjust ports for multiple replicas
	ports := make([]types.PortMapping, len(deployment.Ports))
	copy(ports, deployment.Ports)

	for i := range ports {
		if deployment.Replicas > 1 && ports[i].HostPort > 0 {
			ports[i].HostPort = ports[i].HostPort + index
		}
	}

	container := &types.Container{
		ID:           uuid.New().String(),
		Name:         containerName,
		DeploymentID: deployment.ID,
		Status:       types.ContainerPending,
		Health:       types.HealthUnknown,
		RestartCount: 0,
		Ports:        ports,
	}

	return container, nil
}

func (dm *DeploymentManager) removeContainers(deployment *types.Deployment, count int) error {
	containers, err := dm.storage.ListContainersByDeployment(deployment.ID)
	if err != nil {
		return fmt.Errorf("failed to list containers: %w", err)
	}

	// Find running containers to remove
	var runningContainers []*types.Container
	for _, container := range containers {
		if container.Status == types.ContainerRunning {
			runningContainers = append(runningContainers, container)
		}
	}

	if len(runningContainers) < count {
		count = len(runningContainers)
	}

	// Remove containers (starting from the end)
	for i := len(runningContainers) - count; i < len(runningContainers); i++ {
		container := runningContainers[i]

		if err := dm.stopContainer(container); err != nil {
			return fmt.Errorf("failed to stop container %s: %w", container.ID, err)
		}

		if err := dm.storage.DeleteContainer(container.ID); err != nil {
			return fmt.Errorf("failed to delete container %s: %w", container.ID, err)
		}

		delete(deployment.Containers, container.ID)
	}

	return nil
}

func (dm *DeploymentManager) stopContainer(container *types.Container) error {
	fmt.Printf("Stopping container %s on node %s\n", container.Name, container.NodeID)

	// Get current hostname to check if container is on local node
	currentHostname, err := os.Hostname()
	if err != nil {
		return fmt.Errorf("failed to get hostname: %w", err)
	}

	// Get node info
	node, err := dm.storage.GetNode(container.NodeID)
	if err != nil {
		return fmt.Errorf("failed to get node: %w", err)
	}

	// Stop container if it's on the local node
	if node.Hostname == currentHostname {
		if dm.dockerClient != nil && container.DockerID != "" {
			if err := dm.dockerClient.StopContainer(container.DockerID); err != nil {
				fmt.Printf("Warning: Failed to stop Docker container %s: %v\n", container.DockerID, err)
			}
			if err := dm.dockerClient.RemoveContainer(container.DockerID, true); err != nil {
				fmt.Printf("Warning: Failed to remove Docker container %s: %v\n", container.DockerID, err)
			}
		}
	} else {
		// TODO: Send container stop command to worker node via gRPC
		fmt.Printf("TODO: Stop container %s on remote node %s\n", container.Name, node.Hostname)
	}

	container.Status = types.ContainerStopped
	now := time.Now()
	container.FinishedAt = &now

	return dm.storage.SaveContainer(container)
}

func (dm *DeploymentManager) UpdateContainerStatus(containerID string, event *types.ContainerEvent) error {
	container, err := dm.storage.GetContainer(containerID)
	if err != nil {
		return fmt.Errorf("container not found: %w", err)
	}

	// Update container status
	container.Status = event.NewStatus
	container.LastHeartbeat = event.Timestamp

	if event.ExitCode != nil {
		container.ExitCode = event.ExitCode
	}

	if event.EventType == "restart" {
		container.RestartCount = event.RestartCount
	}

	// Update timestamps
	switch event.NewStatus {
	case types.ContainerRunning:
		if container.StartedAt == nil {
			container.StartedAt = &event.Timestamp
		}
	case types.ContainerStopped, types.ContainerFailed:
		container.FinishedAt = &event.Timestamp
	}

	// Save updated container
	if err := dm.storage.SaveContainer(container); err != nil {
		return fmt.Errorf("failed to save container: %w", err)
	}

	// Update deployment status
	return dm.updateDeploymentStatus(container.DeploymentID)
}

func (dm *DeploymentManager) startContainer(container *types.Container, deployment *types.Deployment, node *types.Node) error {
	// Get current hostname to check if container should be started locally
	currentHostname, err := os.Hostname()
	if err != nil {
		return fmt.Errorf("failed to get hostname: %w", err)
	}

	// Start container if it's on the local node
	if node.Hostname == currentHostname {
		if dm.dockerClient == nil {
			return fmt.Errorf("Docker client not available - Docker daemon may not be running")
		}

		fmt.Printf("📦 Pulling image %s...\n", deployment.Image)
		// Pull image if needed
		if err := dm.dockerClient.PullImage(deployment.Image); err != nil {
			fmt.Printf("⚠️  Failed to pull image %s: %v\n", deployment.Image, err)
			fmt.Printf("   Continuing with local image (if available)...\n")
			// Continue anyway - image might already exist locally
		} else {
			fmt.Printf("✅ Image %s pulled successfully\n", deployment.Image)
		}

		// Prepare container configuration
		dockerConfig := &docker.ContainerConfig{
			Image:         deployment.Image,
			Name:          container.Name,
			Ports:         dm.convertPortMappings(container.Ports),
			Environment:   dm.convertEnvironmentMap(deployment.Environment),
			Volumes:       dm.convertVolumeMappings(deployment.Volumes),
			Entrypoint:    deployment.Entrypoint,
			Cmd:           deployment.Command,
			RestartPolicy: deployment.RestartPolicy,
		}

		fmt.Printf("🚀 Starting container %s...\n", container.Name)
		// Start the container
		dockerContainerID, err := dm.dockerClient.RunContainer(dockerConfig)
		if err != nil {
			return fmt.Errorf("failed to start Docker container: %w", err)
		}
		fmt.Printf("✅ Container %s started with ID: %s\n", container.Name, dockerContainerID[:12])

		// Update container with Docker container ID and running status
		container.DockerID = dockerContainerID
		container.Status = types.ContainerRunning
		now := time.Now()
		container.StartedAt = &now
		container.LastHeartbeat = now

		return dm.storage.SaveContainer(container)
	} else {
		// TODO: Send container start command to worker node via gRPC
		fmt.Printf("TODO: Start container %s on remote node %s\n", container.Name, node.Hostname)
		// For now, mark as failed since we can't start on remote nodes
		container.Status = types.ContainerFailed
		return dm.storage.SaveContainer(container)
	}
}

func (dm *DeploymentManager) convertPortMappings(ports []types.PortMapping) []string {
	var portStrings []string
	for _, port := range ports {
		if port.HostPort > 0 {
			portStrings = append(portStrings, strconv.Itoa(port.HostPort)+":"+strconv.Itoa(port.ContainerPort))
		}
	}
	return portStrings
}

func (dm *DeploymentManager) convertEnvironmentMap(env map[string]string) []string {
	var envStrings []string
	for key, value := range env {
		envStrings = append(envStrings, key+"="+value)
	}
	return envStrings
}

func (dm *DeploymentManager) convertVolumeMappings(volumes []types.VolumeMapping) []string {
	var volumeStrings []string
	for _, volume := range volumes {
		volumeStr := volume.HostPath + ":" + volume.ContainerPath
		if volume.ReadOnly {
			volumeStr += ":ro"
		}
		volumeStrings = append(volumeStrings, volumeStr)
	}
	return volumeStrings
}

func (dm *DeploymentManager) updateDeploymentStatus(deploymentID string) error {
	deployment, err := dm.storage.GetDeployment(deploymentID)
	if err != nil {
		return fmt.Errorf("deployment not found: %w", err)
	}

	containers, err := dm.storage.ListContainersByDeployment(deploymentID)
	if err != nil {
		return fmt.Errorf("failed to list containers: %w", err)
	}

	runningCount := 0
	failedCount := 0

	for _, container := range containers {
		switch container.Status {
		case types.ContainerRunning:
			runningCount++
		case types.ContainerFailed:
			failedCount++
		}
	}

	// Determine deployment status
	if runningCount == deployment.Replicas {
		deployment.Status = types.DeploymentRunning
	} else if runningCount > 0 {
		deployment.Status = types.DeploymentRunning // Partially running
	} else if failedCount > 0 {
		deployment.Status = types.DeploymentFailed
	} else {
		deployment.Status = types.DeploymentPending
	}

	deployment.UpdatedAt = time.Now()
	return dm.storage.SaveDeployment(deployment)
}
