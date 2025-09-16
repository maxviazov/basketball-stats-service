# Deployment

This project ships as a stateless container. You can run it locally with Docker Compose or deploy to a cloud platform orchestrator.

## Local (Docker Compose)

Prerequisites: Docker, Docker Compose.

1) Copy environment template and adjust if needed:
```bash
cp .env.example .env
```
2) Start the stack (Postgres + app):
```bash
docker compose up -d --build
```
3) Apply DB migrations (one-time):
```bash
make migrate-up
```
4) Open the app:
- http://localhost:8080/docs (Swagger UI)
- http://localhost:8080/openapi.yaml (OpenAPI)

## Production (cloud sketch)

A minimal production setup on AWS could look like this:

- Networking & LB: Amazon ALB (or NLB) terminates TLS and forwards to app tasks.
- Compute: Amazon ECS (Fargate) or EKS (Kubernetes) runs stateless app containers with desired count ≥ 2.
- Database: Amazon RDS for PostgreSQL with multi-AZ, automated backups, and monitoring.
- Secrets: AWS Secrets Manager for DB credentials; inject via task definitions / env.
- Observability: CloudWatch Logs + metrics (container and app), alarms on error rate/latency.
- CI/CD: GitHub Actions builds/pushes the Docker image to ECR and updates ECS service.

### ECS deployment flow
1. GitHub Actions builds the image (multi-stage Dockerfile) and pushes to ECR.
2. Task definition references the ECR image and injects env vars (APP_POSTGRES_*).
3. ECS service with desired count (e.g., 3) behind an ALB target group.
4. ALB health checks call /ready.

### Kubernetes (EKS) flow
- Deployments with HPA (CPU/latency based), Service (ClusterIP) + Ingress (ALB Ingress Controller).
- Config via ConfigMap/Secret; mount or env-inject APP_POSTGRES_*.
- Liveness/readiness probes pointing to /live and /ready.

## Scaling & performance notes
- App is stateless; horizontal scaling is straightforward.
- Tune pgx pool via config (max/min conns, timeouts) per environment and DB capacity.
- Use connection reuse and avoid long transactions; upserts are idempotent and short-lived.

## Migrations in CI/CD
- Apply goose migrations as a dedicated step before rolling out a new app version.
- For zero-downtime, keep migrations backward-compatible with the previous app version.
