package docker

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/registry"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
)

type Client struct {
	cli *client.Client
	ctx context.Context
}

type ContainerConfig struct {
	Image         string
	Name          string
	Ports         []string
	Environment   []string
	Volumes       []string
	WorkingDir    string
	Entrypoint    []string
	Cmd           []string
	Detach        bool
	RestartPolicy string
}

func NewClient() (*Client, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("failed to create Docker client: %w", err)
	}

	return &Client{
		cli: cli,
		ctx: context.Background(),
	}, nil
}

func NewRemoteClient(host string) (*Client, error) {
	cli, err := client.NewClientWithOpts(
		client.WithHost(host),
		client.WithAPIVersionNegotiation(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create Docker client for %s: %w", host, err)
	}

	return &Client{
		cli: cli,
		ctx: context.Background(),
	}, nil
}

func (c *Client) Ping() error {
	_, err := c.cli.Ping(c.ctx)
	return err
}

func (c *Client) RunContainer(config *ContainerConfig) (string, error) {
	// Parse port mappings
	portBindings, exposedPorts, err := c.parsePortMappings(config.Ports)
	if err != nil {
		return "", fmt.Errorf("failed to parse port mappings: %w", err)
	}

	// Parse volume mappings
	binds := config.Volumes

	// Create container configuration
	containerConfig := &container.Config{
		Image:        config.Image,
		Env:          config.Environment,
		ExposedPorts: exposedPorts,
		WorkingDir:   config.WorkingDir,
		Entrypoint:   config.Entrypoint,
		Cmd:          config.Cmd,
	}

	// Create host configuration
	hostConfig := &container.HostConfig{
		PortBindings: portBindings,
		Binds:        binds,
	}

	// Set restart policy
	if config.RestartPolicy != "" {
		hostConfig.RestartPolicy = container.RestartPolicy{
			Name: container.RestartPolicyMode(config.RestartPolicy),
		}
	}

	// Create container
	resp, err := c.cli.ContainerCreate(c.ctx, containerConfig, hostConfig, nil, nil, config.Name)
	if err != nil {
		return "", fmt.Errorf("failed to create container: %w", err)
	}

	// Start container
	if err := c.cli.ContainerStart(c.ctx, resp.ID, container.StartOptions{}); err != nil {
		return "", fmt.Errorf("failed to start container: %w", err)
	}

	return resp.ID, nil
}

func (c *Client) parsePortMappings(ports []string) (nat.PortMap, nat.PortSet, error) {
	portBindings := make(nat.PortMap)
	exposedPorts := make(nat.PortSet)

	for _, port := range ports {
		parts := strings.Split(port, ":")
		if len(parts) != 2 {
			return nil, nil, fmt.Errorf("invalid port mapping format: %s", port)
		}

		hostPort := parts[0]
		containerPortStr := parts[1]

		// Add protocol if not specified
		if !strings.Contains(containerPortStr, "/") {
			containerPortStr += "/tcp"
		}

		containerPort, err := nat.NewPort("tcp", strings.Split(containerPortStr, "/")[0])
		if err != nil {
			return nil, nil, fmt.Errorf("invalid container port: %s", containerPortStr)
		}

		portBindings[containerPort] = []nat.PortBinding{{HostPort: hostPort}}
		exposedPorts[containerPort] = struct{}{}
	}

	return portBindings, exposedPorts, nil
}

func (c *Client) StopContainer(containerID string) error {
	timeout := 30 // seconds
	return c.cli.ContainerStop(c.ctx, containerID, container.StopOptions{Timeout: &timeout})
}

func (c *Client) RemoveContainer(containerID string, force bool) error {
	return c.cli.ContainerRemove(c.ctx, containerID, container.RemoveOptions{Force: force})
}

func (c *Client) RestartContainer(containerID string) error {
	timeout := 30 // seconds
	return c.cli.ContainerRestart(c.ctx, containerID, container.StopOptions{Timeout: &timeout})
}

func (c *Client) ListContainers(all bool) ([]types.Container, error) {
	return c.cli.ContainerList(c.ctx, container.ListOptions{All: all})
}

func (c *Client) GetContainerLogs(containerID string, follow bool) (io.ReadCloser, error) {
	return c.cli.ContainerLogs(c.ctx, containerID, container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     follow,
		Timestamps: true,
	})
}

func (c *Client) PullImage(imageName string) error {
	reader, err := c.cli.ImagePull(c.ctx, imageName, image.PullOptions{})
	if err != nil {
		return fmt.Errorf("failed to pull image %s: %w", imageName, err)
	}
	defer reader.Close()

	// Read the response to completion
	_, err = io.Copy(io.Discard, reader)
	return err
}

func (c *Client) ListImages() ([]image.Summary, error) {
	return c.cli.ImageList(c.ctx, image.ListOptions{})
}

func (c *Client) RemoveImage(imageID string, force bool) error {
	_, err := c.cli.ImageRemove(c.ctx, imageID, image.RemoveOptions{Force: force})
	return err
}

func (c *Client) SearchImages(term string) ([]registry.SearchResult, error) {
	return c.cli.ImageSearch(c.ctx, term, registry.SearchOptions{Limit: 25})
}

func (c *Client) Close() error {
	return c.cli.Close()
}
