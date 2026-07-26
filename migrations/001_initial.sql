-- DeployMate initial schema
-- Run: psql -d deploymate -f migrations/001_initial.sql

BEGIN;

-- Organizations (billing boundary, SSO config, global policies)
CREATE TABLE IF NOT EXISTS organizations (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        VARCHAR(255) NOT NULL UNIQUE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Projects (git repo mapping, team RBAC, default target)
CREATE TABLE IF NOT EXISTS projects (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id      UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name        VARCHAR(255) NOT NULL,
    repo_url    VARCHAR(1024),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(org_id, name)
);

-- Environments (production, staging, preview)
CREATE TABLE IF NOT EXISTS environments (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id  UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    name        VARCHAR(100) NOT NULL,
    target_type VARCHAR(50) NOT NULL,  -- cloudrun, ecs, k8s, vm
    policy_ref  VARCHAR(255),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(project_id, name)
);

-- Deployments (runtime state + desired state)
CREATE TABLE IF NOT EXISTS deployments (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id      UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    project_id  UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    env_id      UUID REFERENCES environments(id),
    environment VARCHAR(100) NOT NULL,
    service     VARCHAR(255) NOT NULL,
    image       VARCHAR(1024) NOT NULL,
    replicas    INT NOT NULL DEFAULT 1,
    cpu         VARCHAR(50),
    memory      VARCHAR(50),
    env_vars    JSONB DEFAULT '{}',
    policy_ref  VARCHAR(255),
    target_ref  VARCHAR(255),
    phase       VARCHAR(50) DEFAULT 'pending',
    endpoint    VARCHAR(1024),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_deployments_org_id ON deployments(org_id);
CREATE INDEX idx_deployments_project_id ON deployments(project_id);
CREATE INDEX idx_deployments_env ON deployments(environment);

-- Deployment history (audit trail)
CREATE TABLE IF NOT EXISTS deployment_history (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    deployment_id   UUID NOT NULL REFERENCES deployments(id) ON DELETE CASCADE,
    image           VARCHAR(1024) NOT NULL,
    phase           VARCHAR(50) NOT NULL,
    message         TEXT,
    triggered_by    VARCHAR(255),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_deployment_history_deployment_id ON deployment_history(deployment_id);

-- Agents (edge agent registry)
CREATE TABLE IF NOT EXISTS agents (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id          UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name            VARCHAR(255) NOT NULL,
    target_type     VARCHAR(50) NOT NULL,
    target_ref      VARCHAR(1024),
    status          VARCHAR(50) DEFAULT 'offline',
    last_heartbeat  TIMESTAMPTZ,
    metadata        JSONB DEFAULT '{}',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_agents_org_id ON agents(org_id);

-- Policy bundles (signed OPA bundles)
CREATE TABLE IF NOT EXISTS policy_bundles (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    version     VARCHAR(50) NOT NULL,
    sha256      VARCHAR(64) NOT NULL,
    signature   TEXT,
    cert_pem    TEXT,
    bundle_url  VARCHAR(1024) NOT NULL,
    expires_at  TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(version)
);

-- SSE event replay cache (last 50 per deployment)
CREATE TABLE IF NOT EXISTS sse_events (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    deployment_id   UUID NOT NULL,
    event_type      VARCHAR(100) NOT NULL,
    payload         JSONB NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_sse_events_deployment_id ON sse_events(deployment_id, created_at DESC);

-- Rollback guardrails (max 3 per hour per environment)
CREATE TABLE IF NOT EXISTS rollback_events (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    deployment_id   UUID NOT NULL REFERENCES deployments(id) ON DELETE CASCADE,
    environment     VARCHAR(100) NOT NULL,
    reason          TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_rollback_events_env_time ON rollback_events(environment, created_at DESC);

COMMIT;
