package store

import (
	"context"
	"database/sql"
	"errors"

	"github.com/jackc/pgx/v5/pgxpool"
	"deploymate/internal/model"
)

var (
	ErrNotFound = errors.New("not found")
)

type Store struct {
	pool *pgxpool.Pool
}

func NewStore(ctx context.Context, databaseURL string) (*Store, error) {
	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, err
	}

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, err
	}

	if err := pool.Ping(ctx); err != nil {
		return nil, err
	}

	return &Store{pool: pool}, nil
}

func (s *Store) GetDesiredState(ctx context.Context, orgID, deploymentID string) (*model.DeploymentSpec, error) {
	query := `
		SELECT id, project_id, environment, service, image, replicas,
		       cpu, memory, env_vars, policy_ref, target_ref, created_at, updated_at
		FROM deployments
		WHERE org_id = $1 AND id = $2
	`

	var spec model.DeploymentSpec
	var cpu, memory string
	var envVars []byte

	err := s.pool.QueryRow(ctx, query, orgID, deploymentID).Scan(
		&spec.ID,
		&spec.ProjectID,
		&spec.Environment,
		&spec.Service,
		&spec.Image,
		&spec.Replicas,
		&cpu,
		&memory,
		&envVars,
		&spec.PolicyRef,
		&spec.TargetRef,
		&spec.CreatedAt,
		&spec.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	spec.Resources = model.ResourceSpec{
		CPU:    cpu,
		Memory: memory,
	}

	return &spec, nil
}

func (s *Store) CreateDeployment(ctx context.Context, spec *model.DeploymentSpec) error {
	query := `
		INSERT INTO deployments (id, org_id, project_id, environment, service, image, replicas,
		                         cpu, memory, env_vars, policy_ref, target_ref, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
	`
	_, err := s.pool.Exec(ctx, query,
		spec.ID,
		spec.OrgID,
		spec.ProjectID,
		spec.Environment,
		spec.Service,
		spec.Image,
		spec.Replicas,
		spec.Resources.CPU,
		spec.Resources.Memory,
		"{}",
		spec.PolicyRef,
		spec.TargetRef,
		spec.CreatedAt,
		spec.UpdatedAt,
	)
	return err
}

func (s *Store) ListDeployments(ctx context.Context, orgID string, envFilter string) ([]model.DeploymentSpec, error) {
	query := `
		SELECT id, project_id, environment, service, image, replicas,
		       cpu, memory, env_vars, policy_ref, target_ref, created_at, updated_at
		FROM deployments
		WHERE org_id = $1
	`
	args := []interface{}{orgID}
	if envFilter != "" {
		query += " AND environment = $2"
		args = append(args, envFilter)
	}
	query += " ORDER BY updated_at DESC"

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var specs []model.DeploymentSpec
	for rows.Next() {
		var spec model.DeploymentSpec
		var cpu, memory string
		var envVars []byte

		if err := rows.Scan(
			&spec.ID,
			&spec.ProjectID,
			&spec.Environment,
			&spec.Service,
			&spec.Image,
			&spec.Replicas,
			&cpu,
			&memory,
			&envVars,
			&spec.PolicyRef,
			&spec.TargetRef,
			&spec.CreatedAt,
			&spec.UpdatedAt,
		); err != nil {
			return nil, err
		}

		spec.Resources = model.ResourceSpec{CPU: cpu, Memory: memory}
		spec.OrgID = orgID
		specs = append(specs, spec)
	}
	return specs, rows.Err()
}

func (s *Store) ListAgents(ctx context.Context, orgID string) ([]model.Agent, error) {
	query := `
		SELECT id, name, target_type, status, last_heartbeat
		FROM agents
		WHERE org_id = $1
		ORDER BY name
	`

	rows, err := s.pool.Query(ctx, query, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var agents []model.Agent
	for rows.Next() {
		var agent model.Agent
		if err := rows.Scan(
			&agent.ID,
			&agent.Name,
			&agent.TargetType,
			&agent.Status,
			&agent.LastHeartbeat,
		); err != nil {
			return nil, err
		}
		agents = append(agents, agent)
	}
	return agents, rows.Err()
}

func (s *Store) UpdateDeploymentStatus(ctx context.Context, deploymentID string, status *model.DeploymentStatus) error {
	query := `
		UPDATE deployments
		SET phase = $1, endpoint = $2, updated_at = NOW()
		WHERE id = $3
	`
	_, err := s.pool.Exec(ctx, query, status.Phase, status.Message, deploymentID)
	return err
}

func (s *Store) UpdateAgentHeartbeat(ctx context.Context, agentID, orgID string) error {
	query := `
		UPDATE agents
		SET last_heartbeat = NOW(), status = 'online', updated_at = NOW()
		WHERE id = $1 AND org_id = $2
	`
	_, err := s.pool.Exec(ctx, query, agentID, orgID)
	return err
}

func (s *Store) GetDeploymentEvents(ctx context.Context, deploymentID string, limit int) ([]model.SSEEvent, error) {
	query := `
		SELECT id, deployment_id, event_type, payload, created_at
		FROM sse_events
		WHERE deployment_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`

	rows, err := s.pool.Query(ctx, query, deploymentID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []model.SSEEvent
	for rows.Next() {
		var event model.SSEEvent
		if err := rows.Scan(
			&event.ID,
			&event.DeploymentID,
			&event.EventType,
			&event.Payload,
			&event.CreatedAt,
		); err != nil {
			return nil, err
		}
		events = append(events, event)
	}
	return events, rows.Err()
}

func (s *Store) Close() {
	s.pool.Close()
}

func (s *Store) GetAgentDeployments(ctx context.Context, agentID string) ([]model.DeploymentSpec, error) {
	// ponytail: two-query lookup because agent-to-deployment is currently indirect via org_id
	// upgrade path: add explicit agent_id FK on deployments once schema permits
	orgQuery := `SELECT org_id FROM agents WHERE id = $1`
	var orgID string
	if err := s.pool.QueryRow(ctx, orgQuery, agentID).Scan(&orgID); err != nil {
		return nil, err
	}

	deployQuery := `
		SELECT id, project_id, environment, service, image, replicas,
		       cpu, memory, env_vars, policy_ref, target_ref,
		       created_at, updated_at
		FROM deployments
		WHERE org_id = $1
		ORDER BY updated_at DESC
	`

	rows, err := s.pool.Query(ctx, deployQuery, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var deployments []model.DeploymentSpec
	for rows.Next() {
		var spec model.DeploymentSpec
		var cpu, memory string
		var envVars []byte

		if err := rows.Scan(
			&spec.ID, &spec.ProjectID, &spec.Environment, &spec.Service,
			&spec.Image, &spec.Replicas, &cpu, &memory, &envVars,
			&spec.PolicyRef, &spec.TargetRef, &spec.CreatedAt, &spec.UpdatedAt,
		); err != nil {
			return nil, err
		}

		spec.Resources = model.ResourceSpec{CPU: cpu, Memory: memory}
		deployments = append(deployments, spec)
	}

	return deployments, nil
}
