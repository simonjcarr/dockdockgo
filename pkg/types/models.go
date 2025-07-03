package types

import (
	"time"
)

// DeploymentStatus represents the current state of a deployment
type DeploymentStatus string

const (
	DeploymentPending   DeploymentStatus = "pending"
	DeploymentRunning   DeploymentStatus = "running"
	DeploymentFailed    DeploymentStatus = "failed"
	DeploymentStopped   DeploymentStatus = "stopped"
	DeploymentScaling   DeploymentStatus = "scaling"
	DeploymentUpdating  DeploymentStatus = "updating"
)

// ContainerStatus represents the current state of a container
type ContainerStatus string

const (
	ContainerPending    ContainerStatus = "pending"
	ContainerRunning    ContainerStatus = "running"
	ContainerStopped    ContainerStatus = "stopped"
	ContainerFailed     ContainerStatus = "failed"
	ContainerRestarting ContainerStatus = "restarting"
	ContainerUnknown    ContainerStatus = "unknown"
)

// NodeStatus represents the current state of a cluster node
type NodeStatus string

const (
	NodeOnline   NodeStatus = "online"
	NodeOffline  NodeStatus = "offline"
	NodeDraining NodeStatus = "draining"
	NodeFailed   NodeStatus = "failed"
)

// HealthStatus represents the health state of a container
type HealthStatus string

const (
	HealthHealthy   HealthStatus = "healthy"
	HealthUnhealthy HealthStatus = "unhealthy"
	HealthUnknown   HealthStatus = "unknown"
)

// Deployment represents a named container deployment
type Deployment struct {
	ID              string                 `json:"id"`
	Name            string                 `json:"name"`
	Image           string                 `json:"image"`
	Command         []string               `json:"command,omitempty"`
	Entrypoint      []string               `json:"entrypoint,omitempty"`
	Environment     map[string]string      `json:"environment,omitempty"`
	Ports           []PortMapping          `json:"ports,omitempty"`
	Volumes         []VolumeMapping        `json:"volumes,omitempty"`
	Replicas        int                    `json:"replicas"`
	Placement       *PlacementConfig       `json:"placement,omitempty"`
	HealthCheck     *HealthCheckConfig     `json:"health_check,omitempty"`
	RestartPolicy   string                 `json:"restart_policy"`
	Status          DeploymentStatus       `json:"status"`
	Containers      map[string]*Container  `json:"containers"`
	CreatedAt       time.Time              `json:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at"`
	CreatedBy       string                 `json:"created_by"`
}

// Container represents a running container instance
type Container struct {
	ID             string           `json:"id"`
	Name           string           `json:"name"`
	DeploymentID   string           `json:"deployment_id"`
	NodeID         string           `json:"node_id"`
	DockerID       string           `json:"docker_id"`
	Status         ContainerStatus  `json:"status"`
	Health         HealthStatus     `json:"health"`
	ExitCode       *int             `json:"exit_code,omitempty"`
	RestartCount   int              `json:"restart_count"`
	StartedAt      *time.Time       `json:"started_at,omitempty"`
	FinishedAt     *time.Time       `json:"finished_at,omitempty"`
	LastHeartbeat  time.Time        `json:"last_heartbeat"`
	Resources      *ResourceUsage   `json:"resources,omitempty"`
	Ports          []PortMapping    `json:"ports,omitempty"`
}

// Node represents a cluster member
type Node struct {
	ID            string            `json:"id"`
	Hostname      string            `json:"hostname"`
	IPAddress     string            `json:"ip_address"`
	Port          int               `json:"port"`
	Status        NodeStatus        `json:"status"`
	Role          string            `json:"role"` // master, worker
	Version       string            `json:"version"`
	Labels        map[string]string `json:"labels,omitempty"`
	Resources     *NodeResources    `json:"resources,omitempty"`
	LastHeartbeat time.Time         `json:"last_heartbeat"`
	JoinedAt      time.Time         `json:"joined_at"`
}

// PortMapping defines a port mapping for a container
type PortMapping struct {
	HostPort      int    `json:"host_port"`
	ContainerPort int    `json:"container_port"`
	Protocol      string `json:"protocol"` // tcp, udp
}

// VolumeMapping defines a volume mount for a container
type VolumeMapping struct {
	HostPath      string `json:"host_path"`
	ContainerPath string `json:"container_path"`
	ReadOnly      bool   `json:"read_only"`
}

