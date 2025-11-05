# Setup Guide - Synaptica.ai Platform

## Current Status

✅ **All code files are in your local directory**: `/Users/manasbhole/Synaptica.ai`

## Project Structure

```
Synaptica.ai/
├── cmd/                          # Microservices (11 services)
│   ├── api-gateway/             # API Gateway (Port 8080)
│   ├── ingestion-service/       # Ingestion Service (Port 8081)
│   ├── dlp-service/             # DLP & PHI Detector (Port 8082)
│   ├── deid-service/            # De-ID & Token Vault (Port 8083)
│   ├── normalizer-service/      # Schema Normalizer (Port 8084)
│   ├── linkage-service/         # Record Linkage (Port 8085)
│   ├── llm-service/             # LLM Pipelines (Port 8086)
│   ├── cohort-service/          # Cohort/Query Engine (Port 8087)
│   ├── training-service/        # Model Training (Port 8088)
│   ├── serving-service/         # Model Serving (Port 8089)
│   └── cleanroom-service/       # Clean Room (Port 8090)
├── pkg/                          # Shared packages
│   ├── common/                   # Common utilities
│   │   ├── config/              # Configuration management
│   │   ├── database/            # Database connections (Postgres, Redis)
│   │   ├── kafka/               # Kafka producer/consumer
│   │   ├── logger/              # Logging
│   │   └── models/              # Data models
│   ├── gateway/                  # API Gateway components
│   │   ├── auth/                # OIDC authentication
│   │   ├── middleware/          # Middleware (RLS, auth, logging)
│   │   └── routes/              # Route handlers
│   └── storage/                  # Storage layer
│       ├── lakehouse.go         # Lakehouse storage
│       ├── rtolap.go            # RT OLAP storage
│       └── featurestore.go      # Feature store
├── scripts/                      # Build scripts
├── docker-compose.yml            # Local infrastructure
├── go.mod                        # Go module definition
├── Makefile                      # Build commands
├── README.md                     # Project documentation
├── ARCHITECTURE.md              # Architecture documentation
└── .env.example                  # Environment variables template
```

## Step 1: Verify Local Files

All your code is currently **LOCAL** in `/Users/manasbhole/Synaptica.ai`. To see all files:

```bash
cd /Users/manasbhole/Synaptica.ai
find . -type f | grep -v ".git" | sort
```

## Step 2: Initialize Git Repository (Already Done ✅)

Git repository has been initialized. Check status:

```bash
git status
```

## Step 3: Create Initial Commit

```bash
# Add all files
git add .

# Create initial commit
git commit -m "Initial commit: Synaptica.ai healthcare data pipeline platform"
```

## Step 4: Create GitHub Repository

You have two options:

### Option A: Create New Repository on GitHub

1. Go to https://github.com/new
2. Repository name: `Synaptica.ai` (or `synaptica-platform`)
3. Description: "Healthcare data pipeline platform with microservices architecture"
4. Choose **Private** or **Public**
5. **DO NOT** initialize with README, .gitignore, or license (we already have these)
6. Click "Create repository"

### Option B: Use GitHub CLI (if installed)

```bash
# Install GitHub CLI if not installed
# brew install gh

# Login to GitHub
gh auth login

# Create repository
gh repo create Synaptica.ai --private --source=. --remote=origin --push
```

## Step 5: Connect Local Repo to GitHub

After creating the repository on GitHub, run:

```bash
# Add remote (replace YOUR_USERNAME with your GitHub username)
git remote add origin https://github.com/YOUR_USERNAME/Synaptica.ai.git

# Or if you prefer SSH
git remote add origin git@github.com:YOUR_USERNAME/Synaptica.ai.git

# Verify remote
git remote -v

# Push to GitHub
git branch -M main  # Rename branch to main (if needed)
git push -u origin main
```

## Step 6: Verify on GitHub

After pushing, visit:
- `https://github.com/YOUR_USERNAME/Synaptica.ai`

You should see all your code files there!

## Quick Start Commands

```bash
# Install dependencies
go mod download

# Start infrastructure (Postgres, Redis, Kafka, ClickHouse)
make docker-up
# or
docker-compose up -d

# Build all services
make build
# or
./scripts/build.sh

# Run individual services (in separate terminals)
make run-gateway
make run-ingestion
make run-dlp
# ... etc

# Or run all services (requires process manager)
```

## Environment Setup

1. Copy `.env.example` to `.env`:
```bash
cp .env.example .env
```

2. Update `.env` with your configuration (API keys, database passwords, etc.)

## Next Steps

1. ✅ Code is ready - all 11 microservices implemented
2. ⏭️ Create GitHub repository
3. ⏭️ Push code to GitHub
4. ⏭️ Set up CI/CD (GitHub Actions workflow included)
5. ⏭️ Deploy to cloud (AWS/GCP/Azure)

## Troubleshooting

**Q: Where are my code files?**
A: All files are in `/Users/manasbhole/Synaptica.ai`. They're not on GitHub yet - you need to push them.

**Q: How do I push to GitHub?**
A: Follow Step 4 and Step 5 above to create a repo and push.

**Q: Can I see the code structure?**
A: Run `tree -L 3` or `find . -type f | head -30` to see all files.

**Q: How many services are there?**
A: 11 microservices total:
- API Gateway
- Ingestion Service
- DLP Service
- De-ID Service
- Normalizer Service
- Linkage Service
- LLM Service
- Cohort Service
- Training Service
- Serving Service
- Clean Room Service

