package api

import (
	"bytes"
	"dockdockgo/internal/cluster"
	"dockdockgo/internal/storage"
	"dockdockgo/pkg/types"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

type Server struct {
	router  *mux.Router
	port    string
	host    string
	storage *storage.Storage
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
	api.HandleFunc("/health", s.handleHealth).Methods("GET")
}

func (s *Server) Start() error {
	addr := fmt.Sprintf("%s:%s", s.host, s.port)
	log.Printf("Starting API server on %s", addr)
	return http.ListenAndServe(addr, s.router)
}

func (s *Server) Close() error {
	if s.storage != nil {
		return s.storage.Close()
	}
	return nil
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

	// Use advertise-addr or hostname as IP
	ipAddress := req.AdvertiseAddr
	if ipAddress == "" {
		ipAddress = hostname
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
		existingNode.Role = req.Role
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
		thisNode = &types.Node{
			ID:            uuid.New().String(),
			Hostname:      hostname,
			IPAddress:     hostname, // TODO: Get actual IP
			Port:          8443,
			Status:        types.NodeOnline,
			Role:          req.Role,
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
		Role:         thisNode.Role,
		Master:       req.MasterAddr,
		ClusterNodes: len(finalNodes),
	}

	s.sendJSON(w, Response{Success: true, Data: response})
}

func (s *Server) fetchClusterStateFromMaster(masterAddr string) ([]*types.Node, error) {
	// Try to fetch cluster state from master node's API
	url := fmt.Sprintf("http://%s:8080/api/v1/nodes", masterAddr)

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
	url := fmt.Sprintf("http://%s:8080/api/v1/nodes/register", masterAddr)
	
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