// PlacementConfig defines placement constraints and preferences
type PlacementConfig struct {
	Strategy    string            `json:"strategy"`    // spread, pack, binpack
	Constraints []string          `json:"constraints"` // node.labels.foo==bar
	NodeLabels  map[string]string `json:"node_labels,omitempty"`
	TargetNodes []string          `json:"target_nodes,omitempty"`
}

// HealthCheckConfig defines container health checking
type HealthCheckConfig struct {
	HTTPGet     *HTTPHealthCheck `json:"http_get,omitempty"`
	TCPSocket   *TCPHealthCheck  `json:"tcp_socket,omitempty"`
	Exec        *ExecHealthCheck `json:"exec,omitempty"`
	Interval    time.Duration    `json:"interval"`
	Timeout     time.Duration    `json:"timeout"`
	Retries     int              `json:"retries"`
	StartPeriod time.Duration    `json:"start_period"`
}

// HTTPHealthCheck defines HTTP-based health checking
type HTTPHealthCheck struct {
	Path   string            `json:"path"`
	Port   int               `json:"port"`
	Scheme string            `json:"scheme"` // http, https
	Headers map[string]string `json:"headers,omitempty"`
}

// TCPHealthCheck defines TCP-based health checking
type TCPHealthCheck struct {
	Port int `json:"port"`
}

// ExecHealthCheck defines command-based health checking
type ExecHealthCheck struct {
	Command []string `json:"command"`
}

// ResourceUsage represents current resource consumption
type ResourceUsage struct {
	CPUPercent    float64 `json:"cpu_percent"`
	MemoryUsageMB uint64  `json:"memory_usage_mb"`
	MemoryLimitMB uint64  `json:"memory_limit_mb"`
	NetworkRX     uint64  `json:"network_rx"`
	NetworkTX     uint64  `json:"network_tx"`
	DiskRead      uint64  `json:"disk_read"`
	DiskWrite     uint64  `json:"disk_write"`
}

// NodeResources represents node capacity and usage
type NodeResources struct {
	CPUCores        int     `json:"cpu_cores"`
	CPUUsagePercent float64 `json:"cpu_usage_percent"`
	MemoryTotalMB   uint64  `json:"memory_total_mb"`
	MemoryUsageMB   uint64  `json:"memory_usage_mb"`
	DiskTotalGB     uint64  `json:"disk_total_gb"`
	DiskUsageGB     uint64  `json:"disk_usage_gb"`
	ContainerCount  int     `json:"container_count"`
	MaxContainers   int     `json:"max_containers"`
}

// ContainerEvent represents a container state change
type ContainerEvent struct {
	ContainerID   string          `json:"container_id"`
	DeploymentID  string          `json:"deployment_id"`
	NodeID        string          `json:"node_id"`
	EventType     string          `json:"event_type"` // start, stop, die, restart
	OldStatus     ContainerStatus `json:"old_status"`
	NewStatus     ContainerStatus `json:"new_status"`
	Timestamp     time.Time       `json:"timestamp"`
	ExitCode      *int            `json:"exit_code,omitempty"`
	Reason        string          `json:"reason,omitempty"`
	RestartCount  int             `json:"restart_count"`
}

// DeploymentSpec represents the desired state for deployment creation
type DeploymentSpec struct {
	Name          string                `json:"name"`
	Image         string                `json:"image"`
	Command       []string              `json:"command,omitempty"`
	Entrypoint    []string              `json:"entrypoint,omitempty"`
	Environment   map[string]string     `json:"environment,omitempty"`
	Ports         []PortMapping         `json:"ports,omitempty"`
	Volumes       []VolumeMapping       `json:"volumes,omitempty"`
	Replicas      int                   `json:"replicas"`
	Placement     *PlacementConfig      `json:"placement,omitempty"`
	HealthCheck   *HealthCheckConfig    `json:"health_check,omitempty"`
	RestartPolicy string                `json:"restart_policy"`
}

// ClusterState represents the overall cluster state
type ClusterState struct {
	Deployments    map[string]*Deployment `json:"deployments"`
	Containers     map[string]*Container  `json:"containers"`
	Nodes          map[string]*Node       `json:"nodes"`
	MasterNodeID   string                 `json:"master_node_id"`
	LeadershipTerm int64                  `json:"leadership_term"`
	LastLogIndex   int64                  `json:"last_log_index"`
	UpdatedAt      time.Time              `json:"updated_at"`
}