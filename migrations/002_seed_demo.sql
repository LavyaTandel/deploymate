-- DeployMate demo seed data
-- Run: psql -d deploymate -f migrations/002_seed_demo.sql

BEGIN;

-- Demo organization
INSERT INTO organizations (id, name, created_at, updated_at)
VALUES ('550e8400-e29b-41d4-a716-446655440000', 'Demo Corp', NOW(), NOW())
ON CONFLICT (id) DO NOTHING;

-- Demo project
INSERT INTO projects (id, org_id, name, repo_url, created_at, updated_at)
VALUES ('550e8400-e29b-41d4-a716-446655440001', '550e8400-e29b-41d4-a716-446655440000', 'web-app', 'https://github.com/demo/web-app', NOW(), NOW())
ON CONFLICT (id) DO NOTHING;

-- Demo environments
INSERT INTO environments (id, project_id, name, target_type, policy_ref, created_at, updated_at)
VALUES
  ('550e8400-e29b-41d4-a716-446655440010', '550e8400-e29b-41d4-a716-446655440001', 'production', 'cloudrun', 'default', NOW(), NOW()),
  ('550e8400-e29b-41d4-a716-446655440011', '550e8400-e29b-41d4-a716-446655440001', 'staging', 'cloudrun', 'default', NOW(), NOW())
ON CONFLICT (project_id, name) DO NOTHING;

-- Demo deployments
INSERT INTO deployments (id, org_id, project_id, environment, service, image, replicas, cpu, memory, env_vars, policy_ref, target_ref, phase, endpoint, created_at, updated_at)
VALUES
  ('550e8400-e29b-41d4-a716-446655440020', '550e8400-e29b-41d4-a716-446655440000', '550e8400-e29b-41d4-a716-446655440001', 'production', 'api-gateway', 'gcr.io/demo/api-gateway:v1.2.3', 3, '500m', '512Mi', '{"LOG_LEVEL":"info"}', 'default', 'cloudrun/api-gateway', 'running', 'https://api-gateway-abc123.run.app', NOW() - INTERVAL '2 hours', NOW()),
  ('550e8400-e29b-41d4-a716-446655440021', '550e8400-e29b-41d4-a716-446655440000', '550e8400-e29b-41d4-a716-446655440001', 'production', 'auth-service', 'gcr.io/demo/auth-service:v2.1.0', 2, '250m', '256Mi', '{"DB_HOST":"postgres"}', 'default', 'cloudrun/auth-service', 'deploying', 'https://auth-service-def456.run.app', NOW() - INTERVAL '30 minutes', NOW()),
  ('550e8400-e29b-41d4-a716-446655440022', '550e8400-e29b-41d4-a716-446655440000', '550e8400-e29b-41d4-a716-446655440001', 'staging', 'payment-worker', 'gcr.io/demo/payment:v3.0.0', 1, '100m', '128Mi', '{}', 'default', 'cloudrun/payment-worker', 'failed', '', NOW() - INTERVAL '1 hour', NOW())
ON CONFLICT (id) DO NOTHING;

-- Demo agents
INSERT INTO agents (id, org_id, name, target_type, status, last_heartbeat, created_at, updated_at)
VALUES
  ('550e8400-e29b-41d4-a716-446655440030', '550e8400-e29b-41d4-a716-446655440000', 'prod-agent-1', 'cloudrun', 'online', NOW() - INTERVAL '5 seconds', NOW(), NOW()),
  ('550e8400-e29b-41d4-a716-446655440031', '550e8400-e29b-41d4-a716-446655440000', 'staging-agent-1', 'cloudrun', 'online', NOW() - INTERVAL '10 seconds', NOW(), NOW()),
  ('550e8400-e29b-41d4-a716-446655440032', '550e8400-e29b-41d4-a716-446655440000', 'backup-agent', 'k8s', 'offline', NOW() - INTERVAL '2 hours', NOW(), NOW())
ON CONFLICT (id) DO NOTHING;

-- Demo deployment history
INSERT INTO deployment_history (deployment_id, image, phase, message, triggered_by, created_at)
VALUES
  ('550e8400-e29b-41d4-a716-446655440020', 'gcr.io/demo/api-gateway:v1.2.2', 'running', 'Previous deployment', 'user@demo.com', NOW() - INTERVAL '3 hours'),
  ('550e8400-e29b-41d4-a716-446655440020', 'gcr.io/demo/api-gateway:v1.2.3', 'running', 'Current deployment', 'user@demo.com', NOW() - INTERVAL '2 hours')
ON CONFLICT DO NOTHING;

-- Demo policy bundle
INSERT INTO policy_bundles (id, version, sha256, signature, cert_pem, expires_at, created_at)
VALUES ('550e8400-e29b-41d4-a716-446655440040', 'v1.0.0', 'abc123def456', 'sig-placeholder', 'cert-placeholder', NOW() + INTERVAL '30 days', NOW())
ON CONFLICT (version) DO NOTHING;

COMMIT;
