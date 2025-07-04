package cluster

import (
	"bytes"
	"context"
	"dockdockgo/internal/storage"
	"dockdockgo/pkg/types"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
)

// SyncManager handles cluster state synchronization
type SyncManager struct {
	storage   *storage.Storage
	currentNode *types.Node
	isRunning bool
	stopChan  chan struct{}
	mu        sync.RWMutex
}

// NewSyncManager creates a new cluster sync manager
func NewSyncManager(storage *storage.Storage, currentNode *types.Node) *SyncManager {
	return &SyncManager{
		storage:     storage,
		currentNode: currentNode,
		stopChan:    make(chan struct{}),
	}
}

// Start begins the cluster synchronization process
func (sm *SyncManager) Start() error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if sm.isRunning {
		return fmt.Errorf("sync manager is already running")
	}

	sm.isRunning = true
	go sm.syncLoop()
	
	log.Println("Cluster sync manager started")
	return nil
}

// Stop stops the cluster synchronization process
func (sm *SyncManager) Stop() error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if !sm.isRunning {
		return nil
	}

	close(sm.stopChan)
	sm.isRunning = false
	
	log.Println("Cluster sync manager stopped")
	return nil
}

// syncLoop runs the periodic synchronization
func (sm *SyncManager) syncLoop() {
	ticker := time.NewTicker(30 * time.Second) // Sync every 30 seconds
	defer ticker.Stop()

	for {
		select {
		case <-sm.stopChan:
			return
		case <-ticker.C:
			if err := sm.performSync(); err != nil {
				log.Printf("Sync error: %v", err)
			}
		}
	}
}

// performSync synchronizes cluster state
func (sm *SyncManager) performSync() error {
	// Get all nodes in the cluster
	nodes, err := sm.storage.ListNodes()
	if err != nil {
		return fmt.Errorf("failed to list nodes: %w", err)
	}

	// If this is the master node, push state to workers
	if sm.currentNode.Role == types.NodeRoleMaster {
		return sm.pushStateToWorkers(nodes)
	}

	// If this is a worker node, pull state from master
	return sm.pullStateFromMaster(nodes)
}

// pushStateToWorkers pushes cluster state to all worker nodes
func (sm *SyncManager) pushStateToWorkers(nodes []*types.Node) error {
	// Get current cluster state
	deployments, err := sm.storage.ListDeployments()
	if err != nil {
		return fmt.Errorf("failed to get deployments: %w", err)
	}

	containers, err := sm.storage.ListContainers()
	if err != nil {
		return fmt.Errorf("failed to get containers: %w", err)
	}

	// Create sync payload
	syncData := &ClusterSyncData{
		Timestamp:   time.Now(),
		Deployments: deployments,
		Containers:  containers,
		Nodes:       nodes,
	}

	// Send to all worker nodes
	for _, node := range nodes {
		if node.Role == types.NodeRoleWorker && node.Status == types.NodeStatusOnline {
			if err := sm.sendSyncData(node, syncData); err != nil {
				log.Printf("Failed to sync with worker %s: %v", node.Hostname, err)
			}
		}
	}

	return nil
}

// pullStateFromMaster pulls cluster state from the master node
func (sm *SyncManager) pullStateFromMaster(nodes []*types.Node) error {
	// Find master node
	var masterNode *types.Node
	for _, node := range nodes {
		if node.Role == types.NodeRoleMaster {
			masterNode = node
			break
		}
	}

	if masterNode == nil {
		return fmt.Errorf("no master node found")
	}

	// Request sync data from master
	syncData, err := sm.requestSyncData(masterNode)
	if err != nil {
		return fmt.Errorf("failed to request sync data from master: %w", err)
	}

	// Update local storage with master's state
	return sm.applySyncData(syncData)
}

// sendSyncData sends sync data to a specific node
func (sm *SyncManager) sendSyncData(node *types.Node, data *ClusterSyncData) error {
	url := fmt.Sprintf("http://%s:%d/api/v1/cluster/sync", node.IPAddress, node.Port)
	
	payload, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal sync data: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(payload))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send sync data: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("sync request failed with status %d", resp.StatusCode)
	}

	return nil
}

// requestSyncData requests sync data from master node
func (sm *SyncManager) requestSyncData(masterNode *types.Node) (*ClusterSyncData, error) {
	url := fmt.Sprintf("http://%s:%d/api/v1/cluster/sync", masterNode.IPAddress, masterNode.Port)
	
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to request sync data: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("sync request failed with status %d", resp.StatusCode)
	}

	var syncData ClusterSyncData
	if err := json.NewDecoder(resp.Body).Decode(&syncData); err != nil {
		return nil, fmt.Errorf("failed to decode sync data: %w", err)
	}

	return &syncData, nil
}

// applySyncData applies received sync data to local storage
func (sm *SyncManager) applySyncData(data *ClusterSyncData) error {
	// Update deployments
	for _, deployment := range data.Deployments {
		if err := sm.storage.SaveDeployment(deployment); err != nil {
			log.Printf("Failed to save deployment %s: %v", deployment.Name, err)
		}
	}

	// Update containers
	for _, container := range data.Containers {
		if err := sm.storage.SaveContainer(container); err != nil {
			log.Printf("Failed to save container %s: %v", container.Name, err)
		}
	}

	// Update nodes
	for _, node := range data.Nodes {
		if err := sm.storage.SaveNode(node); err != nil {
			log.Printf("Failed to save node %s: %v", node.Hostname, err)
		}
	}

	log.Printf("Applied sync data with %d deployments, %d containers, %d nodes", 
		len(data.Deployments), len(data.Containers), len(data.Nodes))
	
	return nil
}

// ClusterSyncData represents the data synchronized between cluster nodes
type ClusterSyncData struct {
	Timestamp   time.Time           `json:"timestamp"`
	Deployments []*types.Deployment `json:"deployments"`
	Containers  []*types.Container  `json:"containers"`
	Nodes       []*types.Node       `json:"nodes"`
}

// SyncOnDemand triggers an immediate sync
func (sm *SyncManager) SyncOnDemand() error {
	return sm.performSync()
}