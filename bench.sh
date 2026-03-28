#!/bin/bash

echo "Starting 5k batch benchmark..."

# start process
/usr/bin/time -v ./awb-gen --input 5k_batch.json --output 5k_out.pdf --pprof localhost:6060 > awb_gen.log 2>&1 &
PID=$!

echo "AWB Generator started with PID $PID. Waiting 1 second for HTTP server to come up..."
sleep 1

echo "Collecting heap profile to mem.prof..."
curl -s http://localhost:6060/debug/pprof/heap > mem.prof

echo "Collecting 5-second CPU profile to cpu.prof..."
curl -s "http://localhost:6060/debug/pprof/profile?seconds=5" > cpu.prof

echo "Waiting for batch run to complete..."
wait $PID
echo "Done!"
