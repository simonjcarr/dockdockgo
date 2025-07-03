#!/bin/bash

# DockDockGo Uninstall Script
# This script removes DockDockGo and all its components from the system

set -e

# Configuration (must match install.sh)
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

confirm_uninstall() {
    echo -e "${YELLOW}WARNING: This will completely remove DockDockGo from your system.${NC}"
    echo "The following will be removed:"
    echo "  - DockDockGo binary ($INSTALL_DIR/$BINARY_NAME)"
    echo "  - DockDockGo service ($SERVICE_FILE)"
    echo "  - DockDockGo user account"
    echo "  - Configuration directory ($CONFIG_DIR)"
    echo "  - Data directory ($DATA_DIR)"
    echo "  - Log directory ($LOG_DIR)"
    echo "  - This uninstall script ($UNINSTALL_SCRIPT)"
    echo ""
    
    if [[ "${FORCE_UNINSTALL:-}" != "true" ]]; then
        read -p "Are you sure you want to continue? (y/N): " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            log_info "Uninstall cancelled by user."
            exit 0
        fi
    fi
}

check_root() {
    if [[ $EUID -ne 0 ]]; then
        log_error "This script must be run as root (use sudo)"
        exit 1
    fi
}

stop_and_disable_service() {
    if systemctl is-active --quiet dockdockgo 2>/dev/null; then
        log_info "Stopping DockDockGo service..."
        systemctl stop dockdockgo
    fi
    
    if systemctl is-enabled --quiet dockdockgo 2>/dev/null; then
        log_info "Disabling DockDockGo service..."
        systemctl disable dockdockgo
    fi
}

remove_service_file() {
    if [[ -f "$SERVICE_FILE" ]]; then
        log_info "Removing systemd service file..."
        rm -f "$SERVICE_FILE"
        systemctl daemon-reload
    fi
}

remove_binary() {
    if [[ -f "$INSTALL_DIR/$BINARY_NAME" ]]; then
        log_info "Removing DockDockGo binary..."
        rm -f "$INSTALL_DIR/$BINARY_NAME"
    fi
}

remove_user() {
    if id "$BINARY_NAME" &>/dev/null; then
        log_info "Removing DockDockGo user..."
        # Stop any processes running as the user
        pkill -u "$BINARY_NAME" 2>/dev/null || true
        # Remove the user
        userdel "$BINARY_NAME" 2>/dev/null || true
    fi
}

remove_directories() {
    local dirs=("$DATA_DIR" "$CONFIG_DIR" "$LOG_DIR")
    
    for dir in "${dirs[@]}"; do
        if [[ -d "$dir" ]]; then
            log_info "Removing directory: $dir"
            rm -rf "$dir"
        fi
    done
}

remove_firewall_rules() {
    # Remove firewall rules if ufw is available
    if command -v ufw &> /dev/null; then
        log_info "Removing firewall rules..."
        ufw delete allow 8080/tcp 2>/dev/null || true
        ufw delete allow 8443/tcp 2>/dev/null || true
    fi
}

remove_uninstall_script() {
    if [[ -f "$UNINSTALL_SCRIPT" ]]; then
        log_info "Removing uninstall script..."
        rm -f "$UNINSTALL_SCRIPT"
    fi
}

main() {
    log_info "Starting DockDockGo uninstall..."
    
    # Check if running as root
    check_root
    
    # Confirm uninstall
    confirm_uninstall
    
    # Stop and disable service
    stop_and_disable_service
    
    # Remove service file
    remove_service_file
    
    # Remove binary
    remove_binary
    
    # Remove user
    remove_user
    
    # Remove directories
    remove_directories
    
    # Remove firewall rules
    remove_firewall_rules
    
    # Remove uninstall script (this script removes itself)
    remove_uninstall_script
    
    log_success "DockDockGo has been completely removed from your system."
    log_info "Thank you for using DockDockGo!"
}

# Show usage if help requested
if [[ "${1:-}" == "--help" ]] || [[ "${1:-}" == "-h" ]]; then
    cat << EOF
DockDockGo Uninstall Script

Usage: $0 [OPTIONS]

Options:
  -h, --help    Show this help message
  --force       Skip confirmation prompt

This script will completely remove DockDockGo from your system including:
  - DockDockGo binary
  - Systemd service
  - User account
  - Configuration files
  - Data files
  - Log files
  - Firewall rules

Examples:
  sudo $0                    # Interactive uninstall
  sudo $0 --force            # Force uninstall without confirmation

EOF
    exit 0
fi

# Set force flag if requested
if [[ "${1:-}" == "--force" ]]; then
    export FORCE_UNINSTALL="true"
fi

# Run main uninstall
main "$@"