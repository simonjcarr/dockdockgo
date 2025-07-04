package api

import (
	"bytes"
	"context"
	"dockdockgo/internal/cluster"
	"dockdockgo/internal/storage"
	"dockdockgo/pkg/types"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

type Server struct {
	router  *mux.Router
	port    string
	host    string
	storage *storage.Storage
	server  *http.Server
}

type Response struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

type ClusterInitRequest struct {
	AdvertiseAddr string `json:"advertise_addr,omitempty"`
}

type ClusterJoinRequest struct {
	MasterAddr string `json:"master_addr"`
	Role       string `json:"role,omitempty"`
}

type ClusterInitResponse struct {
	NodeID      string `json:"node_id"`
	Hostname    string `json:"hostname"`
	IPAddress   string `json:"ip_address"`
	Port        int    `json:"port"`
	JoinCommand string `json:"join_command"`
}

type ClusterJoinResponse struct {
	NodeID       string `json:"node_id"`
	Role         string `json:"role"`
	Master       string `json:"master"`
	ClusterNodes int    `json:"cluster_nodes"`
}

func NewServer(host, port string) *Server {
	storage, err := storage.NewDefaultStorage()
	if err != nil {
		log.Fatalf("Failed to initialize storage: %v", err)
	}

	s := &Server{
		router:  mux.NewRouter(),
		port:    port,
		host:    host,
		storage: storage,
	}
	s.setupRoutes()
	return s
}

func (s *Server) setupRoutes() {
	api := s.router.PathPrefix("/api/v1").Subrouter()

	api.HandleFunc("/containers", s.handleContainers).Methods("GET", "POST")
	api.HandleFunc("/containers/{id}", s.handleContainer).Methods("GET", "DELETE")
	api.HandleFunc("/deployments", s.handleDeployments).Methods("GET", "POST")
	api.HandleFunc("/deployments/{id}", s.handleDeployment).Methods("GET", "DELETE")
	api.HandleFunc("/images/search", s.handleImageSearch).Methods("GET")
	api.HandleFunc("/compose", s.handleCompose).Methods("POST")
	api.HandleFunc("/nodes", s.handleNodes).Methods("GET")
	api.HandleFunc("/nodes/register", s.handleNodeRegister).Methods("POST")
	api.HandleFunc("/cluster/init", s.handleClusterInit).Methods("POST")
	api.HandleFunc("/cluster/join", s.handleClusterJoin).Methods("POST")
	api.HandleFunc("/cluster/sync", s.handleClusterSync).Methods("GET", "POST")
	api.HandleFunc("/health", s.handleHealth).Methods("GET")
}

func (s *Server) Start() error {
	addr := fmt.Sprintf("%s:%s", s.host, s.port)
	log.Printf("Starting API server on %s", addr)

	// Create HTTP server
	s.server = &http.Server{
		Addr:    addr,
		Handler: s.router,
	}

	// Setup graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan

		log.Println("Shutting down API server...")

		// Create a context with timeout for shutdown
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// Shutdown the server gracefully
		if err := s.server.Shutdown(ctx); err != nil {
			log.Printf("Error during server shutdown: %v", err)
		}

		// Close database connection
		if err := s.Close(); err != nil {
			log.Printf("Error closing database: %v", err)
		}

		log.Println("API server stopped")
	}()

	// Start the server
	if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("failed to start server: %w", err)
	}

	return nil
}

func (s *Server) Close() error {
	// Close the connection manager instead of individual storage
	return storage.GetInstance().Close()
}

func (s *Server) handleContainers(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		s.sendJSON(w, Response{Success: true, Data: []string{"container1", "container2"}})
	case "POST":
		s.sendJSON(w, Response{Success: true, Data: "Container created"})
	}
}

func (s *Server) handleContainer(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	switch r.Method {
	case "GET":
		s.sendJSON(w, Response{Success: true, Data: fmt.Sprintf("Container: %s", id)})
	case "DELETE":
		s.sendJSON(w, Response{Success: true, Data: fmt.Sprintf("Container %s deleted", id)})
	}
}

func (s *Server) handleImageSearch(w http.ResponseWriter, r *http.Request) {
	term := r.URL.Query().Get("q")
	s.sendJSON(w, Response{Success: true, Data: fmt.Sprintf("Search results for: %s", term)})
}

func (s *Server) handleCompose(w http.ResponseWriter, r *http.Request) {
	s.sendJSON(w, Response{Success: true, Data: "Compose deployment started"})
}

