package provider

import (
	"context"
	"os/exec"
	"deploymate/internal/model"
	"testing"
)

// Compile-time interface checks
var (
	_ Deployer = (*CloudRunProvider)(nil)
	_ Deployer = (*ECSProvider)(nil)
	_ Deployer = (*VMProvider)(nil)
)

func hasGcloud() bool {
	_, err := exec.LookPath("gcloud")
	return err == nil
}

func TestCloudRunProvider_Deploy(t *testing.T) {
	if !hasGcloud() {
		t.Skip("gcloud not installed, skipping")
	}
	p := NewCloudRunProvider("my-project", "us-central1")
	spec := model.DeploymentSpec{ID: "test-1", Service: "api", Image: "gcr.io/proj/api:v1", Replicas: 3}
	result, err := p.Deploy(context.Background(), spec)
	if err != nil {
		t.Fatalf("Deploy() error = %v", err)
	}
	if result.ID != "test-1" {
		t.Errorf("ID = %q, want %q", result.ID, "test-1")
	}
}

func TestECSProvider_Deploy(t *testing.T) {
	p := NewECSProvider("prod-cluster", "us-east-1")
	spec := model.DeploymentSpec{ID: "test-2", Service: "worker", Image: "123456.dkr.ecr.us-east-1.amazonaws.com/worker:v1", Replicas: 2}
	result, err := p.Deploy(context.Background(), spec)
	if err != nil {
		t.Fatalf("Deploy() error = %v", err)
	}
	if result.Phase != "deploying" {
		t.Errorf("Phase = %q, want %q", result.Phase, "deploying")
	}
}

func TestVMProvider_Deploy(t *testing.T) {
	p := NewVMProvider("10.0.0.1", "deploy", "/home/deploy/.ssh/id_rsa")
	spec := model.DeploymentSpec{ID: "test-3", Service: "legacy-app", Image: "binary-v2.1.0", Replicas: 1}
	result, err := p.Deploy(context.Background(), spec)
	if err != nil {
		t.Fatalf("Deploy() error = %v", err)
	}
	if result.Phase != "deploying" {
		t.Errorf("Phase = %q, want %q", result.Phase, "deploying")
	}
}

func TestProviderStatus(t *testing.T) {
	tests := []struct {
		name     string
		provider Deployer
		id       string
	}{
		{"ecs", NewECSProvider("cluster", "us-east-1"), "dep-2"},
		{"vm", NewVMProvider("host", "user", "/key"), "dep-3"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, err := tt.provider.Status(context.Background(), tt.id)
			if err != nil {
				t.Fatalf("Status() error = %v", err)
			}
			if status.ID != tt.id {
				t.Errorf("ID = %q, want %q", status.ID, tt.id)
			}
		})
	}
}
