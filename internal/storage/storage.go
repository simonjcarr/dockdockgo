package storage

import (
	"dockdockgo/pkg/types"
	"encoding/json"
	"fmt"
	"time"

	"go.etcd.io/bbolt"
)

const (
	DeploymentsBucket = "deployments"
	ContainersBucket  = "containers"
	NodesBucket       = "nodes"
	ClusterBucket     = "cluster"
	RaftBucket        = "raft"
)

type Storage struct {
	db *bbolt.DB
}

func NewStorage(dbPath string) (*Storage, error) {
	const maxRetries = 3
	const retryDelay = 2 * time.Second

	var db *bbolt.DB
	var err error

	for i := 0; i < maxRetries; i++ {
		db, err = bbolt.Open(dbPath, 0600, &bbolt.Options{Timeout: 10 * time.Second})
		if err == nil {
			break
		}

		if i < maxRetries-1 {
			time.Sleep(retryDelay)
		}
	}

	if err != nil {
		return nil, fmt.Errorf("failed to open database after %d retries: %w", maxRetries, err)
	}

	storage := &Storage{db: db}

	// Create buckets if they don't exist
	if err := storage.initBuckets(); err != nil {
		return nil, fmt.Errorf("failed to initialize buckets: %w", err)
	}

	return storage, nil
}

func (s *Storage) initBuckets() error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		buckets := []string{
			DeploymentsBucket,
			ContainersBucket,
			NodesBucket,
			ClusterBucket,
			RaftBucket,
		}

		for _, bucket := range buckets {
			if _, err := tx.CreateBucketIfNotExists([]byte(bucket)); err != nil {
				return fmt.Errorf("failed to create bucket %s: %w", bucket, err)
			}
		}

		return nil
	})
}

func (s *Storage) Close() error {
	return s.db.Close()
}

// Deployment operations
func (s *Storage) SaveDeployment(deployment *types.Deployment) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(DeploymentsBucket))

		deployment.UpdatedAt = time.Now()
		data, err := json.Marshal(deployment)
		if err != nil {
			return fmt.Errorf("failed to marshal deployment: %w", err)
		}

		return bucket.Put([]byte(deployment.ID), data)
	})
}

func (s *Storage) GetDeployment(id string) (*types.Deployment, error) {
	var deployment *types.Deployment

	err := s.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(DeploymentsBucket))
		data := bucket.Get([]byte(id))

		if data == nil {
			return fmt.Errorf("deployment %s not found", id)
		}

		deployment = &types.Deployment{}
		if err := json.Unmarshal(data, deployment); err != nil {
			return fmt.Errorf("failed to unmarshal deployment: %w", err)
		}

		return nil
	})

	return deployment, err
}

func (s *Storage) GetDeploymentByName(name string) (*types.Deployment, error) {
	var deployment *types.Deployment

	err := s.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(DeploymentsBucket))
		cursor := bucket.Cursor()

		for k, v := cursor.First(); k != nil; k, v = cursor.Next() {
			var dep types.Deployment
			if err := json.Unmarshal(v, &dep); err != nil {
				continue
			}

			if dep.Name == name {
				deployment = &dep
				return nil
			}
		}

		return fmt.Errorf("deployment with name %s not found", name)
	})

	return deployment, err
}

func (s *Storage) ListDeployments() ([]*types.Deployment, error) {
	var deployments []*types.Deployment

	err := s.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(DeploymentsBucket))
		cursor := bucket.Cursor()

		for k, v := cursor.First(); k != nil; k, v = cursor.Next() {
			var deployment types.Deployment
			if err := json.Unmarshal(v, &deployment); err != nil {
				continue
			}
			deployments = append(deployments, &deployment)
		}

		return nil
	})

	return deployments, err
}

func (s *Storage) DeleteDeployment(id string) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(DeploymentsBucket))
		return bucket.Delete([]byte(id))
	})
}

// Container operations
func (s *Storage) SaveContainer(container *types.Container) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(ContainersBucket))

		data, err := json.Marshal(container)
		if err != nil {
			return fmt.Errorf("failed to marshal container: %w", err)
		}

		return bucket.Put([]byte(container.ID), data)
	})
}

