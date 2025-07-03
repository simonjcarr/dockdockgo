package cluster

import (
	"dockdockgo/internal/storage"
	"dockdockgo/pkg/types"
	"fmt"
	"math"
	"strings"
)

type Scheduler struct {
	storage *storage.Storage
}

func NewScheduler(storage *storage.Storage) *Scheduler {
	return &Scheduler{
		storage: storage,
	}
}

func (s *Scheduler) ScheduleContainer(container *types.Container, placement *types.PlacementConfig) (*types.Node, error) {
	// Get available nodes
	nodes, err := s.getAvailableNodes()
	if err != nil {
		return nil, fmt.Errorf("failed to get available nodes: %w", err)
	}

	if len(nodes) == 0 {
		return nil, fmt.Errorf("no available nodes found")
	}

	// Apply placement constraints
	if placement != nil {
		nodes = s.applyPlacementConstraints(nodes, placement)
		if len(nodes) == 0 {
			return nil, fmt.Errorf("no nodes match placement constraints")
		}
	}

	// Choose scheduling strategy
	strategy := "spread" // default
	if placement != nil && placement.Strategy != "" {
		strategy = placement.Strategy
	}

	var selectedNode *types.Node
	switch strategy {
	case "spread":
		selectedNode = s.scheduleSpread(nodes, container)
	case "pack":
		selectedNode = s.schedulePack(nodes, container)
	case "binpack":
		selectedNode = s.scheduleBinPack(nodes, container)
	default:
		selectedNode = s.scheduleSpread(nodes, container)
	}

	if selectedNode == nil {
		return nil, fmt.Errorf("failed to find suitable node")
	}

	return selectedNode, nil
}

func (s *Scheduler) getAvailableNodes() ([]*types.Node, error) {
	allNodes, err := s.storage.ListNodes()
	if err != nil {
		return nil, err
	}

	var availableNodes []*types.Node
	for _, node := range allNodes {
		if node.Status == types.NodeOnline && node.Role == "worker" {
			availableNodes = append(availableNodes, node)
		}
	}

	return availableNodes, nil
}

func (s *Scheduler) applyPlacementConstraints(nodes []*types.Node, placement *types.PlacementConfig) []*types.Node {
	var filteredNodes []*types.Node

	for _, node := range nodes {
		if s.nodeMatchesConstraints(node, placement) {
			filteredNodes = append(filteredNodes, node)
		}
	}

	return filteredNodes
}

