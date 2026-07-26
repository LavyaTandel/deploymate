package preview

import "time"

// PreviewRequest is a request to create a preview environment
type PreviewRequest struct {
	RepoURL  string `json:"repo_url"`
	PRNumber int    `json:"pr_number"`
	Branch   string `json:"branch"`
	Service  string `json:"service"`
	Image    string `json:"image"`
	CommitSHA string `json:"commit_sha"`
}

// PreviewEnv represents a preview environment
type PreviewEnv struct {
	ID        string    `json:"id"`
	PRNumber  int       `json:"pr_number"`
	URL       string    `json:"url"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at,omitempty"`
}

// Preview status constants
const (
	StatusPending   = "pending"
	StatusCreating  = "creating"
	StatusRunning   = "running"
	StatusDestroying = "destroying"
	StatusDestroyed = "destroyed"
	StatusFailed    = "failed"
)
