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

func (s *Store) Close() {
	s.pool.Close()
}
