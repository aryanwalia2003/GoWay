#!/bin/bash

COUNT=${1:-5000}

echo "["
for i in $(seq 1 $COUNT); do
cat <<EOF
  {
    "awb_number": "AWB$(printf "%08d" $i)",
    "order_id": "ORD$(printf "%08d" $i)",
    "sender": "Test Sender Inc. 123 Logistics Park",
    "receiver": "Test Receiver $i",
    "address": "A/19 Example Block, Demo City, India",
    "pincode": "110001",
    "weight": "1.5 Kg",
    "sku_details": "Books (2), Electronics (1)"
  }$(if [ $i -lt $COUNT ]; then echo ","; fi)
EOF
done
echo "]"
