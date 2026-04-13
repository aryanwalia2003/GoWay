#!/bin/bash
set -e

# GoWay Throughput Analyzer
# Parses JSON logs to calculate RPS and Latency.

CONTAINER_NAME="goway-goway-1"

echo "Analyzing logs for $CONTAINER_NAME..."

# 1. Get logs and filter for request completed lines
# The application logs lines like: 
# {"level":"info","ts":"...","msg":"request completed","trace_id":"...","duration_ms":123,...}

# Get logs from the last hour (or all if not specified)
LOGS=$(docker compose logs goway --no-log-prefix | grep "request completed" || true)

if [ -z "$LOGS" ]; then
    echo "No 'request completed' logs found."
    exit 0
fi

# 2. Total Requests
TOTAL_REQS=$(echo "$LOGS" | wc -l)

# 3. Latency Metrics (using jq)
AVG_LATENCY=$(echo "$LOGS" | jq -s 'map(.duration_ms) | add / length' | awk '{printf "%.2f", $1}')
MAX_LATENCY=$(echo "$LOGS" | jq -s 'map(.duration_ms) | max')
MIN_LATENCY=$(echo "$LOGS" | jq -s 'map(.duration_ms) | min')

# 4. Throughput (RPS)
# Calculate time span of logs
START_TS=$(echo "$LOGS" | head -n 1 | jq -r .ts)
END_TS=$(echo "$LOGS" | tail -n 1 | jq -r .ts)

# Convert ISO8601 to seconds
START_SEC=$(date -d "$START_TS" +%s)
END_SEC=$(date -d "$END_TS" +%s)
DURATION=$((END_SEC - START_SEC))

if [ "$DURATION" -le 0 ]; then
    RPS=$TOTAL_REQS
else
    RPS=$(echo "scale=2; $TOTAL_REQS / $DURATION" | bc -l)
fi

echo "----------------------------------------"
echo "Analysis Results"
echo "----------------------------------------"
echo "Total Requests  : $TOTAL_REQS"
echo "Avg Latency     : ${AVG_LATENCY}ms"
echo "Max Latency     : ${MAX_LATENCY}ms"
echo "Min Latency     : ${MIN_LATENCY}ms"
echo "Time Span (sec) : $DURATION"
echo "Throughput (RPS): $RPS"
echo "----------------------------------------"
