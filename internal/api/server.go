package api

import (
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
	NodeID        string `json:"node_id"`
	Hostname      string `json:"hostname"`
	IPAddress     string `json:"ip_address"`
	Port          int    `json:"port"`
	JoinCommand   string `json:"join_command"`
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
	api.HandleFunc("/images/search", s.handleImageSearch).Methods("GET")
	api.HandleFunc("/compose", s.handleCompose).Methods("POST")
	api.HandleFunc("/nodes", s.handleNodes).Methods("GET")
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
	thisNode := &types.Node{
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

	response := ClusterJoinResponse{
		NodeID:       thisNode.ID,
		Role:         thisNode.Role,
		Master:       req.MasterAddr,
		ClusterNodes: len(clusterState) + 1,
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

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	s.sendJSON(w, Response{Success: true, Data: "OK"})
}

func (s *Server) sendJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}
