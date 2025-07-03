#!/bin/bash

# DockDockGo Installation Script
# This script downloads and installs the latest DockDockGo release

set -e

# Configuration
GITHUB_REPO="simonjcarr/dockdockgo"  # Update with your actual GitHub username/repo
VERSION=""  # Will be automatically updated by CI
BINARY_NAME="dockdockgo"
INSTALL_DIR="/usr/local/bin"
DATA_DIR="/var/lib/dockdockgo"
CONFIG_DIR="/etc/dockdockgo"
LOG_DIR="/var/log/dockdockgo"
SERVICE_FILE="/etc/systemd/system/dockdockgo.service"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

check_root() {
    if [[ $EUID -ne 0 ]]; then
        log_error "This script must be run as root (use sudo)"
        exit 1
    fi
}

check_dependencies() {
    local deps=("curl" "tar" "systemctl")
    for dep in "${deps[@]}"; do
        if ! command -v "$dep" &> /dev/null; then
            log_error "Required dependency '$dep' is not installed"
            exit 1
        fi
    done
}

detect_architecture() {
    local arch=$(uname -m)
    case $arch in
        x86_64)
            echo "amd64"
            ;;
        *)
            log_error "Unsupported architecture: $arch"
            log_error "DockDockGo only supports Linux AMD64"
            exit 1
            ;;
    esac
}

get_latest_version() {
    if [[ -n "$VERSION" ]]; then
        echo "$VERSION"
        return
    fi
    
    local latest_version
    latest_version=$(curl -s "https://api.github.com/repos/$GITHUB_REPO/releases" | grep '"tag_name":' | head -1 | sed -E 's/.*"tag_name": *"([^"]+)".*/\1/')
    
    if [[ -z "$latest_version" ]]; then
        log_error "Failed to get latest version from GitHub"
        exit 1
    fi
    
    echo "$latest_version"
}

download_binary() {
    local version="$1"
    local arch="$2"
    local download_url="https://github.com/$GITHUB_REPO/releases/download/$version/dockdockgo-linux-$arch.tar.gz"
    local temp_dir=$(mktemp -d)
    local archive_file="$temp_dir/dockdockgo.tar.gz"
    
    log_info "Downloading DockDockGo $version for Linux $arch..." >&2
    
    if ! curl -L -o "$archive_file" "$download_url"; then
        log_error "Failed to download DockDockGo" >&2
        rm -rf "$temp_dir"
        exit 1
    fi
    
    log_info "Extracting binary..." >&2
    if ! tar -xzf "$archive_file" -C "$temp_dir"; then
        log_error "Failed to extract binary" >&2
        rm -rf "$temp_dir"
        exit 1
    fi
    
    echo "$temp_dir/$BINARY_NAME"
}

install_binary() {
    local binary_path="$1"
    
    log_info "Installing binary to $INSTALL_DIR..."
    
    # Stop service if running
    if systemctl is-active --quiet dockdockgo 2>/dev/null; then
        log_info "Stopping DockDockGo service..."
        systemctl stop dockdockgo
    fi
    
    # Install binary
    cp "$binary_path" "$INSTALL_DIR/$BINARY_NAME"
    chmod +x "$INSTALL_DIR/$BINARY_NAME"
    
    log_success "Binary installed to $INSTALL_DIR/$BINARY_NAME"
}

install_uninstall_script() {
    local temp_dir="$1"
    local uninstall_script="$temp_dir/uninstall.sh"
    local target_path="/usr/local/bin/dockdockgo-uninstall.sh"
    
    if [[ -f "$uninstall_script" ]]; then
        log_info "Installing uninstall script to $target_path..."
        cp "$uninstall_script" "$target_path"
        chmod +x "$target_path"
        log_success "Uninstall script installed to $target_path"
    else
        log_warning "Uninstall script not found in release archive"
    fi
}

create_user() {
    if ! id "$BINARY_NAME" &>/dev/null; then
        log_info "Creating dockdockgo user..."
        useradd --system --shell /bin/false --home-dir "$DATA_DIR" --create-home "$BINARY_NAME"
    fi
}

