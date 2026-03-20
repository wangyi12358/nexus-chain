#!/bin/bash

# Build script for Nexus Chain

echo "Building Nexus Chain..."
go build -o bin/nexus-chain cmd/nexus-chain/main.go
echo "Build complete."