func (s *Scheduler) nodeMatchesConstraints(node *types.Node, placement *types.PlacementConfig) bool {
	// Check target nodes
	if len(placement.TargetNodes) > 0 {
		found := false
		for _, targetNode := range placement.TargetNodes {
			if node.ID == targetNode || node.Hostname == targetNode {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Check node labels
	if len(placement.NodeLabels) > 0 {
		for key, value := range placement.NodeLabels {
			if nodeValue, exists := node.Labels[key]; !exists || nodeValue != value {
				return false
			}
		}
	}

	// Check constraints (format: "node.labels.key==value" or "node.hostname!=value")
	for _, constraint := range placement.Constraints {
		if !s.evaluateConstraint(node, constraint) {
			return false
		}
	}

	return true
}

func (s *Scheduler) evaluateConstraint(node *types.Node, constraint string) bool {
	// Parse constraint (simplified implementation)
	if strings.Contains(constraint, "==") {
		parts := strings.Split(constraint, "==")
		if len(parts) != 2 {
			return false
		}
		
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		
		return s.getNodeProperty(node, key) == value
	} else if strings.Contains(constraint, "!=") {
		parts := strings.Split(constraint, "!=")
		if len(parts) != 2 {
			return false
		}
		
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		
		return s.getNodeProperty(node, key) != value
	}

	return true
}

func (s *Scheduler) getNodeProperty(node *types.Node, property string) string {
	switch {
	case property == "node.hostname":
		return node.Hostname
	case property == "node.id":
		return node.ID
	case strings.HasPrefix(property, "node.labels."):
		labelKey := strings.TrimPrefix(property, "node.labels.")
		if value, exists := node.Labels[labelKey]; exists {
			return value
		}
	}
	return ""
}

// Spread strategy: distribute containers evenly across nodes
func (s *Scheduler) scheduleSpread(nodes []*types.Node, container *types.Container) *types.Node {
	// Get container counts per node
	containerCounts := s.getContainerCountsPerNode()
	
	var bestNode *types.Node
	minContainers := math.MaxInt32
	
	for _, node := range nodes {
		if s.hasCapacity(node, container) {
			count := containerCounts[node.ID]
			if count < minContainers {
				minContainers = count
				bestNode = node
			}
		}
	}
	
	return bestNode
}

// Pack strategy: fill nodes before moving to next one
func (s *Scheduler) schedulePack(nodes []*types.Node, container *types.Container) *types.Node {
	// Get container counts per node
	containerCounts := s.getContainerCountsPerNode()
	
	var bestNode *types.Node
	maxContainers := -1
	
	for _, node := range nodes {
		if s.hasCapacity(node, container) {
			count := containerCounts[node.ID]
			if count > maxContainers {
				maxContainers = count
				bestNode = node
			}
		}
	}
	
	return bestNode
}

// BinPack strategy: choose node with best resource fit
func (s *Scheduler) scheduleBinPack(nodes []*types.Node, container *types.Container) *types.Node {
	var bestNode *types.Node
	bestScore := float64(-1)
	
	for _, node := range nodes {
		if s.hasCapacity(node, container) {
			score := s.calculateBinPackScore(node, container)
			if score > bestScore {
				bestScore = score
				bestNode = node
			}
		}
	}
	
	return bestNode
}

func (s *Scheduler) calculateBinPackScore(node *types.Node, container *types.Container) float64 {
	if node.Resources == nil {
		return 0.5 // neutral score if no resource info
	}
	
	// Calculate resource utilization (higher is better for bin packing)
	cpuUtil := node.Resources.CPUUsagePercent / 100.0
	memUtil := float64(node.Resources.MemoryUsageMB) / float64(node.Resources.MemoryTotalMB)
	
	// Combine CPU and memory utilization
	return (cpuUtil + memUtil) / 2.0
}

func (s *Scheduler) hasCapacity(node *types.Node, container *types.Container) bool {
	if node.Resources == nil {
		return true // assume capacity if no resource info
	}
	
	// Check container count limit
	if node.Resources.ContainerCount >= node.Resources.MaxContainers {
		return false
	}
	
	// Check CPU usage (don't schedule if > 90% used)
	if node.Resources.CPUUsagePercent > 90.0 {
		return false
	}
	
	// Check memory usage (don't schedule if > 90% used)
	memoryUsagePercent := float64(node.Resources.MemoryUsageMB) / float64(node.Resources.MemoryTotalMB) * 100.0
	if memoryUsagePercent > 90.0 {
		return false
	}
	
	// Check port conflicts
	if s.hasPortConflicts(node, container.Ports) {
		return false
	}
	
	return true
}

func (s *Scheduler) hasPortConflicts(node *types.Node, ports []types.PortMapping) bool {
	// Get all containers on this node
	containers, err := s.storage.ListContainersByNode(node.ID)
	if err != nil {
		return false // assume no conflicts if we can't check
	}
	
	// Build map of used ports
	usedPorts := make(map[int]bool)
	for _, container := range containers {
		if container.Status == types.ContainerRunning || container.Status == types.ContainerPending {
			for _, port := range container.Ports {
				if port.HostPort > 0 {
					usedPorts[port.HostPort] = true
				}
			}
		}
	}
	
	// Check for conflicts
	for _, port := range ports {
		if port.HostPort > 0 && usedPorts[port.HostPort] {
			return true
		}
	}
	
	return false
}

func (s *Scheduler) getContainerCountsPerNode() map[string]int {
	counts := make(map[string]int)
	
	containers, err := s.storage.ListContainers()
	if err != nil {
		return counts
	}
	
	for _, container := range containers {
		if container.Status == types.ContainerRunning || container.Status == types.ContainerPending {
			counts[container.NodeID]++
		}
	}
	
	return counts
}