# Deployment Guide

This repository now includes container templates and automation to deploy the entire Synaptica
platform on free/open-source infrastructure.

## Container Images (GHCR)
- Workflow `.github/workflows/publish-containers.yml` builds every service and the frontend
  and publishes images to GitHub Container Registry on each push to `main`.
- Trigger the workflow manually or build locally:
  ```bash
  SERVICE=api-gateway
  DOCKER_BUILDKIT=1 docker build \
    -f infra/docker/go.Dockerfile \
    --build-arg SERVICE_PATH=${SERVICE} \
    -t synaptica/${SERVICE}:local .

  cd frontend
  DOCKER_BUILDKIT=1 docker build -t synaptica/frontend:local .
  ```

## One-node Hosting (Docker Compose)
`deploy/compose.cloud.yml` runs postgres, redis, kafka, and all Go services + frontend:
```bash
cd deploy
docker compose -f compose.cloud.yml up --build -d
# destroy
docker compose -f compose.cloud.yml down
```

## Free-tier Blueprint
- **Frontend**: Vercel free plan (connect GitHub repo, set `NEXT_PUBLIC_API_BASE`).
- **Backend**: Deploy GHCR images to Fly.io, Railway, Render, or a small VPS.
  - Use managed Postgres/Redis/Kafka from the provider.
  - Mirror the env-vars in `compose.cloud.yml` in your host.

## Database Seeds
Run the SQL files once against the hosted Postgres instance:
```bash
psql "$DATABASE_URL" -f db/schema.sql
psql "$DATABASE_URL" -f db/seed/ingestion_requests.sql
psql "$DATABASE_URL" -f db/seed/normalized_records.sql
psql "$DATABASE_URL" -f db/seed/patient_linkages.sql
psql "$DATABASE_URL" -f db/seed/storage_facts.sql
psql "$DATABASE_URL" -f db/seed/training_jobs.sql
```

## Observability
- Each service exposes `/health` and `/ready` endpoints.
- Hook them into free uptime monitors (UptimeRobot) or Fly.io health checks.

## Local Development against hosted API
- Use `NEXT_PUBLIC_API_BASE=https://api.synaptica.dev` (or your host) when running `npm run dev`.
- Continue iterating locally, push to GitHub, let the CI build and publish containers, then redeploy
  by pulling the latest GHCR tags.
