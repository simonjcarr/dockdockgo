#!/bin/bash

# DockDockGo Fresh Deployment Script
# Fast deployment script for testing - complete clean install of latest version
# Usage: curl https://raw.githubusercontent.com/simonjcarr/dockdockgo/develop/deploy-fresh.sh | sudo bash

set -e

# Configuration
GITHUB_REPO="simonjcarr/dockdockgo"
BINARY_NAME="dockdockgo"
INSTALL_DIR="/usr/local/bin"
DATA_DIR="/var/lib/dockdockgo"
CONFIG_DIR="/etc/dockdockgo"
LOG_DIR="/var/log/dockdockgo"
SERVICE_FILE="/etc/systemd/system/dockdockgo.service"
UNINSTALL_SCRIPT="/usr/local/bin/dockdockgo-uninstall.sh"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Functions
log_info() {
    printf "${BLUE}🔄 [INFO]${NC} %s\n" "$1" >&2
}

log_success() {
    printf "${GREEN}✅ [SUCCESS]${NC} %s\n" "$1" >&2
}

log_warning() {
    printf "${YELLOW}⚠️  [WARNING]${NC} %s\n" "$1" >&2
}

log_error() {
    printf "${RED}❌ [ERROR]${NC} %s\n" "$1" >&2
}

check_root() {
    if [ "$(id -u)" -ne 0 ]; then
        log_error "This script must be run as root (use sudo)"
        exit 1
    fi
}

check_dependencies() {
    local deps="curl tar systemctl"
    for dep in $deps; do
        if ! command -v "$dep" >/dev/null 2>&1; then
            log_error "Required dependency '$dep' is not installed"
            exit 1
        fi
    done
}

uninstall_current() {
    log_info "Uninstalling current DockDockGo installation..."
    
    # Stop service if running
    if systemctl is-active --quiet dockdockgo 2>/dev/null; then
        log_info "Stopping DockDockGo service..."
        systemctl stop dockdockgo
    fi
    
    # Disable service if enabled
    if systemctl is-enabled --quiet dockdockgo 2>/dev/null; then
        log_info "Disabling DockDockGo service..."
        systemctl disable dockdockgo
    fi
    
    # Remove service file
    if [ -f "$SERVICE_FILE" ]; then
        log_info "Removing systemd service file..."
        rm -f "$SERVICE_FILE"
        systemctl daemon-reload
    fi
    
    # Remove binary
    if [ -f "$INSTALL_DIR/$BINARY_NAME" ]; then
        log_info "Removing DockDockGo binary..."
        rm -f "$INSTALL_DIR/$BINARY_NAME"
    fi
    
    # Remove user
    if id "$BINARY_NAME" >/dev/null 2>&1; then
        log_info "Removing DockDockGo user..."
        pkill -u "$BINARY_NAME" 2>/dev/null || true
        userdel "$BINARY_NAME" 2>/dev/null || true
    fi
    
    # Remove directories
    for dir in "$DATA_DIR" "$CONFIG_DIR" "$LOG_DIR"; do
        if [ -d "$dir" ]; then
            log_info "Removing directory: $dir"
            rm -rf "$dir"
        fi
    done
    
    # Remove firewall rules if ufw is available
    if command -v ufw >/dev/null 2>&1; then
        log_info "Removing firewall rules..."
        ufw delete allow 8080/tcp 2>/dev/null || true
        ufw delete allow 8443/tcp 2>/dev/null || true
    fi
    
    # Remove uninstall script
    if [ -f "$UNINSTALL_SCRIPT" ]; then
        rm -f "$UNINSTALL_SCRIPT"
    fi
    
    log_success "Current installation removed"
}

clean_current_directory() {
    log_info "Cleaning current directory of DockDockGo files..."
    
    # Remove any dockdockgo related files in current directory
    rm -f dockdockgo* 2>/dev/null || true
    rm -f *.tar.gz 2>/dev/null || true
    
    log_success "Current directory cleaned"
}

get_latest_version() {
    log_info "Getting latest version from GitHub..."
    
    local latest_version
    latest_version=$(curl -s "https://api.github.com/repos/$GITHUB_REPO/releases" | grep '"tag_name":' | head -1 | sed -E 's/.*"tag_name": *"([^"]+)".*/\1/')
    
    if [ -z "$latest_version" ]; then
        log_error "Failed to get latest version from GitHub"
        exit 1
    fi
    
    log_info "Latest version: $latest_version"
    echo "$latest_version"
}

get_arch() {
    local arch
    case "$(uname -m)" in
        x86_64)
            arch="amd64"
            ;;
        aarch64|arm64)
            arch="arm64"
            ;;
        armv7l)
            arch="arm"
            ;;
        *)
            log_error "Unsupported architecture: $(uname -m)"
            exit 1
            ;;
    esac
    echo "$arch"
}

