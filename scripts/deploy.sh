#!/bin/bash
set -e

# GoWay Deployment & Lifecycle Script

RED='\033[0;31m'
GREEN='\033[0;32m'
NC='\033[0m' # No Color

function usage() {
    echo "Usage: $0 {up|down|restart|logs|check}"
    echo "  up      : Build and start the service in background"
    echo "  down    : Stop and remove containers"
    echo "  restart : Restart the service"
    echo "  logs    : Tail the service logs"
    echo "  check   : Verify health endpoints"
    exit 1
}

if [ -z "$1" ]; then
    usage
fi

case "$1" in
    up)
        echo -e "${GREEN}Deploying GoWay...${NC}"
        docker compose up -d --build
        echo -e "${GREEN}Deployment started. Run '$0 check' to verify.${NC}"
        ;;
    down)
        echo "Stopping GoWay..."
        docker compose down
        echo -e "${GREEN}Service stopped.${NC}"
        ;;
    restart)
        echo "Restarting GoWay..."
        docker compose restart
        echo -e "${GREEN}Service restarted.${NC}"
        ;;
    logs)
        docker compose logs -f goway
        ;;
    check)
        echo "Checking service health..."
        # Extract port from .env or default to 8080
        PORT=$(grep "^PORT=" .env | cut -d '=' -f2)
        PORT=${PORT:-8080}
        
        echo "Testing http://localhost:${PORT}/readyz ..."
        if curl -s -f "http://localhost:${PORT}/readyz" > /dev/null; then
            echo -e "${GREEN}Service is READY.${NC}"
        else
            echo -e "${RED}Service is NOT READY or NOT RUNNING.${NC}"
            exit 1
        fi

        echo "Testing http://localhost:${PORT}/livez ..."
        if curl -s -f "http://localhost:${PORT}/livez" > /dev/null; then
             echo -e "${GREEN}Service is ALIVE.${NC}"
        else
            echo -e "${RED}Service is NOT responding.${NC}"
            exit 1
        fi
        ;;
    *)
        usage
        ;;
esac
