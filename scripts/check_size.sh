#!/bin/bash
set -e

echo "--------------------------------------------------"
echo "GoWay Docker Image Sizes"
echo "--------------------------------------------------"
docker images --format "table {{.Repository}}\t{{.Tag}}\t{{.Size}}" | grep goway
echo "--------------------------------------------------"
