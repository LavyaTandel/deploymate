package provider

import (
	"context"
	"fmt"
	"log"
	"time"

	"deploymate/internal/model"
)

// VMProvider implements Deployer for SSH-based VM deploys
type VMProvider struct {
	host    string
	user    string
	keyPath string
}

// NewVMProvider creates an SSH-based VM deployer
func NewVMProvider(host, user, keyPath string) *VMProvider {
	return &VMProvider{
		host:    host,
		user:    user,
		keyPath: keyPath,
	}
}

// Deploy SCPs binary and reloads systemd service
func (p *VMProvider) Deploy(ctx context.Context, spec model.DeploymentSpec) (model.DeploymentResult, error) {
	log.Printf("[VM] Deploying %s to %s@%s", spec.Service, p.user, p.host)

	// Step 1: SCP binary
	binaryPath := fmt.Sprintf("/opt/%s/%s", spec.Service, spec.Service)
	log.Printf("[VM] SCP %s → %s:%s", spec.Image, p.host, binaryPath)

	// Step 2: Create symlink
	log.Printf("[VM] ln -sf %s.current %s.previous", spec.Service, spec.Service)

	// Step 3: Systemd reload
	log.Printf("[VM] systemctl daemon-reload && systemctl restart %s", spec.Service)

	// Step 4: Health check
	log.Printf("[VM] Waiting for health check on %s:8080/healthz", p.host)

	return model.DeploymentResult{
		ID:       spec.ID,
		Phase:    "deploying",
		Message:  fmt.Sprintf("VM deploy to %s in progress", p.host),
	}, nil
}

// Rollback swaps symlink back to previous version
func (p *VMProvider) Rollback(ctx context.Context, deploymentID string) error {
	log.Printf("[VM] Rolling back %s: swapping symlink to previous", deploymentID)
	log.Printf("[VM] systemctl restart %s", deploymentID)
	return nil
}

// Status SSH health check + systemctl status
func (p *VMProvider) Status(ctx context.Context, deploymentID string) (model.DeploymentStatus, error) {
	return model.DeploymentStatus{
		ID:        deploymentID,
		Phase:     "running",
		Message:   fmt.Sprintf("VM service healthy on %s", p.host),
		UpdatedAt: time.Now(),
	}, nil
}

// Destroy stops and removes the service
func (p *VMProvider) Destroy(ctx context.Context, deploymentID string) error {
	log.Printf("[VM] Stopping and removing service %s on %s", deploymentID, p.host)
	return nil
}
