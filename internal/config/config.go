package config

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Servers    map[string]*ServerConfig `yaml:"servers"`
	Defaults   *DefaultConfig           `yaml:"defaults"`
	API        *APIConfig               `yaml:"api"`
	Cluster    *ClusterConfig           `yaml:"cluster"`
	Registry   *RegistryConfig          `yaml:"registry"`
	Monitoring *MonitoringConfig        `yaml:"monitoring"`
}

type ServerConfig struct {
	Host        string `yaml:"host"`
	Port        string `yaml:"port"`
	User        string `yaml:"user"`
	KeyPath     string `yaml:"key_path"`
	Password    string `yaml:"password"`
	DockerHost  string `yaml:"docker_host"`
	Labels      map[string]string `yaml:"labels"`
	MaxReplicas int    `yaml:"max_replicas"`
}

type DefaultConfig struct {
	SSHUser     string `yaml:"ssh_user"`
	SSHKeyPath  string `yaml:"ssh_key_path"`
	DockerUser  string `yaml:"docker_user"`
	NetworkMode string `yaml:"network_mode"`
	LogDriver   string `yaml:"log_driver"`
	Detach      bool   `yaml:"detach"`
}

type APIConfig struct {
	Host        string `yaml:"host"`
	Port        string `yaml:"port"`
	TLS         bool   `yaml:"tls"`
	CertFile    string `yaml:"cert_file"`
	KeyFile     string `yaml:"key_file"`
	TokenExpiry string `yaml:"token_expiry"`
}

type ClusterConfig struct {
	Name           string   `yaml:"name"`
	ZookeeperHosts []string `yaml:"zookeeper_hosts"`
	RedisPassword  string   `yaml:"redis_password"`
	ElectionTTL    int      `yaml:"election_ttl"`
	HealthInterval int      `yaml:"health_interval"`
}

type RegistryConfig struct {
	DefaultRegistry string                     `yaml:"default_registry"`
	Registries      map[string]*RegistryInfo   `yaml:"registries"`
}

type RegistryInfo struct {
	URL      string `yaml:"url"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	Insecure bool   `yaml:"insecure"`
}

type MonitoringConfig struct {
	Enabled         bool   `yaml:"enabled"`
	MetricsPort     string `yaml:"metrics_port"`
	LogLevel        string `yaml:"log_level"`
	HealthCheckPort string `yaml:"health_check_port"`
}

var globalConfig *Config

func Load(configPath string) (*Config, error) {
	if configPath == "" {
		configPath = getDefaultConfigPath()
	}

	data, err := ioutil.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Create default config if it doesn't exist
			config := getDefaultConfig()
			if err := Save(config, configPath); err != nil {
				return nil, fmt.Errorf("failed to create default config: %w", err)
			}
			return config, nil
		}
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Set defaults
	config.setDefaults()

	globalConfig = &config
	return &config, nil
}

func Save(config *Config, configPath string) error {
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Create directory if it doesn't exist
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	if err := ioutil.WriteFile(configPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

func Get() *Config {
	if globalConfig == nil {
		config, err := Load("")
		if err != nil {
			return getDefaultConfig()
		}
		return config
	}
	return globalConfig
}

func getDefaultConfig() *Config {
	homeDir, _ := os.UserHomeDir()
	defaultSSHKey := filepath.Join(homeDir, ".ssh", "id_rsa")

	return &Config{
		Servers: make(map[string]*ServerConfig),
		Defaults: &DefaultConfig{
			SSHUser:     os.Getenv("USER"),
			SSHKeyPath:  defaultSSHKey,
			DockerUser:  "docker",
			NetworkMode: "bridge",
			LogDriver:   "json-file",
			Detach:      true,
		},
		API: &APIConfig{
			Host:        "0.0.0.0",
			Port:        "8080",
			TLS:         false,
			TokenExpiry: "24h",
		},
		Cluster: &ClusterConfig{
			Name:           "dockdockgo-cluster",
			ZookeeperHosts: []string{"localhost:2181"},
			ElectionTTL:    30,
			HealthInterval: 10,
		},
		Registry: &RegistryConfig{
			DefaultRegistry: "docker.io",
			Registries: map[string]*RegistryInfo{
				"docker.io": {
					URL:      "https://index.docker.io/v1/",
					Insecure: false,
				},
			},
		},
		Monitoring: &MonitoringConfig{
			Enabled:         true,
			MetricsPort:     "9090",
			LogLevel:        "info",
			HealthCheckPort: "8081",
		},
	}
}

func (c *Config) setDefaults() {
	if c.Defaults == nil {
		c.Defaults = getDefaultConfig().Defaults
	}
	if c.API == nil {
		c.API = getDefaultConfig().API
	}
	if c.Cluster == nil {
		c.Cluster = getDefaultConfig().Cluster
	}
	if c.Registry == nil {
		c.Registry = getDefaultConfig().Registry
	}
	if c.Monitoring == nil {
		c.Monitoring = getDefaultConfig().Monitoring
	}
}

func getDefaultConfigPath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "./dockdockgo.yaml"
	}
	return filepath.Join(homeDir, ".dockdockgo", "config.yaml")
}

func (c *Config) GetServer(name string) (*ServerConfig, bool) {
	server, exists := c.Servers[name]
	return server, exists
}

func (c *Config) AddServer(name string, server *ServerConfig) {
	if c.Servers == nil {
		c.Servers = make(map[string]*ServerConfig)
	}
	c.Servers[name] = server
}

func (c *Config) RemoveServer(name string) {
	delete(c.Servers, name)
}

func (c *Config) ListServers() []string {
	servers := make([]string, 0, len(c.Servers))
	for name := range c.Servers {
		servers = append(servers, name)
	}
	return servers
}