download_and_install() {
    local version="$1"
    local arch
    arch=$(get_arch)
    local download_url="https://github.com/$GITHUB_REPO/releases/download/$version/dockdockgo-linux-$arch.tar.gz"
    local temp_dir=$(mktemp -d)
    local archive_file="$temp_dir/dockdockgo.tar.gz"
    
    log_info "Detected architecture: $arch"
    log_info "Downloading DockDockGo $version..."
    
    if ! curl -L -o "$archive_file" "$download_url"; then
        log_error "Failed to download DockDockGo"
        rm -rf "$temp_dir"
        exit 1
    fi
    
    log_info "Extracting archive..."
    cd "$temp_dir"
    if ! tar -xzf "$archive_file"; then
        log_error "Failed to extract archive"
        rm -rf "$temp_dir"
        exit 1
    fi
    
    # Find the binary (it might be in a subdirectory)
    local binary_path
    binary_path=$(find "$temp_dir" -name "$BINARY_NAME" -type f | head -1)
    
    if [ -z "$binary_path" ]; then
        log_error "Binary not found in archive"
        rm -rf "$temp_dir"
        exit 1
    fi
    
    log_info "Installing binary..."
    cp "$binary_path" "$INSTALL_DIR/$BINARY_NAME"
    chmod +x "$INSTALL_DIR/$BINARY_NAME"
    
    # Create user
    if ! id "$BINARY_NAME" >/dev/null 2>&1; then
        useradd --system --no-create-home --shell /bin/false "$BINARY_NAME"
    fi
    
    # Create directories
    for dir in "$DATA_DIR" "$CONFIG_DIR" "$LOG_DIR"; do
        mkdir -p "$dir"
        chown "$BINARY_NAME:$BINARY_NAME" "$dir"
    done
    
    # Create systemd service
    log_info "Creating systemd service..."
    cat > "$SERVICE_FILE" << EOF
[Unit]
Description=DockDockGo Container Orchestration Service
After=network.target
Wants=network.target

[Service]
Type=simple
User=$BINARY_NAME
Group=$BINARY_NAME
ExecStart=$INSTALL_DIR/$BINARY_NAME api start --host 0.0.0.0 --port 8080
Restart=always
RestartSec=5
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
    
    # Enable and start service
    systemctl daemon-reload
    systemctl enable dockdockgo
    systemctl start dockdockgo
    
    # Copy uninstall script
    if [ -f "$temp_dir/uninstall.sh" ]; then
        cp "$temp_dir/uninstall.sh" "$UNINSTALL_SCRIPT"
        chmod +x "$UNINSTALL_SCRIPT"
    fi
    
    # Configure firewall if ufw is available
    if command -v ufw >/dev/null 2>&1; then
        log_info "Configuring firewall..."
        ufw allow 8080/tcp 2>/dev/null || true
        ufw allow 8443/tcp 2>/dev/null || true
    fi
    
    # Cleanup
    rm -rf "$temp_dir"
    
    log_success "Installation completed"
}

verify_installation() {
    log_info "Verifying installation..."
    
    # Wait a moment for service to start
    sleep 3
    
    # Check if binary exists and is executable
    if [ ! -x "$INSTALL_DIR/$BINARY_NAME" ]; then
        log_error "Binary not found or not executable"
        return 1
    fi
    
    # Check version
    local version_output
    version_output=$("$INSTALL_DIR/$BINARY_NAME" -v 2>/dev/null || echo "Failed to get version")
    log_info "Installed version: $version_output"
    
    # Check service status
    if systemctl is-active --quiet dockdockgo; then
        log_success "Service is running"
    else
        log_warning "Service is not running - checking status..."
        systemctl status dockdockgo --no-pager -l
        return 1
    fi
    
    # Test basic connectivity
    if curl -s http://localhost:8080/api/v1/health > /dev/null; then
        log_success "API is responding"
    else
        log_warning "API is not responding yet (may take a moment to start)"
    fi
    
    return 0
}

main() {
    echo "🚀 DockDockGo Fresh Deployment Script" >&2
    echo "======================================" >&2
    
    # Pre-flight checks
    check_root
    check_dependencies
    
    # Main deployment process
    uninstall_current
    clean_current_directory
    
    local latest_version
    latest_version=$(get_latest_version)
    
    download_and_install "$latest_version"
    
    if verify_installation; then
        echo "" >&2
        log_success "🎉 Deployment completed successfully!"
        log_info "📋 Version: $("$INSTALL_DIR/$BINARY_NAME" -v 2>/dev/null | head -1)"
        log_info "🟢 Service: $(systemctl is-active dockdockgo)"
        log_info "🌐 API: http://localhost:8080"
        echo "" >&2
        echo "Ready to test! Try: dockdockgo server list" >&2
    else
        log_error "❌ Deployment completed but verification failed"
        log_info "Check service status: sudo systemctl status dockdockgo"
        exit 1
    fi
}

# Run main function
main "$@"