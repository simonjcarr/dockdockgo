package storage

import (
	"os"
	"path/filepath"
	"strings"
)

// GetDatabasePath returns the appropriate database path based on the environment
func GetDatabasePath() string {
	if IsProduction() {
		return getProductionDatabasePath()
	}
	return getDevelopmentDatabasePath()
}

// IsProduction determines if the application is running in production
func IsProduction() bool {
	// Check if binary is installed in system paths
	executable, err := os.Executable()
	if err == nil {
		if strings.HasPrefix(executable, "/usr/") ||
			strings.HasPrefix(executable, "/opt/") {
			return true
		}
	}

	// Check environment variable
	if env := os.Getenv("DOCKDOCKGO_ENV"); env == "production" {
		return true
	}

	return false
}

// getProductionDatabasePath returns the FHS-compliant production database path
func getProductionDatabasePath() string {
	dbDir := "/var/lib/dockdockgo"

	// Ensure directory exists
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		// Fallback to current directory if we can't create /var/lib/dockdockgo
		return "./dockdockgo.db"
	}

	return filepath.Join(dbDir, "dockdockgo.db")
}

// getDevelopmentDatabasePath returns the development database path
func getDevelopmentDatabasePath() string {
	return "./dockdockgo.db"
}

// NewDefaultStorage creates a new storage instance with the appropriate database path
func NewDefaultStorage() (*Storage, error) {
	dbPath := GetDatabasePath()
	return NewStorage(dbPath)
}
