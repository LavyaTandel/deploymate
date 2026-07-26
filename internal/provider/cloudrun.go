package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"deploymate/internal/model"
)

// CloudRunProvider implements Deployer for Google Cloud Run using the gcloud
// CLI in push-mode. It delegates all mutating and query operations to the
// gcloud run sub-commands and parses their JSON or text output.
type CloudRunProvider struct {
	project string
	region  string
}

// NewCloudRunProvider returns a CloudRunProvider targeting the given GCP
// project and region. The caller is expected to ensure that gcloud is
// installed, authenticated, and authorised for the target project before
// invoking any Deployer methods.
func NewCloudRunProvider(project, region string) *CloudRunProvider {
	return &CloudRunProvider{
		project: project,
		region:  region,
	}
}

// deployCommandResult captures the subset of gcloud run deploy JSON output
// that we care about.
type deployCommandResult struct {
	ServiceName string `json:"serviceName"`
	RevisionName string `json:"revisionName"`
	URL         string `json:"url"`
	Conditions  []struct {
		Type               string `json:"type"`
		Status             string `json:"status"`
		Message            string `json:"message"`
		LastTransitionTime string `json:"lastTransitionTime"`
	} `json:"conditions"`
}

// revisionResult represents a single entry in `gcloud run revisions list --json`.
type revisionResult struct {
	Metadata struct {
		Name   string            `json:"name"`
		Labels map[string]string `json:"labels"`
	} `json:"metadata"`
	Spec struct {
		Containers []struct {
			Image string `json:"image"`
		} `json:"containers"`
	} `json:"spec"`
	Status struct {
		URL     string `json:"url"`
		Conditions []struct {
			Type   string `json:"type"`
			Status string `json:"status"`
			Message string `json:"message"`
		} `json:"conditions"`
	} `json:"status"`
}

// serviceResult is the JSON shape returned by gcloud run services describe.
type serviceResult struct {
	Status struct {
		URL     string `json:"url"`
		Traffic []struct {
			RevisionName string `json:"revisionName"`
			Percent      int    `json:"percent"`
		} `json:"traffic"`
		Conditions []struct {
			Type    string `json:"type"`
			Status  string `json:"status"`
			Message string `json:"message"`
		} `json:"conditions"`
	} `json:"status"`
}

// gcloudOut runs a gcloud CLI command with the given arguments. The context
// governs cancellation and deadline propagation. On success it returns the
// combined stdout; on failure it returns a wrapped error containing stderr.
func gcloudOut(ctx context.Context, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "gcloud", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		stderrText := strings.TrimSpace(stderr.String())
		if stderrText != "" {
			return "", fmt.Errorf("gcloud %s failed: %w\nstderr: %s", strings.Join(args, " "), err, stderrText)
		}
		return "", fmt.Errorf("gcloud %s failed: %w", strings.Join(args, " "), err)
	}
	return stdout.String(), nil
}

// ---------------------------------------------------------------------------
// Deploy
// ---------------------------------------------------------------------------

func (p *CloudRunProvider) Deploy(ctx context.Context, spec model.DeploymentSpec) (model.DeploymentResult, error) {
	args := p.buildDeployArgs(spec)

	stdout, err := gcloudOut(ctx, args...)
	if err != nil {
		return model.DeploymentResult{}, fmt.Errorf("deploy service %q: %w", spec.Service, err)
	}

	serviceURL := extractServiceURL(stdout)
	if serviceURL == "" {
		serviceURL = fmt.Sprintf("https://%s-%s-%s.a.run.app", spec.Service, p.region, p.project)
	}

	return model.DeploymentResult{
		ID:          spec.ID,
		Phase:       "created",
		Message:     "service deployed successfully",
		Endpoint:    serviceURL,
		ImageDigest: spec.Image,
		CreatedAt:   time.Now().UTC(),
	}, nil
}

// buildDeployArgs constructs the full argument list for `gcloud run deploy`.
// The method is exported via a lowercase name so it can be tested from the
// same package.
func (p *CloudRunProvider) buildDeployArgs(spec model.DeploymentSpec) []string {
	args := []string{
		"run", "deploy", spec.Service,
		"--image", spec.Image,
		"--region", p.region,
		"--project", p.project,
		"--format=json",
	}

	if spec.Resources.CPU != "" {
		args = append(args, "--cpu", spec.Resources.CPU)
	}
	if spec.Resources.Memory != "" {
		args = append(args, "--memory", spec.Resources.Memory)
	}
	if spec.Replicas > 0 {
		args = append(args,
			"--min-instances", fmt.Sprintf("%d", spec.Replicas),
			"--max-instances", fmt.Sprintf("%d", spec.Replicas),
		)
	}

	// Pass environment variables as comma-separated KEY=VALUE pairs.
	if len(spec.EnvVars) > 0 {
		envPairs := make([]string, 0, len(spec.EnvVars))
		for k, v := range spec.EnvVars {
			envPairs = append(envPairs, k+"="+v)
		}
		args = append(args, "--env-vars", strings.Join(envPairs, ","))
	}

	// Deploy without rolling traffic to the new revision immediately; the
	// release pipeline is responsible for promoting traffic after validation.
	args = append(args, "--no-traffic", "--quiet")
	return args
}

