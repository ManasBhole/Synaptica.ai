# Synaptica.ai Platform Architecture

## Overview

This document describes the comprehensive microservices architecture for the Synaptica.ai healthcare data pipeline platform.

## Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         UPSTREAM PRODUCERS                                   │
├─────────────────────────────────────────────────────────────────────────────┤
│ Hospitals/EHR (FHIR/HL7/ABDM)  │  Labs/Diagnostics  │  Imaging (DICOM)     │
│ Wearables/IoT (CGM/HR/HRV)     │  Telehealth/Notes                          │
└─────────────────────────────────────────────────────────────────────────────┘
                                      │
                                      ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                         EDGE / INGRESS                                      │
├─────────────────────────────────────────────────────────────────────────────┤
│ API Gateway (OIDC, mTLS, RLS)                                              │
│         │                                                                   │
│         ▼                                                                   │
│ Ingestion Service (FHIR, device JSON, file drops)                           │
│         │                                                                   │
│         ▼                                                                   │
│ Event Bus (Kafka/Pub-Sub) - UPSTREAM ▶ events                              │
└─────────────────────────────────────────────────────────────────────────────┘
                                      │
                                      ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                    PRIVACY & NORMALIZATION                                  │
├─────────────────────────────────────────────────────────────────────────────┤
│ DLP & PHI Detector (regex+NER+LLM) - sanitize                              │
│         │                                                                   │
│         ▼                                                                   │
│ De-ID & Token Vault (k/l-diversity) - de-identify                          │
│         │                                                                   │
│         ▼                                                                   │
│ Schema Normalizer (FHIR→canonical, SNOMED/LOINC/ICD) - canonical rows      │
│         │                                                                   │
│         ▼                                                                   │
│ Record Linkage (deterministic + probabilistic) - canonical rows            │
└─────────────────────────────────────────────────────────────────────────────┘
                                      │
                                      ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                         DATA PLANES                                         │
├─────────────────────────────────────────────────────────────────────────────┤
│ Lakehouse (Delta/BigQuery)        │  DOWNSTREAM ▶ immutable facts          │
│ RT OLAP (ClickHouse/Pinot)        │  DOWNSTREAM ▶ denormalized facts       │
│ OLTP Truth (Postgres/AlloyDB)      │  consents/ids                          │
│ Feature Store (Offline+Online)     │  features build (batch)                │
│ Online FS Cache (Redis)            │  features p95<10ms                     │
└─────────────────────────────────────────────────────────────────────────────┘
                                      │
                                      ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                         AI / ANALYTICS                                      │
├─────────────────────────────────────────────────────────────────────────────┤
│ LLM Pipelines (NL→Cohort, Notes NLP, Code map) - NL schema/context         │
│ Cohort/Query Engine (DSL + verifier) - cohort scans, sub-sec slicing      │
│ Model Training (Batch/AutoML) - training data, feature views               │
│ Model Serving (Triton/Vertex/TF Serving) - model.artifacts, features       │
│ Clean Room (DP ε budgets, lineage) - DP queries                           │
└─────────────────────────────────────────────────────────────────────────────┘
                                      │
                                      ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                      DOWNSTREAM CONSUMERS                                   │
