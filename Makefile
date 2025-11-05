.PHONY: help build run test clean docker-up docker-down

help:
	@echo "Available targets:"
	@echo "  build       - Build all services"
	@echo "  run-gateway - Run API Gateway"
	@echo "  run-ingestion - Run Ingestion Service"
	@echo "  run-dlp     - Run DLP Service"
	@echo "  run-deid    - Run De-ID Service"
	@echo "  run-normalizer - Run Normalizer Service"
	@echo "  run-linkage - Run Linkage Service"
	@echo "  run-llm     - Run LLM Service"
	@echo "  run-cohort  - Run Cohort Service"
	@echo "  run-training - Run Training Service"
	@echo "  run-serving - Run Serving Service"
	@echo "  run-cleanroom - Run Clean Room Service"
	@echo "  docker-up   - Start Docker infrastructure"
	@echo "  docker-down - Stop Docker infrastructure"
	@echo "  test        - Run tests"
	@echo "  clean       - Clean build artifacts"

build:
	@echo "Building all services..."
	@go build -o bin/api-gateway ./cmd/api-gateway
	@go build -o bin/ingestion-service ./cmd/ingestion-service
	@go build -o bin/dlp-service ./cmd/dlp-service
	@go build -o bin/deid-service ./cmd/deid-service
	@go build -o bin/normalizer-service ./cmd/normalizer-service
	@go build -o bin/linkage-service ./cmd/linkage-service
	@go build -o bin/llm-service ./cmd/llm-service
	@go build -o bin/cohort-service ./cmd/cohort-service
	@go build -o bin/training-service ./cmd/training-service
	@go build -o bin/serving-service ./cmd/serving-service
	@go build -o bin/cleanroom-service ./cmd/cleanroom-service

run-gateway:
	@go run ./cmd/api-gateway

run-ingestion:
	@go run ./cmd/ingestion-service

run-dlp:
	@go run ./cmd/dlp-service

run-deid:
	@go run ./cmd/deid-service

run-normalizer:
	@go run ./cmd/normalizer-service

run-linkage:
	@go run ./cmd/linkage-service

run-llm:
	@go run ./cmd/llm-service

run-cohort:
	@go run ./cmd/cohort-service

run-training:
	@go run ./cmd/training-service

run-serving:
	@go run ./cmd/serving-service

run-cleanroom:
	@go run ./cmd/cleanroom-service

docker-up:
	@echo "Starting Docker infrastructure..."
	@docker-compose up -d
	@echo "Waiting for services to be ready..."
	@sleep 5
	@echo "Infrastructure is ready!"

docker-down:
	@echo "Stopping Docker infrastructure..."
	@docker-compose down

test:
	@go test ./...

clean:
	@echo "Cleaning build artifacts..."
	@rm -rf bin/
	@go clean

