#!/bin/bash

# DockDockGo Build Script
# This script builds the DockDockGo binary for the current platform

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Script variables
BINARY_NAME="dockdockgo"
BUILD_DIR="./bin"
MAIN_FILE="main.go"

echo -e "${YELLOW}Building DockDockGo...${NC}"

# Check if Go is installed
if ! command -v go &> /dev/null; then
    echo -e "${RED}Error: Go is not installed or not in PATH${NC}"
    exit 1
fi

# Check if main.go exists
if [ ! -f "$MAIN_FILE" ]; then
    echo -e "${RED}Error: $MAIN_FILE not found${NC}"
    exit 1
fi

# Create build directory if it doesn't exist
mkdir -p "$BUILD_DIR"

# Get version info
# Try to get semantic version from git tags first (including annotated tags)
VERSION=$(git describe --tags --exact-match 2>/dev/null || git describe --tags 2>/dev/null || echo "")

if [ -z "$VERSION" ]; then
    # If no tags, use semantic versioning based on branch
    BRANCH=$(git rev-parse --abbrev-ref HEAD 2>/dev/null || echo "unknown")
    COMMIT_COUNT=$(git rev-list --count HEAD 2>/dev/null || echo "0")
    COMMIT_SHORT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
    
    if [ "$BRANCH" = "main" ]; then
        VERSION="1.0.0-dev"
    elif [ "$BRANCH" = "develop" ]; then
        # Use 0.x.y format for development
        MINOR=$((COMMIT_COUNT / 10))  # Increment minor every 10 commits
        PATCH=$((COMMIT_COUNT % 10))  # Patch is remainder
        VERSION="0.${MINOR}.${PATCH}-dev"
    else
        VERSION="0.0.0-dev-${COMMIT_SHORT}"
    fi
else
    # Clean up git describe output (remove 'v' prefix if present)
    VERSION=$(echo "$VERSION" | sed 's/^v//')
fi

COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME=$(date -u '+%Y-%m-%d_%H:%M:%S')

# Build flags
LDFLAGS="-X main.Version=$VERSION -X main.Commit=$COMMIT -X main.BuildTime=$BUILD_TIME"

echo -e "${YELLOW}Version: $VERSION${NC}"
echo -e "${YELLOW}Commit: $COMMIT${NC}"
echo -e "${YELLOW}Build Time: $BUILD_TIME${NC}"

# Download dependencies
echo -e "${YELLOW}Downloading dependencies...${NC}"
go mod download

# Build the binary
echo -e "${YELLOW}Compiling binary...${NC}"
go build -ldflags "$LDFLAGS" -o "$BUILD_DIR/$BINARY_NAME" "$MAIN_FILE"

# Check if build was successful
if [ -f "$BUILD_DIR/$BINARY_NAME" ]; then
    echo -e "${GREEN}✓ Build successful!${NC}"
    echo -e "${GREEN}Binary location: $BUILD_DIR/$BINARY_NAME${NC}"
    
    # Show binary info
    echo -e "${YELLOW}Binary size: $(du -h "$BUILD_DIR/$BINARY_NAME" | cut -f1)${NC}"
    
    # Make executable (in case it's not already)
    chmod +x "$BUILD_DIR/$BINARY_NAME"
    
    echo -e "${GREEN}✓ Build complete! Run with: ./$BUILD_DIR/$BINARY_NAME${NC}"
else
    echo -e "${RED}✗ Build failed!${NC}"
    exit 1
fi