├─────────────────────────────────────────────────────────────────────────────┤
│ Clinician Apps/Alerts       │  risk scores                                 │
│ Pharma/CRO RWD/RWE          │  aggregates only                             │
│ Insurers/TPA Analytics      │  actuarial sets                              │
│ Internal Ops & Dashboards   │  ops queries/metrics                         │
└─────────────────────────────────────────────────────────────────────────────┘
```

## Service Architecture

### 1. API Gateway (`cmd/api-gateway`)
- **Port**: 8080
- **Features**:
  - OIDC authentication
  - mTLS support
  - Row-Level Security (RLS)
  - Request routing and forwarding
- **Endpoints**:
  - `GET /health` - Health check
  - `POST /api/v1/ingest` - Forward to ingestion service

### 2. Ingestion Service (`cmd/ingestion-service`)
- **Port**: 8081
- **Features**:
  - Handles FHIR, HL7, ABDM formats
  - Processes device JSON and file drops
  - Publishes events to Kafka
- **Endpoints**:
  - `POST /api/v1/ingest` - Accept upstream data
  - `GET /api/v1/ingest/status/{id}` - Check ingestion status

### 3. DLP Service (`cmd/dlp-service`)
- **Port**: 8082
- **Features**:
  - PHI detection using regex patterns
  - NER (Named Entity Recognition)
  - LLM-based detection
- **Endpoints**:
  - `POST /api/v1/detect` - Detect PHI in data
- **Kafka**: Consumes `upstream-events`, publishes `sanitized-events`

### 4. De-ID Service (`cmd/deid-service`)
- **Port**: 8083
- **Features**:
  - Tokenization of PHI
  - Token vault storage
  - k/l-diversity checks
- **Endpoints**:
  - `POST /api/v1/deid` - De-identify data
- **Kafka**: Consumes `sanitized-events`, publishes `deidentified-events`

### 5. Normalizer Service (`cmd/normalizer-service`)
- **Port**: 8084
- **Features**:
  - Converts to canonical FHIR format
  - Code mapping (SNOMED, LOINC, ICD)
- **Endpoints**:
  - `POST /api/v1/normalize` - Normalize data
- **Kafka**: Consumes `deidentified-events`, publishes `normalized-events`

### 6. Linkage Service (`cmd/linkage-service`)
- **Port**: 8085
- **Features**:
  - Deterministic record matching
  - Probabilistic record matching
  - Master patient ID creation
- **Endpoints**:
  - `POST /api/v1/link` - Link records
- **Kafka**: Consumes `normalized-events`, publishes to multiple downstream topics

### 7. LLM Service (`cmd/llm-service`)
- **Port**: 8086
- **Features**:
  - Natural language to cohort query conversion
  - Clinical notes NLP
  - Medical code mapping
- **Endpoints**:
  - `POST /api/v1/nl-to-cohort` - Convert NL to cohort query
  - `POST /api/v1/notes-nlp` - Process clinical notes
  - `POST /api/v1/code-map` - Map medical codes

### 8. Cohort Service (`cmd/cohort-service`)
- **Port**: 8087
- **Features**:
  - Cohort query execution
  - DSL verification
  - Query optimization
- **Endpoints**:
  - `POST /api/v1/cohort/query` - Execute cohort query
  - `POST /api/v1/cohort/verify` - Verify DSL
  - `GET /api/v1/cohort/{id}` - Get cohort by ID

### 9. Training Service (`cmd/training-service`)
- **Port**: 8088
- **Features**:
  - Model training orchestration
  - AutoML support
  - Training job management
- **Endpoints**:
  - `POST /api/v1/training/jobs` - Create training job
  - `GET /api/v1/training/jobs/{id}` - Get job status

### 10. Serving Service (`cmd/serving-service`)
- **Port**: 8089
- **Features**:
  - Model serving (Triton/Vertex/TF Serving)
  - Low-latency predictions (p95 < 10ms)
  - Feature retrieval from cache
- **Endpoints**:
  - `POST /api/v1/predict` - Get predictions
  - `GET /api/v1/models` - List available models

### 11. Clean Room Service (`cmd/cleanroom-service`)
- **Port**: 8090
- **Features**:
  - Differential privacy (DP) queries
  - ε-budget tracking
  - Data lineage tracking
- **Endpoints**:
  - `POST /api/v1/cleanroom/query` - Execute DP query
  - `GET /api/v1/cleanroom/query/{id}` - Get query result
  - `GET /api/v1/cleanroom/lineage/{id}` - Get data lineage

## Data Flow

### Upstream Flow
1. **Producers** → API Gateway → Ingestion Service
2. Ingestion Service → **Event Bus (Kafka)**
3. Event Bus → DLP Service → De-ID Service → Normalizer Service → Linkage Service
4. Linkage Service → **Data Planes** (Lakehouse, RT OLAP, OLTP)

### Downstream Flow
1. **Cohort Service** queries Lakehouse and RT OLAP
2. **LLM Service** processes queries and notes
3. **Training Service** uses Lakehouse and Feature Store
4. **Serving Service** uses Feature Store cache for predictions
5. **Clean Room Service** provides aggregated, privacy-preserving data

## Technology Stack

- **Language**: Go 1.21+
- **Message Queue**: Kafka
- **Databases**:
  - PostgreSQL (OLTP)
  - ClickHouse (RT OLAP)
  - Redis (Cache)
- **Storage**: Delta Lake / BigQuery (Lakehouse)
- **ML Serving**: Triton / Vertex AI / TensorFlow Serving
- **LLM**: OpenAI / Anthropic / Custom models

## Database Reliability & Speed

### OLTP (PostgreSQL/AlloyDB)
- **Reliability**: ⭐⭐⭐⭐⭐ (ACID compliance, strong consistency)
- **Speed**: ⭐⭐⭐⭐ (Transactional queries, <10ms for point lookups)
- **Use Case**: Consents, patient IDs, transactional data

### RT OLAP (ClickHouse/Pinot)
- **Reliability**: ⭐⭐⭐⭐ (High availability, replication)
- **Speed**: ⭐⭐⭐⭐⭐ (Sub-second analytical queries, optimized aggregations)
- **Use Case**: Real-time dashboards, interactive analytics

### Online FS Cache (Redis)
- **Reliability**: ⭐⭐⭐ (In-memory, requires persistence strategy)
- **Speed**: ⭐⭐⭐⭐⭐ (p95 < 10ms, in-memory access)
- **Use Case**: Hot feature serving for ML models

### Lakehouse (Delta/BigQuery)
- **Reliability**: ⭐⭐⭐⭐⭐ (Immutable, versioned, ACID)
- **Speed**: ⭐⭐⭐ (Complex analytical queries, batch processing)
- **Use Case**: Historical data, training data, complex analytics

## Security & Privacy

- **OIDC Authentication**: Industry-standard authentication
- **mTLS**: Mutual TLS for service-to-service communication
- **RLS**: Row-Level Security for data access control
- **PHI Detection**: Multi-layer detection (regex, NER, LLM)
- **De-identification**: Tokenization with k/l-diversity
- **Differential Privacy**: ε-budget tracking for privacy-preserving queries

## Scalability

- **Horizontal Scaling**: All services are stateless and can scale horizontally
- **Event-Driven**: Asynchronous processing via Kafka
- **Caching**: Redis for hot feature serving
- **Database Sharding**: Supported for high-volume data

## Monitoring & Observability

- **Logging**: Structured JSON logging via logrus
- **Health Checks**: All services expose `/health` endpoints
- **Metrics**: (To be implemented) Prometheus metrics
- **Tracing**: (To be implemented) Distributed tracing

