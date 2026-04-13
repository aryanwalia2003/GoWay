#!/bin/bash
set -e

# GoWay Setup Script
# Bootstraps the environment for development and deployment

RED='\033[0;31m'
GREEN='\033[0;32m'
NC='\033[0m' # No Color

echo -e "${GREEN}===> Starting GoWay Setup...${NC}"

# 1. Check Requirements
echo "Checking requirements..."
if ! command -v docker &> /dev/null; then
    echo -e "${RED}Error: docker is not installed.${NC}"
    exit 1
fi

if ! command -v go &> /dev/null; then
    echo -e "${RED}Error: go is not installed.${NC}"
    exit 1
fi
echo -e "${GREEN}Requirements OK.${NC}"

# 2. Initialize .env
if [ ! -f .env ]; then
    echo "Creating .env from .env.example..."
    cp .env.example .env
    echo -e "${GREEN}.env created. Please review it before deploying.${NC}"
else
    echo ".env already exists."
fi

# 3. Ensure Templates and Assets
echo "Verifying templates..."
if [ ! -d templates ]; then
    echo -e "${RED}Error: templates directory missing.${NC}"
    exit 1
fi

if [ ! -f templates/uc_shipping_label.html ]; then
    echo -e "${RED}Error: uc_shipping_label.html template missing.${NC}"
    exit 1
fi

if [ ! -f templates/zippee_logo_new.jpeg ]; then
    echo -e "${RED}Warning: Zippee logo missing in templates/. Labels may fail rendering.${NC}"
fi
echo -e "${GREEN}Templates OK.${NC}"

echo -e "${GREEN}===> Setup Complete!${NC}"
echo "You can now run: ./scripts/deploy.sh up"
