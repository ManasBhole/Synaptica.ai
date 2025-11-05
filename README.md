# Synaptica.ai - Healthcare Data Pipeline Platform

A comprehensive microservices architecture for healthcare data ingestion, processing, and analytics with AI/ML capabilities.

## Architecture Overview

The platform consists of multiple microservices that handle:
- **Upstream Data Ingestion**: Hospitals/EHR, Labs, Imaging, Wearables, Telehealth
- **Privacy & Normalization**: PHI detection, de-identification, schema normalization
- **Data Storage**: Lakehouse, RT OLAP, OLTP, Feature Store, Redis Cache
- **AI/Analytics**: LLM pipelines, cohort queries, model training & serving
- **Downstream Consumption**: Clinician apps, Pharma/CRO, Insurers, Internal ops

## Services

### Core Services
- `api-gateway`: OIDC, mTLS, Row-Level Security gateway
- `ingestion-service`: Handles FHIR, device JSON, file drops
- `event-bus`: Kafka/Pub-Sub integration
- `dlp-service`: PHI detection using regex, NER, and LLM
- `deid-service`: De-identification and token vault with k/l-diversity
- `normalizer-service`: Schema normalization to canonical FHIR
- `linkage-service`: Record linkage (deterministic + probabilistic)
- `llm-service`: NL→Cohort, Notes NLP, Code mapping
- `cohort-service`: Cohort/Query Engine with DSL verifier
- `training-service`: Model training with AutoML
- `serving-service`: Model serving with Triton/Vertex/TF Serving
- `cleanroom-service`: Clean room with differential privacy

## Tech Stack

- **Language**: Go 1.21+
- **Databases**: PostgreSQL (OLTP), ClickHouse (RT OLAP), Redis (Cache)
- **Message Queue**: Kafka
- **API Gateway**: Custom implementation with OIDC/mTLS
- **Storage**: Delta Lake / BigQuery (Lakehouse)
- **ML**: TensorFlow Serving, Triton

## Getting Started

### Prerequisites
- Go 1.21+
- Docker & Docker Compose
- PostgreSQL, Redis, Kafka (via Docker)

### Run Locally

```bash
# Start infrastructure
docker-compose up -d

# Run services (each in separate terminal)
cd cmd/api-gateway && go run main.go
cd cmd/ingestion-service && go run main.go
# ... etc
```

### Development

```bash
# Install dependencies
go mod download

# Run tests
go test ./...

# Build all services
./scripts/build.sh
```

## Configuration

Services use environment variables for configuration. See `.env.example` for details.

## License

Proprietary - Synaptica.ai

