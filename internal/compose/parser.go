package compose

import (
	"fmt"
	"io/ioutil"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

type ComposeFile struct {
	Version  string                    `yaml:"version"`
	Services map[string]*Service       `yaml:"services"`
	Networks map[string]*Network       `yaml:"networks"`
	Volumes  map[string]*Volume        `yaml:"volumes"`
	Secrets  map[string]*Secret        `yaml:"secrets"`
	Configs  map[string]*Config        `yaml:"configs"`
	
	// DockDockGo extensions
	DockDockGo *DockDockGoExtension `yaml:"x-dockdockgo"`
}

type Service struct {
	Image         string                 `yaml:"image"`
	Build         *BuildConfig           `yaml:"build"`
	Command       interface{}            `yaml:"command"`
	Entrypoint    interface{}            `yaml:"entrypoint"`
	Environment   interface{}            `yaml:"environment"`
	Ports         []string               `yaml:"ports"`
	Volumes       []string               `yaml:"volumes"`
	Networks      interface{}            `yaml:"networks"`
	DependsOn     interface{}            `yaml:"depends_on"`
	Restart       string                 `yaml:"restart"`
	ContainerName string                 `yaml:"container_name"`
	Hostname      string                 `yaml:"hostname"`
	WorkingDir    string                 `yaml:"working_dir"`
	User          string                 `yaml:"user"`
	Labels        map[string]string      `yaml:"labels"`
	Expose        []string               `yaml:"expose"`
	Links         []string               `yaml:"links"`
	ExternalLinks []string               `yaml:"external_links"`
	Deploy        *DeployConfig          `yaml:"deploy"`
	
	// DockDockGo extensions
	DockDockGo *ServiceDockDockGo `yaml:"x-dockdockgo"`
}

type BuildConfig struct {
	Context    string            `yaml:"context"`
	Dockerfile string            `yaml:"dockerfile"`
	Args       map[string]string `yaml:"args"`
	Target     string            `yaml:"target"`
}

type DeployConfig struct {
	Replicas    int                    `yaml:"replicas"`
	UpdateConfig *UpdateConfig         `yaml:"update_config"`
	RestartPolicy *RestartPolicyConfig `yaml:"restart_policy"`
	Placement   *PlacementConfig       `yaml:"placement"`
	Resources   *ResourcesConfig       `yaml:"resources"`
}

type UpdateConfig struct {
	Parallelism int    `yaml:"parallelism"`
	Delay       string `yaml:"delay"`
	Order       string `yaml:"order"`
}

type RestartPolicyConfig struct {
	Condition string `yaml:"condition"`
	Delay     string `yaml:"delay"`
	MaxAttempts int  `yaml:"max_attempts"`
	Window    string `yaml:"window"`
}

type PlacementConfig struct {
	Constraints []string `yaml:"constraints"`
	Preferences []string `yaml:"preferences"`
}

type ResourcesConfig struct {
	Limits       *ResourceLimits `yaml:"limits"`
	Reservations *ResourceLimits `yaml:"reservations"`
}

type ResourceLimits struct {
	CPUs   string `yaml:"cpus"`
	Memory string `yaml:"memory"`
}

type Network struct {
	Driver     string            `yaml:"driver"`
	DriverOpts map[string]string `yaml:"driver_opts"`
	External   bool              `yaml:"external"`
	IPAM       *IPAMConfig       `yaml:"ipam"`
	Labels     map[string]string `yaml:"labels"`
}

type IPAMConfig struct {
	Driver string       `yaml:"driver"`
	Config []IPAMEntry  `yaml:"config"`
}

type IPAMEntry struct {
	Subnet  string `yaml:"subnet"`
	Gateway string `yaml:"gateway"`
}

type Volume struct {
	Driver     string            `yaml:"driver"`
	DriverOpts map[string]string `yaml:"driver_opts"`
	External   bool              `yaml:"external"`
	Labels     map[string]string `yaml:"labels"`
}

type Secret struct {
	File     string            `yaml:"file"`
	External bool              `yaml:"external"`
	Labels   map[string]string `yaml:"labels"`
}

type Config struct {
	File     string            `yaml:"file"`
	External bool              `yaml:"external"`
	Labels   map[string]string `yaml:"labels"`
}

// DockDockGo specific extensions
type DockDockGoExtension struct {
	DefaultServers []string                      `yaml:"default_servers"`
	ServerGroups   map[string][]string           `yaml:"server_groups"`
	Placement      *GlobalPlacementConfig        `yaml:"placement"`
}

type ServiceDockDockGo struct {
	Servers     []string                `yaml:"servers"`
	ServerGroup string                  `yaml:"server_group"`
	Replicas    int                     `yaml:"replicas"`
	Placement   *ServicePlacementConfig `yaml:"placement"`
}

type GlobalPlacementConfig struct {
	Strategy string            `yaml:"strategy"` // spread, pack, binpack
	Labels   map[string]string `yaml:"labels"`
}

type ServicePlacementConfig struct {
	Strategy string            `yaml:"strategy"`
	Labels   map[string]string `yaml:"labels"`
	Affinity []string          `yaml:"affinity"`
}

func ParseComposeFile(filePath string) (*ComposeFile, error) {
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read compose file: %w", err)
	}

	return ParseComposeData(data)
}

