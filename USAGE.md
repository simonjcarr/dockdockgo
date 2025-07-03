# DockDockGo Usage Guide

## Getting Started

After installation, verify DockDockGo is working:

```bash
# Check version
dockdockgo --version

# View available commands
dockdockgo --help
```

## Basic Commands

### Container Management

```bash
# Run a container
dockdockgo run nginx

# Run with port mapping
dockdockgo run -p 8080:80 nginx

# Run with environment variables
dockdockgo run -e ENV=production nginx

# List running containers
dockdockgo ps

# View container logs
dockdockgo logs <container-name>

# Stop a container
dockdockgo stop <container-name>
```

### Image Management

```bash
# Search for images
dockdockgo search nginx

# Search in specific registry
dockdockgo search --registry docker.io nginx

# List local images
dockdockgo images
```

### Docker Compose Support

```bash
# Deploy from docker-compose.yml
dockdockgo compose up

# Deploy with custom file
dockdockgo compose -f my-compose.yml up

# Stop compose deployment
dockdockgo compose down
```

### Cluster Management

```bash
# Add remote servers
dockdockgo cluster add --host 192.168.1.100 --user root

# List cluster nodes
dockdockgo cluster ls

# Deploy to specific nodes
dockdockgo run --nodes node1,node2 nginx

# Scale deployment
dockdockgo scale my-app=3
```

## Advanced Usage

### Multi-Node Deployments

Deploy containers across multiple servers:

```bash
# Deploy with replicas across nodes
dockdockgo run --replicas 3 --nodes node1,node2,node3 nginx

# Deploy with load balancing
dockdockgo run --replicas 2 --load-balance round-robin nginx
```

### Configuration Management

```bash
# Generate configuration
dockdockgo config generate

# Edit configuration
dockdockgo config edit

# View current configuration
dockdockgo config show
```

### Service Management

```bash
# Check service status
sudo systemctl status dockdockgo

# View service logs
sudo journalctl -u dockdockgo -f

# Restart service
sudo systemctl restart dockdockgo
```

## Examples

### Simple Web Application

```bash
# Deploy nginx with custom configuration
dockdockgo run -p 80:80 -v /etc/nginx:/etc/nginx nginx

# Scale to 3 replicas
dockdockgo scale nginx=3
```

### Database Cluster

```bash
# Deploy PostgreSQL cluster
dockdockgo run --replicas 3 --cluster postgres:13

# Add persistent storage
dockdockgo run -v /data/postgres:/var/lib/postgresql/data postgres:13
```

### Development Environment

```bash
# Deploy full stack from compose
dockdockgo compose -f docker-compose.dev.yml up

# View logs for specific service
dockdockgo logs web-app
```

## Configuration Files

### Extended Docker Compose

DockDockGo supports extended docker-compose files with cluster configuration:

```yaml
version: '3.8'
services:
  web:
    image: nginx
    ports:
      - "80:80"
    x-dockdockgo:
      replicas: 3
      nodes:
        - node1
        - node2
        - node3
      load_balance: round-robin
      
  db:
    image: postgres:13
    environment:
      POSTGRES_DB: myapp
    x-dockdockgo:
      replicas: 1
      nodes:
        - node1
      persistent: true
```

### Cluster Configuration

```yaml
# /etc/dockdockgo/cluster.yml
nodes:
  - name: node1
    host: 192.168.1.100
    user: root
    ssh_key: /root/.ssh/id_rsa
    
  - name: node2
    host: 192.168.1.101
    user: root
    ssh_key: /root/.ssh/id_rsa

load_balancer:
  type: round-robin
  health_check: true
  
registry:
  default: docker.io
  mirrors:
    - registry.local:5000
```

## Best Practices

1. **Use meaningful names**: Name your containers and services descriptively
2. **Resource limits**: Set appropriate CPU and memory limits
3. **Health checks**: Configure health checks for critical services
4. **Persistent storage**: Use volumes for data that needs to persist
5. **Security**: Use non-root users and limit network access
6. **Monitoring**: Regularly check logs and service status

## Troubleshooting

### Common Issues

1. **Container won't start**
   ```bash
   # Check logs
   dockdockgo logs <container-name>
   
   # Check service status
   sudo systemctl status dockdockgo
   ```

2. **Port conflicts**
   ```bash
   # List port usage
   dockdockgo ps --ports
   
   # Use automatic port assignment
   dockdockgo run -P nginx
   ```

3. **Connection issues**
   ```bash
   # Test cluster connectivity
   dockdockgo cluster ping
   
   # Check SSH connectivity
   dockdockgo cluster test-ssh
   ```

### Log Locations

- **System logs**: `/var/log/dockdockgo/`
- **Service logs**: `sudo journalctl -u dockdockgo`
- **Container logs**: `dockdockgo logs <container-name>`