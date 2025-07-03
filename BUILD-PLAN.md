# DockDockGo Build & Development Plan

This document outlines the complete build, deployment, and development workflow for the DockDockGo container orchestration platform.

## Table of Contents
- [Project Overview](#project-overview)
- [Git Workflow Strategy](#git-workflow-strategy)
- [CI/CD Pipeline](#cicd-pipeline)
- [Version Management](#version-management)
- [Installation System](#installation-system)
- [Development Workflow](#development-workflow)
- [Database Strategy](#database-strategy)
- [Implementation Roadmap](#implementation-roadmap)

## Project Overview

DockDockGo is a distributed container orchestration platform designed for Linux environments. It provides Docker Swarm-like functionality with enhanced remote management capabilities through a CLI and API interface.

### Current Architecture
- **Distributed cluster** with master-worker nodes
- **gRPC communication** between cluster nodes (planned)
- **Raft consensus** for master election (planned)
- **BoltDB embedded database** for cluster state persistence
- **Container scheduling** with placement strategies
- **Deployment management** with persistent named deployments

### Technology Stack
- **Language**: Go 1.24.1
- **Database**: BoltDB (embedded)
- **Communication**: gRPC (planned)
- **Consensus**: Raft algorithm (planned)
- **CLI Framework**: Cobra
- **Target Platform**: Linux (amd64)

## Git Workflow Strategy

### Branch Structure
- **`main`** - Production releases (1.0.0+)
  - Protected branch requiring approval
  - Only receives merges from `develop`
  - Triggers production releases
- **`develop`** - Integration and testing branch (0.x.x releases)
  - Protected branch, no approval required
  - Receives feature branch merges
  - Triggers pre-release builds
- **`feature/*`** - Individual feature development
- **`fix/*`** - Bug fix branches
- **`hotfix/*`** - Emergency production fixes

### Conventional Commits Strategy
- **Feature branches**: Informal commit messages allowed during development
- **Pull Request titles**: MUST follow conventional commit format
  - `feat: add container health monitoring`
  - `fix: resolve port conflict detection bug`
  - `feat!: redesign cluster communication protocol` (breaking change)
- **Merge strategy**: Always squash merge PRs into `develop`
- **Semantic release**: Analyzes conventional commits for version bumps

### Branch Protection Rules
- **Main branch**: Requires 1 approval, up-to-date branches
- **Develop branch**: Requires passing CI checks, no approval needed
- **Direct pushes**: Blocked on both protected branches

## CI/CD Pipeline

### GitHub Actions Workflow

#### Pull Request Checks
- **PR Title Validation**: Ensure conventional commit format
- **Build & Test**: Compile binary and run test suite
- **Lint Check**: Code quality and formatting validation
- **Dependency Check**: Security vulnerability scanning

#### Develop Branch Pipeline
- **Build Binary**: Linux amd64 compilation
- **Run Tests**: Complete test suite execution
- **Create Pre-release**: 0.x.x-beta.y versions
- **Update Artifacts**: Development build artifacts

#### Main Branch Pipeline
- **Semantic Release**: Version bump based on conventional commits
- **Build Production Binary**: Optimized Linux build
- **Create GitHub Release**: With changelog and assets
- **Update install.sh**: Automatic script updates
- **Deploy Notifications**: Success/failure notifications

### Build Configuration
```yaml
# Example GitHub Actions workflow
name: Release
on:
  push:
    branches: [main, develop]
  pull_request:
    branches: [develop]

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v3
        with:
          go-version: '1.24.1'
      - run: go build -ldflags "-X main.Version=${{ github.ref_name }}"
```

## Version Management

### Semantic Versioning Strategy
- **Development Phase**: 0.x.x versions
  - 0.1.0, 0.2.0, 0.3.0, etc.
  - Breaking changes allowed without major version bump
- **Production Release**: 1.0.0 for first stable release
- **Version Injection**: Build-time using Go ldflags
  - `-X main.Version=0.2.1`
  - `-X main.Commit=abc123`
  - `-X main.BuildTime=2024-01-15T10:30:00Z`

### CLI Version Command
```bash
$ dockdockgo --version
DockDockGo version 0.2.1
Commit: abc123
Built: 2024-01-15T10:30:00Z
```

### Implementation in main.go
```go
var (
    Version   = "dev"
    Commit    = "unknown"
    BuildTime = "unknown"
)

func init() {
    rootCmd.Flags().BoolP("version", "v", false, "Print version information")
}
```

## Installation System

### Linux FHS Compliance
- **Binary Location**: `/usr/local/bin/dockdockgo`
- **Database Path**: `/var/lib/dockdockgo/dockdockgo.db`
- **Configuration**: `/etc/dockdockgo/config.yaml`
- **Logs**: `/var/log/dockdockgo/`
- **Systemd Service**: `/etc/systemd/system/dockdockgo.service`

### Install Script Features
```bash
#!/bin/bash
# install.sh functionality:
# 1. Detect Linux distribution
# 2. Download latest release from GitHub
# 3. Verify checksums
# 4. Install binary with proper permissions
# 5. Create required directories
# 6. Setup systemd service
# 7. Configure firewall if needed
```

### Installation Commands
```bash
# One-line installation
curl -sSL https://raw.githubusercontent.com/user/dockdockgo/main/install.sh | bash

# Manual installation
wget https://github.com/user/dockdockgo/releases/latest/download/dockdockgo-linux-amd64.tar.gz
tar -xzf dockdockgo-linux-amd64.tar.gz
sudo mv dockdockgo /usr/local/bin/
```

### Systemd Service Configuration
```ini
[Unit]
Description=DockDockGo Container Orchestration
After=network.target

[Service]
Type=simple
User=dockdockgo
Group=dockdockgo
ExecStart=/usr/local/bin/dockdockgo daemon
Restart=always
RestartSec=5
Environment=DOCKDOCKGO_ENV=production

[Install]
WantedBy=multi-user.target
```

## Development Workflow

### GitHub CLI Integration
Using `gh` CLI for complete workflow management:

1. **Feature Development**:
   ```bash
   git checkout develop
   git pull origin develop
   git checkout -b feature/gRPC-cluster-communication
   # Development work
   git add . && git commit -m "Implement gRPC service definitions"
   git push -u origin feature/gRPC-cluster-communication
   ```

2. **Pull Request Creation**:
   ```bash
   gh pr create \
     --title "feat: implement gRPC cluster communication" \
     --body "## Summary
   - Add gRPC service definitions
   - Implement cluster node registration
   - Add secure mTLS communication
   
   ## Testing
   - [x] Unit tests pass
   - [x] Integration tests with cluster
   - [x] Manual testing with multiple nodes"
   ```

3. **PR Management**:
   ```bash
   gh pr checks  # View CI status
   gh pr merge --squash --delete-branch  # Merge when ready
   ```

### Code Review Process
- **Automated Checks**: CI must pass before merge
- **Manual Review**: Optional for develop, required for main
- **Squash Merges**: Maintain clean commit history
- **Branch Cleanup**: Automatic deletion after merge

## Database Strategy

### Production vs Development
- **Production**: `/var/lib/dockdockgo/dockdockgo.db`
  - Persistent across system reboots
  - Proper file permissions (600)
  - Backup strategies implemented
- **Development**: `./dockdockgo.db`
  - Local to development directory
  - Easy cleanup and reset
  - Version controlled in .gitignore

### Database Schema
BoltDB buckets:
- `deployments` - Named container deployments
- `containers` - Individual container instances
- `nodes` - Cluster server information
- `cluster` - Overall cluster state
- `raft` - Raft consensus data (planned)

### Path Detection Logic
```go
func GetDatabasePath() string {
    if IsProduction() {
        return "/var/lib/dockdockgo/dockdockgo.db"
    }
    return "./dockdockgo.db"
}

func IsProduction() bool {
    executable, _ := os.Executable()
    return strings.HasPrefix(executable, "/usr/")
}
```

## Implementation Roadmap

### Phase 1: Build System & Versioning ✅
- [x] Create build.sh script
- [x] Add version injection support
- [ ] Implement --version flag
- [ ] Update main.go for version variables

### Phase 2: CI/CD Pipeline
- [ ] Create GitHub Actions workflow
- [ ] Setup conventional commit validation
- [ ] Configure semantic-release
- [ ] Add automated testing

### Phase 3: Installation System
- [ ] Create install.sh script
- [ ] Add systemd service configuration
- [ ] Implement production database paths
- [ ] Add uninstall functionality

### Phase 4: Core Features (Ongoing)
- [ ] gRPC service definitions
- [ ] Raft consensus implementation
- [ ] Container state monitoring
- [ ] Cluster management APIs

### Phase 5: Production Readiness
- [ ] Security hardening
- [ ] Performance optimization
- [ ] Monitoring and logging
- [ ] Documentation completion

## Current Status

### Completed Features (41/117 - 35%)
- ✅ SSH & Docker integration
- ✅ Configuration system
- ✅ Container deployment engine
- ✅ Docker Compose support
- ✅ Enhanced CLI commands
- ✅ Data models and storage
- ✅ Deployment management
- ✅ Container scheduling
- ✅ New CLI interface

### Next Priority Features
1. **gRPC cluster communication**
2. **Version management implementation**
3. **CI/CD pipeline setup**
4. **Installation system**
5. **Container state monitoring**

---

## Development Notes

### Prerequisites
- Go 1.24.1+
- GitHub CLI (`gh`) installed
- Linux development environment
- Docker for testing

### Quick Start Development
```bash
git clone https://github.com/user/dockdockgo.git
cd dockdockgo
git checkout develop
./build.sh
./bin/dockdockgo --help
```

### Testing Strategy
- Unit tests for core logic
- Integration tests with Docker
- End-to-end cluster testing
- Performance benchmarking

This build plan serves as the definitive guide for DockDockGo development and deployment processes.