func (s *Storage) GetContainer(id string) (*types.Container, error) {
	var container *types.Container

	err := s.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(ContainersBucket))
		data := bucket.Get([]byte(id))

		if data == nil {
			return fmt.Errorf("container %s not found", id)
		}

		container = &types.Container{}
		if err := json.Unmarshal(data, container); err != nil {
			return fmt.Errorf("failed to unmarshal container: %w", err)
		}

		return nil
	})

	return container, err
}

func (s *Storage) ListContainers() ([]*types.Container, error) {
	var containers []*types.Container

	err := s.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(ContainersBucket))
		cursor := bucket.Cursor()

		for k, v := cursor.First(); k != nil; k, v = cursor.Next() {
			var container types.Container
			if err := json.Unmarshal(v, &container); err != nil {
				continue
			}
			containers = append(containers, &container)
		}

		return nil
	})

	return containers, err
}

func (s *Storage) ListContainersByDeployment(deploymentID string) ([]*types.Container, error) {
	var containers []*types.Container

	err := s.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(ContainersBucket))
		cursor := bucket.Cursor()

		for k, v := cursor.First(); k != nil; k, v = cursor.Next() {
			var container types.Container
			if err := json.Unmarshal(v, &container); err != nil {
				continue
			}

			if container.DeploymentID == deploymentID {
				containers = append(containers, &container)
			}
		}

		return nil
	})

	return containers, err
}

func (s *Storage) ListContainersByNode(nodeID string) ([]*types.Container, error) {
	var containers []*types.Container

	err := s.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(ContainersBucket))
		cursor := bucket.Cursor()

		for k, v := cursor.First(); k != nil; k, v = cursor.Next() {
			var container types.Container
			if err := json.Unmarshal(v, &container); err != nil {
				continue
			}

			if container.NodeID == nodeID {
				containers = append(containers, &container)
			}
		}

		return nil
	})

	return containers, err
}

func (s *Storage) DeleteContainer(id string) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(ContainersBucket))
		return bucket.Delete([]byte(id))
	})
}

// Node operations
func (s *Storage) SaveNode(node *types.Node) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(NodesBucket))

		data, err := json.Marshal(node)
		if err != nil {
			return fmt.Errorf("failed to marshal node: %w", err)
		}

		return bucket.Put([]byte(node.ID), data)
	})
}

func (s *Storage) GetNode(id string) (*types.Node, error) {
	var node *types.Node

	err := s.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(NodesBucket))
		data := bucket.Get([]byte(id))

		if data == nil {
			return fmt.Errorf("node %s not found", id)
		}

		node = &types.Node{}
		if err := json.Unmarshal(data, node); err != nil {
			return fmt.Errorf("failed to unmarshal node: %w", err)
		}

		return nil
	})

	return node, err
}

func (s *Storage) ListNodes() ([]*types.Node, error) {
	var nodes []*types.Node

	err := s.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(NodesBucket))
		cursor := bucket.Cursor()

		for k, v := cursor.First(); k != nil; k, v = cursor.Next() {
			var node types.Node
			if err := json.Unmarshal(v, &node); err != nil {
				continue
			}
			nodes = append(nodes, &node)
		}

		return nil
	})

	return nodes, err
}

func (s *Storage) DeleteNode(id string) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(NodesBucket))
		return bucket.Delete([]byte(id))
	})
}

// Cluster state operations
func (s *Storage) SaveClusterState(state *types.ClusterState) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(ClusterBucket))

		state.UpdatedAt = time.Now()
		data, err := json.Marshal(state)
		if err != nil {
			return fmt.Errorf("failed to marshal cluster state: %w", err)
		}

		return bucket.Put([]byte("current"), data)
	})
}

func (s *Storage) GetClusterState() (*types.ClusterState, error) {
	var state *types.ClusterState

	err := s.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(ClusterBucket))
		data := bucket.Get([]byte("current"))

		if data == nil {
			// Return empty state if not found
			state = &types.ClusterState{
				Deployments: make(map[string]*types.Deployment),
				Containers:  make(map[string]*types.Container),
				Nodes:       make(map[string]*types.Node),
				UpdatedAt:   time.Now(),
			}
			return nil
		}

		state = &types.ClusterState{}
		if err := json.Unmarshal(data, state); err != nil {
			return fmt.Errorf("failed to unmarshal cluster state: %w", err)
		}

		return nil
	})

	return state, err
}
