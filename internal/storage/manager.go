package storage

import (
	"fmt"
	"sync"
	"time"
)

// ConnectionManager manages a single database connection to prevent lock contention
type ConnectionManager struct {
	storage *Storage
	mutex   sync.RWMutex
}

var (
	instance *ConnectionManager
	once     sync.Once
)

// GetInstance returns the singleton instance of ConnectionManager
func GetInstance() *ConnectionManager {
	once.Do(func() {
		instance = &ConnectionManager{}
	})
	return instance
}

// GetStorage returns the shared storage instance, creating it if necessary
func (cm *ConnectionManager) GetStorage() (*Storage, error) {
	cm.mutex.RLock()
	if cm.storage != nil {
		defer cm.mutex.RUnlock()
		return cm.storage, nil
	}
	cm.mutex.RUnlock()

	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	// Double-check pattern - another goroutine might have created it
	if cm.storage != nil {
		return cm.storage, nil
	}

	// Create new storage instance
	dbPath := GetDatabasePath()
	storage, err := NewStorage(dbPath)
	if err != nil {
		return nil, err
	}

	cm.storage = storage
	return cm.storage, nil
}

// Close closes the database connection
func (cm *ConnectionManager) Close() error {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	if cm.storage != nil {
		err := cm.storage.Close()
		cm.storage = nil
		return err
	}
	return nil
}

// NewManagedStorage creates a new storage instance using the connection manager
func NewManagedStorage() (*Storage, error) {
	return GetInstance().GetStorage()
}

// IsHealthy checks if the database connection is healthy
func (cm *ConnectionManager) IsHealthy() bool {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	if cm.storage == nil {
		return false
	}

	// Try to perform a quick read operation
	_, err := cm.storage.GetClusterState()
	return err == nil
}

// Reset resets the connection manager (useful for testing)
func (cm *ConnectionManager) Reset() error {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	if cm.storage != nil {
		err := cm.storage.Close()
		cm.storage = nil
		if err != nil {
			return err
		}
	}

	// Reset the singleton
	instance = &ConnectionManager{}
	return nil
}

// WaitForConnection waits for a database connection to become available
func (cm *ConnectionManager) WaitForConnection(timeout time.Duration) (*Storage, error) {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		storage, err := cm.GetStorage()
		if err == nil {
			return storage, nil
		}

		// Wait a bit before retrying
		time.Sleep(100 * time.Millisecond)
	}

	return nil, &ConnectionTimeoutError{Timeout: timeout}
}

// ConnectionTimeoutError represents a timeout error when waiting for database connection
type ConnectionTimeoutError struct {
	Timeout time.Duration
}

func (e *ConnectionTimeoutError) Error() string {
	return fmt.Sprintf("database connection timeout after %v", e.Timeout)
}
