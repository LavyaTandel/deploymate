package model

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestDeploymentSpec_JSONRoundTrip(t *testing.T) {
	fixedTime := time.Date(2026, 7, 24, 10, 30, 0, 0, time.UTC)

	tests := []struct {
		name  string
		spec  DeploymentSpec
		check func(t *testing.T, got DeploymentSpec)
	}{
		{
			name: "full spec round-trip",
			spec: DeploymentSpec{
				ID:          "dep-001",
				OrgID:       "org-acme",
				ProjectID:   "proj-web",
				Environment: "production",
				Service:     "api-gateway",
				Image:       "gcr.io/acme/api-gateway:v2.1.0",
				Replicas:    5,
				Resources: ResourceSpec{
					CPU:    "1000m",
					Memory: "512Mi",
				},
				EnvVars: map[string]string{
					"LOG_LEVEL": "info",
					"DATABASE_URL": "postgres://db:5432/app",
				},
				PolicyRef: "policy-strict",
				TargetRef: "target-gke",
				CreatedAt: fixedTime,
				UpdatedAt: fixedTime,
			},
			check: func(t *testing.T, got DeploymentSpec) {
				if got.ID != "dep-001" {
					t.Errorf("ID = %q, want %q", got.ID, "dep-001")
				}
				if got.OrgID != "org-acme" {
					t.Errorf("OrgID = %q, want %q", got.OrgID, "org-acme")
				}
				if got.ProjectID != "proj-web" {
					t.Errorf("ProjectID = %q, want %q", got.ProjectID, "proj-web")
				}
				if got.Environment != "production" {
					t.Errorf("Environment = %q, want %q", got.Environment, "production")
				}
				if got.Service != "api-gateway" {
					t.Errorf("Service = %q, want %q", got.Service, "api-gateway")
				}
				if got.Image != "gcr.io/acme/api-gateway:v2.1.0" {
					t.Errorf("Image = %q, want %q", got.Image, "gcr.io/acme/api-gateway:v2.1.0")
				}
				if got.Replicas != 5 {
					t.Errorf("Replicas = %d, want %d", got.Replicas, 5)
				}
				if got.Resources.CPU != "1000m" {
					t.Errorf("Resources.CPU = %q, want %q", got.Resources.CPU, "1000m")
				}
				if got.Resources.Memory != "512Mi" {
					t.Errorf("Resources.Memory = %q, want %q", got.Resources.Memory, "512Mi")
				}
				if got.EnvVars["LOG_LEVEL"] != "info" {
					t.Errorf("EnvVars[LOG_LEVEL] = %q, want %q", got.EnvVars["LOG_LEVEL"], "info")
				}
				if got.EnvVars["DATABASE_URL"] != "postgres://db:5432/app" {
					t.Errorf("EnvVars[DATABASE_URL] = %q, want %q", got.EnvVars["DATABASE_URL"], "postgres://db:5432/app")
				}
				if got.PolicyRef != "policy-strict" {
					t.Errorf("PolicyRef = %q, want %q", got.PolicyRef, "policy-strict")
				}
				if got.TargetRef != "target-gke" {
					t.Errorf("TargetRef = %q, want %q", got.TargetRef, "target-gke")
				}
				if !got.CreatedAt.Equal(fixedTime) {
					t.Errorf("CreatedAt = %v, want %v", got.CreatedAt, fixedTime)
				}
				if !got.UpdatedAt.Equal(fixedTime) {
					t.Errorf("UpdatedAt = %v, want %v", got.UpdatedAt, fixedTime)
				}
			},
		},
		{
			name: "minimal spec round-trip",
			spec: DeploymentSpec{
				ID:    "dep-min",
				OrgID: "org-min",
			},
			check: func(t *testing.T, got DeploymentSpec) {
				if got.ID != "dep-min" {
					t.Errorf("ID = %q, want %q", got.ID, "dep-min")
				}
				if got.OrgID != "org-min" {
					t.Errorf("OrgID = %q, want %q", got.OrgID, "org-min")
				}
				if got.Replicas != 0 {
					t.Errorf("Replicas = %d, want 0", got.Replicas)
				}
				if got.EnvVars != nil {
					t.Errorf("EnvVars = %v, want nil", got.EnvVars)
				}
			},
		},
		{
			name: "spec with empty env vars",
			spec: DeploymentSpec{
				ID:      "dep-empty",
				OrgID:   "org-empty",
				EnvVars: map[string]string{},
			},
			check: func(t *testing.T, got DeploymentSpec) {
				if got.EnvVars == nil {
					t.Error("EnvVars is nil after round-trip, want empty map")
				}
				if len(got.EnvVars) != 0 {
					t.Errorf("EnvVars has %d entries, want 0", len(got.EnvVars))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.spec)
			if err != nil {
				t.Fatalf("json.Marshal() error = %v", err)
			}

			var got DeploymentSpec
			if err := json.Unmarshal(data, &got); err != nil {
				t.Fatalf("json.Unmarshal() error = %v", err)
			}

			tt.check(t, got)
		})
	}
}

func TestDeploymentSpec_JSONKeys(t *testing.T) {
	spec := DeploymentSpec{
		ID:          "dep-keys",
		OrgID:       "org-keys",
		ProjectID:   "proj-keys",
		Environment: "staging",
		Service:     "worker",
		Image:       "img:v1",
		Replicas:    2,
		Resources: ResourceSpec{
			CPU:    "250m",
			Memory: "128Mi",
		},
		PolicyRef: "pol",
		TargetRef: "tgt",
	}

	data, err := json.Marshal(spec)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	jsonStr := string(data)

	// Verify expected JSON keys are present.
	expectedKeys := []string{
		`"id"`,
		`"org_id"`,
		`"project_id"`,
		`"environment"`,
		`"service"`,
		`"image"`,
		`"replicas"`,
		`"resources"`,
		`"cpu"`,
		`"memory"`,
		`"policy_ref"`,
		`"target_ref"`,
	}

	for _, key := range expectedKeys {
		if !strings.Contains(jsonStr, key) {
			t.Errorf("JSON output missing key %s", key)
		}
	}
}