setup_directories() {
    log_info "Setting up directories..."
    
    # Create directories
    mkdir -p "$DATA_DIR" "$CONFIG_DIR" "$LOG_DIR"
    
    # Set ownership and permissions
    chown "$BINARY_NAME:$BINARY_NAME" "$DATA_DIR" "$LOG_DIR"
    chmod 755 "$DATA_DIR" "$CONFIG_DIR" "$LOG_DIR"
}

create_systemd_service() {
    log_info "Creating systemd service..."
    
    cat > "$SERVICE_FILE" << EOF
[Unit]
Description=DockDockGo Container Orchestration
Documentation=https://github.com/$GITHUB_REPO
After=network-online.target docker.service
Wants=network-online.target
Requires=docker.service

[Service]
Type=simple
User=$BINARY_NAME
Group=$BINARY_NAME
ExecStart=$INSTALL_DIR/$BINARY_NAME api start
ExecReload=/bin/kill -HUP \$MAINPID
Restart=always
RestartSec=5
Environment=DOCKDOCKGO_ENV=production
Environment=DOCKDOCKGO_DATA_DIR=$DATA_DIR
WorkingDirectory=$DATA_DIR

# Security settings
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=$DATA_DIR $LOG_DIR

[Install]
WantedBy=multi-user.target
EOF

    systemctl daemon-reload
}

configure_firewall() {
    # Configure firewall if ufw is available
    if command -v ufw &> /dev/null; then
        log_info "Configuring firewall..."
        ufw allow 8080/tcp comment "DockDockGo API"
        ufw allow 8443/tcp comment "DockDockGo gRPC"
    fi
}

main() {
    log_info "Starting DockDockGo installation..."
    
    # Checks
    check_root
    check_dependencies
    
    # Get system info
    local arch=$(detect_architecture)
    local version=$(get_latest_version)
    
    log_info "Installing DockDockGo $version for Linux $arch"
    
    # Download and install
    local binary_path=$(download_binary "$version" "$arch")
    local temp_dir=$(dirname "$binary_path")
    install_binary "$binary_path"
    install_uninstall_script "$temp_dir"
    
    # Setup system
    create_user
    setup_directories
    create_systemd_service
    configure_firewall
    
    # Cleanup
    rm -rf "$(dirname "$binary_path")"
    
    # Enable and start service
    log_info "Enabling and starting DockDockGo service..."
    systemctl enable dockdockgo
    systemctl start dockdockgo
    
    # Verify installation
    if "$INSTALL_DIR/$BINARY_NAME" --version &>/dev/null; then
        log_success "DockDockGo installed successfully!"
        log_info "Version: $($INSTALL_DIR/$BINARY_NAME --version | head -n1)"
        log_info "Service status: $(systemctl is-active dockdockgo)"
        log_info ""
        log_info "Useful commands:"
        log_info "  sudo systemctl status dockdockgo    # Check service status"
        log_info "  sudo journalctl -u dockdockgo       # View logs"
        log_info "  dockdockgo --help                   # View CLI help"
        log_info "  sudo dockdockgo-uninstall.sh        # Uninstall DockDockGo"
        log_info ""
        log_info "Data directory: $DATA_DIR"
        log_info "Config directory: $CONFIG_DIR"
        log_info "Log directory: $LOG_DIR"
    else
        log_error "Installation verification failed"
        exit 1
    fi
}

# Show usage if help requested
if [[ "${1:-}" == "--help" ]] || [[ "${1:-}" == "-h" ]]; then
    cat << EOF
DockDockGo Installation Script

Usage: $0 [OPTIONS]

Options:
  -h, --help    Show this help message

This script will:
  1. Download the latest DockDockGo release
  2. Install the binary to $INSTALL_DIR
  3. Install the uninstall script to /usr/local/bin/dockdockgo-uninstall.sh
  4. Create a system user for DockDockGo
  5. Set up directories and permissions
  6. Create and enable a systemd service
  7. Configure firewall rules (if ufw is available)

Requirements:
  - Linux AMD64 system
  - Root privileges (run with sudo)
  - curl, tar, and systemctl commands

Examples:
  sudo $0                    # Install latest version
  curl -sSL https://raw.githubusercontent.com/$GITHUB_REPO/main/install.sh | sudo bash

EOF
    exit 0
fi

# Run main installation
main "$@"