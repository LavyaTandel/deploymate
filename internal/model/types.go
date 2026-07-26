package model

import "time"

type DeploymentSpec struct {
	ID          string            `json:"id"`
	OrgID       string            `json:"org_id"`
	ProjectID   string            `json:"project_id"`
	Environment string            `json:"environment"`
	Service     string            `json:"service"`
	Image       string            `json:"image"`
	Replicas    int               `json:"replicas"`
	Resources   ResourceSpec      `json:"resources"`
	EnvVars     map[string]string `json:"env_vars"`
	PolicyRef   string            `json:"policy_ref"`
	TargetRef   string            `json:"target_ref"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
}

type ResourceSpec struct {
	CPU    string `json:"cpu"`
	Memory string `json:"memory"`
}

type DeploymentResult struct {
	ID          string    `json:"id"`
	Phase       string    `json:"phase"`
	Message     string    `json:"message"`
	Endpoint    string    `json:"endpoint"`
	ImageDigest string    `json:"image_digest"`
	CreatedAt   time.Time `json:"created_at"`
}

type DeploymentStatus struct {
	ID          string    `json:"id"`
	Phase       string    `json:"phase"`
	Message     string    `json:"message"`
	ProgressPct int       `json:"progress_pct"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type PolicyBundle struct {
	Version   string    `json:"version"`
	URL       string    `json:"url"`
	SHA256    string    `json:"sha256"`
	Signature string    `json:"signature"`
	CertPEM   string    `json:"cert_pem"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}