func ParseComposeData(data []byte) (*ComposeFile, error) {
	var compose ComposeFile
	if err := yaml.Unmarshal(data, &compose); err != nil {
		return nil, fmt.Errorf("failed to parse compose file: %w", err)
	}

	// Set defaults
	if compose.Version == "" {
		compose.Version = "3.8"
	}

	// Process services
	for name, service := range compose.Services {
		if service.ContainerName == "" {
			service.ContainerName = name
		}
		if service.Restart == "" {
			service.Restart = "no"
		}
		
		// Process DockDockGo extensions
		if service.DockDockGo != nil {
			// Inherit default servers if not specified
			if len(service.DockDockGo.Servers) == 0 && compose.DockDockGo != nil {
				service.DockDockGo.Servers = compose.DockDockGo.DefaultServers
			}
			
			// Resolve server groups
			if service.DockDockGo.ServerGroup != "" && compose.DockDockGo != nil {
				if servers, exists := compose.DockDockGo.ServerGroups[service.DockDockGo.ServerGroup]; exists {
					service.DockDockGo.Servers = append(service.DockDockGo.Servers, servers...)
				}
			}
		}
	}

	return &compose, nil
}

func (s *Service) GetEnvironmentMap() map[string]string {
	env := make(map[string]string)
	
	switch e := s.Environment.(type) {
	case []interface{}:
		for _, item := range e {
			if str, ok := item.(string); ok {
				parts := strings.SplitN(str, "=", 2)
				if len(parts) == 2 {
					env[parts[0]] = parts[1]
				} else {
					env[parts[0]] = ""
				}
			}
		}
	case map[string]interface{}:
		for key, value := range e {
			if str, ok := value.(string); ok {
				env[key] = str
			} else {
				env[key] = fmt.Sprintf("%v", value)
			}
		}
	case []string:
		for _, item := range e {
			parts := strings.SplitN(item, "=", 2)
			if len(parts) == 2 {
				env[parts[0]] = parts[1]
			} else {
				env[parts[0]] = ""
			}
		}
	}
	
	return env
}

func (s *Service) GetCommandSlice() []string {
	switch c := s.Command.(type) {
	case string:
		return strings.Fields(c)
	case []interface{}:
		cmd := make([]string, len(c))
		for i, item := range c {
			cmd[i] = fmt.Sprintf("%v", item)
		}
		return cmd
	case []string:
		return c
	default:
		return nil
	}
}

func (s *Service) GetEntrypointSlice() []string {
	switch e := s.Entrypoint.(type) {
	case string:
		return strings.Fields(e)
	case []interface{}:
		entrypoint := make([]string, len(e))
		for i, item := range e {
			entrypoint[i] = fmt.Sprintf("%v", item)
		}
		return entrypoint
	case []string:
		return e
	default:
		return nil
	}
}

func (s *Service) GetReplicas() int {
	if s.DockDockGo != nil && s.DockDockGo.Replicas > 0 {
		return s.DockDockGo.Replicas
	}
	if s.Deploy != nil && s.Deploy.Replicas > 0 {
		return s.Deploy.Replicas
	}
	return 1
}

func (s *Service) GetTargetServers() []string {
	if s.DockDockGo != nil {
		return s.DockDockGo.Servers
	}
	return []string{}
}

func (c *ComposeFile) Validate() error {
	if c.Services == nil || len(c.Services) == 0 {
		return fmt.Errorf("compose file must contain at least one service")
	}

	for name, service := range c.Services {
		if service.Image == "" && service.Build == nil {
			return fmt.Errorf("service '%s' must specify either 'image' or 'build'", name)
		}
		
		// Validate port mappings
		for _, port := range service.Ports {
			if !isValidPortMapping(port) {
				return fmt.Errorf("service '%s' has invalid port mapping: %s", name, port)
			}
		}
	}

	return nil
}

func isValidPortMapping(port string) bool {
	parts := strings.Split(port, ":")
	if len(parts) < 1 || len(parts) > 3 {
		return false
	}
	
	for _, part := range parts {
		if portPart := strings.Split(part, "/")[0]; portPart != "" {
			if _, err := strconv.Atoi(portPart); err != nil {
				return false
			}
		}
	}
	
	return true
}