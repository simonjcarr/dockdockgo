package ssh

import (
	"fmt"
	"strings"
)

type Server struct {
	client *Client
	info   *ServerInfo
}

type ServerInfo struct {
	Host            string
	Port            string
	User            string
	OS              string
	Architecture    string
	DockerInstalled bool
	DockerVersion   string
	Available       bool
}

func NewServer(config *Config) *Server {
	return &Server{
		client: NewClient(config),
		info: &ServerInfo{
			Host: config.Host,
			Port: config.Port,
			User: config.User,
		},
	}
}

func (s *Server) Connect() error {
	return s.client.Connect()
}

func (s *Server) Validate() error {
	if err := s.client.Connect(); err != nil {
		return fmt.Errorf("failed to connect to server: %w", err)
	}

	if err := s.gatherServerInfo(); err != nil {
		return fmt.Errorf("failed to gather server information: %w", err)
	}

	s.info.Available = true
	return nil
}

func (s *Server) gatherServerInfo() error {
	// Get OS information
	osInfo, err := s.client.Execute("uname -s")
	if err != nil {
		return fmt.Errorf("failed to get OS info: %w", err)
	}
	s.info.OS = strings.TrimSpace(osInfo)

	// Get architecture
	arch, err := s.client.Execute("uname -m")
	if err != nil {
		return fmt.Errorf("failed to get architecture: %w", err)
	}
	s.info.Architecture = strings.TrimSpace(arch)

	// Check if Docker is installed
	dockerVersion, err := s.client.Execute("docker --version")
	if err != nil {
		s.info.DockerInstalled = false
		s.info.DockerVersion = ""
	} else {
		s.info.DockerInstalled = true
		s.info.DockerVersion = strings.TrimSpace(dockerVersion)
	}

	return nil
}

func (s *Server) IsDockerInstalled() bool {
	return s.info.DockerInstalled
}

func (s *Server) InstallDocker() error {
	if s.info.DockerInstalled {
		return nil // Already installed
	}

	var installCmd string
	switch strings.ToLower(s.info.OS) {
	case "linux":
		// Use Docker's official installation script
		installCmd = `curl -fsSL https://get.docker.com -o get-docker.sh && sh get-docker.sh && rm get-docker.sh`
	case "darwin":
		return fmt.Errorf("Docker installation on macOS must be done manually")
	default:
		return fmt.Errorf("unsupported OS for automatic Docker installation: %s", s.info.OS)
	}

	output, err := s.client.Execute(installCmd)
	if err != nil {
		return fmt.Errorf("failed to install Docker: %w\nOutput: %s", err, output)
	}

	// Verify installation
	if err := s.gatherServerInfo(); err != nil {
		return fmt.Errorf("failed to verify Docker installation: %w", err)
	}

	if !s.info.DockerInstalled {
		return fmt.Errorf("Docker installation completed but Docker is not available")
	}

	return nil
}

func (s *Server) GetInfo() *ServerInfo {
	return s.info
}

func (s *Server) Execute(command string) (string, error) {
	return s.client.Execute(command)
}

func (s *Server) Close() error {
	return s.client.Close()
}

func (s *Server) IsAvailable() bool {
	return s.info.Available
}
