#!/bin/bash
set -ne

# GoWay Zombie Checker
# Inspects processes inside the container to find orphans or stuck renderers.

CONTAINER_NAME="goway-goway-1"

echo "Checking for zombie/orphaned processes in $CONTAINER_NAME..."

# 1. Get process list from container
# We look for processes that are NOT the main process (PID 1) and not our check command.
# We specially look for 'tectonic', 'chrome', or other suspected renderers.

PROCS=$(docker exec "$CONTAINER_NAME" ps aux || echo "Error: ps not available in container")

if [[ "$PROCS" == *"Error:"* ]]; then
    # Fallback to a simpler check if ps is missing (common in slim images)
    echo "Warning: 'ps' not found in container. Attempting fallback via /proc..."
    PROCS=$(docker exec "$CONTAINER_NAME" ls /proc | grep -E '^[0-9]+$' || true)
    
    ZOMBIE_COUNT=0
    for pid in $PROCS; do
        if [ "$pid" -ne 1 ]; then
             CMD=$(docker exec "$CONTAINER_NAME" cat "/proc/$pid/comm" 2>/dev/null || true)
             if [ -n "$CMD" ] && [ "$CMD" != "awb-gen" ]; then
                echo "Found potential orphaned process: PID $pid ($CMD)"
                ZOMBIE_COUNT=$((ZOMBIE_COUNT + 1))
             fi
        fi
    done
else
    # Parse ps aux output
    # Filter out: PID 1 (main), grep, ps, and the shell itself
    ZOMBIES=$(echo "$PROCS" | grep -vE "PID|awb-gen|ps aux|grep|sh -c" || true)
    
    if [ -n "$ZOMBIES" ]; then
        echo "The following potential orphaned processes were found:"
        echo "$ZOMBIES"
        ZOMBIE_COUNT=$(echo "$ZOMBIES" | wc -l)
    else
        ZOMBIE_COUNT=0
    fi
fi

if [ "$ZOMBIE_COUNT" -eq 0 ]; then
    echo -e "\033[0;32mClean! No orphaned processes found.\033[0m"
else
    echo -e "\033[0;31mFound $ZOMBIE_COUNT potential orphaned process(es).\033[0m"
    echo "If these persist, consider restarting the service: ./scripts/deploy.sh restart"
fi
