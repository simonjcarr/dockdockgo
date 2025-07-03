package orchestrator

import (
	"dockdockgo/internal/docker"
	"dockdockgo/internal/ssh"
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Orchestrator struct {
	servers []*ssh.Server
}

type DeploymentConfig struct {
	Image         string
	Name          string
	Servers       []string
	Replicas      int
	Ports         []string
	Environment   []string
	Volumes       []string
	SSHKey        string
	SSHUser       string
	SSHPassword   string
	InstallDocker bool
	Detach        bool
}

func NewOrchestrator() *Orchestrator {
	return &Orchestrator{
		servers: make([]*ssh.Server, 0),
	}
}

func (o *Orchestrator) DeployContainer(config *DeploymentConfig) error {
	if len(config.Servers) == 0 {
		// Deploy locally
		return o.deployLocally(config)
	}

	// Deploy to remote servers
	return o.deployToRemoteServers(config)
}

func (o *Orchestrator) deployLocally(config *DeploymentConfig) error {
	fmt.Println("Deploying container locally...")

	dockerClient, err := docker.NewClient()
	if err != nil {
		return fmt.Errorf("failed to create Docker client: %w", err)
	}
	defer dockerClient.Close()

	// Check if Docker is available
	if err := dockerClient.Ping(); err != nil {
		return fmt.Errorf("Docker is not available locally: %w", err)
	}

	// Pull image if needed
	fmt.Printf("Pulling image: %s\n", config.Image)
	if err := dockerClient.PullImage(config.Image); err != nil {
		fmt.Printf("Warning: failed to pull image, using local version: %v\n", err)
	}

	// Deploy replicas locally
	for i := 0; i < config.Replicas; i++ {
		containerName := config.Name
		if config.Replicas > 1 {
			containerName = fmt.Sprintf("%s-%d", config.Name, i+1)
		}

		// Adjust ports for multiple replicas
		ports := o.adjustPortsForReplica(config.Ports, i)

		dockerConfig := &docker.ContainerConfig{
			Image:       config.Image,
			Name:        containerName,
			Ports:       ports,
			Environment: config.Environment,
			Volumes:     config.Volumes,
			Detach:      config.Detach,
		}

		containerID, err := dockerClient.RunContainer(dockerConfig)
		if err != nil {
			return fmt.Errorf("failed to run container %s: %w", containerName, err)
		}

		fmt.Printf("✓ Container %s started (ID: %s)\n", containerName, containerID[:12])
	}

	return nil
}

func (o *Orchestrator) deployToRemoteServers(config *DeploymentConfig) error {
	fmt.Printf("Deploying to %d remote servers...\n", len(config.Servers))

	// Connect to all servers first
	servers := make([]*ssh.Server, 0, len(config.Servers))

	for _, serverAddr := range config.Servers {
		server, err := o.connectToServer(serverAddr, config)
		if err != nil {
			return fmt.Errorf("failed to connect to server %s: %w", serverAddr, err)
		}
		servers = append(servers, server)
	}

	// Distribute replicas across servers
	replicasPerServer := config.Replicas / len(servers)
	remainingReplicas := config.Replicas % len(servers)

	replicaCount := 0
	for i, server := range servers {
		serverReplicas := replicasPerServer
		if i < remainingReplicas {
			serverReplicas++
		}

		for j := 0; j < serverReplicas; j++ {
			containerName := config.Name
			if config.Replicas > 1 {
				containerName = fmt.Sprintf("%s-%d", config.Name, replicaCount+1)
			}

			err := o.deployToServer(server, config, containerName, replicaCount)
			if err != nil {
				return fmt.Errorf("failed to deploy to server %s: %w", server.GetInfo().Host, err)
			}

			replicaCount++
		}
	}

	return nil
}

func (o *Orchestrator) connectToServer(serverAddr string, config *DeploymentConfig) (*ssh.Server, error) {
	// Parse server address (handle user@host:port format)
	parts := strings.Split(serverAddr, "@")
	var user, hostPort string

	if len(parts) == 2 {
		user = parts[0]
		hostPort = parts[1]
	} else {
		user = config.SSHUser
		hostPort = serverAddr
	}

	// Parse host and port
	host := hostPort
	port := "22"
	if strings.Contains(hostPort, ":") {
		hostPortParts := strings.Split(hostPort, ":")
		host = hostPortParts[0]
		port = hostPortParts[1]
	}

	// Use default user if not specified
	if user == "" {
		user = os.Getenv("USER")
		if user == "" {
			user = "root"
		}
	}

	sshConfig := &ssh.Config{
		Host:     host,
		Port:     port,
		User:     user,
		Password: config.SSHPassword,
		KeyPath:  config.SSHKey,
	}

	// Use default SSH key if none specified
	if sshConfig.KeyPath == "" && sshConfig.Password == "" {
		sshConfig.KeyPath = ssh.GetDefaultKeyPath()
	}

	server := ssh.NewServer(sshConfig)

	fmt.Printf("Connecting to %s@%s:%s...\n", user, host, port)
	if err := server.Validate(); err != nil {
		return nil, err
	}

	fmt.Printf("✓ Connected to %s\n", host)

	// Check Docker installation
	if !server.IsDockerInstalled() {
		if config.InstallDocker {
			fmt.Printf("Installing Docker on %s...\n", host)
			if err := server.InstallDocker(); err != nil {
				return nil, fmt.Errorf("failed to install Docker: %w", err)
			}
			fmt.Printf("✓ Docker installed on %s\n", host)
		} else {
			fmt.Printf("Docker is not installed on %s. Use --install-docker flag to install automatically.\n", host)
			return nil, fmt.Errorf("Docker not available on %s", host)
		}
	}

	return server, nil
}

func (o *Orchestrator) deployToServer(server *ssh.Server, config *DeploymentConfig, containerName string, replicaIndex int) error {
	host := server.GetInfo().Host

	// Pull image
	fmt.Printf("Pulling image %s on %s...\n", config.Image, host)
	_, err := server.Execute(fmt.Sprintf("docker pull %s", config.Image))
	if err != nil {
		fmt.Printf("Warning: failed to pull image on %s: %v\n", host, err)
	}

	// Build docker run command
	cmd := o.buildDockerRunCommand(config, containerName, replicaIndex)

	fmt.Printf("Starting container %s on %s...\n", containerName, host)
	fmt.Printf("Command: %s\n", cmd)

	output, err := server.Execute(cmd)
	if err != nil {
		return fmt.Errorf("failed to run container: %w\nOutput: %s", err, output)
	}

	containerID := strings.TrimSpace(output)
	if len(containerID) < 12 {
		fmt.Printf("✓ Container %s started on %s (ID: %s)\n", containerName, host, containerID)
	} else {
		fmt.Printf("✓ Container %s started on %s (ID: %s)\n", containerName, host, containerID[:12])
	}

	return nil
}

func (o *Orchestrator) buildDockerRunCommand(config *DeploymentConfig, containerName string, replicaIndex int) string {
	cmd := []string{"docker", "run"}

	// Always run in detached mode for remote deployments to prevent SSH hanging
	cmd = append(cmd, "-d")

	// Add name
	if containerName != "" {
		cmd = append(cmd, "--name", containerName)
	}

	// Add port mappings
	ports := o.adjustPortsForReplica(config.Ports, replicaIndex)
	for _, port := range ports {
		cmd = append(cmd, "-p", port)
	}

	// Add environment variables
	for _, env := range config.Environment {
		cmd = append(cmd, "-e", env)
	}

	// Add volumes
	for _, volume := range config.Volumes {
		cmd = append(cmd, "-v", volume)
	}

	// Add image
	cmd = append(cmd, config.Image)

	return strings.Join(cmd, " ")
}

func (o *Orchestrator) adjustPortsForReplica(ports []string, replicaIndex int) []string {
	if replicaIndex == 0 {
		return ports
	}

	adjustedPorts := make([]string, len(ports))
	for i, port := range ports {
		parts := strings.Split(port, ":")
		if len(parts) == 2 {
			hostPort, err := strconv.Atoi(parts[0])
			if err == nil {
				adjustedPorts[i] = fmt.Sprintf("%d:%s", hostPort+replicaIndex, parts[1])
			} else {
				adjustedPorts[i] = port
			}
		} else {
			adjustedPorts[i] = port
		}
	}
	return adjustedPorts
}
