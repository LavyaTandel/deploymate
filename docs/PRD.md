# DeployMate — Product Requirements Document

**Version:** 1.0
**Created:** 2026-07-21
**Last Updated:** 2026-07-24
**Status:** MVP Implementation Complete

---

## Table of Contents

1. [Problem Statement](#problem-statement)
2. [Solution](#solution)
3. [User Stories](#user-stories)
4. [Architecture Decisions](#architecture-decisions)
5. [Technical Design](#technical-design)
6. [Implementation Status](#implementation-status)
7. [Phased Delivery](#phased-delivery)
8. [Testing Strategy](#testing-strategy)
9. [Non-Goals](#non-goals)
10. [Open Questions](#open-questions)

---

## Problem Statement

Teams want GitOps deployment ("push to main → deploy") with preview environments, policy-as-code, instant rollback, and cost attribution — without the operational complexity of ArgoCD/Flux or deep Kubernetes expertise. Existing solutions are either too heavy (full K8s operators), too limited (CI/CD pipelines), or lock users into a single cloud provider.

---

## Solution

**DeployMate:** A lightweight GitOps deployment engine that watches Git repos, builds Docker images, and deploys to multiple target types (ECS, Cloud Run, K8s, VMs) via a unified provider interface. Features autonomous edge agents for zero-trust policy enforcement, Sigstore-signed policy bundles, preview environments per PR, and per-service cost tracking — all controlled from a Next.js dashboard with real-time SSE updates.

### Core Value Propositions

1. **Zero-Config GitOps** — Push to main, auto-deploy. No YAML gymnastics.
2. **Multi-Cloud** — Same workflow for Cloud Run, ECS, K8s, VMs.
3. **Policy-as-Code** — OPA Rego policies enforced at central + edge.
4. **Instant Rollback** — One-click or automatic rollback with 3/hr guardrails.
5. **Cost Attribution** — Per-service, per-environment, per-team cost breakdowns.

---

## User Stories

### Core Deployment (5)

| ID | Story | Priority |
|----|-------|----------|
| US-01 | As a platform engineer, I want to connect a Git repo and configure deployment targets (ECS, Cloud Run, GKE, VMs), so pushes to main automatically build and deploy my services. | P0 |
| US-02 | As a developer, I want every PR to auto-spin up a preview environment with a unique URL, so I can share working features before merge. | P0 |
| US-03 | As a devops engineer, I want one-click rollback to the previous image digest, so I can recover from bad deploys in seconds. | P0 |
| US-04 | As a team lead, I want deployment policies as code (OPA Rego) — e.g., "no prod deploy without approval", "cpu limit < 2" — so guardrails are enforced automatically. | P1 |
| US-05 | As a finance manager, I want per-service, per-environment, per-team cost breakdowns updated daily, so I can attribute cloud spend accurately. | P1 |

### Policy & Security (3)

| ID | Story | Priority |
|----|-------|----------|
| US-06 | As a security engineer, I want policy bundles signed with Sigstore keyless signing, so agents verify bundle integrity before applying. | P1 |
| US-07 | As an SRE, I want edge agents to enforce policies locally (cached bundle, 48h staleness max), so deployments continue during engine outages. | P1 |
| US-08 | As a compliance officer, I want an immutable audit log of every deployment, rollback, and policy evaluation, so I can pass SOC2 audits. | P2 |

### Multi-Target (4)

| ID | Story | Priority |
|----|-------|----------|
| US-09 | As a platform engineer, I want to deploy to Google Cloud Run via `gcloud run deploy`, so I can use managed serverless. | P0 |
| US-10 | As a platform engineer, I want to deploy to Kubernetes via client-go, so I can run stateful workloads. | P0 |
| US-11 | As a platform engineer, I want to deploy to AWS ECS via AWS SDK, so I can run containerized workloads on Fargate. | P2 |
| US-12 | As a platform engineer, I want to deploy to VMs via SSH + systemd, so I can run legacy applications. | P3 |

### Observability (3)

| ID | Story | Priority |
|----|-------|----------|
| US-13 | As an on-call engineer, I want real-time deployment status via SSE, so I can monitor rollouts without refreshing. | P1 |
| US-14 | As an on-call engineer, I want automatic rollback when health checks or OPA drift scans fail, with max-3-per-hour guardrail. | P1 |
| US-15 | As a platform engineer, I want OpenTelemetry auto-instrumentation for all engine components, so I can trace requests end-to-end. | P2 |

---

## Architecture Decisions

### AD-01: Hybrid Push/Pull Execution

**Decision:** Engine pushes desired state to agents; agents pull and reconcile locally.
**Rationale:** Push for immediacy (rollback), pull for resilience (agent continues during engine outage).
**Trade-off:** Slightly more complex than pure push/pull, but handles both speed and reliability requirements.

### AD-02: Chi Router (Go)

**Decision:** Use `go-chi/chi` over Gin/Echo.
**Rationale:** Standard library compatible, lightweight, excellent middleware ecosystem. Gin benchmarks faster but Chi is "Go-idiomatic" — accepts interfaces, returns structs.
**Trade-off:** ~5% slower raw throughput vs Gin, but better maintainability.

### AD-03: Redis Cache (Agent Polling)

**Decision:** Redis with 5s TTL for desired-state cache.
**Rationale:** Agents poll every 10s; 5s TTL ensures freshness without hammering Postgres.
**Trade-off:** Extra infrastructure dependency, but Redis is already in most stacks.

### AD-04: Dual OPA Evaluation

**Decision:** Central OPA (engine) + Edge OPA (agent) evaluation.
**Rationale:** Central for pre-deploy gates, edge for runtime drift detection. Edge uses cached bundle (max 48h staleness).
**Trade-off:** Dual eval = potential inconsistency, but edge cache ensures availability.

### AD-05: Sigstore Keyless Signing

**Decision:** Use Sigstore/Fulcio for policy bundle signing (OIDC-based, no key management).
**Rationale:** Zero key management overhead. Agent verifies bundle integrity before applying.
**Trade-off:** Requires internet access for initial signing, but bundles are cached locally.

### AD-06: State Segregation

**Decision:** Git = desired state source of truth, DB = runtime state + audit trail.
**Rationale:** Git provides version control and audit; DB provides fast queries and SSE event history.
**Trade-off:** Dual source of truth requires reconciliation, but each system does what it's best at.

### AD-07: Provider Interface

**Decision:** Unified `Deployer` interface: Deploy/Rollback/Status/Destroy.
**Rationale:** Each cloud provider implements the same interface. Engine is provider-agnostic.
**Trade-off:** Abstracts away provider-specific features, but ensures consistent behavior.

### AD-08: Autonomous Rollback (3/hr Guardrail)

**Decision:** Auto-rollback on health check failure or OPA drift detection, max 3 per hour per environment.
**Rationale:** Self-healing without human intervention, but guardrails prevent rollback storms.
**Trade-off:** May delay rollback if guardrail hit, but prevents cascading failures.

### AD-09: Provider-Native Traffic Routing

**Decision:** Use provider-native traffic splitting (Cloud Run revisions, ECS blue/green) + optional Envoy Gateway.
**Rationale:** Native routing is simpler and more reliable than custom load balancers.
**Trade-off:** Vendor-specific, but provider interface abstracts this.

### AD-10: Cost Attribution via Agent Telemetry

**Decision:** Agents report resource usage via OTel metrics; engine aggregates per-service costs.
**Rationale:** Agent-level visibility into actual resource consumption, not just billing API estimates.
**Trade-off:** Requires OTel infrastructure, but provides granular cost data.

### AD-11: Preview Environment Lifecycle

**Decision:** PR open → create preview env; PR close → destroy preview env.
**Rationale:** Automatic cleanup prevents resource sprawl. Unique URLs per PR for sharing.
**Trade-off:** Preview envs consume resources during PR review, but auto-cleanup limits waste.

### AD-12: SSE (Not WebSockets)

**Decision:** Server-Sent Events for real-time dashboard updates.
**Rationale:** Asymmetrical data flow (server → client) fits SSE better. Simpler than WebSockets.
**Trade-off:** No client→server real-time channel, but REST POSTs handle that.

### AD-13: Open-Core Business Model

**Decision:** Core engine + K8s agent open source; dashboard + enterprise features commercial.
**Rationale:** Community adoption via open source; revenue via premium features (SSO, RBAC, audit export).
**Trade-off:** Must maintain clear feature boundary between open and commercial.

### AD-14: PostgreSQL for Runtime State

**Decision:** PostgreSQL for deployments, agents, policy bundles, SSE events, rollback guardrails.
**Rationale:** ACID compliance, JSONB for flexible schemas, mature ecosystem.
**Trade-off:** Heavier than SQLite, but necessary for production multi-tenancy.

### AD-15: Docker Multi-Stage Builds

**Decision:** Multi-stage Dockerfiles: Go build → distroless runtime.
**Rationale:** Small attack surface (distroless), fast builds (Go cache).
**Trade-off:** No shell in runtime (debugging harder), but security benefit outweighs.

### AD-16: OTel Auto-Instrumentation

**Decision:** OpenTelemetry SDK with auto-instrumentation for HTTP, gRPC, database.
**Rationale:** Vendor-neutral observability, works with Jaeger/Zipkin/Datadog/etc.
**Trade-off:** Slight runtime overhead (~1-2%), but observability benefit is critical.

### AD-17: Snapshot Metric Gates

**Decision:** Pre-deploy: snapshot current metrics; post-deploy: compare; auto-rollback if degradation detected.
**Rationale:** Catch performance regressions before they reach users.
**Trade-off:** Adds latency to deploy pipeline, but prevents degraded deployments.

### AD-18: VM Graceful Reload + Low-Mem Pre-Check

**Decision:** For VM deploys: check available memory before reload, graceful SIGTERM with drain period.
**Rationale:** Prevents OOM kills during deployment, ensures in-flight requests complete.
**Trade-off:** More complex deploy logic, but prevents downtime on resource-constrained VMs.

---

## Technical Design

### Project Structure

```
deploymate/
├── cmd/
│   ├── engine/          # Main API server
│   │   └── main.go
│   └── agent/           # K8s edge agent
│       └── main.go
├── internal/
│   ├── agent/           # Agent reconciler
│   │   └── reconciler.go
│   ├── auth/            # OIDC authentication
│   │   ├── oidc.go
│   │   └── oidc_test.go
│   ├── cache/           # Redis cache layer
│   │   ├── cache.go
│   │   └── cache_test.go
│   ├── config/          # Configuration
│   │   └── config.go
│   ├── handler/         # HTTP handlers
│   │   ├── desired_state.go
│   │   └── desired_state_test.go
│   ├── model/           # Domain models
│   │   ├── types.go
│   │   └── types_test.go
│   ├── provider/        # Deployer interface + implementations
│   │   ├── deployer.go
│   │   └── cloudrun.go
│   └── store/           # PostgreSQL store
│       └── postgres.go
├── migrations/          # Database migrations
│   └── 001_initial.sql
├── pkg/                 # Shared packages
├── docs/                # Documentation
│   └── PRD.md
├── Dockerfile.engine    # Engine container
├── Dockerfile.agent     # Agent container
├── .dockerignore
├── go.mod
└── go.sum
```

### API Endpoints

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/healthz` | No | Health check |
| GET | `/api/v1/deployments/{id}/desired-state` | OIDC | Get desired state (cache-first) |
| POST | `/api/v1/deployments/{id}/rollback` | OIDC | Trigger rollback |
| GET | `/api/v1/events` | No | SSE event stream |
| GET | `/api/v1/agents/{agentID}/deployments` | OIDC | Agent deployment list |
| POST | `/api/v1/agents/{agentID}/deployments/{id}/status` | OIDC | Agent status report |

### Deployer Interface

```go
type Deployer interface {
    Deploy(ctx context.Context, spec model.DeploymentSpec) (model.DeploymentResult, error)
    Rollback(ctx context.Context, deploymentID string) error
    Status(ctx context.Context, deploymentID string) (model.DeploymentStatus, error)
    Destroy(ctx context.Context, deploymentID string) error
}
```

### Database Schema

**Tables:**
- `organizations` — Billing boundary, SSO config
- `projects` — Git repo mapping, team RBAC
- `environments` — Production, staging, preview
- `deployments` — Runtime state + desired state
- `deployment_history` — Audit trail
- `agents` — Edge agent registry
- `policy_bundles` — Signed OPA bundles
- `sse_events` — Event replay cache
- `rollback_events` — Guardrail tracking

### K8s Agent Architecture

```
┌─────────────────────────────────────────┐
│  DeployMate Engine (Cloud)              │
│  ┌─────────┐  ┌─────────┐  ┌─────────┐│
│  │ Chi API │  │ Redis   │  │ Postgres ││
│  └────┬────┘  └────┬────┘  └────┬────┘│
│       └────────────┼────────────┘      │
└────────────────────┼───────────────────┘
                     │ HTTP (Bearer Token)
┌────────────────────┼───────────────────┐
│  K8s Agent (Cluster)                   │
│  ┌─────────────────┴─────────────────┐ │
│  │  Polling Loop (10s)               │ │
│  │  GET /desired-state               │ │
│  │  POST /status                     │ │
│  └─────────────────┬─────────────────┘ │
│  ┌─────────────────┴─────────────────┐ │
│  │  Reconciler (client-go)           │ │
│  │  - Create/Update Deployment       │ │
│  │  - Create/Update Service          │ │
│  │  - Report Status                  │ │
│  └───────────────────────────────────┘ │
└─────────────────────────────────────────┘
```

---

## Implementation Status

### Phase 1: MVP (Weeks 1-6) — ✅ COMPLETE

| Component | Status | Files |
|-----------|--------|-------|
| Go module + project structure | ✅ | `go.mod`, `go.sum` |
| Chi router + middleware | ✅ | `cmd/engine/main.go` |
| OIDC authentication | ✅ | `internal/auth/oidc.go`, `oidc_test.go` |
| Redis cache layer | ✅ | `internal/cache/cache.go`, `cache_test.go` |
| PostgreSQL store | ✅ | `internal/store/postgres.go` |
| /desired-state endpoint | ✅ | `internal/handler/desired_state.go`, `desired_state_test.go` |
| Rollback handler | ✅ | `internal/handler/desired_state.go` |
| SSE events handler | ✅ | `internal/handler/desired_state.go` |
| Deployer interface | ✅ | `internal/provider/deployer.go` |
| Cloud Run provider | ✅ | `internal/provider/cloudrun.go` |
| K8s agent + reconciler | ✅ | `cmd/agent/main.go`, `internal/agent/reconciler.go` |
| Domain models | ✅ | `internal/model/types.go`, `types_test.go` |
| Configuration | ✅ | `internal/config/config.go` |
| Database migrations | ✅ | `migrations/001_initial.sql` |
| Dockerfiles | ✅ | `Dockerfile.engine`, `Dockerfile.agent` |
| Unit tests | ✅ | 4 test suites, all passing |

### Build Status

```
go build ./...  → EXIT:0
go test ./...   → 4/4 PASS
go vet ./...    → EXIT:0
```

### Next Phases (Pending)

| Phase | Weeks | Features |
|-------|-------|----------|
| Policy Hardening | 7-10 | Edge OPA, Sigstore signing, bundle distribution, dual-eval |
| Multi-Target | 11-14 | ECS provider, VM agent (NGINX), preview env automation |
| Observability | 15-18 | SSE dashboard (Next.js), cost pipeline, auto-rollback, air-gap mode |
| Enterprise | 19+ | SSO, RBAC, BYO Rekor, audit export, Terraform provider |

---

## Testing Strategy

### Unit Tests

| Package | Coverage | Pattern |
|---------|----------|---------|
| `internal/auth` | OIDC validation, token expiry, signature verification | Table-driven with t.Run |
| `internal/cache` | Get/Set/Del round-trips, cache miss, desired state operations | Table-driven with mock |
| `internal/handler` | HTTP handlers, cache hit/miss, error responses, SSE headers | httptest.NewRecorder |
| `internal/model` | JSON serialization round-trips | Table-driven |

### Integration Tests (Planned)

- Engine ↔ PostgreSQL (testcontainers)
- Engine ↔ Redis (testcontainers)
- Agent ↔ Engine (mock HTTP server)
- Agent ↔ K8s (envtest)

### E2E Tests (Planned)

- Full deploy cycle: Git push → build → deploy → verify
- Rollback flow: deploy → rollback → verify
- Preview env: PR open → create → PR close → destroy

---

## Non-Goals

- **Multi-cloud VPC peering** — Does not configure networking between clouds.
- **Transit Gateway / Service Connect** — No service mesh configuration.
- **ApplicationSet patterns** — Single repo per project, no ApplicationSet generator.
- **ML-based cost forecasting** — Rule-based projection only.
- **Custom load balancers** — Uses provider-native routing.

---

## Open Questions

1. **Engine API auth:** OIDC tokens (recommended over mTLS for air-gap compatibility).
2. **Terraform provider scope:** Strictly configuration (Projects, Targets, Policies) — NOT deployment state.
3. **Cost ingestion at launch:** AWS and GCP only (proven schemas). Azure in Enterprise phase.
4. **Dashboard real-time:** SSE + RESTful POSTs (not WebSockets) — asymmetrical data flow fits better.

---

## Competitive Landscape

| Feature | DeployMate | ArgoCD | Flux | Drone | GitHub Actions |
|---------|------------|--------|------|-------|----------------|
| Multi-cloud | ✅ | ❌ (K8s only) | ❌ (K8s only) | ❌ | ❌ |
| Preview envs | ✅ | ❌ | ❌ | ❌ | Partial |
| Policy-as-code | ✅ (OPA) | ❌ | ❌ | ❌ | ❌ |
| Cost attribution | ✅ | ❌ | ❌ | ❌ | ❌ |
| One-click rollback | ✅ | ✅ | ✅ | ❌ | ❌ |
| Self-hosted | ✅ | ✅ | ✅ | ✅ | ❌ |
| Complexity | Low | High | Medium | Low | Low |

---

## Business Model

### Open Core

| Feature | Open Source | Commercial |
|---------|-------------|------------|
| Engine API | ✅ | ✅ |
| K8s Agent | ✅ | ✅ |
| Cloud Run Provider | ✅ | ✅ |
| Policy Engine (OPA) | ✅ | ✅ |
| Sigstore Signing | ✅ | ✅ |
| Dashboard (Next.js) | ❌ | ✅ |
| ECS Provider | ❌ | ✅ |
| VM Agent | ❌ | ✅ |
| SSO / RBAC | ❌ | ✅ |
| Audit Export | ❌ | ✅ |
| Terraform Provider | ❌ | ✅ |
| Priority Support | ❌ | ✅ |

### Pricing Tiers

| Tier | Price | Features |
|------|-------|----------|
| Community | Free | Engine + K8s + Cloud Run + OPA |
| Pro | $99/mo | + Dashboard + ECS + VM + Preview Envs |
| Enterprise | $499/mo | + SSO + RBAC + Audit + Terraform + SLA |

---

*This PRD is a living document. Update as implementation progresses.*
