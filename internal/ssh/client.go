package ssh

import (
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

type Config struct {
	Host     string
	Port     string
	User     string
	Password string
	KeyPath  string
	Timeout  time.Duration
}

type Client struct {
	config     *Config
	connection *ssh.Client
}

func NewClient(config *Config) *Client {
	if config.Port == "" {
		config.Port = "22"
	}
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}
	return &Client{config: config}
}

func (c *Client) Connect() error {
	var authMethods []ssh.AuthMethod

	// Try key-based authentication first
	if c.config.KeyPath != "" {
		keyAuth, err := c.getKeyAuth(c.config.KeyPath)
		if err != nil {
			return fmt.Errorf("failed to load SSH key: %w", err)
		}
		authMethods = append(authMethods, keyAuth)
	}

	// Add password authentication if provided
	if c.config.Password != "" {
		authMethods = append(authMethods, ssh.Password(c.config.Password))
	}

	if len(authMethods) == 0 {
		return fmt.Errorf("no authentication method provided (key or password)")
	}

	config := &ssh.ClientConfig{
		User:            c.config.User,
		Auth:            authMethods,
		Timeout:         c.config.Timeout,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // TODO: Implement proper host key verification
	}

	addr := net.JoinHostPort(c.config.Host, c.config.Port)
	conn, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		return fmt.Errorf("failed to connect to %s: %w", addr, err)
	}

	c.connection = conn
	return nil
}

func (c *Client) getKeyAuth(keyPath string) (ssh.AuthMethod, error) {
	// Try to read the key file
	key, err := ioutil.ReadFile(keyPath)
	if err != nil {
		return nil, fmt.Errorf("unable to read private key: %w", err)
	}

	// Create the Signer for this private key
	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		return nil, fmt.Errorf("unable to parse private key: %w", err)
	}

	return ssh.PublicKeys(signer), nil
}

func (c *Client) Execute(command string) (string, error) {
	if c.connection == nil {
		return "", fmt.Errorf("not connected to server")
	}

	session, err := c.connection.NewSession()
	if err != nil {
		return "", fmt.Errorf("failed to create session: %w", err)
	}
	defer session.Close()

	// Set a timeout for long-running commands
	done := make(chan error, 1)
	var output []byte
	
	go func() {
		output, err = session.CombinedOutput(command)
		done <- err
	}()

	select {
	case err := <-done:
		if err != nil {
			return string(output), fmt.Errorf("command failed: %w", err)
		}
		return string(output), nil
	case <-time.After(60 * time.Second): // 60 second timeout
		session.Close() // Close the session instead of trying to signal
		return "", fmt.Errorf("command timed out after 60 seconds")
	}
}

func (c *Client) ExecuteWithStdin(command, stdin string) (string, error) {
	if c.connection == nil {
		return "", fmt.Errorf("not connected to server")
	}

	session, err := c.connection.NewSession()
	if err != nil {
		return "", fmt.Errorf("failed to create session: %w", err)
	}
	defer session.Close()

	session.Stdin = strings.NewReader(stdin)
	output, err := session.CombinedOutput(command)
	if err != nil {
		return string(output), fmt.Errorf("command failed: %w", err)
	}

	return string(output), nil
}

func (c *Client) Close() error {
	if c.connection != nil {
		return c.connection.Close()
	}
	return nil
}

func (c *Client) IsConnected() bool {
	return c.connection != nil
}

// GetDefaultKeyPath returns the default SSH key path for the current user
func GetDefaultKeyPath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return homeDir + "/.ssh/id_rsa"
}