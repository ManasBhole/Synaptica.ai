#!/bin/bash

# Build script for all services

set -e

echo "Building Synaptica.ai Platform..."

# Create bin directory
mkdir -p bin

# Build all services
echo "Building API Gateway..."
go build -o bin/api-gateway ./cmd/api-gateway

echo "Building Ingestion Service..."
go build -o bin/ingestion-service ./cmd/ingestion-service

echo "Building DLP Service..."
go build -o bin/dlp-service ./cmd/dlp-service

echo "Building De-ID Service..."
go build -o bin/deid-service ./cmd/deid-service

echo "Building Normalizer Service..."
go build -o bin/normalizer-service ./cmd/normalizer-service

echo "Building Linkage Service..."
go build -o bin/linkage-service ./cmd/linkage-service

echo "Building LLM Service..."
go build -o bin/llm-service ./cmd/llm-service

echo "Building Cohort Service..."
go build -o bin/cohort-service ./cmd/cohort-service

echo "Building Training Service..."
go build -o bin/training-service ./cmd/training-service

echo "Building Serving Service..."
go build -o bin/serving-service ./cmd/serving-service

echo "Building Clean Room Service..."
go build -o bin/cleanroom-service ./cmd/cleanroom-service

echo "Build complete! Binaries are in ./bin"