func TestDeploymentResult_JSONRoundTrip(t *testing.T) {
	fixedTime := time.Date(2026, 7, 24, 10, 30, 0, 0, time.UTC)

	tests := []struct {
		name   string
		result DeploymentResult
	}{
		{
			name: "full result round-trip",
			result: DeploymentResult{
				ID:          "res-001",
				Phase:       "completed",
				Message:     "Deployed successfully",
				Endpoint:    "https://api.example.com",
				ImageDigest: "sha256:abc123",
				CreatedAt:   fixedTime,
			},
		},
		{
			name: "minimal result",
			result: DeploymentResult{
				ID:    "res-min",
				Phase: "pending",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.result)
			if err != nil {
				t.Fatalf("json.Marshal() error = %v", err)
			}

			var got DeploymentResult
			if err := json.Unmarshal(data, &got); err != nil {
				t.Fatalf("json.Unmarshal() error = %v", err)
			}

			if got.ID != tt.result.ID {
				t.Errorf("ID = %q, want %q", got.ID, tt.result.ID)
			}
			if got.Phase != tt.result.Phase {
				t.Errorf("Phase = %q, want %q", got.Phase, tt.result.Phase)
			}
			if got.Message != tt.result.Message {
				t.Errorf("Message = %q, want %q", got.Message, tt.result.Message)
			}
		})
	}
}

func TestDeploymentStatus_JSONRoundTrip(t *testing.T) {
	fixedTime := time.Date(2026, 7, 24, 10, 30, 0, 0, time.UTC)

	tests := []struct {
		name   string
		status DeploymentStatus
	}{
		{
			name: "full status round-trip",
			status: DeploymentStatus{
				ID:          "st-001",
				Phase:       "in_progress",
				Message:     "Rolling update",
				ProgressPct: 67,
				UpdatedAt:   fixedTime,
			},
		},
		{
			name: "zero progress",
			status: DeploymentStatus{
				ID:    "st-zero",
				Phase: "pending",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.status)
			if err != nil {
				t.Fatalf("json.Marshal() error = %v", err)
			}

			var got DeploymentStatus
			if err := json.Unmarshal(data, &got); err != nil {
				t.Fatalf("json.Unmarshal() error = %v", err)
			}

			if got.ID != tt.status.ID {
				t.Errorf("ID = %q, want %q", got.ID, tt.status.ID)
			}
			if got.Phase != tt.status.Phase {
				t.Errorf("Phase = %q, want %q", got.Phase, tt.status.Phase)
			}
			if got.ProgressPct != tt.status.ProgressPct {
				t.Errorf("ProgressPct = %d, want %d", got.ProgressPct, tt.status.ProgressPct)
			}
		})
	}
}

func TestPolicyBundle_JSONRoundTrip(t *testing.T) {
	fixedTime := time.Date(2026, 7, 24, 10, 30, 0, 0, time.UTC)

	tests := []struct {
		name   string
		bundle PolicyBundle
	}{
		{
			name: "full bundle round-trip",
			bundle: PolicyBundle{
				Version:   "v1.2.0",
				URL:       "https://policies.example.com/bundle-v1.2.0.tar.gz",
				SHA256:    "abc123def456",
				Signature: "sig-base64-data",
				CertPEM:   "-----BEGIN CERTIFICATE-----\ntest\n-----END CERTIFICATE-----",
				ExpiresAt: fixedTime.Add(30 * 24 * time.Hour),
				CreatedAt: fixedTime,
			},
		},
		{
			name: "minimal bundle",
			bundle: PolicyBundle{
				Version: "v0.1.0",
				URL:     "https://example.com/bundle",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.bundle)
			if err != nil {
				t.Fatalf("json.Marshal() error = %v", err)
			}

			var got PolicyBundle
			if err := json.Unmarshal(data, &got); err != nil {
				t.Fatalf("json.Unmarshal() error = %v", err)
			}

			if got.Version != tt.bundle.Version {
				t.Errorf("Version = %q, want %q", got.Version, tt.bundle.Version)
			}
			if got.URL != tt.bundle.URL {
				t.Errorf("URL = %q, want %q", got.URL, tt.bundle.URL)
			}
			if got.SHA256 != tt.bundle.SHA256 {
				t.Errorf("SHA256 = %q, want %q", got.SHA256, tt.bundle.SHA256)
			}
		})
	}
}

func TestResourceSpec_JSONRoundTrip(t *testing.T) {
	tests := []struct {
		name string
		res  ResourceSpec
	}{
		{
			name: "standard resources",
			res:  ResourceSpec{CPU: "500m", Memory: "256Mi"},
		},
		{
			name: "large resources",
			res:  ResourceSpec{CPU: "4000m", Memory: "8Gi"},
		},
		{
			name: "empty resources",
			res:  ResourceSpec{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.res)
			if err != nil {
				t.Fatalf("json.Marshal() error = %v", err)
			}

			var got ResourceSpec
			if err := json.Unmarshal(data, &got); err != nil {
				t.Fatalf("json.Unmarshal() error = %v", err)
			}

			if got.CPU != tt.res.CPU {
				t.Errorf("CPU = %q, want %q", got.CPU, tt.res.CPU)
			}
			if got.Memory != tt.res.Memory {
				t.Errorf("Memory = %q, want %q", got.Memory, tt.res.Memory)
			}
		})
	}
}
