package api

import (
	"bytes"
	"dockdockgo/pkg/types"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
}

func NewClient(host, port string) *Client {
	return &Client{
		baseURL: fmt.Sprintf("http://%s:%s/api/v1", host, port),
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (c *Client) IsServiceRunning() bool {
	resp, err := c.httpClient.Get(c.baseURL + "/health")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

func (c *Client) ClusterInit(advertiseAddr string) (*ClusterInitResponse, error) {
	req := ClusterInitRequest{
		AdvertiseAddr: advertiseAddr,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := c.httpClient.Post(c.baseURL+"/cluster/init", "application/json", bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to API: %w", err)
	}
	defer resp.Body.Close()

	var response Response
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if !response.Success {
		return nil, fmt.Errorf("API error: %s", response.Error)
	}

	// Convert response.Data to ClusterInitResponse
	dataBytes, err := json.Marshal(response.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal response data: %w", err)
	}

	var initResponse ClusterInitResponse
	if err := json.Unmarshal(dataBytes, &initResponse); err != nil {
		return nil, fmt.Errorf("failed to unmarshal cluster init response: %w", err)
	}

	return &initResponse, nil
}

func (c *Client) ClusterJoin(masterAddr, role string) (*ClusterJoinResponse, error) {
	req := ClusterJoinRequest{
		MasterAddr: masterAddr,
		Role:       role,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := c.httpClient.Post(c.baseURL+"/cluster/join", "application/json", bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to API: %w", err)
	}
	defer resp.Body.Close()

	var response Response
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if !response.Success {
		return nil, fmt.Errorf("API error: %s", response.Error)
	}

	// Convert response.Data to ClusterJoinResponse
	dataBytes, err := json.Marshal(response.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal response data: %w", err)
	}

	var joinResponse ClusterJoinResponse
	if err := json.Unmarshal(dataBytes, &joinResponse); err != nil {
		return nil, fmt.Errorf("failed to unmarshal cluster join response: %w", err)
	}

	return &joinResponse, nil
}

func (c *Client) ListNodes() ([]*types.Node, error) {
	resp, err := c.httpClient.Get(c.baseURL + "/nodes")
	if err != nil {
		return nil, fmt.Errorf("failed to connect to API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var nodes []*types.Node
	if err := json.NewDecoder(resp.Body).Decode(&nodes); err != nil {
		return nil, fmt.Errorf("failed to decode nodes: %w", err)
	}

	return nodes, nil
}

func (c *Client) CreateDeployment(spec *types.DeploymentSpec) (*types.Deployment, error) {
	body, err := json.Marshal(spec)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal deployment spec: %w", err)
	}

	resp, err := c.httpClient.Post(c.baseURL+"/deployments", "application/json", bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to API: %w", err)
	}
	defer resp.Body.Close()

	var response Response
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if !response.Success {
		return nil, fmt.Errorf("API error: %s", response.Error)
	}

	// Convert response.Data to Deployment
	dataBytes, err := json.Marshal(response.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal response data: %w", err)
	}

	var deployment types.Deployment
	if err := json.Unmarshal(dataBytes, &deployment); err != nil {
		return nil, fmt.Errorf("failed to unmarshal deployment: %w", err)
	}

	return &deployment, nil
}

func (c *Client) ListDeployments() ([]*types.Deployment, error) {
	resp, err := c.httpClient.Get(c.baseURL + "/deployments")
	if err != nil {
		return nil, fmt.Errorf("failed to connect to API: %w", err)
	}
	defer resp.Body.Close()

	var response Response
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if !response.Success {
		return nil, fmt.Errorf("API error: %s", response.Error)
	}

	// Convert response.Data to []*Deployment
	dataBytes, err := json.Marshal(response.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal response data: %w", err)
	}

	var deployments []*types.Deployment
	if err := json.Unmarshal(dataBytes, &deployments); err != nil {
		return nil, fmt.Errorf("failed to unmarshal deployments: %w", err)
	}

	return deployments, nil
}

func (c *Client) GetDeployment(id string) (*types.Deployment, error) {
	resp, err := c.httpClient.Get(c.baseURL + "/deployments/" + id)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to API: %w", err)
	}
	defer resp.Body.Close()

	var response Response
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if !response.Success {
		return nil, fmt.Errorf("API error: %s", response.Error)
	}

	// Convert response.Data to Deployment
	dataBytes, err := json.Marshal(response.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal response data: %w", err)
	}

	var deployment types.Deployment
	if err := json.Unmarshal(dataBytes, &deployment); err != nil {
		return nil, fmt.Errorf("failed to unmarshal deployment: %w", err)
	}

	return &deployment, nil
}

func (c *Client) DeleteDeployment(id string) error {
	req, err := http.NewRequest("DELETE", c.baseURL+"/deployments/"+id, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to connect to API: %w", err)
	}
	defer resp.Body.Close()

	var response Response
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	if !response.Success {
		return fmt.Errorf("API error: %s", response.Error)
	}

	return nil
}
