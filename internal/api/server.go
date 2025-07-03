package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

type Server struct {
	router *mux.Router
	port   string
	host   string
}

type Response struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

func NewServer(host, port string) *Server {
	s := &Server{
		router: mux.NewRouter(),
		port:   port,
		host:   host,
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
	api.HandleFunc("/health", s.handleHealth).Methods("GET")
}

func (s *Server) Start() error {
	addr := fmt.Sprintf("%s:%s", s.host, s.port)
	log.Printf("Starting API server on %s", addr)
	return http.ListenAndServe(addr, s.router)
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

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	s.sendJSON(w, Response{Success: true, Data: "OK"})
}

func (s *Server) sendJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}
