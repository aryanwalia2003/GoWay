#!/usr/bin/env bash
# scripts/gen_fixture.sh
#
# Generates a JSON fixture file of N AWB records for load testing.
# Usage:
#   ./scripts/gen_fixture.sh 5000 > testdata/bench_5000.json
#   ./scripts/gen_fixture.sh 1000 > testdata/bench_1000.json
#
# Then benchmark with:
#   time ./awb-gen --input testdata/bench_5000.json --output /tmp/bench.pdf

set -euo pipefail

N="${1:-1000}"

skus=("Widget Pro x1" "Gadget Max x2" "Doohickey v3 x1" "Thingamajig x5" "Gizmo Plus x2, Accessory Pack x1")
cities=("Mumbai, Maharashtra" "Delhi, Delhi" "Bangalore, Karnataka" "Chennai, Tamil Nadu" "Hyderabad, Telangana" "Pune, Maharashtra" "Kolkata, West Bengal")
pincodes=("400001" "110001" "560001" "600001" "500001" "411001" "700001")
senders=("GlobalMart Store" "FastShip India" "QuickDeliver Co" "RetailHub Pro" "EcomExpress")

printf '[\n'
for i in $(seq 1 "$N"); do
  sku="${skus[$((RANDOM % ${#skus[@]}))]}"
  city="${cities[$((RANDOM % ${#cities[@]}))]}"
  pin="${pincodes[$((RANDOM % ${#pincodes[@]}))]}"
  sender="${senders[$((RANDOM % ${#senders[@]}))]}"
  weight="$((RANDOM % 10 + 1)).$(printf '%01d' $((RANDOM % 10)))kg"
  awb="ZFW$(printf '%012d' "$i")"

  comma=","
  if [ "$i" -eq "$N" ]; then
    comma=""
  fi

  printf '  {"awb_number":"%s","order_id":"#%d","sender":"%s","receiver":"Customer %d","address":"%d Elm Road, %s","pincode":"%s","weight":"%s","sku_details":"%s"}%s\n' \
    "$awb" "$i" "$sender" "$i" "$((RANDOM % 999 + 1))" "$city" "$pin" "$weight" "$sku" "$comma"
done
printf ']\n'
