#!/bin/bash

set -ex
echo "Step 1: Install on local OS..."
echo "Step 1a: Installing dependencies..."
go install ./cmd/uptui
echo "Step 1b: Building binary/executable..."
go build -o uptui ./cmd/uptui

echo "Step 2: Docker image build..."
echo "Step 2a: Stop the container"
docker compose down
echo "Step 2b: Build the new image from scratch"
docker compose build --no-cache
echo "Step 2c: Start the container on new image"
docker compose up -d --force-recreate
echo "Step 2d: Get rid of garbage"
docker system prune -af
