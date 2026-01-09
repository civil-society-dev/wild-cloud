#!/bin/bash

echo "🛑 Stopping wild-cloud-central background services..."

if docker ps | grep -q wild-central-bg; then
    docker stop wild-central-bg
    docker rm wild-central-bg
    echo "✅ Services stopped and container removed."
else
    echo "ℹ️  No background services running."
fi