func (s *Server) handleNodes(w http.ResponseWriter, r *http.Request) {
	nodes, err := s.storage.ListNodes()
	if err != nil {
		s.sendJSON(w, Response{Success: false, Error: fmt.Sprintf("Failed to list nodes: %v", err)})
		return
	}

	// Return nodes directly for cluster join functionality
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(nodes)
}

func (s *Server) handleClusterInit(w http.ResponseWriter, r *http.Request) {
	var req ClusterInitRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.sendJSON(w, Response{Success: false, Error: "Invalid request format"})
		return
	}

	// Get hostname and IP
	hostname, err := os.Hostname()
	if err != nil {
		s.sendJSON(w, Response{Success: false, Error: fmt.Sprintf("Failed to get hostname: %v", err)})
		return
	}

	// Use advertise-addr or auto-detect IP address
	ipAddress := req.AdvertiseAddr
	if ipAddress == "" {
		// Try to get actual IP address instead of hostname
		if detectedIP := s.getLocalIPAddress(); detectedIP != "" {
			ipAddress = detectedIP
		} else {
			// Fallback to hostname if IP detection fails
			ipAddress = hostname
		}
	}

	// Check if cluster already exists
	nodes, err := s.storage.ListNodes()
	if err != nil {
		s.sendJSON(w, Response{Success: false, Error: fmt.Sprintf("Failed to check existing cluster: %v", err)})
		return
	}

	if len(nodes) > 0 {
		s.sendJSON(w, Response{Success: false, Error: fmt.Sprintf("Cluster already initialized. Found %d existing nodes", len(nodes))})
		return
	}

	// Create master node
	masterNode := &types.Node{
		ID:            uuid.New().String(),
		Hostname:      hostname,
		IPAddress:     ipAddress,
		Port:          8443,
		Status:        types.NodeOnline,
		Role:          "master",
		Version:       "1.0.0", // TODO: Get actual version
		Labels:        map[string]string{"cluster.role": "master"},
		LastHeartbeat: time.Now(),
		JoinedAt:      time.Now(),
	}

	// Save master node
	if err := s.storage.SaveNode(masterNode); err != nil {
		s.sendJSON(w, Response{Success: false, Error: fmt.Sprintf("Failed to initialize cluster: %v", err)})
		return
	}

	response := ClusterInitResponse{
		NodeID:      masterNode.ID,
		Hostname:    masterNode.Hostname,
		IPAddress:   masterNode.IPAddress,
		Port:        masterNode.Port,
		JoinCommand: fmt.Sprintf("dockdockgo cluster join %s", ipAddress),
	}

	s.sendJSON(w, Response{Success: true, Data: response})
}

func (s *Server) handleClusterJoin(w http.ResponseWriter, r *http.Request) {
	var req ClusterJoinRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.sendJSON(w, Response{Success: false, Error: "Invalid request format"})
		return
	}

	if req.MasterAddr == "" {
		s.sendJSON(w, Response{Success: false, Error: "Master address is required"})
		return
	}

	if req.Role == "" {
		req.Role = "worker"
	}

	// Get hostname
	hostname, err := os.Hostname()
	if err != nil {
		s.sendJSON(w, Response{Success: false, Error: fmt.Sprintf("Failed to get hostname: %v", err)})
		return
	}

	// Check if this node already exists in the cluster
	existingNodes, err := s.storage.ListNodes()
	if err != nil {
		s.sendJSON(w, Response{Success: false, Error: fmt.Sprintf("Failed to check existing nodes: %v", err)})
		return
	}

	// Check for duplicate by hostname
	var existingNode *types.Node
	for _, node := range existingNodes {
		if node.Hostname == hostname {
			existingNode = node
			break
		}
	}

	var thisNode *types.Node
	if existingNode != nil {
		// Update existing node
		existingNode.Role = types.NodeRole(req.Role)
		existingNode.Status = types.NodeOnline
		existingNode.LastHeartbeat = time.Now()
		if err := s.storage.SaveNode(existingNode); err != nil {
			s.sendJSON(w, Response{Success: false, Error: fmt.Sprintf("Failed to update this node: %v", err)})
			return
		}
		thisNode = existingNode
	} else {
		// Try to fetch cluster state from master
		clusterState, err := s.fetchClusterStateFromMaster(req.MasterAddr)
		if err != nil {
			s.sendJSON(w, Response{Success: false, Error: fmt.Sprintf("Failed to connect to master node: %v", err)})
			return
		}

		// Save all nodes from cluster state
		for _, node := range clusterState {
			if err := s.storage.SaveNode(node); err != nil {
				log.Printf("Warning: Failed to save node %s: %v", node.Hostname, err)
			}
		}

		// Create this node and add it to the cluster
		nodeIPAddress := hostname
		if detectedIP := s.getLocalIPAddress(); detectedIP != "" {
			nodeIPAddress = detectedIP
		}

		thisNode = &types.Node{
			ID:            uuid.New().String(),
			Hostname:      hostname,
			IPAddress:     nodeIPAddress,
			Port:          8443,
			Status:        types.NodeOnline,
			Role:          types.NodeRole(req.Role),
			Version:       "1.0.0", // TODO: Get actual version
			Labels:        map[string]string{"cluster.role": req.Role},
			LastHeartbeat: time.Now(),
			JoinedAt:      time.Now(),
		}

		if err := s.storage.SaveNode(thisNode); err != nil {
			s.sendJSON(w, Response{Success: false, Error: fmt.Sprintf("Failed to save this node: %v", err)})
			return
		}

		// Register this node with the master
		if err := s.registerWithMaster(req.MasterAddr, thisNode); err != nil {
			log.Printf("Warning: Failed to register with master: %v", err)
			// Don't fail the join operation, just log the warning
		}
	}

	// Get final cluster count
	finalNodes, _ := s.storage.ListNodes()

	response := ClusterJoinResponse{
		NodeID:       thisNode.ID,
		Role:         string(thisNode.Role),
		Master:       req.MasterAddr,
		ClusterNodes: len(finalNodes),
	}

	s.sendJSON(w, Response{Success: true, Data: response})
}

