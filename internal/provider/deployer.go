package provider

import (
	"context"

	"deploymate/internal/model"
)

// Deployer defines the contract for a deployment provider.
// Each cloud provider (Cloud Run, ECS, Azure Container Apps, etc.)
// implements this interface to perform lifecycle operations against its target.
type Deployer interface {
	// Deploy pushes a container image and creates or updates a service
	// according to the given spec. It returns a result containing the
	// deployment ID, the service endpoint, and the resolved image digest.
	Deploy(ctx context.Context, spec model.DeploymentSpec) (model.DeploymentResult, error)

	// Rollback reverts a deployment identified by id to its previous
	// healthy revision. The exact semantics are provider-specific: for
	// Cloud Run this means rolling back to the most recent non-current
	// revision; for other providers it may mean restoring a snapshot.
	Rollback(ctx context.Context, id string) error

	// Status returns the current deployment status for the given id,
	// including phase, message, and progress information.
	Status(ctx context.Context, id string) (model.DeploymentStatus, error)

	// Destroy permanently removes the deployment identified by id and
	// all of its associated resources (service, revisions, IAM bindings).
	Destroy(ctx context.Context, id string) error
}
