# DockDockGo Installation Guide

## Quick Installation

### Option 1: Using the Install Script (Recommended)

Download and run the installation script:

```bash
curl -sSL https://raw.githubusercontent.com/simoncarr/dockdockgo/main/install.sh | sudo bash
```

Or download the script first to review it:

```bash
wget https://raw.githubusercontent.com/simoncarr/dockdockgo/main/install.sh
chmod +x install.sh
sudo ./install.sh
```

### Option 2: Manual Installation

1. Download the latest release:
   ```bash
   wget https://github.com/simoncarr/dockdockgo/releases/latest/download/dockdockgo-linux-amd64.tar.gz
   ```

2. Extract the archive:
   ```bash
   tar -xzf dockdockgo-linux-amd64.tar.gz
   ```

3. Make the binary executable and move to system path:
   ```bash
   chmod +x dockdockgo
   sudo mv dockdockgo /usr/local/bin/
   ```

## System Requirements

- Linux AMD64 system
- Docker installed and running
- Root privileges for system installation
- Required packages: `curl`, `tar`, `systemctl`

## Verification

After installation, verify that DockDockGo is working:

```bash
# Check version
dockdockgo --version

# Check service status (if installed via install script)
sudo systemctl status dockdockgo

# View logs
sudo journalctl -u dockdockgo -f

# Test basic functionality
dockdockgo --help
```

## Configuration

### System Service

If installed via the install script, DockDockGo runs as a systemd service:

- **Data directory**: `/var/lib/dockdockgo`
- **Config directory**: `/etc/dockdockgo`
- **Log directory**: `/var/log/dockdockgo`
- **Service file**: `/etc/systemd/system/dockdockgo.service`

### Service Management

```bash
# Start the service
sudo systemctl start dockdockgo

# Stop the service
sudo systemctl stop dockdockgo

# Restart the service
sudo systemctl restart dockdockgo

# Enable auto-start on boot
sudo systemctl enable dockdockgo

# Disable auto-start on boot
sudo systemctl disable dockdockgo

# Check service status
sudo systemctl status dockdockgo
```

## Firewall Configuration

The install script automatically configures firewall rules for:
- Port 8080 (API)
- Port 8443 (gRPC)

## Troubleshooting

### Common Issues

1. **Permission denied errors**
   - Ensure you're running with sudo privileges
   - Check file permissions in `/var/lib/dockdockgo`

2. **Service fails to start**
   - Check that Docker is installed and running
   - Review logs: `sudo journalctl -u dockdockgo`

3. **Binary not found**
   - Ensure `/usr/local/bin` is in your PATH
   - Try running with full path: `/usr/local/bin/dockdockgo`

### Getting Help

- Check the logs: `sudo journalctl -u dockdockgo -f`
- View service status: `sudo systemctl status dockdockgo`
- Test configuration: `dockdockgo --help`

## Uninstallation

To remove DockDockGo:

```bash
# Stop and disable service
sudo systemctl stop dockdockgo
sudo systemctl disable dockdockgo

# Remove service file
sudo rm /etc/systemd/system/dockdockgo.service

# Remove binary
sudo rm /usr/local/bin/dockdockgo

# Remove data directories (optional)
sudo rm -rf /var/lib/dockdockgo
sudo rm -rf /etc/dockdockgo
sudo rm -rf /var/log/dockdockgo

# Remove user (optional)
sudo userdel dockdockgo

# Reload systemd
sudo systemctl daemon-reload
```