func (s *Server) fetchClusterStateFromMaster(masterAddr string) ([]*types.Node, error) {
	// Try to fetch cluster state from master node's API
	// Use the same port as our server is running on
	url := fmt.Sprintf("http://%s:%s/api/v1/nodes", masterAddr, s.port)

	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to master API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("master API returned status %d", resp.StatusCode)
	}

	var nodes []*types.Node
	if err := json.NewDecoder(resp.Body).Decode(&nodes); err != nil {
		return nil, fmt.Errorf("failed to parse cluster state: %w", err)
	}

	return nodes, nil
}

func (s *Server) handleNodeRegister(w http.ResponseWriter, r *http.Request) {
	var node types.Node
	if err := json.NewDecoder(r.Body).Decode(&node); err != nil {
		s.sendJSON(w, Response{Success: false, Error: "Invalid node data"})
		return
	}

	// Check if node already exists
	existingNodes, err := s.storage.ListNodes()
	if err != nil {
		s.sendJSON(w, Response{Success: false, Error: fmt.Sprintf("Failed to check existing nodes: %v", err)})
		return
	}

	var existingNode *types.Node
	for _, existing := range existingNodes {
		if existing.Hostname == node.Hostname {
			existingNode = existing
			break
		}
	}

	if existingNode != nil {
		// Update existing node
		existingNode.Role = node.Role
		existingNode.Status = node.Status
		existingNode.IPAddress = node.IPAddress
		existingNode.Port = node.Port
		existingNode.LastHeartbeat = time.Now()

		if err := s.storage.SaveNode(existingNode); err != nil {
			s.sendJSON(w, Response{Success: false, Error: fmt.Sprintf("Failed to update node: %v", err)})
			return
		}
		s.sendJSON(w, Response{Success: true, Data: "Node updated successfully"})
	} else {
		// Register new node
		node.LastHeartbeat = time.Now()
		if err := s.storage.SaveNode(&node); err != nil {
			s.sendJSON(w, Response{Success: false, Error: fmt.Sprintf("Failed to register node: %v", err)})
			return
		}
		s.sendJSON(w, Response{Success: true, Data: "Node registered successfully"})
	}
}

func (s *Server) registerWithMaster(masterAddr string, node *types.Node) error {
	url := fmt.Sprintf("http://%s:%s/api/v1/nodes/register", masterAddr, s.port)

	nodeData, err := json.Marshal(node)
	if err != nil {
		return fmt.Errorf("failed to marshal node data: %w", err)
	}

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(nodeData))
	if err != nil {
		return fmt.Errorf("failed to connect to master: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("master returned status %d", resp.StatusCode)
	}

	return nil
}

