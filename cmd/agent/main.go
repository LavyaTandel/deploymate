package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"deploymate/internal/agent"
	"deploymate/internal/model"
)

// config holds all agent configuration read from environment variables.
type config struct {
	EngineURL    string
	EngineToken  string
	AgentID      string
	PollInterval time.Duration
	Kubeconfig   string // empty means in-cluster
}

func loadConfig() config {
	pollSec, err := strconv.Atoi(envOrDefault("POLL_INTERVAL", "10"))
	if err != nil {
		pollSec = 10
	}
	return config{
		EngineURL:    mustEnv("ENGINE_URL"),
		EngineToken:  mustEnv("ENGINE_TOKEN"),
		AgentID:      mustEnv("AGENT_ID"),
		PollInterval: time.Duration(pollSec) * time.Second,
		Kubeconfig:   os.Getenv("KUBECONFIG"),
	}
}

func main() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	cfg := loadConfig()

	k8sClient, err := buildK8sClient(cfg.Kubeconfig)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to build kubernetes client")
	}

	reconciler := agent.NewReconciler(k8sClient)

	agentID := cfg.AgentID

	log.Info().
		Str("agent_id", agentID).
		Str("engine_url", cfg.EngineURL).
		Dur("poll_interval", cfg.PollInterval).
		Msg("starting agent")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Graceful shutdown on SIGINT / SIGTERM.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		log.Info().Msg("received shutdown signal")
		cancel()
	}()

	// Healthcheck server.
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "ok")
	})
	healthSrv := &http.Server{Addr: ":8081", Handler: mux}
	go func() {
		log.Info().Str("addr", ":8081").Msg("healthcheck server listening")
		if err := healthSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error().Err(err).Msg("healthcheck server error")
		}
	}()

	engineClient := newEngineClient(cfg.EngineURL, cfg.EngineToken, agentID)

	pollLoop(ctx, engineClient, reconciler, agentID, cfg.PollInterval)

	// Shutdown health server.
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	if err := healthSrv.Shutdown(shutdownCtx); err != nil {
		log.Error().Err(err).Msg("healthcheck shutdown error")
	}

	log.Info().Msg("agent stopped")
}

// ---------------------------------------------------------------------------
// K8s client builder
// ---------------------------------------------------------------------------

func buildK8sClient(kubeconfig string) (kubernetes.Interface, error) {
	if kubeconfig == "" {
		// In-cluster config: requires ServiceAccount token mounted.
		cfg, err := rest.InClusterConfig()
		if err != nil {
			return nil, fmt.Errorf("in-cluster config: %w", err)
		}
		return kubernetes.NewForConfig(cfg)
	}

	// Local / explicit kubeconfig.
	cfg, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return nil, fmt.Errorf("kubeconfig %s: %w", kubeconfig, err)
	}
	return kubernetes.NewForConfig(cfg)
}

// ---------------------------------------------------------------------------
// Engine API client
// ---------------------------------------------------------------------------

type engineClient struct {
	httpClient *http.Client
	baseURL    string
	token      string
	agentID    string
}

func newEngineClient(baseURL, token, agentID string) *engineClient {
	return &engineClient{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		baseURL:    baseURL,
		token:      token,
		agentID:    agentID,
	}
}

// fetchDesiredState retrieves the desired state for a single deployment.
func (c *engineClient) fetchDesiredState(ctx context.Context, deploymentID string) (*model.DeploymentSpec, error) {
	url := c.baseURL + "/api/v1/deployments/" + deploymentID + "/desired-state"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("X-Agent-ID", c.agentID)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("GET %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GET %s returned %d", url, resp.StatusCode)
	}

	var spec model.DeploymentSpec
	if err := json.NewDecoder(resp.Body).Decode(&spec); err != nil {
		return nil, fmt.Errorf("decode desired state: %w", err)
	}
	return &spec, nil
}

// reportStatus posts a status update back to the engine.
func (c *engineClient) reportStatus(ctx context.Context, status model.DeploymentStatus) error {
	url := c.baseURL + "/api/v1/deployments/" + status.ID + "/status"
	body, err := json.Marshal(status)
	if err != nil {
		return fmt.Errorf("marshal status: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytesReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("X-Agent-ID", c.agentID)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("POST %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("POST %s returned %d", url, resp.StatusCode)
	}
	return nil
}

func bytesReader(b []byte) *bytes.Reader {
	return bytes.NewReader(b)
}

// ---------------------------------------------------------------------------
// Polling loop
// ---------------------------------------------------------------------------

func pollLoop(ctx context.Context, client *engineClient, reconciler *agent.Reconciler, agentID string, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	log.Info().Dur("interval", interval).Msg("poll loop started")

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			reconcile(ctx, client, reconciler, agentID)
		}
	}
}

func reconcile(ctx context.Context, client *engineClient, reconciler *agent.Reconciler, agentID string) {
	// The engine returns a list of deployment IDs this agent owns.
	ids, err := fetchDeploymentIDs(ctx, client, agentID)
	if err != nil {
		log.Error().Err(err).Msg("failed to fetch deployment list")
		return
	}

	for _, id := range ids {
		spec, err := client.fetchDesiredState(ctx, id)
		if err != nil {
			log.Error().Err(err).Str("deployment_id", id).Msg("failed to fetch desired state")
			continue
		}

		if err := reconciler.Reconcile(ctx, *spec); err != nil {
			log.Error().Err(err).Str("deployment_id", id).Msg("reconcile failed")
			continue
		}

		status, err := reconciler.Status(ctx, *spec)
		if err != nil {
			log.Error().Err(err).Str("deployment_id", id).Msg("failed to get status")
			continue
		}
		if err := client.reportStatus(ctx, status); err != nil {
			log.Error().Err(err).Str("deployment_id", id).Msg("failed to report status")
		}
	}
}

func fetchDeploymentIDs(ctx context.Context, client *engineClient, agentID string) ([]string, error) {
	url := client.baseURL + "/api/v1/agents/" + agentID + "/deployments"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+client.token)
	req.Header.Set("X-Agent-ID", agentID)

	resp, err := client.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("GET %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GET %s returned %d", url, resp.StatusCode)
	}

	var result struct {
		IDs []string `json:"ids"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode deployment list: %w", err)
	}
	return result.IDs, nil
}

// ---------------------------------------------------------------------------
// Env helpers
// ---------------------------------------------------------------------------

func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatal().Str("key", key).Msg("required environment variable not set")
	}
	return v
}

func envOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
