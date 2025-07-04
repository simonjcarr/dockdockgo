package dockdockgo

import (
	"dockdockgo/internal/storage"
	"dockdockgo/pkg/types"
	"os"
)

// getAPIEndpoint returns the API host and port to connect to
// First tries environment variables, then tries to discover master node,
// finally falls back to localhost
func getAPIEndpoint() (string, string) {
	// Check environment variables first
	if host := os.Getenv("DOCKDOCKGO_MASTER_HOST"); host != "" {
		port := os.Getenv("DOCKDOCKGO_MASTER_PORT")
		if port == "" {
			port = "8080"
		}
		return host, port
	}

	// Try to discover master node from local storage
	if masterHost := discoverMasterNode(); masterHost != "" {
		return masterHost, "8080"
	}

	// Fallback to localhost
	return "localhost", "8080"
}

// discoverMasterNode tries to find the master node from local storage
func discoverMasterNode() string {
	storage, err := storage.NewDefaultStorage()
	if err != nil {
		return ""
	}
	defer storage.Close()

	nodes, err := storage.ListNodes()
	if err != nil {
		return ""
	}

	for _, node := range nodes {
		if node.Role == "master" && node.Status == types.NodeOnline {
			return node.IPAddress
		}
	}

	return ""
}
