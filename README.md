# DeployMate

A lightweight GitOps deployment engine that watches Git repos, builds Docker images, and deploys to multiple target types (ECS, Cloud Run, K8s, VMs) via a unified provider interface.

## Features

- **Multi-Cloud Deployments** — Cloud Run, ECS Fargate, Kubernetes, VMs
- **Policy-as-Code** — OPA Rego policies with bundle signing
- **Preview Environments** — PR-triggered, auto-destroy on close
- **Real-Time Dashboard** — SSE event streaming, dark mode UI
- **Instant Rollback** — One-click or automatic with 3/hr guardrails
- **Edge Agents** — Zero-trust policy enforcement at the edge

## Architecture

```
┌─────────────────────────────────────────────────────────┐
│  DeployMate Engine (Go)                                 │
│  ┌─────────┐  ┌─────────┐  ┌─────────┐  ┌──────────┐ │
│  │ Chi API │  │ Redis   │  │ Postgres │  │ OPA      │ │
│  └────┬────┘  └────┬────┘  └────┬────┘  └────┬─────┘ │
│       └────────────┼────────────┼─────────────┘       │
└────────────────────┼────────────┼─────────────────────┘
                     │            │
    ┌────────────────┼────────────┼────────────────┐
    │                │            │                │
    ▼                ▼            ▼                ▼
┌────────┐    ┌──────────┐  ┌──────────┐    ┌──────────┐
│ K8s    │    │ Cloud    │  │ ECS      │    │ VM       │
│ Agent  │    │ Run      │  │ Fargate  │    │ SSH      │
└────────┘    └──────────┘  └──────────┘    └──────────┘
```

## Quick Start

### Prerequisites

- Go 1.23+
- PostgreSQL 15+
- Redis 7+
- Node.js 18+ (for dashboard)

### Engine API

```bash
# Start PostgreSQL and Redis
docker-compose up -d

# Run migrations
psql -d deploymate -f migrations/001_initial.sql

# Start engine
go run ./cmd/engine
```

### K8s Agent

```bash
# Build agent
go build -o agent ./cmd/agent

# Run in cluster
./agent
```

### Dashboard

```bash
cd apps/dashboard
npm install
npm run dev
```

## Project Structure

```
deploymate/
├── cmd/
│   ├── engine/          # Main API server
│   └── agent/           # K8s edge agent
├── internal/
│   ├── agent/           # Agent reconciler
│   ├── auth/            # OIDC authentication
│   ├── cache/           # Redis cache layer
│   ├── config/          # Configuration
│   ├── handler/         # HTTP handlers
│   ├── model/           # Domain models
│   ├── policy/          # OPA policy engine
│   ├── preview/         # Preview environments
│   ├── provider/        # Deployer implementations
│   └── store/           # PostgreSQL store
├── apps/dashboard/      # Next.js dashboard
├── migrations/          # Database migrations
├── docs/                # Documentation
├── Dockerfile.engine    # Engine container
└── Dockerfile.agent     # Agent container
```

## API Endpoints

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/healthz` | No | Health check |
| GET | `/api/v1/deployments/{id}/desired-state` | OIDC | Get desired state |
| POST | `/api/v1/deployments/{id}/rollback` | OIDC | Trigger rollback |
| GET | `/api/v1/events` | No | SSE event stream |
| POST | `/api/v1/webhooks/github` | HMAC | GitHub webhook |

## Configuration

Environment variables:

```bash
# Server
SERVER_HOST=0.0.0.0
SERVER_PORT=8080

# Database
DB_HOST=localhost
DB_PORT=5432
DB_USER=deploymate
DB_PASSWORD=deploymate
DB_NAME=deploymate

# Redis
REDIS_HOST=localhost
REDIS_PORT=6379

# OIDC
OIDC_ISSUER=https://token.actions.githubusercontent.com
OIDC_AUDIENCE=deploymate
```

## Testing

```bash
# Run all tests
go test ./internal/... -v

# Run with coverage
go test ./internal/... -cover

# Run specific package
go test ./internal/policy/... -v
```

## Deployment

### Docker

```bash
# Build engine
docker build -f Dockerfile.engine -t deploymate/engine .

# Build agent
docker build -f Dockerfile.agent -t deploymate/agent .
```

### Kubernetes

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: deploymate-engine
spec:
  replicas: 2
  selector:
    matchLabels:
      app: deploymate-engine
  template:
    metadata:
      labels:
        app: deploymate-engine
    spec:
      containers:
      - name: engine
        image: deploymate/engine:latest
        ports:
        - containerPort: 8080
```

## Architecture Decisions

See [docs/PRD.md](docs/PRD.md) for full architecture decisions (18 total).

Key decisions:
- **Hybrid Push/Pull** — Engine pushes, agents pull for resilience
- **Dual OPA Eval** — Central + edge policy enforcement
- **Sigstore Signing** — Keyless bundle verification
- **SSE over WebSockets** — Simpler, asymmetrical data flow

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests
5. Submit a pull request

## License

MIT
