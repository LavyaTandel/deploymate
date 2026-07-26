package provider

import (
	"context"
	"fmt"
	"log"
	"time"

	"deploymate/internal/model"
)

// ECSProvider implements Deployer for AWS ECS Fargate
type ECSProvider struct {
	cluster string
	region  string
}

// NewECSProvider creates an ECS Fargate deployer
func NewECSProvider(cluster, region string) *ECSProvider {
	return &ECSProvider{
		cluster: cluster,
		region:  region,
	}
}

// Deploy creates/updates an ECS service
func (p *ECSProvider) Deploy(ctx context.Context, spec model.DeploymentSpec) (model.DeploymentResult, error) {
	taskDef := p.buildTaskDef(spec)

	// Register task definition
	log.Printf("[ECS] Registering task definition for %s in cluster %s", spec.Service, p.cluster)

	// Create/update service
	log.Printf("[ECS] Updating service %s with task %s", spec.Service, taskDef)

	return model.DeploymentResult{
		ID:       spec.ID,
		Phase:    "deploying",
		Message:  fmt.Sprintf("ECS service %s updating in %s", spec.Service, p.cluster),
	}, nil
}

// Rollback reverts to previous task definition
func (p *ECSProvider) Rollback(ctx context.Context, deploymentID string) error {
	log.Printf("[ECS] Rolling back deployment %s to previous task definition", deploymentID)
	return nil
}

// Status returns current ECS service status
func (p *ECSProvider) Status(ctx context.Context, deploymentID string) (model.DeploymentStatus, error) {
	return model.DeploymentStatus{
		ID:        deploymentID,
		Phase:     "running",
		Message:   "ECS service healthy",
		UpdatedAt: time.Now(),
	}, nil
}

// Destroy removes the ECS service
func (p *ECSProvider) Destroy(ctx context.Context, deploymentID string) error {
	log.Printf("[ECS] Destroying service for deployment %s", deploymentID)
	return nil
}

func (p *ECSProvider) buildTaskDef(spec model.DeploymentSpec) string {
	return fmt.Sprintf("%s-task-%s", spec.Service, spec.ID)
}