func (s *Server) handleDeployments(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		deployments, err := s.storage.ListDeployments()
		if err != nil {
			s.sendJSON(w, Response{Success: false, Error: fmt.Sprintf("Failed to list deployments: %v", err)})
			return
		}
		s.sendJSON(w, Response{Success: true, Data: deployments})
	case "POST":
		var spec types.DeploymentSpec
		if err := json.NewDecoder(r.Body).Decode(&spec); err != nil {
			s.sendJSON(w, Response{Success: false, Error: "Invalid request format"})
			return
		}

		// Create deployment using the cluster deployment manager
		deploymentManager := s.getDeploymentManager()
		deployment, err := deploymentManager.CreateDeployment(&spec)
		if err != nil {
			s.sendJSON(w, Response{Success: false, Error: fmt.Sprintf("Failed to create deployment: %v", err)})
			return
		}

		s.sendJSON(w, Response{Success: true, Data: deployment})
	}
}

func (s *Server) handleDeployment(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	switch r.Method {
	case "GET":
		deployment, err := s.storage.GetDeployment(id)
		if err != nil {
			s.sendJSON(w, Response{Success: false, Error: fmt.Sprintf("Failed to get deployment: %v", err)})
			return
		}
		s.sendJSON(w, Response{Success: true, Data: deployment})
	case "DELETE":
		deploymentManager := s.getDeploymentManager()
		err := deploymentManager.DeleteDeployment(id)
		if err != nil {
			s.sendJSON(w, Response{Success: false, Error: fmt.Sprintf("Failed to delete deployment: %v", err)})
			return
		}
		s.sendJSON(w, Response{Success: true, Data: fmt.Sprintf("Deployment %s deleted", id)})
	}
}

func (s *Server) getDeploymentManager() *cluster.DeploymentManager {
	return cluster.NewDeploymentManager(s.storage)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	s.sendJSON(w, Response{Success: true, Data: "OK"})
}

func (s *Server) sendJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func (s *Server) sendError(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(Response{Success: false, Error: message})
}

// getLocalIPAddress returns the first non-loopback IP address
func (s *Server) getLocalIPAddress() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ""
	}

	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}
	return ""
}

// handleClusterSync handles cluster state synchronization
func (s *Server) handleClusterSync(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		s.handleGetClusterSync(w, r)
	case "POST":
		s.handlePostClusterSync(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleGetClusterSync returns current cluster state for synchronization
func (s *Server) handleGetClusterSync(w http.ResponseWriter, r *http.Request) {
	// Get current cluster state
	deployments, err := s.storage.ListDeployments()
	if err != nil {
		s.sendError(w, fmt.Sprintf("Failed to get deployments: %v", err), http.StatusInternalServerError)
		return
	}

	containers, err := s.storage.ListContainers()
	if err != nil {
		s.sendError(w, fmt.Sprintf("Failed to get containers: %v", err), http.StatusInternalServerError)
		return
	}

	nodes, err := s.storage.ListNodes()
	if err != nil {
		s.sendError(w, fmt.Sprintf("Failed to get nodes: %v", err), http.StatusInternalServerError)
		return
	}

	syncData := map[string]interface{}{
		"timestamp":   time.Now(),
		"deployments": deployments,
		"containers":  containers,
		"nodes":       nodes,
	}

	s.sendJSON(w, Response{Success: true, Data: syncData})
}

// handlePostClusterSync receives and applies cluster state from master
func (s *Server) handlePostClusterSync(w http.ResponseWriter, r *http.Request) {
	var syncData struct {
		Timestamp   time.Time           `json:"timestamp"`
		Deployments []*types.Deployment `json:"deployments"`
		Containers  []*types.Container  `json:"containers"`
		Nodes       []*types.Node       `json:"nodes"`
	}

	if err := json.NewDecoder(r.Body).Decode(&syncData); err != nil {
		s.sendError(w, fmt.Sprintf("Failed to decode sync data: %v", err), http.StatusBadRequest)
		return
	}

	// Apply the sync data
	for _, deployment := range syncData.Deployments {
		if err := s.storage.SaveDeployment(deployment); err != nil {
			log.Printf("Failed to save deployment %s: %v", deployment.Name, err)
		}
	}

	for _, container := range syncData.Containers {
		if err := s.storage.SaveContainer(container); err != nil {
			log.Printf("Failed to save container %s: %v", container.Name, err)
		}
	}

	for _, node := range syncData.Nodes {
		if err := s.storage.SaveNode(node); err != nil {
			log.Printf("Failed to save node %s: %v", node.Hostname, err)
		}
	}

	log.Printf("Applied sync data with %d deployments, %d containers, %d nodes",
		len(syncData.Deployments), len(syncData.Containers), len(syncData.Nodes))

	s.sendJSON(w, Response{Success: true, Data: "Sync data applied successfully"})
}
