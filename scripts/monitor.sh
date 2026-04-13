#!/bin/bash
set -e

# GoWay Resource Monitor
# Records CPU and Memory usage of the goway container to a CSV file.

OUTPUT_FILE="stats.csv"
INTERVAL=2 # seconds
CONTAINER_NAME="goway-goway-1"

# Create header if file doesn't exist
if [ ! -f "$OUTPUT_FILE" ]; then
    echo "timestamp,container,cpu_perc,mem_usage,mem_limit,mem_perc" > "$OUTPUT_FILE"
fi

echo "Monitoring $CONTAINER_NAME every $INTERVAL seconds..."
echo "Press Ctrl+C to stop. Data is saved to $OUTPUT_FILE"

while true; do
    # Get stats for the specific container
    # Format: timestamp, name, cpu%, memUsage / memLimit, mem%
    STATS=$(docker stats "$CONTAINER_NAME" --no-stream --format "{{.Name}},{{.CPUPerc}},{{.MemUsage}},{{.MemPerc}}")
    
    if [ -n "$STATS" ]; then
        TIMESTAMP=$(date +"%Y-%m-%dT%H:%M:%S")
        
        # Parse MemUsage which looks like "10MiB / 1GiB"
        NAME=$(echo "$STATS" | cut -d',' -f1)
        CPU=$(echo "$STATS" | cut -d',' -f2)
        MEM_ALL=$(echo "$STATS" | cut -d',' -f3)
        MEM_PERC=$(echo "$STATS" | cut -d',' -f4)
        
        # Split mem_all into usage and limit
        MEM_USAGE=$(echo "$MEM_ALL" | awk '{print $1}')
        MEM_LIMIT=$(echo "$MEM_ALL" | awk '{print $3}')

        echo "$TIMESTAMP,$NAME,$CPU,$MEM_USAGE,$MEM_LIMIT,$MEM_PERC" >> "$OUTPUT_FILE"
    fi
    
    sleep "$INTERVAL"
done
