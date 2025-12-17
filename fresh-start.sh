#!/bin/bash

set -e

echo "🔄 Starting complete fresh reset..."

cd "$(dirname "$0")"

# Stop and remove everything
echo "1. Stopping services..."
docker compose down -v 2>/dev/null || true

# Remove volumes
echo "2. Removing volumes..."
docker volume ls | grep -E "questdb|redpanda|polymarket" | awk '{print $2}' | xargs -r docker volume rm 2>/dev/null || echo "   No volumes to remove"

# Start services
echo "3. Starting services..."
docker compose up -d

# Wait for services
echo "4. Waiting for services to initialize (45 seconds)..."
sleep 45

# Check services
echo "5. Checking services..."
docker ps --format "   {{.Names}}: {{.Status}}" | grep -E "questdb|redpanda|redpanda-connect"

# Create topics
echo "6. Creating topics..."
docker exec redpanda rpk topic create dlq-questdb --brokers localhost:9092 2>/dev/null || echo "   DLQ topic already exists"
docker exec redpanda rpk topic create polymarket-trades --brokers localhost:9092 --partitions 1 2>/dev/null || echo "   Topic already exists"

# Wait for Redpanda Connect to be ready
echo "7. Waiting for Redpanda Connect to initialize..."
sleep 5

# Final status
echo ""
echo "✅ Fresh start complete!"
echo ""
echo "Services:"
docker ps --format "   ✓ {{.Names}}: {{.Status}}" | grep -E "questdb|redpanda|redpanda-connect"
echo ""
echo "Topics:"
docker exec redpanda rpk topic list --brokers localhost:9092 2>&1 | tail -3
echo ""
echo "Redpanda Connect status:"
docker logs redpanda-connect --tail 10 2>&1 | grep -E "Running|Connected|Error" || echo "   Checking logs..."
echo ""
echo "🎉 Ready for new messages!"

