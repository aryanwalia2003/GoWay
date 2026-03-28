#!/bin/bash

echo "Downloading Go 1.22.1..."
wget -q https://go.dev/dl/go1.22.1.linux-amd64.tar.gz

echo "Extracting and upgrading Go to 1.22.1 (requires sudo)..."
sudo rm -rf /usr/local/go
sudo tar -C /usr/local -xzf go1.22.1.linux-amd64.tar.gz

echo "Cleaning up..."
rm go1.22.1.linux-amd64.tar.gz

echo "Applying PATH changes..."
export PATH=$PATH:/usr/local/go/bin

echo "Go upgrade complete. New version:"
go version