// ---------------------------------------------------------------------------
// Rollback
// ---------------------------------------------------------------------------

func (p *CloudRunProvider) Rollback(ctx context.Context, id string) error {
	serviceName := extractServiceName(id)

	// 1. Fetch all revisions, most recently created first.
	revisionJSON, err := gcloudOut(ctx,
		"run", "revisions", "list",
		"--service", serviceName,
		"--region", p.region,
		"--project", p.project,
		"--sort-by=CREATED",
		"--format=json",
		"--quiet",
	)
	if err != nil {
		return fmt.Errorf("list revisions for service %q: %w", serviceName, err)
	}

	var revisions []revisionResult
	if err := json.Unmarshal([]byte(revisionJSON), &revisions); err != nil {
		return fmt.Errorf("parse revision list for %q: %w", serviceName, err)
	}
	if len(revisions) < 2 {
		return fmt.Errorf("rollback not possible for %q: need at least 2 revisions, got %d", serviceName, len(revisions))
	}

	// 2. The second revision in the list is the previous one (index 0 is current).
	targetRevision := revisions[1].Metadata.Name

	// 3. Point traffic at the previous revision.
	_, err = gcloudOut(ctx,
		"run", "services", "update-traffic", serviceName,
		"--region", p.region,
		"--project", p.project,
		"--to-revisions", targetRevision+"=100",
		"--quiet",
	)
	if err != nil {
		return fmt.Errorf("rollback service %q to revision %q: %w", serviceName, targetRevision, err)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Status
// ---------------------------------------------------------------------------

func (p *CloudRunProvider) Status(ctx context.Context, id string) (model.DeploymentStatus, error) {
	serviceName := extractServiceName(id)

	describeJSON, err := gcloudOut(ctx,
		"run", "services", "describe", serviceName,
		"--region", p.region,
		"--project", p.project,
		"--format=json",
		"--quiet",
	)
	if err != nil {
		return model.DeploymentStatus{}, fmt.Errorf("describe service %q: %w", serviceName, err)
	}

	var svc serviceResult
	if err := json.Unmarshal([]byte(describeJSON), &svc); err != nil {
		return model.DeploymentStatus{}, fmt.Errorf("parse service describe for %q: %w", serviceName, err)
	}

	phase, message, progress := classifyServiceConditions(svc.Status.Conditions)

	return model.DeploymentStatus{
		ID:          id,
		Phase:       phase,
		Message:     message,
		ProgressPct: progress,
		UpdatedAt:   time.Now().UTC(),
	}, nil
}

// classifyServiceConditions maps Cloud Run condition arrays to a single
// deployment phase, human-readable message, and a progress percentage.
func classifyServiceConditions(conditions []struct {
	Type    string `json:"type"`
	Status  string `json:"status"`
	Message string `json:"message"`
}) (phase, message string, progress int) {
	for _, c := range conditions {
		if c.Type == "Ready" && c.Status == "True" {
			return "served", "service is ready and serving traffic", 100
		}
		if c.Type == "Ready" && c.Status == "False" {
			return "failed", c.Message, 0
		}
		if c.Type == "ConfigMapsReady" && c.Status == "False" {
			return "failed", fmt.Sprintf("configmap error: %s", c.Message), 0
		}
	}
	return "pending", "service is deploying", 50
}

// ---------------------------------------------------------------------------
// Destroy
// ---------------------------------------------------------------------------

func (p *CloudRunProvider) Destroy(ctx context.Context, id string) error {
	serviceName := extractServiceName(id)

	_, err := gcloudOut(ctx,
		"run", "services", "delete", serviceName,
		"--region", p.region,
		"--project", p.project,
		"--quiet",
	)
	if err != nil {
		return fmt.Errorf("delete service %q: %w", serviceName, err)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// extractServiceURL attempts to pull the service URL from gcloud stdout.
// Cloud Run deploy may surface the URL as a plain text line or inside JSON.
func extractServiceURL(output string) string {
	// Try the common plain-text pattern first: "Service [name] has been
	// deployed and is serving 100 percent of traffic.\n\nService URL: https://..."
	re := regexp.MustCompile(`(?i)Service URL:\s*(https?://\S+)`)
	if m := re.FindStringSubmatch(output); len(m) > 1 {
		return strings.TrimSpace(m[1])
	}

	// Fallback: the URL may be embedded in JSON output.
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(output), &parsed); err == nil {
		if url, ok := parsed["url"].(string); ok && url != "" {
			return url
		}
		if status, ok := parsed["status"].(map[string]interface{}); ok {
			if url, ok := status["url"].(string); ok && url != "" {
				return url
			}
		}
	}

	return ""
}

// extractServiceName derives the Cloud Run service name from a deployment ID.
// DeployMate IDs follow the convention "<service>-<orgID>-<projectID>-<hash>".
// We take the first segment as the canonical service name.
func extractServiceName(id string) string {
	if parts := strings.SplitN(id, "-", 2); len(parts) > 0 {
		return parts[0]
	}
	return id
}
